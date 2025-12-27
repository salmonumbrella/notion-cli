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
				if err := os.WriteFile(configPath, []byte(tt.content), 0o600); err != nil {
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
	if info.Mode().Perm() != 0o600 {
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
	if dirInfo.Mode().Perm() != 0o700 {
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
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
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

func TestGetWorkspace(t *testing.T) {
	cfg := &Config{
		Workspaces: map[string]WorkspaceConfig{
			"personal": {
				Name:        "personal",
				TokenSource: "keyring",
				APIURL:      "",
			},
			"work": {
				Name:        "work",
				TokenSource: "env:NOTION_TOKEN_WORK",
				APIURL:      "https://api.notion.com/v1",
			},
		},
	}

	tests := []struct {
		name    string
		wsName  string
		wantErr bool
	}{
		{
			name:    "existing workspace",
			wsName:  "personal",
			wantErr: false,
		},
		{
			name:    "another existing workspace",
			wsName:  "work",
			wantErr: false,
		},
		{
			name:    "non-existent workspace",
			wsName:  "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := cfg.GetWorkspace(tt.wsName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ws == nil {
				t.Error("GetWorkspace() returned nil workspace")
			}
			if !tt.wantErr && ws.Name != tt.wsName {
				t.Errorf("GetWorkspace() returned workspace with name %v, want %v", ws.Name, tt.wsName)
			}
		})
	}
}

func TestGetWorkspace_NilWorkspaces(t *testing.T) {
	cfg := &Config{}
	_, err := cfg.GetWorkspace("any")
	if err == nil {
		t.Error("GetWorkspace() should return error when Workspaces is nil")
	}
}

func TestGetDefaultWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		wantWorkspace string
		wantErr       bool
	}{
		{
			name: "default set via DefaultWorkspace field",
			cfg: &Config{
				DefaultWorkspace: "personal",
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
				},
			},
			wantWorkspace: "personal",
			wantErr:       false,
		},
		{
			name: "single workspace becomes default",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
				},
			},
			wantWorkspace: "personal",
			wantErr:       false,
		},
		{
			name: "no default configured with multiple workspaces",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
				},
			},
			wantErr: true,
		},
		{
			name:    "no workspaces",
			cfg:     &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := tt.cfg.GetDefaultWorkspace()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ws.Name != tt.wantWorkspace {
				t.Errorf("GetDefaultWorkspace() returned workspace %v, want %v", ws.Name, tt.wantWorkspace)
			}
		})
	}
}

func TestGetDefaultWorkspace_OnlyDefaultWorkspaceMatters(t *testing.T) {
	// Test that DefaultWorkspace field is the authoritative source
	// even if old YAML files have the deprecated Default bool field
	cfg := &Config{
		DefaultWorkspace: "work",
		Workspaces: map[string]WorkspaceConfig{
			"personal": {Name: "personal", TokenSource: "keyring"},
			"work":     {Name: "work", TokenSource: "env:TOKEN"},
		},
	}

	ws, err := cfg.GetDefaultWorkspace()
	if err != nil {
		t.Fatalf("GetDefaultWorkspace() error = %v", err)
	}

	if ws.Name != "work" {
		t.Errorf("GetDefaultWorkspace() returned workspace %v, want work", ws.Name)
	}
}

func TestSetDefaultWorkspace(t *testing.T) {
	cfg := &Config{
		DefaultWorkspace: "personal",
		Workspaces: map[string]WorkspaceConfig{
			"personal": {Name: "personal", TokenSource: "keyring"},
			"work":     {Name: "work", TokenSource: "env:TOKEN"},
		},
	}

	// Set work as default
	err := cfg.SetDefaultWorkspace("work")
	if err != nil {
		t.Fatalf("SetDefaultWorkspace() error = %v", err)
	}

	if cfg.DefaultWorkspace != "work" {
		t.Errorf("DefaultWorkspace = %v, want work", cfg.DefaultWorkspace)
	}
}

func TestSetDefaultWorkspace_NonExistent(t *testing.T) {
	cfg := &Config{
		Workspaces: map[string]WorkspaceConfig{
			"personal": {Name: "personal", TokenSource: "keyring"},
		},
	}

	err := cfg.SetDefaultWorkspace("unknown")
	if err == nil {
		t.Error("SetDefaultWorkspace() should return error for non-existent workspace")
	}
}

func TestAddWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		wsName      string
		ws          WorkspaceConfig
		wantErr     bool
		wantDefault string
	}{
		{
			name:   "add first workspace",
			cfg:    &Config{},
			wsName: "personal",
			ws: WorkspaceConfig{
				TokenSource: "keyring",
			},
			wantErr:     false,
			wantDefault: "personal",
		},
		{
			name: "add second workspace",
			cfg: &Config{
				DefaultWorkspace: "personal",
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
				},
			},
			wsName: "work",
			ws: WorkspaceConfig{
				TokenSource: "env:TOKEN",
			},
			wantErr:     false,
			wantDefault: "personal",
		},
		{
			name: "add duplicate workspace",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
				},
			},
			wsName:  "personal",
			ws:      WorkspaceConfig{TokenSource: "keyring"},
			wantErr: true,
		},
		{
			name:    "add workspace with empty name",
			cfg:     &Config{},
			wsName:  "",
			ws:      WorkspaceConfig{TokenSource: "keyring"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.AddWorkspace(tt.wsName, tt.ws)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				ws, exists := tt.cfg.Workspaces[tt.wsName]
				if !exists {
					t.Errorf("workspace %v was not added", tt.wsName)
					return
				}
				if ws.Name != tt.wsName {
					t.Errorf("workspace Name = %v, want %v", ws.Name, tt.wsName)
				}
				if tt.cfg.DefaultWorkspace != tt.wantDefault {
					t.Errorf("DefaultWorkspace = %v, want %v", tt.cfg.DefaultWorkspace, tt.wantDefault)
				}
			}
		})
	}
}

func TestRemoveWorkspace(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *Config
		removeWs          string
		wantErr           bool
		wantDefaultAfter  string
		wantWorkspaceLeft []string
	}{
		{
			name: "remove non-default workspace",
			cfg: &Config{
				DefaultWorkspace: "personal",
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
				},
			},
			removeWs:          "work",
			wantErr:           false,
			wantDefaultAfter:  "personal",
			wantWorkspaceLeft: []string{"personal"},
		},
		{
			name: "remove default workspace with other workspaces",
			cfg: &Config{
				DefaultWorkspace: "personal",
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
					"test":     {Name: "test", TokenSource: "secret_123"},
				},
			},
			removeWs:          "personal",
			wantErr:           false,
			wantDefaultAfter:  "",
			wantWorkspaceLeft: []string{"work", "test"},
		},
		{
			name: "remove default workspace leaving one workspace",
			cfg: &Config{
				DefaultWorkspace: "personal",
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
				},
			},
			removeWs:          "personal",
			wantErr:           false,
			wantDefaultAfter:  "work",
			wantWorkspaceLeft: []string{"work"},
		},
		{
			name: "remove non-existent workspace",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
				},
			},
			removeWs: "unknown",
			wantErr:  true,
		},
		{
			name:     "remove from nil workspaces",
			cfg:      &Config{},
			removeWs: "any",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.RemoveWorkspace(tt.removeWs)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if _, exists := tt.cfg.Workspaces[tt.removeWs]; exists {
					t.Errorf("workspace %v was not removed", tt.removeWs)
				}

				if tt.cfg.DefaultWorkspace != tt.wantDefaultAfter {
					t.Errorf("DefaultWorkspace = %v, want %v", tt.cfg.DefaultWorkspace, tt.wantDefaultAfter)
				}

				if len(tt.cfg.Workspaces) != len(tt.wantWorkspaceLeft) {
					t.Errorf("Workspaces count = %v, want %v", len(tt.cfg.Workspaces), len(tt.wantWorkspaceLeft))
				}

				for _, wsName := range tt.wantWorkspaceLeft {
					if _, exists := tt.cfg.Workspaces[wsName]; !exists {
						t.Errorf("workspace %v should still exist", wsName)
					}
				}
			}
		})
	}
}

func TestListWorkspaces(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantCount int
		wantNames []string
	}{
		{
			name: "multiple workspaces",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
					"work":     {Name: "work", TokenSource: "env:TOKEN"},
					"test":     {Name: "test", TokenSource: "secret_123"},
				},
			},
			wantCount: 3,
			wantNames: []string{"personal", "test", "work"}, // sorted order
		},
		{
			name: "single workspace",
			cfg: &Config{
				Workspaces: map[string]WorkspaceConfig{
					"personal": {Name: "personal", TokenSource: "keyring"},
				},
			},
			wantCount: 1,
			wantNames: []string{"personal"},
		},
		{
			name:      "no workspaces",
			cfg:       &Config{},
			wantCount: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := tt.cfg.ListWorkspaces()
			if len(names) != tt.wantCount {
				t.Errorf("ListWorkspaces() returned %v names, want %v", len(names), tt.wantCount)
			}

			// Check that results are in sorted order
			for i, wantName := range tt.wantNames {
				if i >= len(names) || names[i] != wantName {
					t.Errorf("ListWorkspaces()[%d] = %v, want %v (expected sorted order)", i, names[i], wantName)
				}
			}
		})
	}
}

func TestWorkspaceConfigEnhancedFields(t *testing.T) {
	content := `default_workspace: personal
workspaces:
  personal:
    name: personal
    token_source: keyring
  work:
    name: work
    token_source: env:NOTION_TOKEN_WORK
    api_url: https://api.notion.com/v1
  test:
    name: test
    token_source: secret_abc123
    api_url: https://test.notion.com/v1`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	// Test that DefaultWorkspace is set correctly
	if cfg.DefaultWorkspace != "personal" {
		t.Errorf("DefaultWorkspace = %v, want personal", cfg.DefaultWorkspace)
	}

	// Test personal workspace
	personal, ok := cfg.Workspaces["personal"]
	if !ok {
		t.Fatal("personal workspace not found")
	}
	if personal.Name != "personal" {
		t.Errorf("personal.Name = %v, want personal", personal.Name)
	}
	if personal.TokenSource != "keyring" {
		t.Errorf("personal.TokenSource = %v, want keyring", personal.TokenSource)
	}
	if personal.APIURL != "" {
		t.Errorf("personal.APIURL = %v, want empty", personal.APIURL)
	}

	// Test work workspace
	work, ok := cfg.Workspaces["work"]
	if !ok {
		t.Fatal("work workspace not found")
	}
	if work.Name != "work" {
		t.Errorf("work.Name = %v, want work", work.Name)
	}
	if work.TokenSource != "env:NOTION_TOKEN_WORK" {
		t.Errorf("work.TokenSource = %v, want env:NOTION_TOKEN_WORK", work.TokenSource)
	}
	if work.APIURL != "https://api.notion.com/v1" {
		t.Errorf("work.APIURL = %v, want https://api.notion.com/v1", work.APIURL)
	}

	// Test test workspace
	test, ok := cfg.Workspaces["test"]
	if !ok {
		t.Fatal("test workspace not found")
	}
	if test.Name != "test" {
		t.Errorf("test.Name = %v, want test", test.Name)
	}
	if test.TokenSource != "secret_abc123" {
		t.Errorf("test.TokenSource = %v, want secret_abc123", test.TokenSource)
	}
	if test.APIURL != "https://test.notion.com/v1" {
		t.Errorf("test.APIURL = %v, want https://test.notion.com/v1", test.APIURL)
	}
}
