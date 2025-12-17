package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage Notion comments",
		Long:  `List and create comments on Notion pages and blocks.`,
	}

	cmd.AddCommand(newCommentListCmd())
	cmd.AddCommand(newCommentAddCmd())

	return cmd
}

func newCommentListCmd() *cobra.Command {
	var startCursor string
	var pageSize int

	cmd := &cobra.Command{
		Use:   "list <block-id>",
		Short: "List comments on a page or block",
		Long: `List un-resolved comments from a page or block.

The block-id can be a page ID or block ID.
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.

Example - List all comments on a page:
  notion comment list abc123def456

Example - List comments with pagination:
  notion comment list abc123def456 --page-size 10 --start-cursor cursor123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID := args[0]

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Build options
			opts := &notion.ListCommentsOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			// List comments
			ctx := context.Background()
			result, err := client.ListComments(ctx, blockID, opts)
			if err != nil {
				return fmt.Errorf("failed to list comments: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")

	return cmd
}

func newCommentAddCmd() *cobra.Command {
	var parentID string
	var discussionID string
	var text string
	var mentions []string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a comment",
		Long: `Create a comment on a page or in an existing discussion thread.

You must specify either --parent (to create a new discussion on a page) or
--discussion-id (to add to an existing discussion thread), but not both.

The --text flag is required and contains the comment content.
Use --mention to @-mention users (they will receive notifications).

Example - Create a new comment on a page:
  notion comment add --parent abc123def456 --text "This is my comment"

Example - Create comment with user mention:
  notion comment add --parent abc123def456 --text "Please review" --mention user-id-123

Example - Add to an existing discussion:
  notion comment add --discussion-id thread123 --text "Reply to discussion"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that text is provided
			if text == "" {
				return fmt.Errorf("--text is required")
			}

			// Validate that either parent or discussion-id is provided, but not both
			if parentID == "" && discussionID == "" {
				return fmt.Errorf("either --parent or --discussion-id is required")
			}
			if parentID != "" && discussionID != "" {
				return fmt.Errorf("cannot specify both --parent and --discussion-id")
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Build rich text with mentions
			richText := []notion.RichText{}

			// Add text
			if text != "" {
				richText = append(richText, notion.RichText{
					Type: "text",
					Text: &notion.TextContent{Content: text},
				})
			}

			// Add mentions
			for _, userID := range mentions {
				richText = append(richText, notion.RichText{
					Type: "mention",
					Mention: &notion.Mention{
						Type: "user",
						User: &notion.UserMention{ID: userID},
					},
				})
			}

			// Build request
			req := &notion.CreateCommentRequest{
				RichText: richText,
			}

			if parentID != "" {
				req.Parent = &notion.CommentParent{
					PageID: parentID,
				}
			} else {
				req.DiscussionID = discussionID
			}

			// Create comment
			ctx := context.Background()
			result, err := client.CreateComment(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create comment: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Page ID to create comment on (mutually exclusive with --discussion-id)")
	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Discussion thread ID to add comment to (mutually exclusive with --parent)")
	cmd.Flags().StringVar(&text, "text", "", "Comment text (required)")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "User ID(s) to @-mention (repeatable)")

	return cmd
}
