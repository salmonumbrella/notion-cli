package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
)

func newMCPCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comment",
		Aliases: []string{"cm"},
		Short:   "Manage comments on Notion pages/blocks via MCP",
	}

	cmd.AddCommand(newMCPCommentListCmd())
	cmd.AddCommand(newMCPCommentAddCmd())

	return cmd
}

func newMCPCommentListCmd() *cobra.Command {
	var (
		discussionID     string
		includeAllBlocks bool
		includeResolved  bool
	)

	cmd := &cobra.Command{
		Use:     "list <target-id>",
		Aliases: []string{"ls"},
		Short:   "List comments/discussions for a Notion page or block context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.GetComments(ctx, mcp.GetCommentsParams{
				TargetID:         args[0],
				DiscussionID:     discussionID,
				IncludeAllBlocks: includeAllBlocks,
				IncludeResolved:  includeResolved,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Fetch only a specific discussion ID/URL")
	cmd.Flags().BoolVar(&includeAllBlocks, "include-all-blocks", false, "Include discussions on child blocks")
	cmd.Flags().BoolVar(&includeResolved, "include-resolved", false, "Include resolved discussions")
	return cmd
}

func newMCPCommentAddCmd() *cobra.Command {
	var (
		discussionID string
		selection    string
	)

	cmd := &cobra.Command{
		Use:     "add <target-id> <text>",
		Aliases: []string{"a"},
		Short:   "Add a comment to a Notion page/block context",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureMutuallyExclusiveFlags("--discussion-id", discussionID != "", "--selection", selection != ""); err != nil {
				return err
			}

			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.CreateComment(ctx, mcp.CreateCommentParams{
				TargetID:              args[0],
				Text:                  args[1],
				DiscussionID:          discussionID,
				SelectionWithEllipsis: selection,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&selection, "selection", "", "Selection-with-ellipsis to target specific content")
	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Reply to an existing discussion ID/URL")
	return cmd
}
