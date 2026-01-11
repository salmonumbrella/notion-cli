package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Manage Notion blocks",
		Long:  `Retrieve, append, update, and delete Notion blocks.`,
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
		Use:   "get <block-id>",
		Short: "Get a block by ID",
		Long: `Retrieve a Notion block by its ID.

Example:
  notion block get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Get block
			block, err := client.GetBlock(ctx, blockID)
			if err != nil {
				return fmt.Errorf("failed to get block: %w", err)
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

	cmd := &cobra.Command{
		Use:   "children <block-id>",
		Short: "Get block children",
		Long: `Retrieve the children of a block.

Use the --start-cursor flag to paginate through results.
Use the --page-size flag to control the number of results per page (max 100).
Use --all to fetch all pages of results automatically.

Example:
  notion block children 12345678-1234-1234-1234-123456789012
  notion block children 12345678-1234-1234-1234-123456789012 --page-size 50
  notion block children 12345678-1234-1234-1234-123456789012 --all`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

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
						return fmt.Errorf("failed to get block children: %w", err)
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
				return printer.Print(ctx, allBlocks)
			}

			// Single page request
			opts := &notion.BlockChildrenOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			blockList, err := client.GetBlockChildren(ctx, blockID, opts)
			if err != nil {
				return fmt.Errorf("failed to get block children: %w", err)
			}

			if limit > 0 && len(blockList.Results) > limit {
				blockList.Results = blockList.Results[:limit]
				blockList.HasMore = true
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, blockList)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")

	return cmd
}

func newBlockAppendCmd() *cobra.Command {
	var childrenJSON string
	var afterBlockID string
	var blockType string
	var content string

	cmd := &cobra.Command{
		Use:   "append <block-id>",
		Short: "Append children to a block",
		Long: `Append child blocks to a parent block.

SIMPLE USAGE (--type and --content):
  notion block append PAGE_ID --type paragraph --content "Hello world"
  notion block append PAGE_ID --type heading_2 --content "Section Title"
  notion block append PAGE_ID --type bulleted_list_item --content "List item"

Supported types: paragraph, heading_1, heading_2, heading_3, bulleted_list_item,
numbered_list_item, quote, callout, code, to_do, toggle, divider

ADVANCED USAGE (--children JSON):
  notion block append PAGE_ID \
    --children '[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}]'

Use --after to insert blocks after a specific block instead of at the end.

TIP: For convenience commands, see 'notion block add --help'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Handle convenience flags --type and --content
			if blockType != "" && content != "" {
				block := buildSimpleBlock(blockType, content)
				if block == nil {
					return fmt.Errorf("unsupported block type %q for simple mode\nSupported: paragraph, heading_1, heading_2, heading_3, bulleted_list_item, numbered_list_item, quote, callout, code, to_do, toggle, divider", blockType)
				}
				childrenJSON = mustMarshalJSON([]map[string]interface{}{block})
			} else if blockType != "" && content == "" {
				// Special case for content-less blocks
				if blockType == "divider" {
					block := notion.NewDivider()
					childrenJSON = mustMarshalJSON([]map[string]interface{}{block})
				} else {
					return fmt.Errorf("--content is required when using --type (except for divider)")
				}
			} else if blockType == "" && content != "" {
				return fmt.Errorf("--type is required when using --content")
			}

			// Validate required flag
			if childrenJSON == "" {
				return fmt.Errorf("either --children or both --type and --content are required\n\nSimple usage:\n  notion block append PAGE_ID --type paragraph --content \"Your text\"\n\nAdvanced usage:\n  notion block append PAGE_ID --children '[{\"type\":\"paragraph\",...}]'")
			}

			// Normalize after block ID if provided
			if afterBlockID != "" {
				afterBlockID, err = cmdutil.NormalizeNotionID(afterBlockID)
				if err != nil {
					return fmt.Errorf("invalid --after block ID: %w", err)
				}
			}

			// Resolve and parse children JSON
			var children []map[string]interface{}
			resolved, err := cmdutil.ReadJSONInput(childrenJSON)
			if err != nil {
				return err
			}
			childrenJSON = resolved
			if err := json.Unmarshal([]byte(childrenJSON), &children); err != nil {
				return fmt.Errorf("failed to parse children JSON: %w", err)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Build request
			req := &notion.AppendBlockChildrenRequest{
				Children: children,
				After:    afterBlockID,
			}

			// Append children
			blockList, err := client.AppendBlockChildren(ctx, blockID, req)
			if err != nil {
				return fmt.Errorf("failed to append block children: %w", err)
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, blockList)
		},
	}

	cmd.Flags().StringVar(&childrenJSON, "children", "", "Children blocks as JSON array")
	cmd.Flags().StringVar(&afterBlockID, "after", "", "Insert blocks after this block ID (instead of at end)")
	cmd.Flags().StringVar(&blockType, "type", "", "Block type for simple mode (paragraph, heading_1, etc.)")
	cmd.Flags().StringVar(&content, "content", "", "Text content for simple mode (use with --type)")

	return cmd
}

// buildSimpleBlock creates a block from type and content for simple usage mode
func buildSimpleBlock(blockType, content string) map[string]interface{} {
	switch blockType {
	case "paragraph":
		return notion.NewParagraph(content)
	case "heading_1":
		return notion.NewHeading1(content)
	case "heading_2":
		return notion.NewHeading2(content)
	case "heading_3":
		return notion.NewHeading3(content)
	case "bulleted_list_item":
		return notion.NewBulletedListItem(content)
	case "numbered_list_item":
		return notion.NewNumberedListItem(content)
	case "quote":
		return notion.NewQuote(content)
	case "callout":
		return notion.NewCallout(content, "💡")
	case "code":
		return notion.NewCode(content, "plain text")
	case "to_do":
		return notion.NewToDo(content, false)
	case "toggle":
		return map[string]interface{}{
			"type": "toggle",
			"toggle": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": content,
						},
					},
				},
			},
		}
	case "divider":
		return notion.NewDivider()
	default:
		return nil
	}
}

// mustMarshalJSON marshals to JSON or panics (for internal use only)
func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func newBlockUpdateCmd() *cobra.Command {
	var contentJSON string
	var archived bool
	var setArchived bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update <block-id>",
		Short: "Update a block",
		Long: `Update a block's content.

The --content flag accepts a JSON object with the block type and its content.

Example of updating a paragraph block:
  notion block update 12345678-1234-1234-1234-123456789012 \
    --content '{"paragraph":{"rich_text":[{"type":"text","text":{"content":"Updated text"}}]}}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
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
				if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
					return fmt.Errorf("failed to parse content JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			if dryRun {
				// Fetch current block to show what would be updated
				currentBlock, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return fmt.Errorf("failed to fetch block: %w", err)
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
						return fmt.Errorf("failed to update block (block is archived and auto-unarchive failed): %w", unarchiveErr)
					}
					_, _ = fmt.Fprintf(stderrFromContext(ctx), "Block was archived, auto-unarchived to apply update\n")

					// Retry the original update
					block, err = client.UpdateBlock(ctx, blockID, req)
					if err != nil {
						return fmt.Errorf("failed to update block after unarchiving: %w", err)
					}
				} else {
					return fmt.Errorf("failed to update block: %w", err)
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

	return cmd
}

func newBlockDeleteCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <block-id>",
		Short: "Delete a block",
		Long: `Delete (archive) a block by its ID.

Example:
  notion block delete 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			if dryRun {
				// Fetch current block to show what would be deleted
				block, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return fmt.Errorf("failed to fetch block: %w", err)
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
				return fmt.Errorf("failed to delete block: %w", err)
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
  notion block add-toc abc123 --color blue`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Validate color
			if !validBlockColors[color] {
				return fmt.Errorf("invalid color %q: valid colors are default, gray, brown, orange, yellow, green, blue, purple, pink, red (or their _background variants)", color)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewTableOfContents(color)

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add table of contents: %w", err)
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
  notion block add-breadcrumb abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewBreadcrumb()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add breadcrumb: %w", err)
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
  notion block add-divider abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewDivider()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add divider: %w", err)
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
  notion block add-columns abc123 --columns 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			if columnCount < 2 || columnCount > 5 {
				return fmt.Errorf("column count must be between 2 and 5, got %d", columnCount)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(ctx, token)

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
				return fmt.Errorf("failed to add columns: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&columnCount, "columns", 2, "Number of columns (2-5)")
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
