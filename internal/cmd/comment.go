package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/richtext"
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
			blockID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

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
					if limit > 0 && len(allComments) >= limit {
						allComments = allComments[:limit]
						break
					}

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

			if limit > 0 && len(result.Results) > limit {
				result.Results = result.Results[:limit]
				result.HasMore = true
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
	var verbose bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a comment",
		Long: `Create a comment on a page or in an existing discussion thread.

You must specify either --parent (to create a new discussion on a page) or
--discussion-id (to add to an existing discussion thread), but not both.

The --text flag is required and contains the comment content.
Use --mention to @-mention users (they will receive notifications).

MARKDOWN FORMATTING:
The --text flag supports markdown formatting:
  **bold**     - Bold text
  *italic*     - Italic text (also _italic_)
  ` + "`code`" + `       - Inline code
  ***both***   - Bold and italic combined

When @Name patterns appear in --text, they are replaced with mentions in order.
For example, "Hey @Georges" with --mention user-id will replace @Georges with
a proper mention object at that position.

Example - Create a new comment on a page:
  notion comment add --parent abc123def456 --text "This is my comment"

Example - Create comment with formatting:
  notion comment add --parent abc123def456 --text "This is **bold** and *italic* and ` + "`code`" + `"

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
			if parentID != "" {
				normalized, err := normalizeNotionID(parentID)
				if err != nil {
					return err
				}
				parentID = normalized
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
			richTextContent := buildCommentRichTextVerbose(os.Stderr, text, mentions, verbose)

			// Build request
			req := &notion.CreateCommentRequest{
				RichText: richTextContent,
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
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show how markdown was parsed before creating comment")

	return cmd
}

// buildCommentRichTextVerbose builds rich text from text with mentions, optionally printing
// verbose output about markdown parsing and mention matching. The w parameter specifies where
// verbose output is written (typically os.Stderr in production).
func buildCommentRichTextVerbose(w io.Writer, text string, userIDs []string, verbose bool) []notion.RichText {
	// Parse markdown first (for verbose output if enabled)
	tokens := richtext.ParseMarkdown(text)
	if verbose {
		summary := richtext.SummarizeTokens(tokens)
		_, _ = fmt.Fprintln(w, richtext.FormatSummary(summary))
	}

	// Count @Name patterns to match with user IDs
	matches := richtext.MentionPattern.FindAllStringIndex(text, -1)
	mentionsNeeded := len(matches)

	if verbose && mentionsNeeded > 0 {
		mentionMatches := richtext.MentionPattern.FindAllString(text, -1)
		_, _ = fmt.Fprintf(w, "Mentions:\n")
		for i, name := range mentionMatches {
			if i < len(userIDs) {
				_, _ = fmt.Fprintf(w, "  %s → %s\n", name, userIDs[i])
			} else {
				_, _ = fmt.Fprintf(w, "  %s → (no user ID available)\n", name)
			}
		}
	}

	return richtext.BuildWithMentionsFromTokens(tokens, userIDs)
}
