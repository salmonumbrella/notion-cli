package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

// Column layout constraints defined by the Notion API.
const (
	MinColumnCount = 2
	MaxColumnCount = 5
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "block",
		Aliases: []string{"blocks", "b"},
		Short:   "Manage Notion blocks",
		Long:    `Retrieve, append, update, and delete Notion blocks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// When invoked without subcommand, default to children
			// Note: children requires a block-id argument
			childrenCmd := newBlockChildrenCmd()
			childrenCmd.SetContext(cmd.Context())
			return childrenCmd.RunE(childrenCmd, args)
		},
	}

	cmd.AddCommand(newBlockGetCmd())
	cmd.AddCommand(newBlockChildrenCmd())
	cmd.AddCommand(newBlockAppendCmd())
	cmd.AddCommand(newBlockUpdateCmd())
	cmd.AddCommand(newBlockDeleteCmd())
	cmd.AddCommand(newBlockAddCmd())
	cmd.AddCommand(newBlockAddTOCCmd())
	cmd.AddCommand(newBlockAddBreadcrumbCmd())
	cmd.AddCommand(newBlockAddDividerCmd())
	cmd.AddCommand(newBlockAddColumnsCmd())

	return cmd
}

func newBlockGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <block-id-or-name>",
		Aliases: []string{"g"},
		Short:   "Get a block by ID or name",
		Long: `Retrieve a Notion block by its ID or name.

If you provide a name instead of an ID, the CLI will search for matching pages/blocks.

Example:
  ntn block get 12345678-1234-1234-1234-123456789012
  ntn block get "Meeting Notes"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve ID with search fallback (no filter - blocks can be pages too)
			blockID, err := resolveIDWithSearch(ctx, client, sf, args[0], "")
			if err != nil {
				return err
			}
			blockID, err = cmdutil.NormalizeNotionID(blockID)
			if err != nil {
				return err
			}

			// Get block
			block, err := client.GetBlock(ctx, blockID)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, block)
		},
	}
}

func newBlockChildrenCmd() *cobra.Command {
	var startCursor string
	var pageSize int
	var all bool
	var depth int
	var plain bool

	cmd := &cobra.Command{
		Use:     "children <block-id-or-name>",
		Aliases: []string{"list", "ls"},
		Short:   "Get block children",
		Long: `Retrieve the children of a block.

If you provide a name instead of an ID, the CLI will search for matching pages.

Use the --start-cursor flag to paginate through results.
Use the --page-size flag to control the number of results per page (max 100).
Use --all to fetch all pages of results automatically.
Use --depth to recursively fetch nested children (e.g., content inside toggles, columns).
Note: global --items-only (alias --ro) outputs a bare array, so jq paths should use '.[]' instead of '.results[]'.

Example:
  ntn block children 12345678-1234-1234-1234-123456789012
  ntn block children "Meeting Notes"
  ntn block children 12345678-1234-1234-1234-123456789012 --page-size 50
  ntn block children 12345678-1234-1234-1234-123456789012 --all
  ntn block children 12345678-1234-1234-1234-123456789012 --depth 3 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Get token from context (respects workspace selection)
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve ID with search fallback
			blockID, err := resolveIDWithSearch(ctx, client, sf, args[0], "")
			if err != nil {
				return err
			}
			blockID, err = cmdutil.NormalizeNotionID(blockID)
			if err != nil {
				return err
			}

			// If --depth flag is set, use recursive fetching
			if depth > 0 {
				opts := &notion.BlockChildrenOptions{
					StartCursor: startCursor,
					PageSize:    pageSize,
				}

				blocks, err := client.GetBlockChildrenRecursive(ctx, blockID, depth, opts)
				if err != nil {
					return wrapAPIError(err, "access block", "block", args[0])
				}

				if limit > 0 && len(blocks) > limit {
					blocks = blocks[:limit]
				}

				printer := printerForContext(ctx)
				if plain {
					return printer.Print(ctx, simplifyBlocks(blocks))
				}
				return printer.Print(ctx, blocks)
			}

			// If --all flag is set, fetch all pages
			if all {
				var allBlocks []notion.Block
				cursor := startCursor

				for {
					opts := &notion.BlockChildrenOptions{
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					blockList, err := client.GetBlockChildren(ctx, blockID, opts)
					if err != nil {
						return wrapAPIError(err, "access block", "block", args[0])
					}

					allBlocks = append(allBlocks, blockList.Results...)
					if limit > 0 && len(allBlocks) >= limit {
						allBlocks = allBlocks[:limit]
						break
					}

					if !blockList.HasMore || blockList.NextCursor == nil || *blockList.NextCursor == "" {
						break
					}
					cursor = *blockList.NextCursor
				}

				// Print all results
				printer := printerForContext(ctx)
				if plain {
					return printer.Print(ctx, simplifyBlocks(allBlocks))
				}
				return printer.Print(ctx, allBlocks)
			}

			// Single page request
			opts := &notion.BlockChildrenOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			blockList, err := client.GetBlockChildren(ctx, blockID, opts)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			if limit > 0 && len(blockList.Results) > limit {
				blockList.Results = blockList.Results[:limit]
				blockList.HasMore = true
			}

			// Print result
			printer := printerForContext(ctx)
			if plain {
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     simplifyBlocks(blockList.Results),
					"has_more":    blockList.HasMore,
					"next_cursor": blockList.NextCursor,
				})
			}
			return printer.Print(ctx, blockList)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Recursively fetch nested children up to this depth (0 = direct children only)")
	cmd.Flags().BoolVar(&plain, "plain", false, "Output simplified blocks (id, type, text, children)")

	return cmd
}

func newBlockAppendCmd() *cobra.Command {
	var childrenJSON string
	var childrenFile string
	var afterBlockID string
	var blockType string
	var content string
	var markdownContent string

	cmd := &cobra.Command{
		Use:     "append <block-id>",
		Aliases: []string{"ap"},
		Short:   "Append children to a block",
		Long: `Append child blocks to a parent block.

SIMPLE USAGE (--type and --content):
  ntn block append PAGE_ID --type paragraph --content "Hello world"
  ntn block append PAGE_ID --type heading_2 --content "Section Title"
  ntn block append PAGE_ID --type bulleted_list_item --content "List item"

Supported types: paragraph, heading_1, heading_2, heading_3, bulleted_list_item,
numbered_list_item, quote, callout, code, to_do, toggle, divider

MARKDOWN USAGE (--md):
  ntn block append PAGE_ID --md '# Heading\n\nA paragraph with **bold** text.\n\n- Bullet one\n- Bullet two'
  ntn block append PAGE_ID --md @content.md
  cat content.md | ntn block append PAGE_ID --md -

ADVANCED USAGE (--children JSON):
  ntn block append PAGE_ID \
    --children '[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}]'

ADVANCED USAGE (--children-file):
  ntn block append PAGE_ID --children-file /tmp/blocks.json

Use --after to insert blocks after a specific block instead of at the end.

TIP: For convenience commands, see 'ntn block add --help'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			blockID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Handle convenience flags --type and --content
			if blockType != "" && content != "" {
				block := buildSimpleBlock(blockType, content)
				if block == nil {
					return fmt.Errorf("unsupported block type %q for simple mode\nSupported: paragraph, heading_1, heading_2, heading_3, bulleted_list_item, numbered_list_item, quote, callout, code, to_do, toggle, divider", blockType)
				}
				marshaled, err := marshalJSON([]map[string]interface{}{block})
				if err != nil {
					return err
				}
				childrenJSON = marshaled
			} else if blockType != "" && content == "" {
				// Special case for content-less blocks
				if blockType == "divider" {
					block := notion.NewDivider()
					marshaled, err := marshalJSON([]map[string]interface{}{block})
					if err != nil {
						return err
					}
					childrenJSON = marshaled
				} else {
					return fmt.Errorf("--content is required when using --type (except for divider)")
				}
			} else if blockType == "" && content != "" {
				return fmt.Errorf("--type is required when using --content")
			}

			// Handle --md flag: parse markdown to blocks
			if markdownContent != "" {
				if childrenJSON != "" || childrenFile != "" || blockType != "" {
					return fmt.Errorf("--md cannot be combined with --children, --children-file, or --type")
				}
				// Support @file, stdin (-), or inline string
				mdText := markdownContent
				if strings.HasPrefix(mdText, "@") {
					data, err := os.ReadFile(strings.TrimPrefix(mdText, "@"))
					if err != nil {
						return fmt.Errorf("reading markdown file: %w", err)
					}
					mdText = string(data)
				} else if mdText == "-" {
					data, err := readMarkdownFile("-")
					if err != nil {
						return fmt.Errorf("reading markdown from stdin: %w", err)
					}
					mdText = data
				}
				blocks := parseMarkdownToBlocks(mdText)
				if len(blocks) == 0 {
					return fmt.Errorf("no blocks parsed from markdown content")
				}
				marshaled, err := marshalJSON(blocks)
				if err != nil {
					return err
				}
				childrenJSON = marshaled
			}

			if childrenFile != "" && childrenJSON != "" {
				return fmt.Errorf("use only one of --children or --children-file")
			}

			// Validate required flag
			if childrenJSON == "" && childrenFile == "" {
				return fmt.Errorf("either --children/--children-file, --md, or both --type and --content are required\n\nMarkdown usage:\n  ntn block append PAGE_ID --md '# Heading\\nParagraph text'\n\nSimple usage:\n  ntn block append PAGE_ID --type paragraph --content \"Your text\"\n\nAdvanced usage:\n  ntn block append PAGE_ID --children '[{\"type\":\"paragraph\",...}]'\n  ntn block append PAGE_ID --children-file /tmp/blocks.json")
			}

			// Normalize after block ID if provided
			if afterBlockID != "" {
				afterBlockID, err = cmdutil.NormalizeNotionID(resolveID(sf, afterBlockID))
				if err != nil {
					return fmt.Errorf("invalid --after block ID: %w", err)
				}
			}

			// Resolve and parse children JSON
			var children []map[string]interface{}
			resolved, err := cmdutil.ResolveJSONInput(childrenJSON, childrenFile)
			if err != nil {
				return err
			}
			childrenJSON = resolved

			// Try to unmarshal as array first, then as single object
			if err := cmdutil.UnmarshalJSONInput(childrenJSON, &children); err != nil {
				// If array unmarshal fails, try as single object and wrap in array
				var singleBlock map[string]interface{}
				if singleErr := cmdutil.UnmarshalJSONInput(childrenJSON, &singleBlock); singleErr == nil {
					children = []map[string]interface{}{singleBlock}
				} else {
					return fmt.Errorf("failed to parse children JSON (expected array or single block object): %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Append children in batches (Notion API limit is 100 blocks per request).
			blockList, err := appendBlockChildrenBatched(ctx, client, blockID, children, afterBlockID)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, blockList)
		},
	}

	cmd.Flags().StringVar(&childrenJSON, "children", "", "Children blocks as JSON array")
	cmd.Flags().StringVar(&childrenFile, "children-file", "", "Read children JSON from file (or - for stdin)")
	cmd.Flags().StringVar(&markdownContent, "md", "", "Markdown content to parse and append as blocks (supports @file and - for stdin)")
	cmd.Flags().StringVar(&afterBlockID, "after", "", "Insert blocks after this block ID (instead of at end)")
	cmd.Flags().StringVar(&blockType, "type", "", "Block type for simple mode (paragraph, heading_1, etc.)")
	cmd.Flags().StringVar(&content, "content", "", "Text content for simple mode (use with --type)")

	// Flag aliases
	flagAlias(cmd.Flags(), "children", "ch")
	flagAlias(cmd.Flags(), "children-file", "chf")

	return cmd
}

// buildSimpleBlock creates a block from type and content for simple usage mode.
// It automatically parses markdown formatting like **bold**, *italic*, and `code`.
func buildSimpleBlock(blockType, content string) map[string]interface{} {
	switch blockType {
	case "paragraph":
		return notion.NewParagraphWithMarkdown(content)
	case "heading_1":
		return notion.NewHeading1WithMarkdown(content)
	case "heading_2":
		return notion.NewHeading2WithMarkdown(content)
	case "heading_3":
		return notion.NewHeading3WithMarkdown(content)
	case "bulleted_list_item":
		return notion.NewBulletedListItemWithMarkdown(content)
	case "numbered_list_item":
		return notion.NewNumberedListItemWithMarkdown(content)
	case "quote":
		return notion.NewQuoteWithMarkdown(content)
	case "callout":
		return notion.NewCalloutWithMarkdown(content, "💡")
	case "code":
		// Code blocks don't parse markdown in their content
		return notion.NewCode(content, "plain text")
	case "to_do":
		return notion.NewToDoWithMarkdown(content, false)
	case "toggle":
		return notion.NewToggleWithMarkdown(content)
	case "divider":
		return notion.NewDivider()
	default:
		return nil
	}
}

func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to encode children JSON: %w", err)
	}
	return string(b), nil
}

func newBlockUpdateCmd() *cobra.Command {
	var contentJSON string
	var archived bool
	var setArchived bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "update <block-id>",
		Aliases: []string{"u"},
		Short:   "Update a block",
		Long: `Update a block's content.

The --content flag accepts a JSON object with the block type and its content.

Example of updating a paragraph block:
  ntn block update 12345678-1234-1234-1234-123456789012 \
    --content '{"paragraph":{"rich_text":[{"type":"text","text":{"content":"Updated text"}}]}}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			blockID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Resolve and parse content JSON if provided
			var content map[string]interface{}
			if contentJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(contentJSON)
				if err != nil {
					return err
				}
				contentJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(contentJSON, &content); err != nil {
					return fmt.Errorf("failed to parse content JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			if dryRun {
				// Fetch current block to show what would be updated
				currentBlock, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return wrapAPIError(err, "get block", "block", args[0])
				}

				printer := NewDryRunPrinter(stderrFromContext(ctx))
				printer.Header("update", "block", blockID)
				printer.Field("Type", currentBlock.Type)

				// Show archived status change if applicable
				if setArchived {
					if archived != currentBlock.Archived {
						printer.Change("Archived", fmt.Sprintf("%t", currentBlock.Archived), fmt.Sprintf("%t", archived))
					} else {
						printer.Unchanged("Archived")
					}
				}

				// Show content changes if provided
				if contentJSON != "" {
					printer.Section("Content to update:")
					contentBytes, _ := json.MarshalIndent(content, "  ", "  ")
					_, _ = fmt.Fprintf(stderrFromContext(ctx), "  %s\n", string(contentBytes))
				}

				printer.Footer()
				return nil
			}

			// Build request
			req := &notion.UpdateBlockRequest{
				Content: content,
			}

			// Set archived flag if specified
			if setArchived {
				req.Archived = &archived
			}

			// Update block
			block, err := client.UpdateBlock(ctx, blockID, req)
			if err != nil {
				// Check if the error is due to the block being archived
				// and we're trying to update content (not explicitly archiving)
				if isArchivedBlockError(err) && contentJSON != "" && !setArchived {
					// Auto-unarchive the block first
					unarchiveReq := &notion.UpdateBlockRequest{
						Archived: ptrBool(false),
					}
					_, unarchiveErr := client.UpdateBlock(ctx, blockID, unarchiveReq)
					if unarchiveErr != nil {
						return wrapAPIError(unarchiveErr, "unarchive block", "block", args[0])
					}
					_, _ = fmt.Fprintf(stderrFromContext(ctx), "Block was archived, auto-unarchived to apply update\n")

					// Retry the original update
					block, err = client.UpdateBlock(ctx, blockID, req)
					if err != nil {
						return wrapAPIError(err, "update block", "block", args[0])
					}
				} else {
					return wrapAPIError(err, "update block", "block", args[0])
				}
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, block)
		},
	}

	cmd.Flags().StringVar(&contentJSON, "content", "", "Block content as JSON")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the block")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	// Track if archived flag was explicitly set
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	// Flag aliases
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}

func newBlockDeleteCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "delete <block-id>",
		Aliases: []string{"d"},
		Short:   "Delete a block",
		Long: `Delete (archive) a block by its ID.

Example:
  ntn block delete 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			blockID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			if dryRun {
				// Fetch current block to show what would be deleted
				block, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return wrapAPIError(err, "get block", "block", args[0])
				}

				printer := NewDryRunPrinter(stderrFromContext(ctx))
				printer.Header("delete", "block", blockID)
				printer.Field("Type", block.Type)
				printer.Field("Archived", fmt.Sprintf("%t", block.Archived))
				printer.Field("Has children", fmt.Sprintf("%t", block.HasChildren))

				// Show content preview if available
				if len(block.Content) > 0 {
					// Try to extract text content for common block types
					if richText, ok := block.Content["rich_text"].([]interface{}); ok && len(richText) > 0 {
						if textObj, ok := richText[0].(map[string]interface{}); ok {
							if text, ok := textObj["plain_text"].(string); ok && text != "" {
								printer.Content("Content preview", text)
							}
						}
					}
				}

				// Show parent information
				if parentType, ok := block.Parent["type"].(string); ok {
					parentID := ""
					if id, ok := block.Parent[parentType].(string); ok {
						parentID = id
					}
					printer.Field("Parent", fmt.Sprintf("%s: %s", parentType, parentID))
				}

				printer.Footer()
				return nil
			}

			// Delete block
			block, err := client.DeleteBlock(ctx, blockID)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, block)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without making changes")
	return cmd
}

// validBlockColors contains all valid Notion block colors
var validBlockColors = map[string]bool{
	"default": true, "gray": true, "brown": true, "orange": true,
	"yellow": true, "green": true, "blue": true, "purple": true,
	"pink": true, "red": true,
	"gray_background": true, "brown_background": true, "orange_background": true,
	"yellow_background": true, "green_background": true, "blue_background": true,
	"purple_background": true, "pink_background": true, "red_background": true,
}

func newBlockAddTOCCmd() *cobra.Command {
	var color string

	cmd := &cobra.Command{
		Use:   "add-toc <parent-block-id>",
		Short: "Add a table of contents block",
		Long: `Add a table of contents block to a page or block.

The table of contents automatically shows all headings in the page.

Example:
  ntn block add-toc abc123 --color blue`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			parentID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Validate color
			if !validBlockColors[color] {
				return fmt.Errorf("invalid color %q: valid colors are default, gray, brown, orange, yellow, green, blue, purple, pink, red (or their _background variants)", color)
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			block := notion.NewTableOfContents(color)

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&color, "color", "default", "Block color (default, gray, brown, orange, yellow, green, blue, purple, pink, red, or _background variants)")
	return cmd
}

func newBlockAddBreadcrumbCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-breadcrumb <parent-block-id>",
		Short: "Add a breadcrumb block",
		Long: `Add a breadcrumb navigation block to a page.

The breadcrumb automatically shows the page hierarchy.

Example:
  ntn block add-breadcrumb abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			parentID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			block := notion.NewBreadcrumb()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddDividerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-divider <parent-block-id>",
		Short: "Add a divider block",
		Long: `Add a horizontal divider block.

Example:
  ntn block add-divider abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			parentID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			block := notion.NewDivider()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddColumnsCmd() *cobra.Command {
	var columnCount int

	cmd := &cobra.Command{
		Use:   "add-columns <parent-block-id>",
		Short: "Add a column layout",
		Long: `Add a column layout block with empty columns.

The columns will be created with placeholder paragraph blocks.
You can then add content to each column using its block ID.

Example:
  ntn block add-columns abc123 --columns 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			parentID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			if columnCount < MinColumnCount || columnCount > MaxColumnCount {
				return fmt.Errorf("column count must be between %d and %d, got %d", MinColumnCount, MaxColumnCount, columnCount)
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Create columns with placeholder content
			columns := make([][]map[string]interface{}, columnCount)
			for i := 0; i < columnCount; i++ {
				columns[i] = []map[string]interface{}{
					notion.NewParagraph(fmt.Sprintf("Column %d", i+1)),
				}
			}

			block := notion.NewColumnList(columns...)

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return wrapAPIError(err, "access block", "block", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&columnCount, "columns", MinColumnCount, fmt.Sprintf("Number of columns (%d-%d)", MinColumnCount, MaxColumnCount))
	return cmd
}

// isArchivedBlockError checks if an error is due to trying to edit an archived block
func isArchivedBlockError(err error) bool {
	if err == nil {
		return false
	}
	// Check for the specific Notion API error message about archived blocks
	errStr := err.Error()
	return strings.Contains(errStr, "archived") && strings.Contains(errStr, "edit")
}

// ptrBool returns a pointer to a bool value
func ptrBool(b bool) *bool {
	return &b
}

func appendBlockChildrenBatched(
	ctx context.Context,
	client blockChildrenWriter,
	blockID string,
	children []map[string]interface{},
	afterBlockID string,
) (*notion.BlockList, error) {
	if len(children) == 0 {
		return nil, fmt.Errorf("children are required")
	}

	const batchSize = 100
	nextAfter := afterBlockID
	combined := &notion.BlockList{Object: "list"}

	for i := 0; i < len(children); i += batchSize {
		end := i + batchSize
		if end > len(children) {
			end = len(children)
		}

		req := &notion.AppendBlockChildrenRequest{
			Children: children[i:end],
			After:    nextAfter,
		}

		batchResult, err := client.AppendBlockChildren(ctx, blockID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to append blocks (batch %d-%d): %w", i, end-1, err)
		}

		if batchResult == nil {
			return nil, fmt.Errorf("failed to append blocks (batch %d-%d): empty response", i, end-1)
		}

		if batchResult.Object != "" {
			combined.Object = batchResult.Object
		}
		combined.Results = append(combined.Results, batchResult.Results...)
		combined.NextCursor = batchResult.NextCursor
		combined.HasMore = batchResult.HasMore
		if batchResult.Type != "" {
			combined.Type = batchResult.Type
		}

		// Preserve order across multiple requests by chaining "after" to the
		// last block ID returned from the previous append.
		if end < len(children) {
			if len(batchResult.Results) == 0 || batchResult.Results[len(batchResult.Results)-1].ID == "" {
				return nil, fmt.Errorf(
					"failed to append blocks (batch %d-%d): response missing last block ID for chaining",
					i,
					end-1,
				)
			}
			nextAfter = batchResult.Results[len(batchResult.Results)-1].ID
		}
	}

	return combined, nil
}
