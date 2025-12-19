package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
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
			blockID := args[0]

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get block
			ctx := context.Background()
			block, err := client.GetBlock(ctx, blockID)
			if err != nil {
				return fmt.Errorf("failed to get block: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			blockID := args[0]

			// Validate page size
			if pageSize > 100 {
				return fmt.Errorf("page-size must be between 1 and 100")
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)
			ctx := context.Background()

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

					if !blockList.HasMore || blockList.NextCursor == nil || *blockList.NextCursor == "" {
						break
					}
					cursor = *blockList.NextCursor
				}

				// Print all results
				printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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

	cmd := &cobra.Command{
		Use:   "append <block-id>",
		Short: "Append children to a block",
		Long: `Append child blocks to a parent block.

The --children flag accepts a JSON array of block objects.

Example of a simple paragraph block:
  notion block append 12345678-1234-1234-1234-123456789012 \
    --children '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello world"}}]}}]'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockID := args[0]

			// Validate required flag
			if childrenJSON == "" {
				return fmt.Errorf("--children flag is required")
			}

			// Parse children JSON
			var children []map[string]interface{}
			if err := json.Unmarshal([]byte(childrenJSON), &children); err != nil {
				return fmt.Errorf("failed to parse children JSON: %w", err)
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Build request
			req := &notion.AppendBlockChildrenRequest{
				Children: children,
			}

			// Append children
			ctx := context.Background()
			blockList, err := client.AppendBlockChildren(ctx, blockID, req)
			if err != nil {
				return fmt.Errorf("failed to append block children: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, blockList)
		},
	}

	cmd.Flags().StringVar(&childrenJSON, "children", "", "Children blocks as JSON array (required)")

	return cmd
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
			blockID := args[0]

			// Parse content JSON if provided
			var content map[string]interface{}
			if contentJSON != "" {
				if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
					return fmt.Errorf("failed to parse content JSON: %w", err)
				}
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)
			ctx := context.Background()

			if dryRun {
				// Fetch current block to show what would be updated
				currentBlock, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return fmt.Errorf("failed to fetch block: %w", err)
				}

				printer := NewDryRunPrinter(os.Stderr)
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
					fmt.Fprintf(os.Stderr, "  %s\n", string(contentBytes))
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
				return fmt.Errorf("failed to update block: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			blockID := args[0]

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)
			ctx := context.Background()

			if dryRun {
				// Fetch current block to show what would be deleted
				block, err := client.GetBlock(ctx, blockID)
				if err != nil {
					return fmt.Errorf("failed to fetch block: %w", err)
				}

				printer := NewDryRunPrinter(os.Stderr)
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
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			parentID := args[0]

			// Validate color
			if !validBlockColors[color] {
				return fmt.Errorf("invalid color %q: valid colors are default, gray, brown, orange, yellow, green, blue, purple, pink, red (or their _background variants)", color)
			}

			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)
			ctx := context.Background()

			block := notion.NewTableOfContents(color)

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add table of contents: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			parentID := args[0]

			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)
			ctx := context.Background()

			block := notion.NewBreadcrumb()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add breadcrumb: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			parentID := args[0]

			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)
			ctx := context.Background()

			block := notion.NewDivider()

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add divider: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			parentID := args[0]

			if columnCount < 2 || columnCount > 5 {
				return fmt.Errorf("column count must be between 2 and 5, got %d", columnCount)
			}

			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)
			ctx := context.Background()

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

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&columnCount, "columns", 2, "Number of columns (2-5)")
	return cmd
}
