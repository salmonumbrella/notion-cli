package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOpenClawEnvIfPresent_LoadsValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	unsetEnvWithRestore(t, "NOTION_TOKEN")
	unsetEnvWithRestore(t, "NOTION_WORKSPACE")
	unsetEnvWithRestore(t, "NOTION_OUTPUT")
	unsetEnvWithRestore(t, "NOTION_API_BASE_URL")

	openClawDir := filepath.Join(home, openClawDirName)
	if err := os.MkdirAll(openClawDir, 0o700); err != nil {
		t.Fatalf("failed to create OpenClaw dir: %v", err)
	}

	content := "" +
		"# comment\n" +
		"NOTION_TOKEN=secret_from_openclaw\n" +
		"export NOTION_WORKSPACE=work\n" +
		"NOTION_OUTPUT=\"json\"\n" +
		"NOTION_API_BASE_URL='https://proxy.example'\n"
	path := filepath.Join(openClawDir, openClawEnvFileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write OpenClaw env file: %v", err)
	}

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}

	if got := os.Getenv("NOTION_TOKEN"); got != "secret_from_openclaw" {
		t.Fatalf("NOTION_TOKEN = %q, want %q", got, "secret_from_openclaw")
	}
	if got := os.Getenv("NOTION_WORKSPACE"); got != "work" {
		t.Fatalf("NOTION_WORKSPACE = %q, want %q", got, "work")
	}
	if got := os.Getenv("NOTION_OUTPUT"); got != "json" {
		t.Fatalf("NOTION_OUTPUT = %q, want %q", got, "json")
	}
	if got := os.Getenv("NOTION_API_BASE_URL"); got != "https://proxy.example" {
		t.Fatalf("NOTION_API_BASE_URL = %q, want %q", got, "https://proxy.example")
	}
}

func TestLoadOpenClawEnvIfPresent_DoesNotOverrideExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NOTION_TOKEN", "already-set")

	openClawDir := filepath.Join(home, openClawDirName)
	if err := os.MkdirAll(openClawDir, 0o700); err != nil {
		t.Fatalf("failed to create OpenClaw dir: %v", err)
	}
	path := filepath.Join(openClawDir, openClawEnvFileName)
	if err := os.WriteFile(path, []byte("NOTION_TOKEN=from-file\n"), 0o600); err != nil {
		t.Fatalf("failed to write OpenClaw env file: %v", err)
	}

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}

	if got := os.Getenv("NOTION_TOKEN"); got != "already-set" {
		t.Fatalf("NOTION_TOKEN = %q, want %q", got, "already-set")
	}
}

func TestLoadOpenClawEnvIfPresent_MissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}
}

func TestParseDotEnvLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		key   string
		value string
		ok    bool
	}{
		{name: "simple", line: "A=B", key: "A", value: "B", ok: true},
		{name: "export", line: "export A=B", key: "A", value: "B", ok: true},
		{name: "double quoted", line: "A=\"B C\"", key: "A", value: "B C", ok: true},
		{name: "single quoted", line: "A='B C'", key: "A", value: "B C", ok: true},
		{name: "comment", line: "# test", ok: false},
		{name: "invalid", line: "A", ok: false},
		{name: "empty key", line: "=value", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok := parseDotEnvLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("parseDotEnvLine(%q) ok = %v, want %v", tt.line, ok, tt.ok)
			}
			if !ok {
				return
			}
			if key != tt.key {
				t.Fatalf("parseDotEnvLine(%q) key = %q, want %q", tt.line, key, tt.key)
			}
			if value != tt.value {
				t.Fatalf("parseDotEnvLine(%q) value = %q, want %q", tt.line, value, tt.value)
			}
		})
	}
}

func unsetEnvWithRestore(t *testing.T, key string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
