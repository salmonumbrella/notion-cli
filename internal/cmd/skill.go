// Package cmd contains the CLI commands for notion-cli.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the notion-cli skill file for agents",
		Long: `The skill file provides aliases and context for AI agents using the CLI.

Run 'notion skill init' after authentication to generate a skill file
tailored to your workspace.`,
	}

	cmd.AddCommand(newSkillInitCmd())
	cmd.AddCommand(newSkillSyncCmd())
	cmd.AddCommand(newSkillPathCmd())
	cmd.AddCommand(newSkillEditCmd())

	return cmd
}

func newSkillInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize skill file by scanning your workspace",
		Long: `Scans your Notion workspace and guides you through creating a skill file.

The wizard will:
1. Discover all databases and datasources
2. Ask you to configure aliases for each
3. Discover all users
4. Set up user aliases (including "me" for yourself)
5. Allow custom aliases for frequently-used pages

The generated skill file is saved to ~/.claude/skills/notion-cli/notion-cli.md`,
		RunE: runSkillInit,
	}
}

func newSkillSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Update skill file with current workspace state",
		Long:  `Re-scans your workspace and updates the skill file, preserving existing aliases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement sync
			return fmt.Errorf("not implemented yet")
		},
	}
}

func newSkillPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the skill file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(skill.DefaultPath())
			return nil
		},
	}
}

func newSkillEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open skill file in your editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement edit (open in $EDITOR)
			return fmt.Errorf("not implemented yet")
		},
	}
}

func runSkillInit(cmd *cobra.Command, args []string) error {
	// TODO: Implement the full wizard
	return fmt.Errorf("not implemented yet - will add in next task")
}
