package cmd

import (
	"bytes"
	"testing"
)

// TestDesirePaths documents every command pattern agents naturally attempt.
// Each entry records: what agents try, whether it works, and what it maps to.
// When a new pattern is discovered, add it here BEFORE implementing it.
//
// This test validates that all "implemented" desire paths actually resolve
// to real commands in the command tree.
func TestDesirePaths(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	root := app.RootCommand()

	paths := []struct {
		name        string   // Description of what agent tries
		args        []string // Command args (without "notion")
		implemented bool     // true = should resolve to a command
	}{
		// Plural aliases
		{"plural: pages", []string{"pages", "--help"}, true},
		{"plural: blocks", []string{"blocks", "--help"}, true},
		{"plural: databases", []string{"databases", "--help"}, true},
		{"plural: users", []string{"users", "--help"}, true},
		{"plural: comments", []string{"comments", "--help"}, true},

		// Single-letter abbreviations
		{"abbrev: p for page", []string{"p", "--help"}, true},
		{"abbrev: b for block", []string{"b", "--help"}, true},
		{"abbrev: s for search", []string{"s", "--help"}, true},
		{"abbrev: u for user", []string{"u", "--help"}, true},
		{"abbrev: c for comment", []string{"c", "--help"}, true},

		// Top-level shortcuts
		{"shortcut: login", []string{"login", "--help"}, true},
		{"shortcut: logout", []string{"logout", "--help"}, true},
		{"shortcut: whoami", []string{"whoami", "--help"}, true},
		{"shortcut: open", []string{"open", "--help"}, true},
		{"shortcut: list", []string{"list", "--help"}, true},
		{"shortcut: get", []string{"get", "--help"}, true},
		{"shortcut: create", []string{"create", "--help"}, true},
		{"shortcut: delete", []string{"delete", "--help"}, true},

		// Top-level delete aliases
		{"shortcut: rm (alias for delete)", []string{"rm", "--help"}, true},
		{"shortcut: d (alias for delete)", []string{"d", "--help"}, true},

		// Subcommand aliases
		{"alias: page list -> page ls", []string{"page", "ls", "--help"}, true},
		{"alias: page delete -> page rm", []string{"page", "rm", "--help"}, true},
		{"alias: page delete -> page d", []string{"page", "d", "--help"}, true},
		{"alias: block children -> block list", []string{"block", "list", "--help"}, true},
		{"alias: block children -> block ls", []string{"block", "ls", "--help"}, true},
		{"alias: search -> find", []string{"find", "--help"}, true},

		// db aliases
		{"alias: db -> database", []string{"database", "--help"}, true},
		{"alias: db shorthand", []string{"db", "--help"}, true},
	}

	for _, tt := range paths {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			found := err == nil && cmd != nil && cmd != root

			if tt.implemented && !found {
				t.Errorf("desire path %q should work but command not found (args: %v)", tt.name, tt.args)
			}
			if !tt.implemented && found {
				t.Errorf("desire path %q marked as unimplemented but command exists (args: %v)", tt.name, tt.args)
			}
		})
	}
}
