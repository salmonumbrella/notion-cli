package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

func TestResolveWorkspaceToken(t *testing.T) {
	tests := []struct {
		name         string
		workspace    *config.WorkspaceConfig
		envVars      map[string]string
		setupKeyring func() KeyringProvider
		want         string
		wantErr      bool
		errContains  string
	}{
		{
			name:        "nil workspace",
			workspace:   nil,
			wantErr:     true,
			errContains: "workspace config is nil",
		},
		{
			name: "empty token source defaults to keyring",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "",
			},
			setupKeyring: func() KeyringProvider {
				mock := &MockKeyring{
					items: map[string]keyring.Item{
						KeyName: {Key: KeyName, Data: []byte("keyring-token-123")},
					},
				}
				return mock
			},
			want:    "keyring-token-123",
			wantErr: false,
		},
		{
			name: "explicit keyring token source",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "keyring",
			},
			setupKeyring: func() KeyringProvider {
				mock := &MockKeyring{
					items: map[string]keyring.Item{
						KeyName: {Key: KeyName, Data: []byte("keyring-token-456")},
					},
				}
				return mock
			},
			want:    "keyring-token-456",
			wantErr: false,
		},
		{
			name: "env token source with valid env var",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "env:MY_NOTION_TOKEN",
			},
			envVars: map[string]string{
				"MY_NOTION_TOKEN": "env-token-789",
			},
			want:    "env-token-789",
			wantErr: false,
		},
		{
			name: "env token source with missing env var",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "env:MISSING_VAR",
			},
			wantErr:     true,
			errContains: "environment variable MISSING_VAR is not set",
		},
		{
			name: "env token source with empty env var",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "env:EMPTY_VAR",
			},
			envVars: map[string]string{
				"EMPTY_VAR": "",
			},
			wantErr:     true,
			errContains: "environment variable EMPTY_VAR is not set",
		},
		{
			name: "direct token value",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "secret_ABC123XYZ",
			},
			want:    "secret_ABC123XYZ",
			wantErr: false,
		},
		{
			name: "direct token value with special chars",
			workspace: &config.WorkspaceConfig{
				Name:        "test",
				TokenSource: "secret_token-with.special_chars123",
			},
			want:    "secret_token-with.special_chars123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			// Setup keyring mock if provided
			if tt.setupKeyring != nil {
				originalProvider := defaultProvider
				defaultProvider = func() (KeyringProvider, error) {
					return tt.setupKeyring(), nil
				}
				defer func() { defaultProvider = originalProvider }()
			}

			got, err := ResolveWorkspaceToken(tt.workspace)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveWorkspaceToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ResolveWorkspaceToken() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveWorkspaceToken() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveWorkspaceToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkspaceToken_WithConfig(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		config        *config.Config
		setupEnv      map[string]string
		setupKeyring  func() KeyringProvider
		want          string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "named workspace with env token",
			workspaceName: "production",
			config: &config.Config{
				Workspaces: map[string]config.WorkspaceConfig{
					"production": {
						Name:        "production",
						TokenSource: "env:PROD_TOKEN",
					},
				},
			},
			setupEnv: map[string]string{
				"PROD_TOKEN": "prod-token-123",
			},
			want:    "prod-token-123",
			wantErr: false,
		},
		{
			name:          "default workspace when no name specified",
			workspaceName: "",
			config: &config.Config{
				DefaultWorkspace: "dev",
				Workspaces: map[string]config.WorkspaceConfig{
					"dev": {
						Name:        "dev",
						TokenSource: "env:DEV_TOKEN",
					},
					"prod": {
						Name:        "prod",
						TokenSource: "env:PROD_TOKEN",
					},
				},
			},
			setupEnv: map[string]string{
				"DEV_TOKEN":  "dev-token-456",
				"PROD_TOKEN": "prod-token-789",
			},
			want:    "dev-token-456",
			wantErr: false,
		},
		{
			name:          "single workspace auto-selected as default",
			workspaceName: "",
			config: &config.Config{
				Workspaces: map[string]config.WorkspaceConfig{
					"only": {
						Name:        "only",
						TokenSource: "direct-token",
					},
				},
			},
			want:    "direct-token",
			wantErr: false,
		},
		{
			name:          "workspace not found",
			workspaceName: "nonexistent",
			config: &config.Config{
				Workspaces: map[string]config.WorkspaceConfig{
					"existing": {
						Name:        "existing",
						TokenSource: "token",
					},
				},
			},
			wantErr:     true,
			errContains: "workspace \"nonexistent\" not found",
		},
		{
			name:          "workspace with keyring token source",
			workspaceName: "work",
			config: &config.Config{
				Workspaces: map[string]config.WorkspaceConfig{
					"work": {
						Name:        "work",
						TokenSource: "keyring",
					},
				},
			},
			setupKeyring: func() KeyringProvider {
				return &MockKeyring{
					items: map[string]keyring.Item{
						KeyName: {Key: KeyName, Data: []byte("work-keyring-token")},
					},
				}
			},
			want:    "work-keyring-token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config directory and file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Save config to temp path
			if err := tt.config.SaveToPath(configPath); err != nil {
				t.Fatalf("Failed to save config: %v", err)
			}

			// Override config path via environment
			originalHome := os.Getenv("HOME")
			_ = os.Setenv("HOME", tmpDir)
			defer func() { _ = os.Setenv("HOME", originalHome) }()

			// Create .config/notion-cli directory structure
			notionCliDir := filepath.Join(tmpDir, ".config", "notion-cli")
			if err := os.MkdirAll(notionCliDir, 0o700); err != nil {
				t.Fatalf("Failed to create config dir: %v", err)
			}

			// Copy config to expected location
			notionConfigPath := filepath.Join(notionCliDir, "config.yaml")
			if err := tt.config.SaveToPath(notionConfigPath); err != nil {
				t.Fatalf("Failed to save config to expected location: %v", err)
			}

			// Setup environment variables
			for k, v := range tt.setupEnv {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			// Setup keyring mock if provided
			if tt.setupKeyring != nil {
				originalProvider := defaultProvider
				defaultProvider = func() (KeyringProvider, error) {
					return tt.setupKeyring(), nil
				}
				defer func() { defaultProvider = originalProvider }()
			}

			got, err := GetWorkspaceToken(tt.workspaceName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetWorkspaceToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetWorkspaceToken() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("GetWorkspaceToken() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetWorkspaceToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkspaceToken_NoConfigFallback(t *testing.T) {
	// Setup temp directory with no config file
	tmpDir := t.TempDir()

	// Override HOME to point to temp dir
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Setup keyring mock
	originalProvider := defaultProvider
	defaultProvider = func() (KeyringProvider, error) {
		return &MockKeyring{
			items: map[string]keyring.Item{
				KeyName: {Key: KeyName, Data: []byte("fallback-token")},
			},
		}, nil
	}
	defer func() { defaultProvider = originalProvider }()

	// Should fall back to keyring when no config exists
	got, err := GetWorkspaceToken("")
	if err != nil {
		t.Fatalf("GetWorkspaceToken() unexpected error = %v", err)
	}
	if got != "fallback-token" {
		t.Errorf("GetWorkspaceToken() = %v, want %v", got, "fallback-token")
	}
}
