package cmd

import "github.com/spf13/cobra"

type canonicalAliasRule struct {
	token   string
	aliases []string
}

var canonicalVerbAliasRules = []canonicalAliasRule{
	{token: "list", aliases: []string{"ls"}},
	{token: "get", aliases: []string{"g"}},
	{token: "show", aliases: []string{"g"}},
	{token: "create", aliases: []string{"mk", "cr"}},
	{token: "update", aliases: []string{"up"}},
	{token: "edit", aliases: []string{"up"}},
	{token: "delete", aliases: []string{"rm", "del"}},
	{token: "remove", aliases: []string{"rm"}},
	{token: "search", aliases: []string{"q"}},
	{token: "query", aliases: []string{"q"}},
	{token: "find", aliases: []string{"q"}},
}

func applyCanonicalVerbAliases(root *cobra.Command) {
	if root == nil {
		return
	}
	applyCanonicalVerbAliasesRecursive(root)
}

func applyCanonicalVerbAliasesRecursive(cmd *cobra.Command) {
	addCanonicalVerbAliases(cmd)
	for _, sub := range cmd.Commands() {
		applyCanonicalVerbAliasesRecursive(sub)
	}
}

func addCanonicalVerbAliases(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	for _, rule := range canonicalVerbAliasRules {
		if !commandHasNameOrAlias(cmd, rule.token) {
			continue
		}
		for _, alias := range rule.aliases {
			addCommandAliasIfSafe(cmd, alias)
		}
	}
}

func addCommandAliasIfSafe(cmd *cobra.Command, alias string) {
	if cmd == nil || alias == "" || alias == cmd.Name() || commandHasAlias(cmd, alias) {
		return
	}

	parent := cmd.Parent()
	if parent != nil {
		for _, sibling := range parent.Commands() {
			if sibling == cmd {
				continue
			}
			if sibling.Name() == alias || commandHasAlias(sibling, alias) {
				return
			}
		}
	}

	cmd.Aliases = append(cmd.Aliases, alias)
}

func commandHasNameOrAlias(cmd *cobra.Command, token string) bool {
	if cmd == nil || token == "" {
		return false
	}
	if cmd.Name() == token {
		return true
	}
	return commandHasAlias(cmd, token)
}

func commandHasAlias(cmd *cobra.Command, alias string) bool {
	if cmd == nil {
		return false
	}
	for _, existing := range cmd.Aliases {
		if existing == alias {
			return true
		}
	}
	return false
}
