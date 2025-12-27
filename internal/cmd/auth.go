package cmd

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/salmonumbrella/notion-cli/internal/auth"
)

const (
	// ProxyURL is the OAuth proxy URL
	ProxyURL = "https://notion-cli.fly.dev"
	// CallbackTimeout is how long to wait for the OAuth callback
	CallbackTimeout = 2 * time.Minute
)

//go:embed templates/auth_success.html
var notionSuccessPage string

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Notion API authentication",
		Long:  `Manage authentication tokens for the Notion API. Tokens are securely stored in the system keyring.`,
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthAddTokenCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthLogoutCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with Notion OAuth",
		Long: `Authenticate with Notion using OAuth.

This will open your browser to authorize notion-cli with your Notion account.
The authorization uses OAuth, allowing you to act as yourself (not as a bot).

If browser-based authentication fails, you can use 'ntn auth add-token'
to manually enter an integration token.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOAuthLogin(cmd.Context(), noBrowser)
		},
	}

	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not auto-open browser; print auth URL instead")
	return cmd
}

func runOAuthLogin(ctx context.Context, noBrowser bool) error {
	noBrowser = noBrowser || envTruthy("NOTION_NO_BROWSER") || envTruthy("NO_BROWSER")

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Start local callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	// Channel to receive the token
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Set up callback handler
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Handle both POST (secure, from proxy) and GET (fallback)
		var stateParam, token, errParam, errDesc string

		if r.Method == http.MethodPost {
			// Parse POST form data (secure - token not in URL/history)
			if err := r.ParseForm(); err != nil {
				errCh <- fmt.Errorf("failed to parse form: %w", err)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			stateParam = r.PostFormValue("state")
			token = r.PostFormValue("token")
			errParam = r.PostFormValue("error")
			errDesc = r.PostFormValue("error_description")
		} else {
			// GET fallback (for manual flows)
			stateParam = r.URL.Query().Get("state")
			token = r.URL.Query().Get("token")
			errParam = r.URL.Query().Get("error")
			errDesc = r.URL.Query().Get("error_description")
		}

		// Verify state
		if stateParam != state {
			errCh <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Check for errors
		if errParam != "" {
			if errDesc == "" {
				errDesc = errParam
			}
			errCh <- fmt.Errorf("authorization failed: %s", errDesc)
			http.Error(w, errDesc, http.StatusBadRequest)
			return
		}

		// Get token
		if token == "" {
			errCh <- fmt.Errorf("no token received")
			http.Error(w, "No token received", http.StatusBadRequest)
			return
		}

		tokenCh <- token

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(notionSuccessPage))
	})

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	// Build OAuth URL
	authURL := fmt.Sprintf("%s/auth/start?redirect_uri=%s&state=%s",
		ProxyURL,
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)

	if noBrowser {
		_, _ = fmt.Fprintln(stderrFromContext(ctx), "Browser auto-open disabled.")
		_, _ = fmt.Fprintf(stderrFromContext(ctx), "Visit this URL to authorize notion-cli:\n%s\n\n", authURL)
	} else {
		// Open browser
		_, _ = fmt.Fprintln(stderrFromContext(ctx), "Opening browser to authorize notion-cli...")
		_, _ = fmt.Fprintln(stderrFromContext(ctx))
		if err := openBrowser(authURL); err != nil {
			_, _ = fmt.Fprintf(stderrFromContext(ctx), "Could not open browser. Please visit:\n%s\n\n", authURL)
		}
	}

	_, _ = fmt.Fprintln(stderrFromContext(ctx), "Waiting for authorization...")

	// Wait for token or timeout
	select {
	case token := <-tokenCh:
		return storeOAuthToken(ctx, token)
	case err := <-errCh:
		return err
	case <-time.After(CallbackTimeout):
		_, _ = fmt.Fprintln(stderrFromContext(ctx))
		_, _ = fmt.Fprintln(stderrFromContext(ctx), "Timed out waiting for authorization.")
		_, _ = fmt.Fprintln(stderrFromContext(ctx), "If you completed authorization, you can paste the token manually.")
		_, _ = fmt.Fprint(stderrFromContext(ctx), "Enter token (or press Enter to cancel): ")

		tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(stderrFromContext(ctx))
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}

		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			return fmt.Errorf("authorization timed out")
		}

		return storeOAuthToken(ctx, token)
	}
}

func envTruthy(name string) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func storeOAuthToken(ctx context.Context, token string) error {
	// Validate token format
	if !isValidNotionTokenFormat(token) {
		return fmt.Errorf("invalid token format")
	}

	// Store token with source metadata
	if err := auth.StoreTokenWithSource(token, "oauth"); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Store auth source for backwards compatibility
	if err := auth.StoreAuthSource(auth.SourceOAuth); err != nil {
		// Non-fatal, continue
		_, _ = fmt.Fprintf(stderrFromContext(ctx), "Warning: could not store auth source: %v\n", err)
	}

	// Fetch and store user info
	client := NewNotionClient(ctx, token)
	user, err := client.GetSelf(ctx)

	var userInfo *auth.UserInfo
	if err == nil && user != nil {
		userInfo = &auth.UserInfo{
			ID:        user.ID,
			Type:      user.Type,
			Name:      user.Name,
			AvatarURL: user.AvatarURL,
		}
		if user.Person != nil {
			userInfo.Email = user.Person.Email
		}
		_ = auth.StoreUserInfo(userInfo)
	}

	// Print success
	_, _ = fmt.Fprintln(stderrFromContext(ctx))
	_, _ = fmt.Fprintln(stderrFromContext(ctx), "Successfully logged in!")
	if userInfo != nil {
		if userInfo.Email != "" {
			_, _ = fmt.Fprintf(stderrFromContext(ctx), "Logged in as: %s (%s)\n", userInfo.Name, userInfo.Email)
		} else {
			_, _ = fmt.Fprintf(stderrFromContext(ctx), "Logged in as: %s\n", userInfo.Name)
		}
	}

	return nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func newAuthAddTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-token",
		Short: "Add Notion API token manually",
		Long: `Store a Notion internal integration token in the system keyring.

Use this command for internal integrations (bot tokens). For personal OAuth
authentication, use 'ntn auth login' instead.

The token will be stored securely using your operating system's keyring:
  - macOS: Keychain
  - Linux: Secret Service (GNOME Keyring, KWallet), with encrypted file fallback
  - Windows: Credential Manager

You will be prompted to enter your token interactively. Input will be hidden.

Get your internal integration token from: https://www.notion.so/my-integrations`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Prompt for token (use stderr for prompts so stdout is clean)
			_, _ = fmt.Fprint(stderrFromContext(ctx), "Enter your Notion API token: ")

			// Read token with hidden input
			tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			_, _ = fmt.Fprintln(stderrFromContext(ctx)) // Print newline after hidden input

			// Trim whitespace
			token := strings.TrimSpace(string(tokenBytes))
			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			// Validate token format
			if !isValidNotionTokenFormat(token) {
				return fmt.Errorf("invalid token format: Notion tokens should start with 'secret_' or 'ntn_'")
			}

			// Store token in keyring with source metadata
			if err := auth.StoreTokenWithSource(token, "internal"); err != nil {
				return fmt.Errorf("failed to store token: %w", err)
			}

			// Store auth source as internal integration for backwards compatibility
			if err := auth.StoreAuthSource(auth.SourceInternal); err != nil {
				// Non-fatal
				_, _ = fmt.Fprintf(stderrFromContext(ctx), "Warning: could not store auth source: %v\n", err)
			}

			// Fetch and store user info
			client := NewNotionClient(ctx, token)
			user, err := client.GetSelf(ctx)

			var userInfo *auth.UserInfo
			if err == nil && user != nil {
				userInfo = &auth.UserInfo{
					ID:        user.ID,
					Type:      user.Type,
					Name:      user.Name,
					AvatarURL: user.AvatarURL,
				}
				if user.Person != nil {
					userInfo.Email = user.Person.Email
				}
				_ = auth.StoreUserInfo(userInfo) // Best effort, don't fail if this fails
			}

			// Print success message
			printer := printerForContext(ctx)
			result := map[string]interface{}{
				"status":  "success",
				"message": "Token stored successfully in keyring",
				"source":  "internal_integration",
			}
			if userInfo != nil {
				result["user"] = map[string]interface{}{
					"id":   userInfo.ID,
					"type": userInfo.Type,
					"name": userInfo.Name,
				}
				if userInfo.Email != "" {
					result["user"].(map[string]interface{})["email"] = userInfo.Email
				}
			}

			return printer.Print(ctx, result)
		},
	}
}

// isValidNotionTokenFormat checks if the token has a valid Notion token prefix
func isValidNotionTokenFormat(token string) bool {
	return strings.HasPrefix(token, "secret_") || strings.HasPrefix(token, "ntn_")
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long: `Display whether a Notion API token is configured.

Shows:
  - Whether you're authenticated
  - Authentication type (OAuth or Internal Integration)
  - Token source (keyring or environment variable)
  - Token age and rotation warnings
  - User information if available

Does not display the actual token value.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Check if token is available
			hasToken := auth.HasToken()

			// Determine token source and auth type
			var tokenSource string
			var authType string
			fromEnvVar := false

			if hasToken {
				if token := os.Getenv(auth.EnvVarName); token != "" {
					tokenSource = "environment variable (" + auth.EnvVarName + ")"
					fromEnvVar = true
					// Can't determine auth type for env var tokens
					authType = "unknown"
				} else {
					tokenSource = "system keyring"
					// Get auth type from keyring (only meaningful for keyring tokens)
					authSource := auth.GetAuthSource()
					switch authSource {
					case auth.SourceOAuth:
						authType = "oauth"
					case auth.SourceInternal:
						authType = "internal_integration"
					default:
						authType = "unknown"
					}
				}
			} else {
				tokenSource = "none"
			}

			// Prepare output
			result := map[string]interface{}{
				"authenticated": hasToken,
				"token_source":  tokenSource,
			}

			if hasToken && !fromEnvVar {
				result["auth_type"] = authType
			}

			// Add token age info if available
			if hasToken && !fromEnvVar {
				if metadata, err := auth.GetTokenMetadata(); err == nil && metadata != nil {
					if !metadata.CreatedAt.IsZero() {
						age := auth.TokenAgeDays(metadata.CreatedAt)
						result["token_age_days"] = age
						result["token_created_at"] = metadata.CreatedAt.Format("2006-01-02")
						result["token_age"] = auth.FormatTokenAge(metadata.CreatedAt)

						// Add warning if token is old
						if auth.IsTokenExpiringSoon(metadata.CreatedAt) {
							result["warning"] = fmt.Sprintf("Token is %d days old. Consider rotating for security.", age)
						}
					}
				}
			}

			// Add user info if available
			if userInfo, _ := auth.GetUserInfo(); userInfo != nil {
				user := map[string]interface{}{
					"id":   userInfo.ID,
					"type": userInfo.Type,
					"name": userInfo.Name,
				}
				if userInfo.Email != "" {
					user["email"] = userInfo.Email
				}
				result["user"] = user
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and remove stored credentials",
		Long: `Remove the stored Notion credentials from the system keyring.

This removes:
  - OAuth or integration token
  - User information
  - Authentication source

Note: If you have set the NOTION_TOKEN environment variable,
you will need to unset it separately.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Delete token from keyring
			if err := auth.DeleteToken(); err != nil {
				return fmt.Errorf("failed to remove token: %w", err)
			}

			// Print success message
			printer := printerForContext(ctx)
			result := map[string]interface{}{
				"status":  "success",
				"message": "Logged out successfully",
			}

			return printer.Print(ctx, result)
		},
	}
}
