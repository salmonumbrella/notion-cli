package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newPageSyncCmd() *cobra.Command {
	var pushFile string
	var pullID string
	var parentID string
	var parentType string
	var outputFile string
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:     "sync",
		Aliases: []string{"sy"},
		Short:   "Sync markdown files with Notion pages",
		Long: `Bidirectional sync between local markdown files and Notion pages.

YAML frontmatter tracks the sync relationship:
  ---
  notion-id: 12345678-1234-1234-1234-123456789012
  title: Meeting Notes
  last-synced: 2026-02-12T10:30:00Z
  ---

PUSH (local -> Notion):
  Push a markdown file to Notion. If the file has a notion-id in frontmatter,
  the existing page content is replaced. If no notion-id, use --parent to create
  a new page (the notion-id is written back to the file).

PULL (Notion -> local):
  Pull a Notion page to a local markdown file with frontmatter. Use -o to write
  to a file, or omit for stdout.

Examples:
  # Update existing page (reads notion-id from frontmatter)
  ntn page sync --push doc.md

  # Create new page under a database, writes notion-id back to file
  ntn page sync --push doc.md --parent <db-id>

  # Export page to local .md with frontmatter
  ntn page sync --pull <page-id> -o doc.md

  # Print to stdout
  ntn page sync --pull <page-id>

  # Show what would change
  ntn page sync --push doc.md --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pushFile == "" && pullID == "" {
				return fmt.Errorf("either --push or --pull is required")
			}
			if pushFile != "" && pullID != "" {
				return fmt.Errorf("--push and --pull are mutually exclusive")
			}

			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}
			stderr := stderrFromContext(ctx)

			if pushFile != "" {
				return runSyncPush(ctx, client, stderr, pushFile, parentID, parentType, dryRun, force)
			}
			return runSyncPull(ctx, client, stderr, pullID, outputFile, dryRun)
		},
	}

	cmd.Flags().StringVar(&pushFile, "push", "", "Markdown file to push to Notion")
	cmd.Flags().StringVar(&pullID, "pull", "", "Page ID to pull from Notion")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (for creating new pages)")
	cmd.Flags().StringVar(&parentType, "parent-type", "", "Parent type: 'page' or 'database' (default: auto-detect)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (for pull)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Push even if the Notion page was edited since last sync")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}

// runSyncPush pushes a local markdown file to Notion.
func runSyncPush(ctx context.Context, client *notion.Client, stderr io.Writer, filePath, parentID, parentType string, dryRun, force bool) error {
	input, err := loadSyncPushInput(filePath)
	if err != nil {
		return err
	}

	if input.NotionID == "" && parentID == "" {
		return fmt.Errorf("no notion-id in frontmatter and no --parent provided; use --parent to create a new page")
	}

	// Check for conflicts on existing pages
	if input.NotionID != "" && !force && !dryRun {
		if err := checkSyncConflict(ctx, client, input.NotionID, input.Frontmatter["last-synced"]); err != nil {
			return err
		}
	}

	if dryRun {
		printer := NewDryRunPrinter(stderr)
		if input.NotionID != "" {
			printer.Header("sync (push)", "page", input.NotionID)
			printer.Field("File", filePath)
			printer.Field("Action", "replace all blocks on existing page")
		} else {
			printer.Header("sync (push)", "new page", filePath)
			printer.Field("Parent", parentID)
			printer.Field("Action", "create new page and write notion-id to frontmatter")
		}
		printer.Field("Blocks to sync", fmt.Sprintf("%d", len(input.Blocks)))
		if len(input.Blocks) > 0 {
			printer.Section("Block types:")
			typeCounts := countBlockTypes(input.Blocks)
			for blockType, count := range typeCounts {
				_, _ = fmt.Fprintf(stderr, "  %s: %d\n", blockType, count)
			}
		}
		printer.Footer()
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if input.NotionID != "" {
		normalizedID, err := syncExistingPage(ctx, client, filePath, input, now)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stderr, "Pushed %d blocks to page %s\n", len(input.Blocks), normalizedID)
		return nil
	}

	pageID, err := createSyncedPage(ctx, client, filePath, parentID, parentType, input, now)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stderr, "Created page %s with %d blocks\n", pageID, len(input.Blocks))
	return nil
}

type syncPushInput struct {
	Frontmatter map[string]string
	Body        string
	Blocks      []map[string]interface{}
	NotionID    string
}

func loadSyncPushInput(filePath string) (*syncPushInput, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fm, body := parseFrontmatter(string(data))
	return &syncPushInput{
		Frontmatter: fm,
		Body:        body,
		Blocks:      parseMarkdownToBlocks(body),
		NotionID:    fm["notion-id"],
	}, nil
}

func checkSyncConflict(ctx context.Context, client *notion.Client, notionID, lastSynced string) error {
	if lastSynced == "" {
		return nil
	}

	checkID, err := cmdutil.NormalizeNotionID(notionID)
	if err != nil {
		return fmt.Errorf("invalid notion-id in frontmatter: %w", err)
	}

	page, err := client.GetPage(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to fetch page for conflict check: %w", err)
	}

	if page.LastEditedTime == "" {
		return nil
	}

	syncedTime, err1 := time.Parse(time.RFC3339, lastSynced)
	editedTime, err2 := time.Parse(time.RFC3339, page.LastEditedTime)
	if err1 == nil && err2 == nil && editedTime.After(syncedTime) {
		return fmt.Errorf("page was modified on Notion since last sync (synced: %s, edited: %s); use --force to overwrite", lastSynced, page.LastEditedTime)
	}
	return nil
}

func deriveSyncTitle(fm map[string]string, body string) string {
	title := fm["title"]
	if title == "" {
		title = extractTitleFromBody(body)
	}
	return title
}

func syncExistingPage(ctx context.Context, client *notion.Client, filePath string, input *syncPushInput, now string) (string, error) {
	normalizedID, err := cmdutil.NormalizeNotionID(input.NotionID)
	if err != nil {
		return "", fmt.Errorf("invalid notion-id in frontmatter: %w", err)
	}

	title := deriveSyncTitle(input.Frontmatter, input.Body)
	if title != "" {
		updateReq := &notion.UpdatePageRequest{
			Properties: map[string]interface{}{
				"title": map[string]interface{}{
					"title": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]interface{}{
								"content": title,
							},
						},
					},
				},
			},
		}
		if _, err := client.UpdatePage(ctx, normalizedID, updateReq); err != nil {
			return "", fmt.Errorf("failed to update page title: %w", err)
		}
	}

	existingBlocks, err := fetchAllBlockChildren(ctx, client, normalizedID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch existing blocks: %w", err)
	}
	for _, block := range existingBlocks {
		if _, err := client.DeleteBlock(ctx, block.ID); err != nil {
			return "", fmt.Errorf("failed to delete block %s: %w", block.ID, err)
		}
	}

	if len(input.Blocks) > 0 {
		if err := appendBlocksInBatches(ctx, client, normalizedID, input.Blocks); err != nil {
			return "", fmt.Errorf("failed to append blocks: %w", err)
		}
	}

	input.Frontmatter["last-synced"] = now
	if err := writeFrontmatterToFile(filePath, input.Frontmatter, input.Body); err != nil {
		return "", fmt.Errorf("failed to update frontmatter: %w", err)
	}

	return normalizedID, nil
}

func createSyncedPage(ctx context.Context, client *notion.Client, filePath, parentID, parentType string, input *syncPushInput, now string) (string, error) {
	normalizedParent, err := cmdutil.NormalizeNotionID(parentID)
	if err != nil {
		return "", fmt.Errorf("invalid parent ID: %w", err)
	}

	title := deriveSyncTitle(input.Frontmatter, input.Body)
	if title == "" {
		title = "Untitled"
	}

	parent, err := resolveParentForSync(ctx, client, normalizedParent, parentType)
	if err != nil {
		return "", err
	}

	req := &notion.CreatePageRequest{
		Parent: parent,
		Properties: map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": title,
						},
					},
				},
			},
		},
	}
	page, err := client.CreatePage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}

	if len(input.Blocks) > 0 {
		if err := appendBlocksInBatches(ctx, client, page.ID, input.Blocks); err != nil {
			return "", fmt.Errorf("failed to append blocks: %w", err)
		}
	}

	input.Frontmatter["notion-id"] = page.ID
	input.Frontmatter["title"] = title
	input.Frontmatter["last-synced"] = now
	if err := writeFrontmatterToFile(filePath, input.Frontmatter, input.Body); err != nil {
		return "", fmt.Errorf("failed to write frontmatter: %w", err)
	}

	return page.ID, nil
}

// runSyncPull pulls a Notion page to a local markdown file.
func runSyncPull(ctx context.Context, client *notion.Client, stderr io.Writer, pageID, outputFile string, dryRun bool) error {
	normalizedID, err := cmdutil.NormalizeNotionID(pageID)
	if err != nil {
		return fmt.Errorf("invalid page ID: %w", err)
	}

	// Fetch page metadata
	page, err := client.GetPage(ctx, normalizedID)
	if err != nil {
		return wrapAPIError(err, "get page", "page", pageID)
	}

	title := pageTitleFromProperties(page.Properties)

	// Fetch blocks
	blocks, err := fetchExportBlocks(ctx, client, normalizedID)
	if err != nil {
		return err
	}

	// Convert blocks to markdown
	markdown := renderMarkdown(blocks, 0)

	// Build frontmatter
	now := time.Now().UTC().Format(time.RFC3339)
	fm := map[string]string{
		"notion-id":   normalizedID,
		"title":       title,
		"last-synced": now,
	}

	if dryRun {
		printer := NewDryRunPrinter(stderr)
		printer.Header("sync (pull)", "page", normalizedID)
		printer.Field("Title", title)
		if outputFile != "" {
			printer.Field("Output", outputFile)
		} else {
			printer.Field("Output", "stdout")
		}
		printer.Field("Blocks", fmt.Sprintf("%d", len(blocks)))
		printer.Footer()
		return nil
	}

	out := buildFrontmatterString(fm) + markdown + "\n"

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(out), 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		_, _ = fmt.Fprintf(stderr, "Pulled page %s to %s\n", normalizedID, outputFile)
		return nil
	}

	// Print to stdout
	stdout := stdoutFromContext(ctx)
	_, _ = fmt.Fprint(stdout, out)
	return nil
}

// parseFrontmatter parses YAML frontmatter from markdown content.
// Returns the frontmatter key-value pairs and the body content after frontmatter.
func parseFrontmatter(content string) (map[string]string, string) {
	fm := make(map[string]string)

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, content
	}

	// Find closing ---
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}

	if closingIdx < 0 {
		// No closing delimiter; treat entire content as body
		return fm, content
	}

	// Parse key: value pairs between delimiters
	for i := 1; i < closingIdx; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				fm[key] = value
			}
		}
	}

	// Body is everything after the closing ---
	bodyStart := closingIdx + 1
	if bodyStart < len(lines) {
		body := strings.Join(lines[bodyStart:], "\n")
		return fm, body
	}

	return fm, ""
}

// buildFrontmatterString builds a YAML frontmatter string from key-value pairs.
// Keys are written in a stable order: notion-id, title, last-synced, then any others.
func buildFrontmatterString(fm map[string]string) string {
	if len(fm) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("---\n")

	// Write known keys in order
	orderedKeys := []string{"notion-id", "title", "last-synced"}
	written := make(map[string]bool)

	for _, key := range orderedKeys {
		if val, ok := fm[key]; ok {
			b.WriteString(key)
			b.WriteString(": ")
			b.WriteString(val)
			b.WriteString("\n")
			written[key] = true
		}
	}

	// Write remaining keys alphabetically
	for key, val := range fm {
		if written[key] {
			continue
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(val)
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	return b.String()
}

// writeFrontmatterToFile writes frontmatter + body back to a file.
func writeFrontmatterToFile(filePath string, fm map[string]string, body string) error {
	content := buildFrontmatterString(fm) + body
	// Ensure file ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}

// extractTitleFromBody looks for the first # heading in the body.
func extractTitleFromBody(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := headingPattern.FindStringSubmatch(trimmed); matches != nil {
			if len(matches[1]) == 1 { // Only # (H1)
				return matches[2]
			}
		}
	}
	return ""
}

// fetchAllBlockChildren fetches all direct block children (non-recursive, handles pagination).
func fetchAllBlockChildren(ctx context.Context, client *notion.Client, blockID string) ([]notion.Block, error) {
	var allBlocks []notion.Block
	cursor := ""

	for {
		opts := &notion.BlockChildrenOptions{StartCursor: cursor, PageSize: 100}
		list, err := client.GetBlockChildren(ctx, blockID, opts)
		if err != nil {
			return nil, err
		}
		allBlocks = append(allBlocks, list.Results...)
		if !list.HasMore || list.NextCursor == nil || *list.NextCursor == "" {
			break
		}
		cursor = *list.NextCursor
	}

	return allBlocks, nil
}

// appendBlocksInBatches appends blocks in batches of 100 (Notion API limit).
func appendBlocksInBatches(ctx context.Context, client *notion.Client, pageID string, blocks []map[string]interface{}) error {
	const batchSize = 100

	for i := 0; i < len(blocks); i += batchSize {
		end := i + batchSize
		if end > len(blocks) {
			end = len(blocks)
		}

		req := &notion.AppendBlockChildrenRequest{
			Children: blocks[i:end],
		}
		if _, err := client.AppendBlockChildren(ctx, pageID, req); err != nil {
			return err
		}
	}

	return nil
}

// resolveParentForSync determines the parent map for creating a new page during sync.
// If parentType is empty, it auto-detects by trying database first, then falling back to page.
func resolveParentForSync(ctx context.Context, client *notion.Client, parentID, parentType string) (map[string]interface{}, error) {
	switch parentType {
	case "page":
		return map[string]interface{}{"page_id": parentID}, nil
	case "database":
		return map[string]interface{}{"database_id": parentID}, nil
	case "":
		// Auto-detect: try database first
		_, err := client.GetDatabase(ctx, parentID)
		if err == nil {
			return map[string]interface{}{"database_id": parentID}, nil
		}
		// Fall back to page
		return map[string]interface{}{"page_id": parentID}, nil
	default:
		return nil, fmt.Errorf("invalid --parent-type: %s (expected 'page' or 'database')", parentType)
	}
}
