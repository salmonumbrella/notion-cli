package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRootHelp_CustomMenu(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	wantSnippets := []string{
		"ntn - Notion CLI for humans and agents",
		"IMPORTANT: Always use the shortest alias",
		"Aliases (resource",
		"Aliases (shortcut",
		"Aliases (subcommand",
		"Quick start:",
		"Lookup:",
		"Search:",
		"Create:",
		"Modify:",
		"Database query:",
		"MCP (capabilities beyond REST):",
		"Output formats:",
		"Common flags:",
		"Exit codes:",
		"Environment:",
		"Skill file:",
		"Discovery:",
		"--help-json",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(got, snippet) {
			t.Fatalf("root help missing %q\nhelp output:\n%s", snippet, got)
		}
	}
}

func TestRootHelp_SubcommandFallback(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"page", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Usage:\n  ntn page [flags]") {
		t.Fatalf("subcommand help did not use default help output:\n%s", got)
	}
	if strings.Contains(got, "IMPORTANT: Always use the shortest alias") {
		t.Fatalf("subcommand help should not include root curated menu:\n%s", got)
	}
}

func TestHelpJSON_Root(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	cmd, ok := findHelpJSONTarget(root, []string{"--help-json"})
	if !ok {
		t.Fatal("findHelpJSONTarget returned false for --help-json")
	}
	if err := printHelpJSON(cmd); err != nil {
		t.Fatalf("printHelpJSON error: %v", err)
	}

	var payload CommandHelp
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out.String())
	}
	if payload.Name != "ntn" {
		t.Fatalf("expected name ntn, got %q", payload.Name)
	}
	if len(payload.Subcommands) == 0 {
		t.Fatal("expected subcommands, got none")
	}
}

func TestHelpJSON_Subcommand(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	cmd, ok := findHelpJSONTarget(root, []string{"page", "--help-json"})
	if !ok {
		t.Fatal("findHelpJSONTarget returned false for page --help-json")
	}
	if err := printHelpJSON(cmd); err != nil {
		t.Fatalf("printHelpJSON error: %v", err)
	}

	var payload CommandHelp
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out.String())
	}
	if payload.Name != "page" {
		t.Fatalf("expected name page, got %q", payload.Name)
	}
	if len(payload.Flags) == 0 {
		t.Fatal("expected flags, got none")
	}
}

func TestHelpJSON_ViaExecute(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	// Execute goes through App.Execute which intercepts --help-json
	// before Cobra arg validation. We can't easily capture stdout from
	// printHelpJSON here because it writes to cmd.OutOrStdout() (os.Stdout
	// by default in Execute). Instead, verify it doesn't error.
	err := app.Execute(context.Background(), []string{"--help-json"})
	if err != nil {
		t.Fatalf("Execute --help-json failed: %v", err)
	}
}
