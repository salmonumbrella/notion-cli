package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newSearchCmd() *cobra.Command {
	var filterType string
	var sortJSON string
	var startCursor string
	var pageSize int
	var all bool
	var resultsOnly bool

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Notion by title",
		Long: `Search for pages and databases in Notion by title.

The query argument is the text to search for (optional).
Use --filter to limit results to "page" or "database" only.
Use --sort to specify sort order (JSON object with "direction" and "timestamp" keys).
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.

Example - Search for all pages and databases:
  notion search

Example - Search by query:
  notion search "project"

Example - Search only pages:
  notion search "meeting notes" --filter page

Example - Search with sort (most recently edited first):
  notion search --sort '{"direction":"descending","timestamp":"last_edited_time"}'

Example - Search with pagination:
  notion search "tasks" --page-size 10 --start-cursor abc123

Example - Fetch all results:
  notion search "project" --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get query from args if provided
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			sortField, sortDesc := output.SortFromContext(ctx)

			// Resolve and parse sort if provided
			var sort map[string]interface{}
			if sortJSON != "" {
				resolved, err := readJSONInput(sortJSON)
				if err != nil {
					return err
				}
				sortJSON = resolved
				if err := json.Unmarshal([]byte(sortJSON), &sort); err != nil {
					return fmt.Errorf("failed to parse sort JSON: %w", err)
				}
			}
			if sortJSON == "" && sortField != "" {
				if sortField != "created_time" && sortField != "last_edited_time" {
					return fmt.Errorf("--sort-by must be created_time or last_edited_time for search")
				}
				direction := "ascending"
				if sortDesc {
					direction = "descending"
				}
				sort = map[string]interface{}{
					"direction": direction,
					"timestamp": sortField,
				}
			}

			// Build filter if filterType is provided
			var filter map[string]interface{}
			if filterType != "" {
				if filterType != "page" && filterType != "database" {
					return fmt.Errorf("filter must be either 'page' or 'database'")
				}
				// Notion API 2025-09-03+ uses "data_source" instead of "database"
				apiFilterValue := filterType
				if filterType == "database" {
					apiFilterValue = "data_source"
				}
				filter = map[string]interface{}{
					"property": "object",
					"value":    apiFilterValue,
				}
			}

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// If --all flag is set, fetch all pages
			format := output.FormatFromContext(ctx)

			if all {
				var allResults []map[string]interface{}
				cursor := startCursor
				hasMore := false
				var nextCursor *string

				for {
					req := &notion.SearchRequest{
						Query:       query,
						Sort:        sort,
						Filter:      filter,
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					result, err := client.Search(ctx, req)
					if err != nil {
						return fmt.Errorf("failed to search: %w", err)
					}

					allResults = append(allResults, result.Results...)
					hasMore = result.HasMore
					nextCursor = result.NextCursor

					if limit > 0 && len(allResults) >= limit {
						allResults = allResults[:limit]
						break
					}

					if !result.HasMore || result.NextCursor == nil || *result.NextCursor == "" {
						break
					}
					cursor = *result.NextCursor
				}

				printer := output.NewPrinter(os.Stdout, GetOutputFormat())
				if resultsOnly || format == output.FormatTable {
					return printer.Print(ctx, allResults)
				}
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     allResults,
					"has_more":    hasMore,
					"next_cursor": nextCursor,
				})
			}

			// Single page request
			req := &notion.SearchRequest{
				Query:       query,
				Sort:        sort,
				Filter:      filter,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.Search(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to search: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			if resultsOnly || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterType, "filter", "", "Filter by object type (page or database)")
	cmd.Flags().StringVar(&sortJSON, "sort", "", "Sort as JSON object (e.g., {\"direction\":\"descending\",\"timestamp\":\"last_edited_time\"})")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")

	return cmd
}
