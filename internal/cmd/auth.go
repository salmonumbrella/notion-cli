package cmd

import (
	"context"
	"crypto/rand"
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

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	// ProxyURL is the OAuth proxy URL
	ProxyURL = "https://notion-cli.fly.dev"
	// CallbackTimeout is how long to wait for the OAuth callback
	CallbackTimeout = 2 * time.Minute
)

// notionSuccessPage is the HTML shown after successful OAuth authorization
const notionSuccessPage = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authorization Successful</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }

        @keyframes cursorBlink {
            0%, 50% { opacity: 1; }
            51%, 100% { opacity: 0; }
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, "Apple Color Emoji", Arial, sans-serif;
            background: #ffffff;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .container {
            text-align: center;
            animation: fadeIn 0.6s ease-out;
        }

        .logo-wrapper {
            position: relative;
            display: inline-block;
            margin-bottom: 32px;
        }

        .logo { width: 80px; height: 80px; }

        .checkmark {
            position: absolute;
            top: -8px;
            right: -12px;
            width: 32px;
            height: 32px;
        }

        .checkmark circle { fill: #00B900; }
        .checkmark path {
            stroke: white;
            stroke-width: 4;
            fill: none;
            stroke-linecap: round;
            stroke-linejoin: round;
        }

        h1 {
            font-size: 24px;
            font-weight: 600;
            color: #37352f;
            margin-bottom: 12px;
            letter-spacing: -0.03em;
        }

        p { font-size: 16px; color: #787774; line-height: 1.5; }

        .terminal-wrapper {
            display: inline-block;
            position: relative;
        }

        .terminal {
            display: inline-block;
            background: #f7f6f3;
            border: 1px solid #e3e2e0;
            border-radius: 4px;
            padding: 2px 8px;
            font-family: "SFMono-Regular", Menlo, Consolas, monospace;
            font-size: 14px;
            color: #37352f;
        }

        .cursor {
            display: inline-block;
            width: 3px;
            height: 17px;
            background: #00B900;
            margin-left: 2px;
            vertical-align: text-bottom;
            animation: cursorBlink 1.2s step-end infinite;
        }

        .sharpie-underlines {
            position: absolute;
            left: 0;
            right: 0;
            bottom: 0px;
            height: 6px;
        }

        .sharpie-underlines svg {
            width: 100%;
            height: 100%;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo-wrapper">
            <svg class="logo" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                <path d="M6.017 4.313l55.333 -4.087c6.797 -0.583 8.543 -0.19 12.817 2.917l17.663 12.443c2.913 2.14 3.883 2.723 3.883 5.053v68.243c0 4.277 -1.553 6.807 -6.99 7.193L24.467 99.967c-4.08 0.193 -6.023 -0.39 -8.16 -3.113L3.3 79.94c-2.333 -3.113 -3.3 -5.443 -3.3 -8.167V11.113c0 -3.497 1.553 -6.413 6.017 -6.8z" fill="#fff"/>
                <path fill-rule="evenodd" clip-rule="evenodd" d="M61.35 0.227l-55.333 4.087C1.553 4.7 0 7.617 0 11.113v60.66c0 2.723 0.967 5.053 3.3 8.167l13.007 16.913c2.137 2.723 4.08 3.307 8.16 3.113l64.257 -3.89c5.433 -0.387 6.99 -2.917 6.99 -7.193V20.64c0 -2.21 -0.873 -2.847 -3.443 -4.733L74.167 3.143c-4.273 -3.107 -6.02 -3.5 -12.817 -2.917zM25.92 19.523c-5.247 0.353 -6.437 0.433 -9.417 -1.99L8.927 11.507c-0.77 -0.78 -0.383 -1.753 1.557 -1.947l53.193 -3.887c4.467 -0.39 6.793 1.167 8.54 2.527l9.123 6.61c0.39 0.197 1.36 1.36 0.193 1.36l-54.933 3.307 -0.68 0.047zM19.803 88.3V30.367c0 -2.53 0.777 -3.697 3.103 -3.893L86 22.78c2.14 -0.193 3.107 1.167 3.107 3.693v57.547c0 2.53 -0.39 4.67 -3.883 4.863l-60.377 3.5c-3.493 0.193 -5.043 -0.97 -5.043 -4.083zm59.6 -54.827c0.387 1.75 0 3.5 -1.75 3.7l-2.91 0.577v42.773c-2.527 1.36 -4.853 2.137 -6.797 2.137 -3.107 0 -3.883 -0.973 -6.21 -3.887l-19.03 -29.94v28.967l6.02 1.363s0 3.5 -4.857 3.5l-13.39 0.777c-0.39 -0.78 0 -2.723 1.357 -3.11l3.497 -0.97v-38.3L30.48 40.667c-0.39 -1.75 0.58 -4.277 3.3 -4.473l14.367 -0.967 19.8 30.327v-26.83l-5.047 -0.58c-0.39 -2.143 1.163 -3.7 3.103 -3.89l13.4 -0.78z" fill="#000"/>
            </svg>
            <svg class="checkmark" viewBox="0 0 36 36">
                <circle cx="18" cy="18" r="18"/>
                <path d="M10 18l5 5 11-11"/>
            </svg>
        </div>
        <h1>Authorization Successful</h1>
        <p>You can now close this window and return to your <span class="terminal-wrapper"><span class="terminal">terminal<span class="cursor"></span></span><span class="sharpie-underlines"><svg viewBox="0 0 60 8" preserveAspectRatio="none">
            <path d="M2 2.5 Q15 1.5, 30 2.8 T58 2" stroke="#00B900" stroke-width="1.8" fill="none" stroke-linecap="round"/>
            <path d="M3 5.5 Q20 4.2, 35 5.8 T57 5" stroke="#00B900" stroke-width="1.5" fill="none" stroke-linecap="round"/>
        </svg></span></span></p>
    </div>
</body>
</html>`

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
	return &cobra.Command{
		Use:   "login",
		Short: "Log in with Notion OAuth",
		Long: `Authenticate with Notion using OAuth.

This will open your browser to authorize notion-cli with your Notion account.
The authorization uses OAuth, allowing you to act as yourself (not as a bot).

If browser-based authentication fails, you can use 'notion auth add-token'
to manually enter an integration token.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOAuthLogin()
		},
	}
}

func runOAuthLogin() error {
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

	// Open browser
	fmt.Fprintln(os.Stderr, "Opening browser to authorize notion-cli...")
	fmt.Fprintln(os.Stderr)
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser. Please visit:\n%s\n\n", authURL)
	}

	fmt.Fprintln(os.Stderr, "Waiting for authorization...")

	// Wait for token or timeout
	select {
	case token := <-tokenCh:
		return storeOAuthToken(token)
	case err := <-errCh:
		return err
	case <-time.After(CallbackTimeout):
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Timed out waiting for authorization.")
		fmt.Fprintln(os.Stderr, "If you completed authorization, you can paste the token manually.")
		fmt.Fprint(os.Stderr, "Enter token (or press Enter to cancel): ")

		tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}

		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			return fmt.Errorf("authorization timed out")
		}

		return storeOAuthToken(token)
	}
}

func storeOAuthToken(token string) error {
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
		fmt.Fprintf(os.Stderr, "Warning: could not store auth source: %v\n", err)
	}

	// Fetch and store user info
	ctx := context.Background()
	client := NewNotionClient(token)
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
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Successfully logged in!")
	if userInfo != nil {
		if userInfo.Email != "" {
			fmt.Fprintf(os.Stderr, "Logged in as: %s (%s)\n", userInfo.Name, userInfo.Email)
		} else {
			fmt.Fprintf(os.Stderr, "Logged in as: %s\n", userInfo.Name)
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
authentication, use 'notion auth login' instead.

The token will be stored securely using your operating system's keyring:
  - macOS: Keychain
  - Linux: Secret Service (GNOME Keyring, KWallet)
  - Windows: Credential Manager

You will be prompted to enter your token interactively. Input will be hidden.

Get your internal integration token from: https://www.notion.so/my-integrations`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prompt for token (use stderr for prompts so stdout is clean)
			fmt.Fprint(os.Stderr, "Enter your Notion API token: ")

			// Read token with hidden input
			tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			fmt.Fprintln(os.Stderr) // Print newline after hidden input

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
				fmt.Fprintf(os.Stderr, "Warning: could not store auth source: %v\n", err)
			}

			// Fetch and store user info
			ctx := context.Background()
			client := NewNotionClient(token)
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
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(cmd.Context(), result)
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
			// Delete token from keyring
			if err := auth.DeleteToken(); err != nil {
				return fmt.Errorf("failed to remove token: %w", err)
			}

			// Print success message
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			result := map[string]interface{}{
				"status":  "success",
				"message": "Logged out successfully",
			}

			return printer.Print(cmd.Context(), result)
		},
	}
}
