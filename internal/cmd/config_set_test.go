package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

func TestConfigSetOutput_AcceptsAndNormalizesFormats(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "ndjson", value: "ndjson", expected: "ndjson"},
		{name: "jsonl alias", value: "jsonl", expected: "ndjson"},
		{name: "uppercase alias", value: "JSONL", expected: "ndjson"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			var out, errBuf bytes.Buffer
			app := &App{Stdout: &out, Stderr: &errBuf}
			root := app.RootCommand()
			root.SetArgs([]string{"config", "set", "output", tt.value})
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("config set output failed: %v\nstderr=%s", err, errBuf.String())
			}

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if cfg.Output != tt.expected {
				t.Fatalf("config output = %q, want %q", cfg.Output, tt.expected)
			}
		})
	}
}

func TestConfigSetOutput_RejectsInvalidFormat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"config", "set", "output", "xml"})

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected config set output xml to fail")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}
