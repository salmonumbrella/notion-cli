package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func newAliasTestRoot(t *testing.T) *App {
	t.Helper()
	return &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
}

func TestRootHiddenFlagAliases(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	tests := []struct {
		base  string
		alias string
	}{
		{base: "output", alias: "out"},
		{base: "query", alias: "qr"},
	}

	for _, tt := range tests {
		t.Run(tt.base+"->"+tt.alias, func(t *testing.T) {
			base := root.PersistentFlags().Lookup(tt.base)
			if base == nil {
				t.Fatalf("base flag --%s not found", tt.base)
			}
			alias := root.PersistentFlags().Lookup(tt.alias)
			if alias == nil {
				t.Fatalf("alias flag --%s not found", tt.alias)
			}
			if !alias.Hidden {
				t.Errorf("alias flag --%s should be hidden", tt.alias)
			}
			if alias.Value.Type() != base.Value.Type() {
				t.Errorf("alias --%s type = %q, want %q", tt.alias, alias.Value.Type(), base.Value.Type())
			}
		})
	}

	// -j is provided by BoolP (native shorthand), not a flagAlias.
	if root.PersistentFlags().ShorthandLookup("j") == nil {
		t.Fatal("-j shorthand not found on --json flag")
	}
	if err := root.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set --json: %v", err)
	}
	jsonEnabled, _ := root.PersistentFlags().GetBool("json")
	if !jsonEnabled {
		t.Error("--json should be enabled")
	}
	if err := root.PersistentFlags().Set("json", "false"); err != nil {
		t.Fatalf("set --json: %v", err)
	}
	jsonAliasEnabled, _ := root.PersistentFlags().GetBool("j")
	if jsonAliasEnabled {
		t.Error("--json should update --j value")
	}

	if err := root.PersistentFlags().Set("out", "yaml"); err != nil {
		t.Fatalf("set --out: %v", err)
	}
	outputVal, _ := root.PersistentFlags().GetString("output")
	if outputVal != "yaml" {
		t.Errorf("--out should set --output, got %q", outputVal)
	}

	if err := root.PersistentFlags().Set("qr", ".id"); err != nil {
		t.Fatalf("set --qr: %v", err)
	}
	queryVal, _ := root.PersistentFlags().GetString("query")
	if queryVal != ".id" {
		t.Errorf("--qr should set --query, got %q", queryVal)
	}

	jqFlag := root.PersistentFlags().Lookup("jq")
	if jqFlag == nil {
		t.Fatal("expected --jq to remain registered")
	}
}

func TestCanonicalVerbAliases(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	tests := []struct {
		name     string
		args     []string
		wantName string
	}{
		{name: "top-level list -> ls", args: []string{"ls", "--help"}, wantName: "list"},
		{name: "top-level get -> g", args: []string{"g", "--help"}, wantName: "get"},
		{name: "top-level create -> mk", args: []string{"mk", "--help"}, wantName: "create"},
		{name: "top-level create -> cr", args: []string{"cr", "--help"}, wantName: "create"},
		{name: "top-level search -> q", args: []string{"q", "--help"}, wantName: "search"},
		{name: "page create -> mk", args: []string{"page", "mk", "--help"}, wantName: "create"},
		{name: "datasource create -> cr", args: []string{"datasource", "cr", "--help"}, wantName: "create"},
		{name: "page update -> up", args: []string{"page", "up", "--help"}, wantName: "update"},
		{name: "block delete -> rm", args: []string{"block", "rm", "--help"}, wantName: "delete"},
		{name: "block delete -> del", args: []string{"block", "del", "--help"}, wantName: "delete"},
		{name: "workspace remove -> rm", args: []string{"workspace", "rm", "--help"}, wantName: "remove"},
		{name: "workspace show -> g", args: []string{"workspace", "g", "--help"}, wantName: "show"},
		{name: "config show -> g", args: []string{"config", "g", "--help"}, wantName: "show"},
		{name: "mcp db create -> mk", args: []string{"mcp", "db", "mk", "--help"}, wantName: "create"},
		{name: "mcp db update -> up", args: []string{"mcp", "db", "up", "--help"}, wantName: "update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			if err != nil {
				t.Fatalf("root.Find(%v) error: %v", tt.args, err)
			}
			if cmd == nil {
				t.Fatalf("root.Find(%v) returned nil command", tt.args)
			}
			if cmd.Name() != tt.wantName {
				t.Errorf("root.Find(%v) resolved to %q, want %q", tt.args, cmd.Name(), tt.wantName)
			}
		})
	}
}

func TestCanonicalVerbAliasesAvoidSiblingConflicts(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	searchCmd, _, err := root.Find([]string{"mcp", "search", "--help"})
	if err != nil {
		t.Fatalf("find mcp search: %v", err)
	}
	if searchCmd == nil {
		t.Fatal("mcp search command not found")
	}
	if hasAlias(searchCmd, "q") {
		t.Error("mcp search should not get alias q because mcp query already owns q")
	}
	if !hasAlias(searchCmd, "s") {
		t.Error("mcp search should retain existing alias s")
	}

	queryCmd, _, err := root.Find([]string{"mcp", "query", "--help"})
	if err != nil {
		t.Fatalf("find mcp query: %v", err)
	}
	if queryCmd == nil {
		t.Fatal("mcp query command not found")
	}
	if !hasAlias(queryCmd, "q") {
		t.Error("mcp query should keep alias q")
	}
}

func hasAlias(cmd *cobra.Command, alias string) bool {
	for _, existing := range cmd.Aliases {
		if existing == alias {
			return true
		}
	}
	return false
}
