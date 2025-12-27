package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/config"
)

func TestWorkspaceFlag_Integration(t *testing.T) {
	// Create a temporary directory for config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Set up config path override
	origConfigPathFunc := config.SetConfigPathFunc(func() (string, error) {
		return configPath, nil
	})
	defer config.SetConfigPathFunc(origConfigPathFunc)

	// Create a config with two workspaces
	cfg := &config.Config{
		DefaultWorkspace: "default",
		Workspaces: map[string]config.WorkspaceConfig{
			"default": {
				Name:        "default",
				TokenSource: "env:DEFAULT_TOKEN",
			},
			"work": {
				Name:        "work",
				TokenSource: "env:WORK_TOKEN",
			},
		},
	}

	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Set up environment variables
	_ = os.Setenv("DEFAULT_TOKEN", "token-default-123")
	_ = os.Setenv("WORK_TOKEN", "token-work-456")
	defer func() {
		_ = os.Unsetenv("DEFAULT_TOKEN")
		_ = os.Unsetenv("WORK_TOKEN")
	}()

	tests := []struct {
		name          string
		workspaceFlag string
		workspaceEnv  string
		expectedToken string
	}{
		{
			name:          "flag takes precedence",
			workspaceFlag: "work",
			workspaceEnv:  "default",
			expectedToken: "token-work-456",
		},
		{
			name:          "env var used when flag empty",
			workspaceFlag: "",
			workspaceEnv:  "work",
			expectedToken: "token-work-456",
		},
		{
			name:          "default workspace when neither set",
			workspaceFlag: "",
			workspaceEnv:  "",
			expectedToken: "token-default-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.workspaceEnv != "" {
				_ = os.Setenv("NOTION_WORKSPACE", tt.workspaceEnv)
			} else {
				_ = os.Unsetenv("NOTION_WORKSPACE")
			}
			defer func() { _ = os.Unsetenv("NOTION_WORKSPACE") }()

			// Simulate context setup like in PersistentPreRunE
			ctx := context.Background()
			ws := tt.workspaceFlag
			if ws == "" {
				ws = os.Getenv("NOTION_WORKSPACE")
			}
			ctx = WithWorkspace(ctx, ws)

			// Get token using the context
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				t.Fatalf("GetTokenFromContext() error = %v", err)
			}

			if token != tt.expectedToken {
				t.Errorf("GetTokenFromContext() = %q, want %q", token, tt.expectedToken)
			}
		})
	}
}

func TestWorkspaceFlag_BackwardCompatibility(t *testing.T) {
	// Create a temporary directory for config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Set up config path override
	origConfigPathFunc := config.SetConfigPathFunc(func() (string, error) {
		return configPath, nil
	})
	defer config.SetConfigPathFunc(origConfigPathFunc)

	// Create empty config (no workspaces)
	cfg := &config.Config{}
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	tests := []struct {
		name          string
		envToken      string
		workspaceFlag string
		workspaceEnv  string
		useKeyring    bool
		expectedToken string
		expectError   bool
	}{
		{
			name:          "uses keyring when no workspace",
			workspaceFlag: "",
			workspaceEnv:  "",
			useKeyring:    true,
			expectedToken: "keyring-token-123",
		},
		{
			name:          "NOTION_TOKEN env var used when no keyring",
			envToken:      "env-token-456",
			workspaceFlag: "",
			workspaceEnv:  "",
			useKeyring:    false,
			expectedToken: "env-token-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up keyring provider for this test
			if tt.useKeyring {
				mockProvider := auth.NewMockKeyringProvider()
				mockProvider.SetToken("keyring-token-123")
				auth.SetProviderFunc(func() (auth.KeyringProvider, error) {
					return mockProvider, nil
				})
			} else {
				// No keyring available
				auth.SetProviderFunc(func() (auth.KeyringProvider, error) {
					return nil, fmt.Errorf("keyring not available")
				})
			}
			defer auth.SetProviderFunc(nil)

			// Set up environment
			if tt.envToken != "" {
				_ = os.Setenv("NOTION_TOKEN", tt.envToken)
			} else {
				_ = os.Unsetenv("NOTION_TOKEN")
			}
			defer func() { _ = os.Unsetenv("NOTION_TOKEN") }()

			if tt.workspaceEnv != "" {
				_ = os.Setenv("NOTION_WORKSPACE", tt.workspaceEnv)
			} else {
				_ = os.Unsetenv("NOTION_WORKSPACE")
			}
			defer func() { _ = os.Unsetenv("NOTION_WORKSPACE") }()

			// Simulate context setup
			ctx := context.Background()
			ws := tt.workspaceFlag
			if ws == "" {
				ws = os.Getenv("NOTION_WORKSPACE")
			}
			ctx = WithWorkspace(ctx, ws)

			// Get token
			token, err := GetTokenFromContext(ctx)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetTokenFromContext() error = %v", err)
			}

			if token != tt.expectedToken {
				t.Errorf("GetTokenFromContext() = %q, want %q", token, tt.expectedToken)
			}
		})
	}
}
