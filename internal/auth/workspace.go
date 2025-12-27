package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

// ResolveWorkspaceToken gets the token for a workspace based on its token_source config.
// Token source can be:
//   - "keyring" - get from system keyring (default)
//   - "env:VAR_NAME" - get from environment variable
//   - Any other value - treat as direct token
func ResolveWorkspaceToken(ws *config.WorkspaceConfig) (string, error) {
	if ws == nil {
		return "", fmt.Errorf("workspace config is nil")
	}

	source := ws.TokenSource
	if source == "" || source == "keyring" {
		return GetToken() // Use existing keyring function
	}

	if strings.HasPrefix(source, "env:") {
		varName := strings.TrimPrefix(source, "env:")
		token := os.Getenv(varName)
		if token == "" {
			return "", fmt.Errorf("environment variable %s is not set", varName)
		}
		return token, nil
	}

	// Treat as direct token value
	return source, nil
}

// GetWorkspaceToken gets the token for a named workspace from config.
// If workspaceName is empty, uses the default workspace.
func GetWorkspaceToken(workspaceName string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	var ws *config.WorkspaceConfig
	if workspaceName == "" {
		ws, err = cfg.GetDefaultWorkspace()
		if err != nil {
			// No default workspace configured, fall back to keyring
			return GetToken()
		}
	} else {
		ws, err = cfg.GetWorkspace(workspaceName)
		if err != nil {
			return "", err
		}
	}

	return ResolveWorkspaceToken(ws)
}
