package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestAutoAliases_AllNonRootCommandsHaveAliases(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	var missing []string
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Parent() != nil && len(cmd.Aliases) == 0 {
			missing = append(missing, cmd.CommandPath())
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(root)

	if len(missing) > 0 {
		t.Fatalf("commands missing aliases: %v", missing)
	}
}

func TestAutoAliases_NoSiblingTokenConflicts(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	var walk func(*cobra.Command)
	walk = func(parent *cobra.Command) {
		owners := map[string]string{}
		for _, child := range parent.Commands() {
			tokens := append([]string{child.Name()}, child.Aliases...)
			for _, token := range tokens {
				if token == "" {
					continue
				}
				if existing, ok := owners[token]; ok {
					t.Fatalf("token %q conflicts under %s (%s vs %s)", token, parent.CommandPath(), existing, child.CommandPath())
				}
				owners[token] = child.CommandPath()
			}
		}
		for _, child := range parent.Commands() {
			walk(child)
		}
	}
	walk(root)
}

func TestAutoFlagShorthands_NoPerCommandConflicts(t *testing.T) {
	root := newAliasTestRoot(t).RootCommand()

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		seen := map[string]string{}
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f == nil || f.Shorthand == "" {
				return
			}
			if existing, ok := seen[f.Shorthand]; ok && existing != f.Name {
				t.Fatalf("%s uses -%s for both --%s and --%s", cmd.CommandPath(), f.Shorthand, existing, f.Name)
			}
			seen[f.Shorthand] = f.Name
		})
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)

	// Smoke-check that -j shorthand is registered (via BoolP, not flagAlias).
	if root.PersistentFlags().ShorthandLookup("j") == nil {
		t.Fatal("expected root shorthand -j to be registered")
	}
}
