package cmd

import (
	"bytes"
	"testing"
)

func newFlagParityTestRoot(t *testing.T) *App {
	t.Helper()
	return &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
}

func TestRootFlagParity_HiddenAliases(t *testing.T) {
	root := newFlagParityTestRoot(t).RootCommand()

	tests := []struct {
		base  string
		alias string
	}{
		{base: "query-file", alias: "qf"},
		{base: "results-only", alias: "ro"},
		{base: "items-only", alias: "io"},
		{base: "items-only", alias: "i"},
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
				t.Fatalf("alias flag --%s should be hidden", tt.alias)
			}
			if alias.Value.Type() != base.Value.Type() {
				t.Fatalf("alias --%s type = %q, want %q", tt.alias, alias.Value.Type(), base.Value.Type())
			}
		})
	}
}

func TestRootFlagParity_ResultsAndItemsOnly(t *testing.T) {
	root := newFlagParityTestRoot(t).RootCommand()

	// --results-only should be hidden (it's an alias for --items-only)
	resultsOnlyFlag := root.PersistentFlags().Lookup("results-only")
	if resultsOnlyFlag == nil {
		t.Fatal("expected --results-only flag")
	}
	if !resultsOnlyFlag.Hidden {
		t.Fatal("--results-only should be hidden")
	}

	if err := root.PersistentFlags().Set("io", "true"); err != nil {
		t.Fatalf("set --io: %v", err)
	}
	resultsOnlyVal, _ := root.PersistentFlags().GetBool("results-only")
	if !resultsOnlyVal {
		t.Fatal("--io should set --results-only")
	}
	itemsOnlyVal, _ := root.PersistentFlags().GetBool("items-only")
	if !itemsOnlyVal {
		t.Fatal("--io should set --items-only")
	}

	if err := root.PersistentFlags().Set("ro", "false"); err != nil {
		t.Fatalf("set --ro: %v", err)
	}
	itemsOnlyVal, _ = root.PersistentFlags().GetBool("items-only")
	if itemsOnlyVal {
		t.Fatal("--ro should update --items-only value")
	}

	if err := root.PersistentFlags().Set("i", "true"); err != nil {
		t.Fatalf("set --i: %v", err)
	}
	itemsOnlyVal, _ = root.PersistentFlags().GetBool("items-only")
	if !itemsOnlyVal {
		t.Fatal("--i should set --items-only")
	}
}

func TestRootFlagParity_QueryFileAlias(t *testing.T) {
	root := newFlagParityTestRoot(t).RootCommand()

	if err := root.PersistentFlags().Set("qf", "-"); err != nil {
		t.Fatalf("set --qf: %v", err)
	}
	queryFileVal, _ := root.PersistentFlags().GetString("query-file")
	if queryFileVal != "-" {
		t.Fatalf("--qf should set --query-file, got %q", queryFileVal)
	}
}

func TestRootFlagParity_NonInteractiveTrio(t *testing.T) {
	root := newFlagParityTestRoot(t).RootCommand()

	yesFlag := root.PersistentFlags().Lookup("yes")
	if yesFlag == nil {
		t.Fatal("expected --yes flag")
	}
	if yesFlag.Shorthand != "y" {
		t.Fatalf("--yes shorthand = %q, want %q", yesFlag.Shorthand, "y")
	}
	noInputFlag := root.PersistentFlags().Lookup("no-input")
	if noInputFlag == nil {
		t.Fatal("expected --no-input flag")
	}
	if !noInputFlag.Hidden {
		t.Fatal("--no-input should be hidden")
	}
	forceFlag := root.PersistentFlags().Lookup("force")
	if forceFlag != nil {
		t.Fatal("--force should not be registered at root (conflicts with page sync --force semantics)")
	}

	if err := root.PersistentFlags().Set("no-input", "true"); err != nil {
		t.Fatalf("set --no-input: %v", err)
	}
	yesVal, _ := root.PersistentFlags().GetBool("yes")
	if !yesVal {
		t.Fatal("--no-input should set --yes")
	}
}
