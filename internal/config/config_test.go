package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantOutput  string
		wantColor   string
		wantDefault string
	}{
		{
			name: "valid config",
			content: `output: json
color: always
default_workspace: personal`,
			wantOutput:  "json",
			wantColor:   "always",
			wantDefault: "personal",
		},
		{
			name:    "empty config",
			content: "",
		},
		{
			name:    "invalid yaml",
			content: "invalid: [yaml",
			wantErr: true,
		},
		{
			name: "partial config",
			content: `output: table
workspaces:
  work:
    token_source: env:NOTION_TOKEN_WORK`,
			wantOutput: "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if tt.content != "" {
				if err := os.WriteFile(configPath, []byte(tt.content), 0600); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
			}

			cfg, err := LoadFromPath(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if cfg.GetOutput() != tt.wantOutput {
				t.Errorf("GetOutput() = %v, want %v", cfg.GetOutput(), tt.wantOutput)
			}
			if cfg.GetColor() != tt.wantColor {
				t.Errorf("GetColor() = %v, want %v", cfg.GetColor(), tt.wantColor)
			}
			if cfg.DefaultWorkspace != tt.wantDefault {
				t.Errorf("DefaultWorkspace = %v, want %v", cfg.DefaultWorkspace, tt.wantDefault)
			}
		})
	}
}

func TestLoadFromPath_NonExistent(t *testing.T) {
	cfg, err := LoadFromPath("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("LoadFromPath() should return empty config for nonexistent file, got error: %v", err)
	}
	if cfg == nil {
		t.Error("LoadFromPath() returned nil config")
	}
}

func TestSaveToPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Output:           "json",
		Color:            "auto",
		DefaultWorkspace: "personal",
		Workspaces: map[string]WorkspaceConfig{
			"personal": {
				TokenSource: "keyring",
			},
			"work": {
				TokenSource: "env:NOTION_TOKEN_WORK",
				Output:      "table",
			},
		},
	}

	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %v, want 0600", info.Mode().Perm())
	}

	// Load it back and verify content
	loaded, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if loaded.Output != cfg.Output {
		t.Errorf("loaded.Output = %v, want %v", loaded.Output, cfg.Output)
	}
	if loaded.Color != cfg.Color {
		t.Errorf("loaded.Color = %v, want %v", loaded.Color, cfg.Color)
	}
	if loaded.DefaultWorkspace != cfg.DefaultWorkspace {
		t.Errorf("loaded.DefaultWorkspace = %v, want %v", loaded.DefaultWorkspace, cfg.DefaultWorkspace)
	}
	if len(loaded.Workspaces) != len(cfg.Workspaces) {
		t.Errorf("loaded.Workspaces len = %v, want %v", len(loaded.Workspaces), len(cfg.Workspaces))
	}
}

func TestSaveToPath_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := &Config{Output: "json"}
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	// Verify directory was created
	dirInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("config path is not a directory")
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("config directory permissions = %v, want 0700", dirInfo.Mode().Perm())
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".config", "notion-cli", "config.yaml")
	if path != expected {
		t.Errorf("DefaultConfigPath() = %v, want %v", path, expected)
	}
}

func TestWorkspaceConfig(t *testing.T) {
	content := `workspaces:
  personal:
    token_source: keyring
    output: json
  work:
    token_source: env:NOTION_TOKEN_WORK
    output: table`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if len(cfg.Workspaces) != 2 {
		t.Errorf("Workspaces len = %v, want 2", len(cfg.Workspaces))
	}

	personal, ok := cfg.Workspaces["personal"]
	if !ok {
		t.Fatal("personal workspace not found")
	}
	if personal.TokenSource != "keyring" {
		t.Errorf("personal.TokenSource = %v, want keyring", personal.TokenSource)
	}
	if personal.Output != "json" {
		t.Errorf("personal.Output = %v, want json", personal.Output)
	}

	work, ok := cfg.Workspaces["work"]
	if !ok {
		t.Fatal("work workspace not found")
	}
	if work.TokenSource != "env:NOTION_TOKEN_WORK" {
		t.Errorf("work.TokenSource = %v, want env:NOTION_TOKEN_WORK", work.TokenSource)
	}
	if work.Output != "table" {
		t.Errorf("work.Output = %v, want table", work.Output)
	}
}
