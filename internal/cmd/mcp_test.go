package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/skill"
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
		"query",
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

func TestMCPQueryFlags(t *testing.T) {
	cmd := newMCPQueryCmd()

	wantFlags := []string{"view", "params"}
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("query command missing --%s flag", name)
		}
	}
}

func TestResolveMCPCreatePeopleIDs_NilSkillFile(t *testing.T) {
	props := map[string]interface{}{"key": "value"}
	got := resolveMCPCreatePeopleIDs(nil, props)
	if got["key"] != "value" {
		t.Fatal("nil SkillFile should return properties unchanged")
	}
}

func TestResolveMCPCreatePeopleIDs(t *testing.T) {
	sf := &skill.SkillFile{
		Users: map[string]skill.UserAlias{
			"isaac": {
				Alias: "isaac",
				Name:  "Isaac",
				ID:    "608a5f14-b513-4fae-b3cc-476d266f227b",
			},
		},
	}

	props := map[string]interface{}{
		"DRI": map[string]interface{}{
			"people": []interface{}{
				map[string]interface{}{"id": "isaac"},
			},
		},
		"Watch": map[string]interface{}{
			"people": []interface{}{
				map[string]interface{}{"id": "already-uuid"},
			},
		},
		"Title": "ignored-non-map-value",
	}

	got := resolveMCPCreatePeopleIDs(sf, props)

	driProp, ok := got["DRI"].(map[string]interface{})
	if !ok {
		t.Fatalf("DRI property type = %T, want map[string]interface{}", got["DRI"])
	}
	driPeople, ok := driProp["people"].([]interface{})
	if !ok || len(driPeople) != 1 {
		t.Fatalf("DRI.people type/len invalid: %T len=%d", driProp["people"], len(driPeople))
	}
	driPerson, ok := driPeople[0].(map[string]interface{})
	if !ok {
		t.Fatalf("DRI.people[0] type = %T, want map[string]interface{}", driPeople[0])
	}
	if driPerson["id"] != "608a5f14-b513-4fae-b3cc-476d266f227b" {
		t.Fatalf("DRI.people[0].id = %v, want resolved user id", driPerson["id"])
	}

	watchProp, ok := got["Watch"].(map[string]interface{})
	if !ok {
		t.Fatalf("Watch property type = %T, want map[string]interface{}", got["Watch"])
	}
	watchPeople, ok := watchProp["people"].([]interface{})
	if !ok || len(watchPeople) != 1 {
		t.Fatalf("Watch.people type/len invalid: %T len=%d", watchProp["people"], len(watchPeople))
	}
	watchPerson, ok := watchPeople[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Watch.people[0] type = %T, want map[string]interface{}", watchPeople[0])
	}
	if watchPerson["id"] != "already-uuid" {
		t.Fatalf("Watch.people[0].id = %v, want unchanged", watchPerson["id"])
	}
}
