package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var filterType string
	var sortJSON string
	var startCursor string
	var pageSize int
	var all bool

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

			// Parse sort if provided
			var sort map[string]interface{}
			if sortJSON != "" {
				if err := json.Unmarshal([]byte(sortJSON), &sort); err != nil {
					return fmt.Errorf("failed to parse sort JSON: %w", err)
				}
			}

			// Build filter if filterType is provided
			var filter map[string]interface{}
			if filterType != "" {
				if filterType != "page" && filterType != "database" {
					return fmt.Errorf("filter must be either 'page' or 'database'")
				}
				filter = map[string]interface{}{
					"property": "object",
					"value":    filterType,
				}
			}

			// Validate page size
			if pageSize > 100 {
				return fmt.Errorf("page-size must be between 1 and 100")
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// If --all flag is set, fetch all pages
			if all {
				var allResults []map[string]interface{}
				cursor := startCursor

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

					if !result.HasMore || result.NextCursor == nil || *result.NextCursor == "" {
						break
					}
					cursor = *result.NextCursor
				}

				// Print all results
				printer := output.NewPrinter(os.Stdout, GetOutputFormat())
				return printer.Print(ctx, allResults)
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
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterType, "filter", "", "Filter by object type (page or database)")
	cmd.Flags().StringVar(&sortJSON, "sort", "", "Sort as JSON object (e.g., {\"direction\":\"descending\",\"timestamp\":\"last_edited_time\"})")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")

	return cmd
}
