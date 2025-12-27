package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// backupMeta stores metadata about the last backup run.
type backupMeta struct {
	LastBackup string `json:"last_backup"`
	PageCount  int    `json:"page_count"`
	Version    int    `json:"version"`
}

func newDBBackupCmd() *cobra.Command {
	var outputDir string
	var includeContent bool
	var incremental bool
	var format string

	cmd := &cobra.Command{
		Use:     "backup <database-id-or-name>",
		Aliases: []string{"bak"},
		Short:   "Backup a database to local files",
		Long: `Export a Notion database to a local directory structure.

Creates a directory named after the database containing schema, page data,
and optionally page content (block children).

Output structure:
  <db-slug>/
    schema.json             # Database schema (properties, title, description)
    .backup-meta.json       # Backup metadata (timestamp, page count)
    pages/
      <page-id>.json        # Page properties
      <page-id>.blocks.json # Page content blocks (with --content)
      <page-id>.md          # Markdown export (with --export-format markdown)

Example - Full backup:
  ntn db backup 12345678-1234-1234-1234-123456789012

Example - Backup with page content:
  ntn db backup "Projects" --content

Example - Incremental backup (only changed pages):
  ntn db backup "Tasks" --incremental

Example - Export as markdown:
  ntn db backup "Notes" --export-format markdown --content

Example - Custom output directory:
  ntn db backup "Tasks" --output-dir ./backups`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)
			stderr := stderrFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Validate format
			format = strings.ToLower(strings.TrimSpace(format))
			if format != "json" && format != "markdown" && format != "md" {
				return fmt.Errorf("invalid --export-format %q (expected json or markdown)", format)
			}
			isMarkdown := format == "markdown" || format == "md"

			// Resolve database ID
			databaseID, err := resolveIDWithSearch(ctx, client, sf, args[0], "database")
			if err != nil {
				return err
			}
			databaseID, err = cmdutil.NormalizeNotionID(databaseID)
			if err != nil {
				return err
			}

			// Step 1: Fetch database schema
			database, err := client.GetDatabase(ctx, databaseID)
			if err != nil {
				return wrapAPIError(err, "get database", "database", args[0])
			}

			// Step 2: Generate db-slug from title
			dbTitle := extractTitlePlainText(toInterfaceSlice(database.Title))
			if dbTitle == "" {
				dbTitle = databaseID
			}
			slug := slugifyDBTitle(dbTitle)

			// Step 3: Create output directory structure
			backupDir := filepath.Join(outputDir, slug)
			pagesDir := filepath.Join(backupDir, "pages")
			if err := os.MkdirAll(pagesDir, 0o755); err != nil {
				return fmt.Errorf("failed to create backup directory: %w", err)
			}

			// Step 4: Write schema.json
			schemaData, err := json.MarshalIndent(database, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal schema: %w", err)
			}
			if err := os.WriteFile(filepath.Join(backupDir, "schema.json"), schemaData, 0o644); err != nil {
				return fmt.Errorf("failed to write schema.json: %w", err)
			}

			// Step 5: Resolve data source for querying
			resolvedDataSourceID, err := resolveDataSourceID(ctx, client, databaseID, "")
			if err != nil {
				return err
			}

			// Step 6: Build query (incremental or full)
			var filter map[string]interface{}
			if incremental {
				metaPath := filepath.Join(backupDir, ".backup-meta.json")
				meta, readErr := readBackupMeta(metaPath)
				if readErr == nil && meta.LastBackup != "" {
					filter = map[string]interface{}{
						"timestamp": "last_edited_time",
						"last_edited_time": map[string]interface{}{
							"after": meta.LastBackup,
						},
					}
				}
				// If no previous meta exists, fall through to full backup
			}

			// Step 7: Query all pages (handle pagination)
			var allPages []notion.Page
			cursor := ""
			for {
				req := &notion.QueryDataSourceRequest{
					Filter:      filter,
					StartCursor: cursor,
					PageSize:    100,
				}

				result, err := client.QueryDataSource(ctx, resolvedDataSourceID, req)
				if err != nil {
					return wrapAPIError(err, "query database", "database", args[0])
				}

				allPages = append(allPages, result.Results...)

				if !result.HasMore || result.NextCursor == nil || *result.NextCursor == "" {
					break
				}
				cursor = *result.NextCursor
			}

			// Step 8: Write each page
			for _, page := range allPages {
				// Write page properties JSON
				pageData, err := json.MarshalIndent(page, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal page %s: %w", page.ID, err)
				}
				if err := os.WriteFile(filepath.Join(pagesDir, page.ID+".json"), pageData, 0o644); err != nil {
					return fmt.Errorf("failed to write page %s: %w", page.ID, err)
				}

				// Fetch and write block children if --content
				if includeContent || isMarkdown {
					blocks, err := fetchExportBlocks(ctx, client, page.ID)
					if err != nil {
						_, _ = fmt.Fprintf(stderr, "Warning: failed to fetch blocks for page %s: %v\n", page.ID, err)
						continue
					}

					if includeContent {
						blocksData, err := json.MarshalIndent(blocks, "", "  ")
						if err != nil {
							return fmt.Errorf("failed to marshal blocks for page %s: %w", page.ID, err)
						}
						if err := os.WriteFile(filepath.Join(pagesDir, page.ID+".blocks.json"), blocksData, 0o644); err != nil {
							return fmt.Errorf("failed to write blocks for page %s: %w", page.ID, err)
						}
					}

					if isMarkdown {
						markdown := renderMarkdown(blocks, 0)
						if title := pageTitleFromProperties(page.Properties); title != "" {
							markdown = "# " + title + "\n\n" + markdown
						}
						if err := os.WriteFile(filepath.Join(pagesDir, page.ID+".md"), []byte(markdown), 0o644); err != nil {
							return fmt.Errorf("failed to write markdown for page %s: %w", page.ID, err)
						}
					}
				}
			}

			// Step 9: Write .backup-meta.json
			meta := backupMeta{
				LastBackup: time.Now().UTC().Format(time.RFC3339),
				PageCount:  len(allPages),
				Version:    1,
			}
			metaData, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal backup metadata: %w", err)
			}
			if err := os.WriteFile(filepath.Join(backupDir, ".backup-meta.json"), metaData, 0o644); err != nil {
				return fmt.Errorf("failed to write .backup-meta.json: %w", err)
			}

			// Step 10: Print summary
			_, _ = fmt.Fprintf(stderr, "Backed up %d pages from '%s' to %s\n", len(allPages), dbTitle, backupDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "d", ".", "Output directory for backup")
	cmd.Flags().BoolVar(&includeContent, "content", false, "Include page body (block children)")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "Only backup pages changed since last run")
	cmd.Flags().StringVar(&format, "export-format", "json", "Export format for pages (json or markdown)")

	return cmd
}

// slugifyDBTitle converts a database title to a filesystem-safe slug.
// Lowercase, spaces replaced with hyphens, non-alphanumeric stripped, trimmed to 50 chars.
func slugifyDBTitle(title string) string {
	// Lowercase
	s := strings.ToLower(title)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Strip non-alphanumeric characters (keep hyphens)
	re := regexp.MustCompile(`[^a-z0-9-]`)
	s = re.ReplaceAllString(s, "")

	// Collapse multiple hyphens
	re = regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Trim to 50 chars
	if len(s) > 50 {
		s = s[:50]
		// Don't end on a hyphen after truncation
		s = strings.TrimRight(s, "-")
	}

	// Fallback if empty
	if s == "" {
		s = "untitled"
	}

	return s
}

// toInterfaceSlice converts []map[string]interface{} to interface{} for extractTitlePlainText.
func toInterfaceSlice(items []map[string]interface{}) interface{} {
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

// readBackupMeta reads and parses a .backup-meta.json file.
func readBackupMeta(path string) (*backupMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta backupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
