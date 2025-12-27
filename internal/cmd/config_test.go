package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

func TestConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a subdirectory so SaveToPath creates it with correct permissions
	tmpConfig := filepath.Join(tmpDir, "notion-cli", "config.yaml")

	cfg := &config.Config{
		Output: "json",
	}

	if err := cfg.SaveToPath(tmpConfig); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(tmpConfig)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("config file permissions = %v, want 0600", info.Mode().Perm())
	}

	// Check directory permissions (the subdirectory we created, not the temp dir)
	dirInfo, err := os.Stat(filepath.Dir(tmpConfig))
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}

	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("config directory permissions = %v, want 0700", dirInfo.Mode().Perm())
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		value string
		want  bool
	}{
		{
			name:  "value exists",
			slice: []string{"a", "b", "c"},
			value: "b",
			want:  true,
		},
		{
			name:  "value does not exist",
			slice: []string{"a", "b", "c"},
			value: "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			value: "a",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.value)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
