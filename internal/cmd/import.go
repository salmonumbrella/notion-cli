package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// Markdown patterns for parsing
var (
	// Headings: # Heading 1, ## Heading 2, ### Heading 3
	headingPattern = regexp.MustCompile(`^(#{1,3})\s+(.+)$`)

	// Horizontal rule / divider: ---, ***, ___
	dividerPattern = regexp.MustCompile(`^(\*{3,}|-{3,}|_{3,})$`)

	// Numbered list: 1. Item, 2. Item, etc.
	numberedListPattern = regexp.MustCompile(`^\d+\.\s+(.+)$`)

	// Bullet list: - Item, * Item, + Item
	bulletListPattern = regexp.MustCompile(`^[-*+]\s+(.+)$`)

	// To-do list: - [ ] Item, - [x] Item
	todoPattern = regexp.MustCompile(`^[-*+]\s+\[([ xX])\]\s+(.+)$`)

	// Quote: > text
	quotePattern = regexp.MustCompile(`^>\s*(.*)$`)

	// Code block fence: ```language or ```
	codeFencePattern = regexp.MustCompile("^```(\\w*)$")

	// Table separator: | --- | --- | (with optional colons for alignment, ignored)
	tableSeparatorPattern = regexp.MustCompile(`^\|[\s:]*-{3,}[\s:]*(\|[\s:]*-{3,}[\s:]*)*\|$`)
)

func newImportCmd() *cobra.Command {
	var filePath string
	var dryRun bool
	var batchSize int

	cmd := &cobra.Command{
		Use:     "import <page-id>",
		Aliases: []string{"im"},
		Short:   "Import a markdown file as Notion blocks",
		Long: `Import a markdown file and convert it to Notion blocks.

Supported markdown elements:
  - # Heading 1, ## Heading 2, ### Heading 3
  - **bold**, *italic*, ` + "`code`" + `, ***bold italic***
  - --- (divider)
  - - Bullet list items
  - 1. Numbered list items
  - - [ ] To-do items (unchecked), - [x] To-do items (checked)
  - > Blockquotes
  - | Tables | with | pipe | syntax |
  - ` + "```" + `language code blocks ` + "```" + `
  - Regular paragraphs

Examples:
  notion import abc123 --file ./document.md
  notion import abc123 --file ./README.md --dry-run
  notion import abc123 --file - < document.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			pageID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Read markdown content
			content, err := readMarkdownFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read markdown file: %w", err)
			}

			// Parse markdown into blocks
			blocks := parseMarkdownToBlocks(content)

			if len(blocks) == 0 {
				_, _ = fmt.Fprintln(stderrFromContext(cmd.Context()), "No blocks parsed from markdown file")
				return nil
			}

			// Dry run mode
			if dryRun {
				printer := NewDryRunPrinter(stderrFromContext(cmd.Context()))
				printer.Header("import", "markdown", filePath)
				printer.Field("Target page", pageID)
				printer.Field("Blocks to create", fmt.Sprintf("%d", len(blocks)))

				printer.Section("Block types:")
				typeCounts := countBlockTypes(blocks)
				for blockType, count := range typeCounts {
					_, _ = fmt.Fprintf(stderrFromContext(cmd.Context()), "  %s: %d\n", blockType, count)
				}

				printer.Footer()
				return nil
			}

			// Get token from context
			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Append blocks in batches (Notion API limit is 100 blocks per request)
			if batchSize <= 0 || batchSize > 100 {
				batchSize = 100
			}

			var totalCreated int
			for i := 0; i < len(blocks); i += batchSize {
				end := i + batchSize
				if end > len(blocks) {
					end = len(blocks)
				}

				batch := blocks[i:end]
				req := &notion.AppendBlockChildrenRequest{
					Children: batch,
				}

				_, err := client.AppendBlockChildren(ctx, pageID, req)
				if err != nil {
					return fmt.Errorf("failed to append blocks (batch %d-%d): %w", i, end-1, err)
				}

				totalCreated += len(batch)
			}

			// Print summary
			_, _ = fmt.Fprintf(stderrFromContext(ctx), "Successfully imported %d blocks to page %s\n", totalCreated, pageID)

			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Markdown file to import (use - for stdin)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be imported without making changes")
	cmd.Flags().IntVar(&batchSize, "batch-size", 100, "Number of blocks to append per API request (max 100)")

	cmd.AddCommand(newImportCSVCmd())

	return cmd
}

// readMarkdownFile reads markdown content from a file or stdin
func readMarkdownFile(path string) (string, error) {
	if path == "-" {
		// Read from stdin
		var builder strings.Builder
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			builder.WriteString(scanner.Text())
			builder.WriteString("\n")
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return builder.String(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseMarkdownToBlocks converts markdown content to Notion blocks
func parseMarkdownToBlocks(content string) []map[string]interface{} {
	var blocks []map[string]interface{}

	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			i++
			continue
		}

		// Check for table
		if isTableRow(trimmed) {
			// Collect all consecutive table rows
			var rawRows []string
			for i < len(lines) {
				rowLine := strings.TrimSpace(lines[i])
				if !isTableRow(rowLine) {
					break
				}
				rawRows = append(rawRows, rowLine)
				i++
			}

			// Need at least 2 rows (header + separator) to be a valid table
			if len(rawRows) >= 2 && tableSeparatorPattern.MatchString(rawRows[1]) {
				// Parse header row
				headerCells := parseTableCells(rawRows[0])

				// Collect data rows (skip separator at index 1)
				var dataRows [][]string
				for _, raw := range rawRows[2:] {
					dataRows = append(dataRows, parseTableCells(raw))
				}

				// Build all rows: header + data
				allRows := [][]string{headerCells}
				allRows = append(allRows, dataRows...)

				blocks = append(blocks, notion.NewTableWithMarkdown(allRows, true))
			} else {
				// Not a valid table (no separator row), treat rows as paragraphs
				for _, raw := range rawRows {
					blocks = append(blocks, notion.NewParagraphWithMarkdown(raw))
				}
			}
			continue
		}

		// Check for code block
		if matches := codeFencePattern.FindStringSubmatch(trimmed); matches != nil {
			language := matches[1]
			if language == "" {
				language = "plain text"
			}

			// Collect code content until closing fence
			var codeLines []string
			i++
			for i < len(lines) {
				if strings.TrimSpace(lines[i]) == "```" {
					break
				}
				codeLines = append(codeLines, lines[i])
				i++
			}
			i++ // Skip closing fence

			codeContent := strings.Join(codeLines, "\n")
			blocks = append(blocks, notion.NewCode(codeContent, language))
			continue
		}

		// Check for divider
		if dividerPattern.MatchString(trimmed) {
			blocks = append(blocks, notion.NewDivider())
			i++
			continue
		}

		// Check for headings
		if matches := headingPattern.FindStringSubmatch(trimmed); matches != nil {
			level := len(matches[1])
			text := matches[2]

			switch level {
			case 1:
				blocks = append(blocks, notion.NewHeading1WithMarkdown(text))
			case 2:
				blocks = append(blocks, notion.NewHeading2WithMarkdown(text))
			case 3:
				blocks = append(blocks, notion.NewHeading3WithMarkdown(text))
			}
			i++
			continue
		}

		// Check for to-do items (must be before bullet list)
		if matches := todoPattern.FindStringSubmatch(trimmed); matches != nil {
			checked := matches[1] == "x" || matches[1] == "X"
			text := matches[2]
			blocks = append(blocks, notion.NewToDoWithMarkdown(text, checked))
			i++
			continue
		}

		// Check for bullet list
		if matches := bulletListPattern.FindStringSubmatch(trimmed); matches != nil {
			text := matches[1]
			blocks = append(blocks, notion.NewBulletedListItemWithMarkdown(text))
			i++
			continue
		}

		// Check for numbered list
		if matches := numberedListPattern.FindStringSubmatch(trimmed); matches != nil {
			text := matches[1]
			blocks = append(blocks, notion.NewNumberedListItemWithMarkdown(text))
			i++
			continue
		}

		// Check for quote
		if matches := quotePattern.FindStringSubmatch(trimmed); matches != nil {
			text := matches[1]
			// Handle multi-line quotes
			for i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if matches := quotePattern.FindStringSubmatch(nextLine); matches != nil {
					text += "\n" + matches[1]
					i++
				} else {
					break
				}
			}
			blocks = append(blocks, notion.NewQuoteWithMarkdown(text))
			i++
			continue
		}

		// Default: treat as paragraph
		// Handle multi-line paragraphs (lines that don't match any pattern and aren't empty)
		paragraphText := trimmed
		for i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "" {
				break
			}
			// Check if next line starts a new block type
			if isBlockStart(nextLine) {
				break
			}
			paragraphText += " " + nextLine
			i++
		}
		blocks = append(blocks, notion.NewParagraphWithMarkdown(paragraphText))
		i++
	}

	return blocks
}

// isBlockStart checks if a line would start a new block type
func isBlockStart(line string) bool {
	return headingPattern.MatchString(line) ||
		dividerPattern.MatchString(line) ||
		bulletListPattern.MatchString(line) ||
		numberedListPattern.MatchString(line) ||
		todoPattern.MatchString(line) ||
		quotePattern.MatchString(line) ||
		codeFencePattern.MatchString(line) ||
		isTableRow(line)
}

// isTableRow checks if a line looks like a markdown table row (starts and ends with |).
func isTableRow(line string) bool {
	return len(line) >= 3 && line[0] == '|' && line[len(line)-1] == '|'
}

// parseTableCells splits a markdown table row into trimmed cell strings.
func parseTableCells(line string) []string {
	// Remove leading/trailing pipes
	inner := line[1 : len(line)-1]
	parts := strings.Split(inner, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

// countBlockTypes counts the number of each block type for dry-run output
func countBlockTypes(blocks []map[string]interface{}) map[string]int {
	counts := make(map[string]int)
	for _, block := range blocks {
		if blockType, ok := block["type"].(string); ok {
			counts[blockType]++
		}
	}
	return counts
}
