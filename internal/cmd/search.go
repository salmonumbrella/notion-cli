package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newSearchCmd() *cobra.Command {
	var filterType string
	var sortJSON string
	var startCursor string
	var pageSize int
	var all bool
	var textQuery string
	var light bool

	cmd := &cobra.Command{
		Use:     "search [query]",
		Aliases: []string{"s", "find"},
		Short:   "Search Notion by title",
		Long: `Search for pages and databases in Notion by title.

The query argument is the text to search for (optional).
Use --filter to limit results to "page" or "database" only.
Use --sort to specify sort order (JSON object with "direction" and "timestamp" keys).
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use --light (or --li) for compact output (id, object, title, url).
Use global --results-only to output just the results array (useful for piping to jq).

Note: The global --query flag is for jq filtering, not the search term.
Use the positional argument or --text for the search term.

Example - Search for all pages and databases:
  ntn search

Example - Search by query:
  ntn search "project"
  ntn search "project" --li

Example - Search using --text flag:
  ntn search --text "project"

Example - Search only pages:
  ntn search "meeting notes" --filter page

Example - Search with sort (most recently edited first):
  ntn search --sort '{"direction":"descending","timestamp":"last_edited_time"}'

Example - Search with pagination:
  ntn search "tasks" --page-size 10 --start-cursor abc123

Example - Fetch all results:
  ntn search "project" --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get query from args or --text flag
			var query string
			if len(args) > 0 {
				query = args[0]
			}
			if textQuery != "" {
				if query != "" {
					return fmt.Errorf("cannot specify both positional query and --text flag")
				}
				query = textQuery
			}

			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			sortField, sortDesc := output.SortFromContext(ctx)

			// Resolve and parse sort if provided
			var sort map[string]interface{}
			if sortJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(sortJSON)
				if err != nil {
					return err
				}
				sortJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(sortJSON, &sort); err != nil {
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
			if filterType != "" && filterType != "page" && filterType != "database" {
				return fmt.Errorf("filter must be either 'page' or 'database'")
			}
			filter := buildSearchFilter(filterType)

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// If --all flag is set, fetch all pages
			if all {
				allResults, nextCursor, hasMore, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]map[string]interface{}, *string, bool, error) {
					req := &notion.SearchRequest{
						Query:       query,
						Sort:        sort,
						Filter:      filter,
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					result, err := client.Search(ctx, req)
					if err != nil {
						return nil, nil, false, err
					}

					return result.Results, result.NextCursor, result.HasMore, nil
				})
				if err != nil {
					return fmt.Errorf("failed to search: %w", err)
				}

				outResults := interface{}(allResults)
				if light {
					outResults = toLightSearchResults(allResults)
				}
				printer := printerForContext(ctx)
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     outResults,
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
			printer := printerForContext(ctx)
			if light {
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     toLightSearchResults(result.Results),
					"has_more":    result.HasMore,
					"next_cursor": result.NextCursor,
				})
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterType, "filter", "", "Filter by object type (page or database)")
	cmd.Flags().StringVar(&sortJSON, "sort", "", "Sort as JSON object (e.g., {\"direction\":\"descending\",\"timestamp\":\"last_edited_time\"})")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().StringVarP(&textQuery, "text", "t", "", "Search text (alternative to positional argument)")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, object, title, url)")

	// Flag aliases
	flagAlias(cmd.Flags(), "filter", "fi")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
