package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

const (
	// CallbackTimeout is how long to wait for the OAuth callback.
	CallbackTimeout = 2 * time.Minute

	// tokenRefreshWindow is how far before expiry to trigger a refresh.
	tokenRefreshWindow = 5 * time.Minute
)

// TokenFile is the JSON structure persisted to disk for MCP OAuth tokens.
type TokenFile struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
}

// tokenPath returns the path to the MCP token file:
// ~/.config/ntn/mcp-token.json (or the platform equivalent via os.UserConfigDir).
func tokenPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(configDir, "ntn", "mcp-token.json"), nil
}

// Login performs the full OAuth 2.0 PKCE flow against the Notion MCP server.
//
// 1. Starts a local HTTP callback server on a random port.
// 2. Generates PKCE code_verifier / code_challenge.
// 3. Uses mcp-go's OAuthHandler to discover endpoints and register the client.
// 4. Opens the browser to the authorization URL.
// 5. Waits for the callback, exchanges the code for a token.
// 6. Persists the token to disk.
func Login(ctx context.Context) error {
	// Start local callback listener on a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build an OAuthHandler via mcp-go's transport layer.
	oauthCfg := transport.OAuthConfig{
		RedirectURI: redirectURI,
		PKCEEnabled: true,
		TokenStore:  transport.NewMemoryTokenStore(),
	}
	handler := transport.NewOAuthHandler(oauthCfg)
	handler.SetBaseURL("https://mcp.notion.com")

	// Dynamic client registration (Notion supports this).
	if err := handler.RegisterClient(ctx, clientName); err != nil {
		return fmt.Errorf("dynamic client registration failed: %w", err)
	}

	// Generate PKCE values and state.
	codeVerifier, err := mcpclient.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := mcpclient.GenerateCodeChallenge(codeVerifier)

	state, err := mcpclient.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	authURL, err := handler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("failed to build authorization URL: %w", err)
	}

	// Channel to receive the callback parameters.
	type callbackResult struct {
		code  string
		state string
		err   error
	}
	callbackCh := make(chan callbackResult, 1)

	// HTTP handler for the OAuth callback.
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = errParam
			}
			callbackCh <- callbackResult{err: fmt.Errorf("authorization failed: %s", desc)}
			http.Error(w, desc, http.StatusBadRequest)
			return
		}

		code := q.Get("code")
		if code == "" {
			callbackCh <- callbackResult{err: fmt.Errorf("no authorization code received")}
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		callbackCh <- callbackResult{code: code, state: q.Get("state")}

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>Authorization successful</h1><p>You can close this window.</p></body></html>"))
	})

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	// Open browser.
	fmt.Fprintf(os.Stderr, "Opening browser for Notion MCP authorization...\n")
	if err := openBrowserURL(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser. Please visit:\n%s\n\n", authURL)
	}
	fmt.Fprintln(os.Stderr, "Waiting for authorization...")

	// Wait for callback or timeout.
	select {
	case result := <-callbackCh:
		if result.err != nil {
			return result.err
		}

		// Exchange code for token via the OAuthHandler.
		if err := handler.ProcessAuthorizationResponse(ctx, result.code, result.state, codeVerifier); err != nil {
			return fmt.Errorf("token exchange failed: %w", err)
		}

		// Retrieve the token from the in-memory store and persist to disk.
		tok, err := oauthCfg.TokenStore.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve token after exchange: %w", err)
		}

		tf := TokenFile{
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			TokenType:    tok.TokenType,
			ExpiresAt:    tok.ExpiresAt,
			ClientID:     handler.GetClientID(),
		}
		if err := saveTokenFile(&tf); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Successfully authenticated with Notion MCP!")
		return nil

	case <-time.After(CallbackTimeout):
		return fmt.Errorf("authorization timed out after %s", CallbackTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// LoadToken reads the persisted MCP token from disk.
// If the token is expired (or within the refresh window), it attempts an
// automatic refresh using the stored refresh token and client ID.
func LoadToken() (*TokenFile, error) {
	p, err := tokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated with Notion MCP — run 'ntn mcp login' first")
		}
		return nil, fmt.Errorf("failed to read MCP token: %w", err)
	}

	var tf TokenFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("failed to parse MCP token file: %w", err)
	}

	// Check if token needs refresh.
	if !tf.ExpiresAt.IsZero() && time.Until(tf.ExpiresAt) < tokenRefreshWindow {
		if tf.RefreshToken != "" && tf.ClientID != "" {
			refreshed, err := refreshToken(context.Background(), &tf)
			if err == nil {
				return refreshed, nil
			}
			// If refresh fails but token hasn't actually expired yet, return it anyway.
			if time.Now().Before(tf.ExpiresAt) {
				return &tf, nil
			}
			return nil, fmt.Errorf("MCP token expired and refresh failed: %w", err)
		}
		if time.Now().After(tf.ExpiresAt) {
			return nil, fmt.Errorf("MCP token expired — run 'ntn mcp login' to re-authenticate")
		}
	}

	return &tf, nil
}

// refreshToken uses the stored refresh token and client ID to obtain a new
// access token from the Notion MCP OAuth server.
func refreshToken(ctx context.Context, tf *TokenFile) (*TokenFile, error) {
	oauthCfg := transport.OAuthConfig{
		ClientID:    tf.ClientID,
		PKCEEnabled: true,
		TokenStore:  transport.NewMemoryTokenStore(),
	}
	handler := transport.NewOAuthHandler(oauthCfg)
	handler.SetBaseURL("https://mcp.notion.com")

	newTok, err := handler.RefreshToken(ctx, tf.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	refreshed := &TokenFile{
		AccessToken:  newTok.AccessToken,
		RefreshToken: newTok.RefreshToken,
		TokenType:    newTok.TokenType,
		ExpiresAt:    newTok.ExpiresAt,
		ClientID:     tf.ClientID,
	}

	if err := saveTokenFile(refreshed); err != nil {
		return nil, err
	}

	return refreshed, nil
}

// Logout removes the persisted MCP token file.
func Logout() error {
	p, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return nil // already logged out
		}
		return fmt.Errorf("failed to remove MCP token: %w", err)
	}
	return nil
}

// Status returns a human-readable map of the current MCP auth state.
func Status() (map[string]interface{}, error) {
	tf, err := LoadToken()
	if err != nil {
		return map[string]interface{}{
			"authenticated": false,
			"error":         err.Error(),
		}, nil
	}

	result := map[string]interface{}{
		"authenticated": true,
		"token_type":    tf.TokenType,
	}

	if !tf.ExpiresAt.IsZero() {
		result["expires_at"] = tf.ExpiresAt.Format(time.RFC3339)
		remaining := time.Until(tf.ExpiresAt)
		if remaining > 0 {
			result["expires_in"] = remaining.Truncate(time.Second).String()
		} else {
			result["expired"] = true
		}
	}

	if tf.ClientID != "" {
		result["client_id"] = tf.ClientID
	}

	return result, nil
}

// saveTokenFile writes the token to disk with restrictive permissions.
func saveTokenFile(tf *TokenFile) error {
	p, err := tokenPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("failed to write MCP token: %w", err)
	}

	return nil
}

// openBrowserURL opens a URL in the user's default browser.
func openBrowserURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
