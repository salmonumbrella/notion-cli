package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
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
	cmd.AddCommand(newCommentGetCmd())

	return cmd
}

func newCommentListCmd() *cobra.Command {
	var startCursor string
	var pageSize int
	var all bool
	var resultsOnly bool

	cmd := &cobra.Command{
		Use:   "list <block-id>",
		Short: "List comments on a page or block",
		Long: `List un-resolved comments from a page or block.

The block-id can be a page ID or block ID.
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use --results-only to output just the results array (useful for piping to jq).

Example - List all comments on a page:
  notion comment list abc123def456

Example - List comments with pagination:
  notion comment list abc123def456 --page-size 10 --start-cursor cursor123

Example - Fetch all comments:
  notion comment list abc123def456 --all

Example - Output only results array:
  notion comment list abc123def456 --all --results-only`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			format := output.FormatFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

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
				printer := printerForContext(ctx)
				if resultsOnly || format == output.FormatTable {
					return printer.Print(ctx, allComments)
				}
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     allComments,
					"has_more":    false,
					"next_cursor": nil,
				})
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
			printer := printerForContext(ctx)
			if resultsOnly || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")

	return cmd
}

func newCommentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <comment-id>",
		Short: "Get a comment by ID",
		Long: `Retrieve a Notion comment by its ID.

Example:
  notion comment get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			comment, err := client.GetComment(ctx, commentID)
			if err != nil {
				return fmt.Errorf("failed to get comment: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, comment)
		},
	}
}

func newCommentAddCmd() *cobra.Command {
	var parentID string
	var discussionID string
	var text string
	var mentions []string
	var pageMentions []string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a comment",
		Long: `Create a comment on a page or in an existing discussion thread.

You must specify either --parent (to create a new discussion on a page) or
--discussion-id (to add to an existing discussion thread), but not both.

The --text flag is required and contains the comment content.
Use --mention to @-mention users (they will receive notifications).
Use --page-mention to @@-mention pages (link to other Notion pages).

MARKDOWN FORMATTING:
The --text flag supports markdown formatting:
  **bold**       - Bold text
  *italic*       - Italic text (also _italic_)
  ` + "`code`" + `         - Inline code
  ***both***     - Bold and italic combined
  [text](url)    - Hyperlink

MENTIONS:
  @Name patterns are replaced with user mentions using --mention IDs.
  @@Name patterns are replaced with page mentions using --page-mention IDs.

Example - Create a new comment on a page:
  notion comment add --parent abc123def456 --text "This is my comment"

Example - Create comment with formatting:
  notion comment add --parent abc123def456 --text "This is **bold** and *italic* and ` + "`code`" + `"

Example - Create comment with a link:
  notion comment add --parent abc123def456 --text "Check [Notion docs](https://notion.so) for help"

Example - Create comment with inline user mention:
  notion comment add --parent abc123def456 --text "Hey @Georges, can you review?" --mention georges-user-id

Example - Create comment with page mention:
  notion comment add --parent abc123def456 --text "See @@RelatedPage for context" --page-mention related-page-id

Example - Create comment with both user and page mentions:
  notion comment add --parent abc123def456 --text "@Alice see @@ProjectPlan for details" \
    --mention alice-user-id --page-mention project-plan-page-id

Example - Add to an existing discussion:
  notion comment add --discussion-id thread123 --text "Reply to discussion"

Combined example (all flags together):
  notion comment add --parent abc123def456 \
    --text "@Alice please **review** @@ProjectPlan and check [docs](https://example.com)" \
    --mention alice-user-id \
    --page-mention project-plan-id \
    --verbose`,
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
				normalized, err := cmdutil.NormalizeNotionID(parentID)
				if err != nil {
					return err
				}
				parentID = normalized
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Build rich text with inline mentions
			// @Name patterns in text are replaced with user mention objects using provided user IDs
			// @@Name patterns in text are replaced with page mention objects using provided page IDs
			richTextContent := buildCommentRichTextVerbose(stderrFromContext(ctx), text, mentions, pageMentions, verbose, true)

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
			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Page ID to create comment on (mutually exclusive with --discussion-id)")
	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Discussion thread ID to add comment to (mutually exclusive with --parent)")
	cmd.Flags().StringVar(&text, "text", "", "Comment text (required)")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "User ID(s) to @-mention (repeatable)")
	cmd.Flags().StringArrayVar(&pageMentions, "page-mention", nil, "Page ID(s) to @@-mention (repeatable)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show how markdown was parsed before creating comment")

	return cmd
}

// buildCommentRichTextVerbose builds rich text from text with mentions, optionally printing
// verbose output about markdown parsing and mention matching. The w parameter specifies where
// verbose output is written (typically os.Stderr in production). If emitWarnings is true,
// warnings are printed when --mention or --page-mention flags are provided but not used.
func buildCommentRichTextVerbose(w io.Writer, text string, userIDs []string, pageIDs []string, verbose bool, emitWarnings bool) []notion.RichText {
	// Parse markdown first (for verbose output if enabled)
	tokens := richtext.ParseMarkdown(text)
	if verbose {
		summary := richtext.SummarizeTokens(tokens)
		_, _ = fmt.Fprintln(w, richtext.FormatSummary(summary))
	}

	// Count @Name patterns to match with user IDs (excluding those in @@Name patterns)
	userMentionsNeeded := richtext.CountUserMentionsOnly(text)
	// Count @@Name patterns to match with page IDs
	pageMentionsNeeded := richtext.CountPageMentions(text)

	if verbose {
		richtext.FormatAllMentionMappings(w, text, userIDs, pageIDs)
	}

	// Emit warnings about unused --mention flags if requested
	if emitWarnings && len(userIDs) > 0 {
		if userMentionsNeeded == 0 {
			_, _ = fmt.Fprintf(w, "warning: %d --mention flag(s) provided but no @Name patterns found in text\n", len(userIDs))
		} else if userMentionsNeeded < len(userIDs) {
			_, _ = fmt.Fprintf(w, "warning: %d of %d --mention flag(s) unused (not enough @Name patterns)\n", len(userIDs)-userMentionsNeeded, len(userIDs))
		}
	}

	// Emit warnings about unused --page-mention flags if requested
	if emitWarnings && len(pageIDs) > 0 {
		if pageMentionsNeeded == 0 {
			_, _ = fmt.Fprintf(w, "warning: %d --page-mention flag(s) provided but no @@Name patterns found in text\n", len(pageIDs))
		} else if pageMentionsNeeded < len(pageIDs) {
			_, _ = fmt.Fprintf(w, "warning: %d of %d --page-mention flag(s) unused (not enough @@Name patterns)\n", len(pageIDs)-pageMentionsNeeded, len(pageIDs))
		}
	}

	return richtext.BuildWithMentionsFromTokens(tokens, userIDs, pageIDs)
}
