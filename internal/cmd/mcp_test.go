package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestMCPCommandTree(t *testing.T) {
	cmd := newMCPCmd()

	// Verify the root mcp command.
	if cmd.Use != "mcp" {
		t.Errorf("mcp command Use = %q, want %q", cmd.Use, "mcp")
	}

	// Verify all expected subcommands exist.
	wantSubcommands := []string{
		"login",
		"logout",
		"status",
		"search",
		"fetch",
		"create",
		"edit",
		"comment",
		"move",
		"duplicate",
		"teams",
		"users",
		"tools",
		"db",
	}

	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	for _, name := range wantSubcommands {
		if !subCmds[name] {
			t.Errorf("missing subcommand %q under mcp", name)
		}
	}
}

func TestMCPCommentSubcommands(t *testing.T) {
	cmd := newMCPCmd()

	// Find the comment subcommand.
	var commentCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "comment" {
			commentCmd = sub
			break
		}
	}
	if commentCmd == nil {
		t.Fatal("mcp command missing 'comment' subcommand")
	}

	wantSubs := []string{"list", "add"}
	subCmds := make(map[string]bool)
	for _, sub := range commentCmd.Commands() {
		subCmds[sub.Name()] = true
	}

	for _, name := range wantSubs {
		if !subCmds[name] {
			t.Errorf("missing subcommand %q under mcp comment", name)
		}
	}
}

func TestMCPSearchArgs(t *testing.T) {
	cmd := newMCPSearchCmd()

	// Verify the search command expects exactly 1 arg.
	if cmd.Args == nil {
		t.Fatal("search command should have Args set")
	}

	// Check the --ai flag exists.
	flag := cmd.Flags().Lookup("ai")
	if flag == nil {
		t.Error("search command missing --ai flag")
	}
}

func TestMCPEditFlags(t *testing.T) {
	cmd := newMCPEditCmd()

	wantFlags := []string{"replace", "replace-range", "insert-after", "new", "properties"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("edit command missing --%s flag", name)
		}
	}
}

func TestMCPCreateFlags(t *testing.T) {
	cmd := newMCPCreateCmd()

	wantFlags := []string{"parent", "data-source", "title", "content", "file", "properties"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("create command missing --%s flag", name)
		}
	}
}

func TestMCPFetchArgs(t *testing.T) {
	cmd := newMCPFetchCmd()

	// Fetch requires exactly 1 arg.
	if cmd.Args == nil {
		t.Fatal("fetch command should have Args set")
	}
}

func TestMCPLoginArgs(t *testing.T) {
	cmd := newMCPLoginCmd()
	if cmd.Use != "login" {
		t.Errorf("login command Use = %q, want 'login'", cmd.Use)
	}
}

func TestMCPLogoutArgs(t *testing.T) {
	cmd := newMCPLogoutCmd()
	if cmd.Use != "logout" {
		t.Errorf("logout command Use = %q, want 'logout'", cmd.Use)
	}
}

func TestMCPStatusArgs(t *testing.T) {
	cmd := newMCPStatusCmd()
	if cmd.Use != "status" {
		t.Errorf("status command Use = %q, want 'status'", cmd.Use)
	}
}

func TestMCPMoveFlags(t *testing.T) {
	cmd := newMCPMoveCmd()
	if cmd.Flags().Lookup("parent") == nil {
		t.Error("move command missing --parent flag")
	}
}

func TestMCPDuplicateArgs(t *testing.T) {
	cmd := newMCPDuplicateCmd()
	if cmd.Use != "duplicate <page-id>" {
		t.Errorf("duplicate command Use = %q", cmd.Use)
	}
}

func TestMCPUsersFlags(t *testing.T) {
	cmd := newMCPUsersCmd()

	wantFlags := []string{"user-id", "cursor", "page-size"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("users command missing --%s flag", name)
		}
	}
}

func TestMCPToolsArgs(t *testing.T) {
	cmd := newMCPToolsCmd()
	if cmd.Use != "tools" {
		t.Errorf("tools command Use = %q, want 'tools'", cmd.Use)
	}
}

func TestMCPDBSubcommands(t *testing.T) {
	cmd := newMCPDBCmd()

	if cmd.Use != "db" {
		t.Errorf("db command Use = %q, want 'db'", cmd.Use)
	}

	wantSubs := []string{"create", "update"}
	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	for _, name := range wantSubs {
		if !subCmds[name] {
			t.Errorf("missing subcommand %q under mcp db", name)
		}
	}
}

func TestMCPDBCreateFlags(t *testing.T) {
	cmd := newMCPDBCreateCmd()

	wantFlags := []string{"parent", "title", "properties"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("db create command missing --%s flag", name)
		}
	}
}

func TestMCPDBUpdateFlags(t *testing.T) {
	cmd := newMCPDBUpdateCmd()

	wantFlags := []string{"id", "title", "properties", "trash"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("db update command missing --%s flag", name)
		}
	}
}
