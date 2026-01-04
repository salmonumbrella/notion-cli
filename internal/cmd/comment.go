package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

// mentionPattern matches @Name patterns in text (alphanumeric, hyphens, underscores)
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]+)`)

// markdownToken represents a parsed markdown segment with its formatting
type markdownToken struct {
	content string
	bold    bool
	italic  bool
	code    bool
}

// parseMarkdown parses text for markdown patterns and returns tokens.
// Supports: **bold**, *italic*, _italic_, `code`, and combinations.
// Unmatched markers are treated as literal text.
func parseMarkdown(text string) []markdownToken {
	if text == "" {
		return nil
	}

	var tokens []markdownToken
	remaining := text

	for len(remaining) > 0 {
		// Find the earliest markdown pattern
		earliest := -1
		var matched string
		var tokenContent string
		var bold, italic, code bool

		// Check for code first (highest priority, doesn't nest)
		if idx := strings.Index(remaining, "`"); idx != -1 {
			endIdx := strings.Index(remaining[idx+1:], "`")
			if endIdx != -1 {
				if earliest == -1 || idx < earliest {
					earliest = idx
					tokenContent = remaining[idx+1 : idx+1+endIdx]
					matched = remaining[idx : idx+1+endIdx+1]
					code = true
					bold = false
					italic = false
				}
			}
		}

		// Check for bold+italic (***text*** or ___text___)
		for _, marker := range []string{"***", "___"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+3:], marker)
				if endIdx != -1 {
					earliest = idx
					tokenContent = remaining[idx+3 : idx+3+endIdx]
					matched = remaining[idx : idx+3+endIdx+3]
					bold = true
					italic = true
					code = false
				}
			}
		}

		// Check for bold (**text** or __text__)
		for _, marker := range []string{"**", "__"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+2:], marker)
				if endIdx != -1 {
					earliest = idx
					tokenContent = remaining[idx+2 : idx+2+endIdx]
					matched = remaining[idx : idx+2+endIdx+2]
					bold = true
					italic = false
					code = false
				}
			}
		}

		// Check for italic (*text* or _text_) - must not be ** or __
		for _, marker := range []string{"*", "_"} {
			doubleMarker := marker + marker
			idx := strings.Index(remaining, marker)
			// Skip if this is actually a double marker
			if idx != -1 && strings.HasPrefix(remaining[idx:], doubleMarker) {
				// Find next single marker that isn't part of a double
				searchFrom := idx + 1
				for searchFrom < len(remaining) {
					nextIdx := strings.Index(remaining[searchFrom:], marker)
					if nextIdx == -1 {
						idx = -1
						break
					}
					actualIdx := searchFrom + nextIdx
					// Check if this is part of a double marker
					if actualIdx > 0 && string(remaining[actualIdx-1]) == marker {
						searchFrom = actualIdx + 1
						continue
					}
					if actualIdx+1 < len(remaining) && string(remaining[actualIdx+1]) == marker {
						searchFrom = actualIdx + 2
						continue
					}
					idx = actualIdx
					break
				}
				if idx == -1 {
					continue
				}
			}
			if idx != -1 && (earliest == -1 || idx < earliest) {
				// Find closing marker that isn't part of a double
				searchEnd := idx + 1
				endIdx := -1
				for searchEnd < len(remaining) {
					nextEnd := strings.Index(remaining[searchEnd:], marker)
					if nextEnd == -1 {
						break
					}
					actualEnd := searchEnd + nextEnd
					// Check if this closing marker is part of a double
					if actualEnd+1 < len(remaining) && string(remaining[actualEnd+1]) == marker {
						searchEnd = actualEnd + 2
						continue
					}
					if actualEnd > 0 && string(remaining[actualEnd-1]) == marker {
						searchEnd = actualEnd + 1
						continue
					}
					endIdx = actualEnd - idx - 1
					break
				}
				if endIdx > 0 {
					earliest = idx
					tokenContent = remaining[idx+1 : idx+1+endIdx]
					matched = remaining[idx : idx+1+endIdx+1]
					bold = false
					italic = true
					code = false
				}
			}
		}

		if earliest == -1 {
			// No more markdown patterns, add remaining as plain text
			tokens = append(tokens, markdownToken{content: remaining})
			break
		}

		// Add text before the pattern
		if earliest > 0 {
			tokens = append(tokens, markdownToken{content: remaining[:earliest]})
		}

		// Add the formatted token
		tokens = append(tokens, markdownToken{
			content: tokenContent,
			bold:    bold,
			italic:  italic,
			code:    code,
		})

		remaining = remaining[earliest+len(matched):]
	}

	return tokens
}

// createAnnotations creates a Notion Annotations object from formatting flags.
// Returns nil if all formatting is default (allows omitempty to work).
func createAnnotations(bold, italic, code bool) *notion.Annotations {
	if !bold && !italic && !code {
		return nil
	}
	return &notion.Annotations{
		Bold:          bold,
		Italic:        italic,
		Strikethrough: false,
		Underline:     false,
		Code:          code,
		Color:         "default",
	}
}

// buildRichTextWithMentions parses text for markdown formatting and @Name patterns,
// replacing them with properly formatted rich text and mention objects.
// Supports: **bold**, *italic*, _italic_, `code`, ***bold italic***, and @mentions.
// Returns the rich_text array with interleaved text and mention objects.
func buildRichTextWithMentions(text string, userIDs []string) []notion.RichText {
	if text == "" && len(userIDs) == 0 {
		return []notion.RichText{}
	}

	// First, parse markdown to get formatted tokens
	tokens := parseMarkdown(text)
	if len(tokens) == 0 && len(userIDs) == 0 {
		return []notion.RichText{}
	}

	// Now process each token, looking for @mentions within them
	var richText []notion.RichText
	userIDIndex := 0

	for _, token := range tokens {
		// Check for @mentions within this token's content
		matches := mentionPattern.FindAllStringIndex(token.content, -1)

		if len(matches) == 0 {
			// No mentions in this token, add it directly with its formatting
			if token.content != "" {
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.content},
					Annotations: createAnnotations(token.bold, token.italic, token.code),
				})
			}
			continue
		}

		// Process mentions within this formatted token
		lastEnd := 0
		for _, match := range matches {
			start, end := match[0], match[1]

			// Add text before this mention (with the token's formatting)
			if start > lastEnd {
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.content[lastEnd:start]},
					Annotations: createAnnotations(token.bold, token.italic, token.code),
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
				// No more user IDs, keep the @Name as plain text with formatting
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.content[start:end]},
					Annotations: createAnnotations(token.bold, token.italic, token.code),
				})
			}

			lastEnd = end
		}

		// Add remaining text after the last mention (with formatting)
		if lastEnd < len(token.content) {
			richText = append(richText, notion.RichText{
				Type:        "text",
				Text:        &notion.TextContent{Content: token.content[lastEnd:]},
				Annotations: createAnnotations(token.bold, token.italic, token.code),
			})
		}
	}

	// If there are extra user IDs (no matching @Name patterns), append them at the end
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
			blockID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			if limit > 0 && (pageSize == 0 || pageSize > limit) {
				pageSize = limit
			}

			if pageSize > 100 {
				return fmt.Errorf("page-size must be between 1 and 100")
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
