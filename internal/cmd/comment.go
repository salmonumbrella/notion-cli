package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/richtext"
)

func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comment",
		Aliases: []string{"comments", "c"},
		Short:   "Manage Notion comments",
		Long:    `List and create comments on Notion pages and blocks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Desire paths:
			//   ntn comment <page-or-block>       -> list
			//   ntn comment <page> "text..."     -> add (new discussion on page)
			switch len(args) {
			case 0:
				return errors.NewUserError(
					"missing target id",
					"Try:\n  • ntn comment <page-or-block-id>\n  • ntn comment <page-id> \"Comment text\"\n  • ntn comment list <page-or-block-id>\n  • ntn comment add <page-id> --text \"...\"",
				)
			case 1:
				listCmd := newCommentListCmd()
				listCmd.SetContext(cmd.Context())
				return listCmd.RunE(listCmd, args)
			default:
				addCmd := newCommentAddCmd()
				addCmd.SetContext(cmd.Context())
				return addCmd.RunE(addCmd, args)
			}
		},
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
	var light bool

	cmd := &cobra.Command{
		Use:     "list <page-or-block-id-or-name>",
		Aliases: []string{"ls"},
		Short:   "List comments on a page or block",
		Long: `List un-resolved comments from a page or block.

The target can be:
  - a page ID
  - a block ID
  - a Notion URL
  - a skill file alias
  - a page name (resolved via search)

Note: Notion search only finds pages/databases; resolving by name is page-only.
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use --light (or --li) for compact output (id, discussion_id, text, created_by).
Use global --results-only to output just the results array (useful for piping to jq).

Example - List all comments on a page:
  ntn comment list abc123def456
  ntn comment list "Meeting Notes"
  ntn comment list https://www.notion.so/Meeting-Notes-abc123def456

Example - List comments with pagination:
  ntn comment list abc123def456 --page-size 10 --start-cursor cursor123

Example - Fetch all comments:
  ntn comment list abc123def456 --all
  ntn comment list abc123def456 --li

Example - Output only results array:
  ntn comment list abc123def456 --all --results-only`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Get token from context (respects workspace selection)
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve target ID (supports skill alias, URL, and page name via search).
			// Block IDs are UUIDs so they bypass search anyway.
			blockID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			blockID, err = cmdutil.NormalizeNotionID(blockID)
			if err != nil {
				return err
			}

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
						return wrapAPIError(err, "list comments", "block", blockID)
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
				if light {
					return printer.Print(ctx, map[string]interface{}{
						"object":      "list",
						"results":     toLightComments(allComments),
						"has_more":    false,
						"next_cursor": nil,
					})
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
				return wrapAPIError(err, "list comments", "block", blockID)
			}

			if limit > 0 && len(result.Results) > limit {
				result.Results = result.Results[:limit]
				result.HasMore = true
			}

			// Print result
			printer := printerForContext(ctx)
			if light {
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     toLightComments(result.Results),
					"has_more":    result.HasMore,
					"next_cursor": result.NextCursor,
				})
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, discussion_id, text, created_by)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func newCommentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <comment-id>",
		Aliases: []string{"g"},
		Short:   "Get a comment by ID",
		Long: `Retrieve a Notion comment by its ID.

Example:
  ntn comment get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			commentID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			comment, err := client.GetComment(ctx, commentID)
			if err != nil {
				return wrapAPIError(err, "get comment", "comment", commentID)
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
		Use:     "add [page-id-or-name] [text...]",
		Aliases: []string{"create", "a"},
		Short:   "Create a comment",
		Long: `Create a comment on a page or in an existing discussion thread.

You must specify either --parent (to create a new discussion on a page) or
--discussion-id (to add to an existing discussion thread), but not both.

The --text flag is required and contains the comment content.
Use --mention to @-mention users (they will receive notifications).
Use --page-mention to @@-mention pages (link to other Notion pages).

DESIRE PATHS:
You can also use positional args instead of flags:
  ntn comment add <page-id-or-name> "Comment text..."
  ntn comment <page-id-or-name> "Comment text..."

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
  ntn comment add --parent abc123def456 --text "This is my comment"
  ntn comment add abc123def456 "This is my comment"
  ntn comment abc123def456 "This is my comment"

Example - Create comment with formatting:
  ntn comment add --parent abc123def456 --text "This is **bold** and *italic* and ` + "`code`" + `"
  ntn comment add abc123def456 "This is **bold** and *italic* and ` + "`code`" + `"

Example - Create comment with a link:
  ntn comment add --parent abc123def456 --text "Check [Notion docs](https://notion.so) for help"

Example - Create comment with inline user mention:
  ntn comment add --parent abc123def456 --text "Hey @Reviewer, can you review?" --mention reviewer-user-id

Example - Create comment with page mention:
  ntn comment add --parent abc123def456 --text "See @@RelatedPage for context" --page-mention related-page-id

Example - Create comment with both user and page mentions:
  ntn comment add --parent abc123def456 --text "@Alice see @@ProjectPlan for details" \
    --mention alice-user-id --page-mention project-plan-page-id

Example - Add to an existing discussion:
  ntn comment add --discussion-id thread123 --text "Reply to discussion"

Combined example (all flags together):
  ntn comment add --parent abc123def456 \
    --text "@Alice please **review** @@ProjectPlan and check [docs](https://example.com)" \
    --mention alice-user-id \
    --page-mention project-plan-id \
    --verbose`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Positional parsing:
			// - If neither --parent nor --discussion-id is set: args[0] is parent, args[1:] is text (if --text is empty).
			// - If --parent/--discussion-id is set: args[:] is text (if --text is empty).
			if parentID == "" && discussionID == "" && len(args) > 0 {
				parentID = args[0]
				if text == "" && len(args) > 1 {
					text = strings.Join(args[1:], " ")
				}
			} else if (parentID != "" || discussionID != "") && text == "" && len(args) > 0 {
				text = strings.Join(args, " ")
			}

			// Validate that either parent or discussion-id is provided, but not both
			if parentID == "" && discussionID == "" {
				return fmt.Errorf("either --parent/--page or --discussion-id is required")
			}
			if parentID != "" && discussionID != "" {
				return fmt.Errorf("cannot specify both --parent and --discussion-id")
			}

			// Validate that text is provided
			if strings.TrimSpace(text) == "" {
				return fmt.Errorf("--text is required (or provide it positionally)")
			}

			// Resolve user aliases in mentions
			resolvedMentions := make([]string, len(mentions))
			for i, m := range mentions {
				resolvedMentions[i] = resolveUserID(sf, m)
			}
			// Resolve page aliases in page mentions
			resolvedPageMentions := make([]string, len(pageMentions))
			for i, p := range pageMentions {
				resolvedPageMentions[i] = resolveID(sf, p)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve parent page ID (supports skill aliases, URLs, and page name via search).
			if parentID != "" {
				resolved, err := resolveIDWithSearch(ctx, client, sf, parentID, "page")
				if err != nil {
					return err
				}
				normalized, err := cmdutil.NormalizeNotionID(resolved)
				if err != nil {
					return err
				}
				parentID = normalized
			}

			// Normalize/resolve page mention IDs (skill aliases, URLs, page names).
			for i, p := range resolvedPageMentions {
				resolved, err := resolveIDWithSearch(ctx, client, sf, p, "page")
				if err != nil {
					return err
				}
				normalized, err := cmdutil.NormalizeNotionID(resolved)
				if err != nil {
					return err
				}
				resolvedPageMentions[i] = normalized
			}

			// Build rich text with inline mentions
			// @Name patterns in text are replaced with user mention objects using provided user IDs
			// @@Name patterns in text are replaced with page mention objects using provided page IDs
			richTextContent := buildCommentRichTextVerbose(stderrFromContext(ctx), text, resolvedMentions, resolvedPageMentions, verbose, true)

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
				target := parentID
				if target == "" {
					target = discussionID
				}
				return wrapAPIError(err, "create comment", "page", target)
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "Page ID to create comment on (mutually exclusive with --discussion-id)")
	cmd.Flags().StringVar(&parentID, "page", "", "Alias for --parent")
	_ = cmd.Flags().MarkHidden("page")
	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Discussion thread ID to add comment to (mutually exclusive with --parent)")
	cmd.Flags().StringVarP(&text, "text", "t", "", "Comment text (required)")
	cmd.Flags().StringArrayVarP(&mentions, "mention", "m", nil, "User ID(s) to @-mention (repeatable)")
	cmd.Flags().StringArrayVar(&pageMentions, "page-mention", nil, "Page ID(s) to @@-mention (repeatable)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show how markdown was parsed before creating comment")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "text", "t")
	flagAlias(cmd.Flags(), "mention", "m")
	flagAlias(cmd.Flags(), "page-mention", "pm")

	return cmd
}

// buildCommentRichTextVerbose builds rich text from text with mentions, optionally printing
// verbose output about markdown parsing and mention matching. The w parameter specifies where
// verbose output is written (typically os.Stderr in production). If emitWarnings is true,
// warnings are printed when --mention or --page-mention flags are provided but not used.
// Link URL validation warnings are always printed when there are issues, regardless of verbose mode.
func buildCommentRichTextVerbose(w io.Writer, text string, userIDs []string, pageIDs []string, verbose bool, emitWarnings bool) []notion.RichText {
	// Sanitize block-level markdown (fenced code blocks) to inline formatting.
	// Notion comments only support inline formatting; triple backticks would
	// corrupt the inline parser by mispairing backtick characters.
	text = richtext.SanitizeForComments(text)

	// Parse markdown first (for verbose output if enabled)
	tokens := richtext.ParseMarkdown(text)
	if verbose {
		summary := richtext.SummarizeTokens(tokens)
		_, _ = fmt.Fprintln(w, richtext.FormatSummary(summary))
	}

	// Validate link URLs and always show warnings (not gated by verbose)
	linkWarnings := richtext.ValidateLinkURLs(tokens)
	richtext.FormatLinkWarnings(w, linkWarnings)

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
