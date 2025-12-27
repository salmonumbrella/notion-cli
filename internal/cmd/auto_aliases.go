package cmd

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// applyDefaultCommandAliases adds a generated alias to any non-root command that
// has no aliases yet. Aliases are only added when safe among siblings.
func applyDefaultCommandAliases(root *cobra.Command) {
	if root == nil {
		return
	}
	applyDefaultCommandAliasesRecursive(root)
}

func applyDefaultCommandAliasesRecursive(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		if len(sub.Aliases) == 0 {
			if alias := chooseGeneratedAlias(sub); alias != "" {
				sub.Aliases = append(sub.Aliases, alias)
			}
		}
		applyDefaultCommandAliasesRecursive(sub)
	}
}

func chooseGeneratedAlias(cmd *cobra.Command) string {
	if cmd == nil || cmd.Parent() == nil {
		return ""
	}
	for _, candidate := range commandAliasCandidates(cmd.Name()) {
		if aliasAvailableForCommand(cmd, candidate) {
			return candidate
		}
	}
	return ""
}

func aliasAvailableForCommand(cmd *cobra.Command, alias string) bool {
	if cmd == nil || alias == "" || alias == cmd.Name() || commandHasAlias(cmd, alias) {
		return false
	}

	parent := cmd.Parent()
	if parent == nil {
		return false
	}

	for _, sibling := range parent.Commands() {
		if sibling == cmd {
			continue
		}
		if sibling.Name() == alias || commandHasAlias(sibling, alias) {
			return false
		}
	}

	return true
}

func commandAliasCandidates(name string) []string {
	parts := tokenParts(name)
	if len(parts) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	var candidates []string
	add := func(v string) {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		candidates = append(candidates, v)
	}

	// Initialism for multi-part tokens (e.g., add-token -> at).
	if len(parts) > 1 {
		var initials strings.Builder
		for _, p := range parts {
			if p != "" {
				initials.WriteByte(p[0])
			}
		}
		add(initials.String())
	}

	joined := strings.Join(parts, "")
	for n := 1; n < len(joined); n++ {
		add(joined[:n])
	}

	// Fallbacks if all natural prefixes are taken by siblings.
	if joined != "" {
		for i := 2; i <= 9; i++ {
			add(joined[:1] + strconv.Itoa(i))
		}
	}

	return candidates
}

func tokenParts(token string) []string {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "" {
		return nil
	}
	parts := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// applyDefaultFlagShorthands adds hidden shorthand aliases (-x) for visible
// flags that do not already have one, while avoiding collisions in command
// contexts (including ancestor persistent flags and descendant command flags).
func applyDefaultFlagShorthands(root *cobra.Command) {
	if root == nil {
		return
	}
	applyDefaultFlagShorthandsRecursive(root)
}

func applyDefaultFlagShorthandsRecursive(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		applyDefaultFlagShorthandsRecursive(sub)
	}
	assignDefaultLocalShorthands(cmd)
	assignDefaultPersistentShorthands(cmd)
}

func assignDefaultLocalShorthands(cmd *cobra.Command) {
	used := collectUsedShorthandsForCommand(cmd)
	flags := visibleFlagsWithoutShorthand(cmd.LocalFlags())
	assignShorthandAliases(cmd.LocalFlags(), flags, used)
}

func assignDefaultPersistentShorthands(cmd *cobra.Command) {
	used := collectUsedShorthandsForCommand(cmd)
	for shorthand := range collectDescendantShorthands(cmd) {
		used[shorthand] = struct{}{}
	}
	flags := visibleFlagsWithoutShorthand(cmd.PersistentFlags())
	assignShorthandAliases(cmd.PersistentFlags(), flags, used)
}

func collectUsedShorthandsForCommand(cmd *cobra.Command) map[string]struct{} {
	used := map[string]struct{}{
		"h": {}, // Cobra help shorthand
	}

	for p := cmd; p != nil; p = p.Parent() {
		p.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f != nil && f.Shorthand != "" {
				used[f.Shorthand] = struct{}{}
			}
		})
	}

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f != nil && f.Shorthand != "" {
			used[f.Shorthand] = struct{}{}
		}
	})

	return used
}

func collectDescendantShorthands(cmd *cobra.Command) map[string]struct{} {
	used := map[string]struct{}{}
	var walk func(*cobra.Command)
	walk = func(cur *cobra.Command) {
		cur.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f != nil && f.Shorthand != "" {
				used[f.Shorthand] = struct{}{}
			}
		})
		cur.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f != nil && f.Shorthand != "" {
				used[f.Shorthand] = struct{}{}
			}
		})
		for _, child := range cur.Commands() {
			walk(child)
		}
	}
	for _, child := range cmd.Commands() {
		walk(child)
	}
	return used
}

func visibleFlagsWithoutShorthand(fs *pflag.FlagSet) []*pflag.Flag {
	if fs == nil {
		return nil
	}
	var flags []*pflag.Flag
	fs.VisitAll(func(f *pflag.Flag) {
		if f == nil || f.Hidden || f.Shorthand != "" || f.Name == "help" {
			return
		}
		flags = append(flags, f)
	})
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

func assignShorthandAliases(fs *pflag.FlagSet, flags []*pflag.Flag, used map[string]struct{}) {
	if fs == nil {
		return
	}
	for _, f := range flags {
		for _, shorthand := range shorthandCandidates(f.Name) {
			if _, taken := used[shorthand]; taken {
				continue
			}
			if addShorthandAlias(fs, f, shorthand) {
				used[shorthand] = struct{}{}
				break
			}
		}
	}
}

func shorthandCandidates(name string) []string {
	parts := tokenParts(name)
	if len(parts) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	var out []string
	add := func(r rune) {
		r = unicode.ToLower(r)
		if !unicode.IsLetter(r) {
			return
		}
		s := string(r)
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, p := range parts {
		for _, r := range p {
			add(r)
			break
		}
	}

	for _, p := range parts {
		for _, r := range p {
			add(r)
		}
	}

	return out
}

func addShorthandAlias(fs *pflag.FlagSet, base *pflag.Flag, shorthand string) bool {
	if fs == nil || base == nil || shorthand == "" {
		return false
	}
	if fs.Lookup(shorthand) != nil || fs.ShorthandLookup(shorthand) != nil {
		return false
	}
	fs.AddFlag(&pflag.Flag{
		Name:        shorthand,
		Shorthand:   shorthand,
		Usage:       base.Usage,
		Value:       base.Value,
		DefValue:    base.DefValue,
		NoOptDefVal: base.NoOptDefVal,
		Hidden:      true,
	})
	return true
}
