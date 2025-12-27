// Package auth provides secure token storage and retrieval for the Notion CLI.
//
// Token storage uses the OS keyring (macOS Keychain, Windows Credential Manager,
// Linux Secret Service) via github.com/99designs/keyring. This ensures tokens
// are stored securely and are not accessible to other applications.
//
// The package also supports fallback to the NOTION_TOKEN environment variable
// for CI/CD environments and scripts where keyring access may not be available.
// Keyring fallback files can be directed to a custom root with:
//   - NOTION_CREDENTIALS_DIR (preferred)
//   - OPENCLAW_CREDENTIALS_DIR (shared fallback)
//
// Priority order for token retrieval:
//  1. NOTION_TOKEN environment variable (highest priority, avoids keychain prompts)
//  2. OS keyring (fallback)
//
// Example usage:
//
//	// Store a token
//	if err := auth.StoreToken("secret_abc123"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Retrieve a token
//	token, err := auth.GetToken()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if a token exists
//	if auth.HasToken() {
//	    fmt.Println("Token is configured")
//	}
//
//	// Delete a token
//	if err := auth.DeleteToken(); err != nil {
//	    log.Fatal(err)
//	}
package auth
