package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

// mentionPattern matches @Name patterns in text (alphanumeric, hyphens, underscores)
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]+)`)

// buildRichTextWithMentions parses text for @Name patterns and replaces them with
// mention objects using the provided user IDs in order. Returns the rich_text array
// with interleaved text and mention objects.
func buildRichTextWithMentions(text string, userIDs []string) []notion.RichText {
	if len(userIDs) == 0 {
		// No mentions to process, return plain text
		if text == "" {
			return []notion.RichText{}
		}
		return []notion.RichText{
			{
				Type: "text",
				Text: &notion.TextContent{Content: text},
			},
		}
	}

	matches := mentionPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// No @Name patterns found, append mentions at the end (legacy behavior)
		richText := []notion.RichText{}
		if text != "" {
			richText = append(richText, notion.RichText{
				Type: "text",
				Text: &notion.TextContent{Content: text},
			})
		}
		for _, userID := range userIDs {
			richText = append(richText, notion.RichText{
				Type: "mention",
				Mention: &notion.Mention{
					Type: "user",
					User: &notion.UserMention{ID: userID},
				},
			})
		}
		return richText
	}

	// Build rich text with inline mentions
	richText := []notion.RichText{}
	lastEnd := 0
	userIDIndex := 0

	for _, match := range matches {
		start, end := match[0], match[1]

		// Add text before this mention
		if start > lastEnd {
			richText = append(richText, notion.RichText{
				Type: "text",
				Text: &notion.TextContent{Content: text[lastEnd:start]},
			})
		}

		// Add mention if we have a user ID for it
		if userIDIndex < len(userIDs) {
			richText = append(richText, notion.RichText{
				Type: "mention",
				Mention: &notion.Mention{
					Type: "user",
					User: &notion.UserMention{ID: userIDs[userIDIndex]},
				},
			})
			userIDIndex++
		} else {
			// No more user IDs, keep the @Name as plain text
			richText = append(richText, notion.RichText{
				Type: "text",
				Text: &notion.TextContent{Content: text[start:end]},
			})
		}

		lastEnd = end
	}

	// Add remaining text after the last mention
	if lastEnd < len(text) {
		richText = append(richText, notion.RichText{
			Type: "text",
			Text: &notion.TextContent{Content: text[lastEnd:]},
		})
	}

	// If there are extra user IDs, append them at the end
	for ; userIDIndex < len(userIDs); userIDIndex++ {
		richText = append(richText, notion.RichText{
			Type: "mention",
			Mention: &notion.Mention{
				Type: "user",
				User: &notion.UserMention{ID: userIDs[userIDIndex]},
			},
		})
	}

	return richText
}

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
	var all bool

	cmd := &cobra.Command{
		Use:   "list <block-id>",
		Short: "List comments on a page or block",
		Long: `List un-resolved comments from a page or block.

The block-id can be a page ID or block ID.
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.

Example - List all comments on a page:
  notion comment list abc123def456

Example - List comments with pagination:
  notion comment list abc123def456 --page-size 10 --start-cursor cursor123

Example - Fetch all comments:
  notion comment list abc123def456 --all`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID := args[0]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// If --all flag is set, fetch all pages
			if all {
				var allComments []*notion.Comment
				cursor := startCursor

				for {
					opts := &notion.ListCommentsOptions{
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					result, err := client.ListComments(ctx, blockID, opts)
					if err != nil {
						return fmt.Errorf("failed to list comments: %w", err)
					}

					allComments = append(allComments, result.Results...)

					if !result.HasMore || result.NextCursor == nil || *result.NextCursor == "" {
						break
					}
					cursor = *result.NextCursor
				}

				// Print all results
				printer := output.NewPrinter(os.Stdout, GetOutputFormat())
				return printer.Print(ctx, allComments)
			}

			// Single page request
			opts := &notion.ListCommentsOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

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
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")

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

When @Name patterns appear in --text, they are replaced with mentions in order.
For example, "Hey @Georges" with --mention user-id will replace @Georges with
a proper mention object at that position.

Example - Create a new comment on a page:
  notion comment add --parent abc123def456 --text "This is my comment"

Example - Create comment with inline user mention:
  notion comment add --parent abc123def456 --text "Hey @Georges, can you review?" --mention georges-user-id

Example - Create comment with multiple mentions:
  notion comment add --parent abc123def456 --text "@Alice and @Bob please review" --mention alice-id --mention bob-id

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

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Build rich text with inline mentions
			// @Name patterns in text are replaced with mention objects using provided user IDs
			richText := buildRichTextWithMentions(text, mentions)

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
