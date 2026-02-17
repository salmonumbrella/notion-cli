package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "db",
		Aliases: []string{"database", "databases"},
		Short:   "Manage Notion databases",
		Long: `Retrieve, query, create, and update Notion databases.

Use 'db' when you need database-level metadata (title, icon, cover, archived) or
when querying by database name. 'db query' resolves the database to its primary
data source automatically. Use 'datasource' instead when you have a data source ID
directly, need to manage templates, or work with multi-source databases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// When invoked without subcommand, default to list
			listCmd := newDBListCmd()
			listCmd.SetContext(cmd.Context())
			return listCmd.RunE(listCmd, args)
		},
	}

	cmd.AddCommand(newDBGetCmd())
	cmd.AddCommand(newDBListCmd())
	cmd.AddCommand(newDBQueryCmd())
	cmd.AddCommand(newDBCreateCmd())
	cmd.AddCommand(newDBUpdateCmd())
	cmd.AddCommand(newDBBackupCmd())

	return cmd
}

func newDBListCmd() *cobra.Command {
	var startCursor string
	var pageSize int
	var all bool
	var titleMatch string
	var light bool

	cmd := &cobra.Command{
		Use:     "list [query]",
		Aliases: []string{"ls"},
		Short:   "List databases",
		Long: `List Notion databases (data sources) with optional title search.

Example - List databases:
  ntn db list

Example - Search by title:
  ntn db list "Vendor"
  ntn db list --li

Example - Fetch all results:
  ntn db list --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)

			var titleRE *regexp.Regexp
			if titleMatch != "" {
				compiled, err := regexp.Compile(titleMatch)
				if err != nil {
					return fmt.Errorf("invalid --title-match regex: %w", err)
				}
				titleRE = compiled
			}

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}
			filter := map[string]interface{}{
				"property": "object",
				"value":    "data_source",
			}

			if all {
				allResults, nextCursor, hasMore, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]map[string]interface{}, *string, bool, error) {
					req := &notion.SearchRequest{
						Query:       query,
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
					return wrapAPIError(err, "list databases", "database", query)
				}

				if titleRE != nil {
					allResults = filterDataSourcesByTitle(allResults, titleRE)
				}

				printer := printerForContext(ctx)
				if light {
					return printer.Print(ctx, map[string]interface{}{
						"object":      "list",
						"results":     toLightSearchResults(allResults),
						"has_more":    hasMore,
						"next_cursor": nextCursor,
					})
				}
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     allResults,
					"has_more":    hasMore,
					"next_cursor": nextCursor,
				})
			}

			req := &notion.SearchRequest{
				Query:       query,
				Filter:      filter,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.Search(ctx, req)
			if err != nil {
				return wrapAPIError(err, "list databases", "database", query)
			}

			if titleRE != nil {
				result.Results = filterDataSourcesByTitle(result.Results, titleRE)
			}
			if limit > 0 && len(result.Results) > limit {
				result.Results = result.Results[:limit]
			}

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

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().StringVar(&titleMatch, "title-match", "", "Regex to match database title (Go syntax, use (?i) for case-insensitive). Note: filtering is applied after fetching, so fewer results may be returned when combined with --limit")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, object, title, url)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func filterDataSourcesByTitle(results []map[string]interface{}, re *regexp.Regexp) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0, len(results))
	for _, item := range results {
		title := extractTitlePlainText(item["title"])
		if title == "" {
			continue
		}
		if re.MatchString(title) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func extractTitlePlainText(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok {
		return ""
	}

	var parts []string
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if plain, ok := entry["plain_text"].(string); ok && plain != "" {
			parts = append(parts, plain)
			continue
		}
		if text, ok := entry["text"].(map[string]interface{}); ok {
			if content, ok := text["content"].(string); ok && content != "" {
				parts = append(parts, content)
			}
		}
	}
	return strings.Join(parts, "")
}

func newDBGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <database-id-or-name>",
		Aliases: []string{"g"},
		Short:   "Get a database by ID or name",
		Long: `Retrieve a Notion database by its ID or name.

If you provide a name instead of an ID, the CLI will search for matching databases.

Example:
  ntn db get 12345678-1234-1234-1234-123456789012
  ntn db get "Projects"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve ID with search fallback
			databaseID, err := resolveIDWithSearch(ctx, client, sf, args[0], "database")
			if err != nil {
				return err
			}
			databaseID, err = cmdutil.NormalizeNotionID(databaseID)
			if err != nil {
				return err
			}

			database, err := client.GetDatabase(ctx, databaseID)
			if err != nil {
				if hinted := maybeDataSourceHintForDatabaseNotFound(ctx, client, err, databaseID); hinted != nil {
					return hinted
				}
				return wrapAPIError(err, "get database", "database", args[0])
			}

			// In API 2025-09-03+, properties live on data sources, not databases.
			// Auto-populate properties from the primary data source for convenience.
			if database.Properties == nil && len(database.DataSources) == 1 {
				dataSource, err := client.GetDataSource(ctx, database.DataSources[0].ID)
				if err == nil && dataSource.Properties != nil {
					// Convert map[string]interface{} to map[string]map[string]interface{}
					database.Properties = make(map[string]map[string]interface{})
					for k, v := range dataSource.Properties {
						if propMap, ok := v.(map[string]interface{}); ok {
							database.Properties[k] = propMap
						}
					}
				}
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, database)
		},
	}
}

func maybeDataSourceHintForDatabaseNotFound(ctx context.Context, client dataSourceGetter, dbErr error, identifier string) error {
	if dbErr == nil || client == nil {
		return nil
	}
	if !looksLikeUUID(identifier) {
		return nil
	}
	if clierrors.APINotFoundError(dbErr, "database", identifier) == dbErr {
		return nil
	}

	ds, err := client.GetDataSource(ctx, identifier)
	if err != nil || ds == nil {
		return nil
	}

	title := dataSourceTitle(ds)
	msg := fmt.Sprintf("The ID %q is a data source, not a database.", identifier)
	if title != "" {
		msg = fmt.Sprintf("The ID %q is the data source %q, not a database.", identifier, title)
	}

	return clierrors.WrapUserError(
		dbErr,
		"failed to get database",
		fmt.Sprintf("%s\n\nUse one of:\n  • ntn datasource get %s\n  • ntn datasource query %s\n  • ntn db query <database-id> --datasource %s",
			msg, identifier, identifier, identifier),
	)
}

func dataSourceTitle(ds *notion.DataSource) string {
	if ds == nil || len(ds.Title) == 0 {
		return ""
	}

	var b strings.Builder
	for _, item := range ds.Title {
		if strings.TrimSpace(item.PlainText) != "" {
			b.WriteString(item.PlainText)
			continue
		}
		if item.Text != nil && strings.TrimSpace(item.Text.Content) != "" {
			b.WriteString(item.Text.Content)
		}
	}

	return strings.TrimSpace(b.String())
}

func newDBQueryCmd() *cobra.Command {
	var filterJSON string
	var filterFile string
	var sortsJSON string
	var sortsFile string
	var startCursor string
	var pageSize int
	var all bool
	var dataSourceID string
	var selectProperty string
	var selectEquals string
	var selectNot string
	var selectMatch string
	var statusEquals string
	var statusProperty string
	var assigneeContains string
	var assigneeProperty string
	var priorityEquals string
	var priorityProperty string

	cmd := &cobra.Command{
		Use:     "query <database-id-or-name>",
		Aliases: []string{"q"},
		Short:   "Query a database",
		Long: `Query a Notion database with optional filters and sorts.

If you provide a name instead of an ID, the CLI will search for matching databases.

The --filter flag accepts a JSON object representing the filter (supports @file or - for stdin).
The --filter-file flag reads filter JSON from a file (useful for complex filters; - for stdin).
The --sorts flag accepts a JSON array of sort objects (supports @file or - for stdin).
The --sorts-file flag reads sorts JSON from a file (- for stdin).
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use --datasource to query a specific data source in a multi-source database.
Use global --results-only to output just the results array.

AGENT-FRIENDLY FILTER SHORTHANDS:
You can avoid writing JSON filters for common fields:
  --status "In Progress"         Filter Status equals value (property type: status or select)
  --assignee me                  Filter Assignee contains user (property type: people)
  --priority High                Filter Priority equals value (property type: select or status)

These shorthands require fetching the data source schema once to determine the
correct filter shape. They combine with --filter using AND.

Example - Query all pages:
  ntn db query 12345678-1234-1234-1234-123456789012

Example - Query by name:
  ntn db query "Projects"

Example - Query with filter (single line recommended):
  ntn db query 12345678-1234-1234-1234-123456789012 --filter '{"property":"Status","select":{"equals":"Done"}}'

Example - Query with filter from file (avoids shell escaping issues):
  ntn db query 12345678-1234-1234-1234-123456789012 --filter @filter.json

Example - Query with sorts:
  ntn db query 12345678-1234-1234-1234-123456789012 --sorts '[{"property":"Created","direction":"descending"}]'

Example - Query with pagination:
  ntn db query 12345678-1234-1234-1234-123456789012 --page-size 10 --start-cursor abc123

Example - Fetch all results:
  ntn db query 12345678-1234-1234-1234-123456789012 --all

Note: When using multi-line commands with backslash (\), ensure there are no
trailing spaces after the backslash. Otherwise the shell may split the command
incorrectly, causing "accepts 1 arg(s), received N" errors.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve ID with search fallback
			databaseID, err := resolveIDWithSearch(ctx, client, sf, args[0], "database")
			if err != nil {
				return err
			}
			databaseID, err = cmdutil.NormalizeNotionID(databaseID)
			if err != nil {
				return err
			}
			if dataSourceID != "" {
				normalized, err := cmdutil.NormalizeNotionID(resolveID(sf, dataSourceID))
				if err != nil {
					return err
				}
				dataSourceID = normalized
			}
			limit := output.LimitFromContext(ctx)
			sortField, sortDesc := output.SortFromContext(ctx)

			if (selectEquals != "" || selectNot != "" || selectMatch != "") && selectProperty == "" {
				return fmt.Errorf("--select-property is required when using --select-equals, --select-not, or --select-match")
			}
			selectionFlags := 0
			if selectEquals != "" {
				selectionFlags++
			}
			if selectNot != "" {
				selectionFlags++
			}
			if selectMatch != "" {
				selectionFlags++
			}
			if selectionFlags > 1 {
				return fmt.Errorf("use only one of --select-equals, --select-not, or --select-match")
			}

			// Back-compat: if --filter-file is set, treat it as the source of --filter.
			// Also enables stdin via --filter-file -.
			if filterFile != "" {
				filterJSON = "@" + filterFile
				if strings.TrimSpace(filterFile) == "-" {
					filterJSON = "-"
				}
			}

			var filter map[string]interface{}
			if filterJSON != "" {
				parsedFilter, resolved, err := readAndDecodeJSON[map[string]interface{}](filterJSON, "failed to parse filter JSON")
				if err != nil {
					return err
				}
				filterJSON = resolved
				filter = parsedFilter
			}

			// Back-compat: if --sorts-file is set, treat it as the source of --sorts.
			// Also enables stdin via --sorts-file -.
			if sortsFile != "" {
				sortsJSON = "@" + sortsFile
				if strings.TrimSpace(sortsFile) == "-" {
					sortsJSON = "-"
				}
			}

			var sorts []map[string]interface{}
			if sortsJSON != "" {
				parsedSorts, resolved, err := readAndDecodeJSON[[]map[string]interface{}](sortsJSON, "failed to parse sorts JSON")
				if err != nil {
					return err
				}
				sortsJSON = resolved
				sorts = parsedSorts
			}
			if sortsJSON == "" {
				if s := buildSortFromFlags(sortField, sortDesc); s != nil {
					sorts = s
				}
			}

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			resolvedDataSourceID, err := resolveDataSourceID(ctx, client, databaseID, dataSourceID)
			if err != nil {
				return err
			}

			// Build server-side shorthand filters (optional).
			shorthandFilters, err := buildDBQueryShorthandFilters(ctx, client, sf, resolvedDataSourceID,
				statusProperty, statusEquals,
				assigneeProperty, assigneeContains,
				priorityProperty, priorityEquals,
			)
			if err != nil {
				return err
			}
			if len(shorthandFilters) > 0 {
				filter = mergeNotionFilters(filter, shorthandFilters)
			}

			// If --all flag is set, fetch all pages
			if all {
				allPages, nextCursor, hasMore, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]notion.Page, *string, bool, error) {
					req := &notion.QueryDataSourceRequest{
						Filter:      filter,
						Sorts:       sorts,
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					result, err := client.QueryDataSource(ctx, resolvedDataSourceID, req)
					if err != nil {
						return nil, nil, false, err
					}

					return result.Results, result.NextCursor, result.HasMore, nil
				})
				if err != nil {
					return wrapAPIError(err, "query database", "database", args[0])
				}

				if selectProperty != "" && (selectEquals != "" || selectNot != "" || selectMatch != "") {
					filtered, err := filterResultsBySelect(allPages, selectProperty, selectEquals, selectNot, selectMatch)
					if err != nil {
						return err
					}
					allPages = filtered
				}

				printer := printerForContext(ctx)
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     allPages,
					"has_more":    hasMore,
					"next_cursor": nextCursor,
				})
			}

			// Single page request
			req := &notion.QueryDataSourceRequest{
				Filter:      filter,
				Sorts:       sorts,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.QueryDataSource(ctx, resolvedDataSourceID, req)
			if err != nil {
				return wrapAPIError(err, "query database", "database", args[0])
			}

			if selectProperty != "" && (selectEquals != "" || selectNot != "" || selectMatch != "") {
				filtered, err := filterResultsBySelect(result.Results, selectProperty, selectEquals, selectNot, selectMatch)
				if err != nil {
					return err
				}
				result.Results = filtered
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterJSON, "filter", "", "Filter as JSON object (@file or - for stdin supported)")
	cmd.Flags().StringVar(&filterFile, "filter-file", "", "Read filter JSON from file (- for stdin)")
	cmd.Flags().StringVar(&sortsJSON, "sorts", "", "Sorts as JSON array (@file or - for stdin supported)")
	cmd.Flags().StringVar(&sortsFile, "sorts-file", "", "Read sorts JSON from file (- for stdin)")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().StringVar(&dataSourceID, "datasource", "", "Data source ID to query (optional)")
	cmd.Flags().StringVar(&selectProperty, "select-property", "", "Property name to match (select, multi_select, or status)")
	cmd.Flags().StringVar(&selectEquals, "select-equals", "", "Match select name exactly")
	cmd.Flags().StringVar(&selectNot, "select-not", "", "Exclude items where select name matches exactly")
	cmd.Flags().StringVar(&selectMatch, "select-match", "", "Match select name with regex (Go syntax, use (?i) for case-insensitive). Note: filtering is applied after fetching, so fewer results may be returned when combined with --limit")
	cmd.Flags().StringVar(&statusEquals, "status", "", "Shorthand: filter Status equals value (type status/select; requires schema lookup)")
	cmd.Flags().StringVar(&statusProperty, "status-prop", "Status", "Property name to use for --status")
	cmd.Flags().StringVar(&assigneeContains, "assignee", "", "Shorthand: filter Assignee contains user (type people; accepts skill alias)")
	cmd.Flags().StringVar(&assigneeContains, "assigned-to", "", "Alias for --assignee")
	_ = cmd.Flags().MarkHidden("assigned-to")
	cmd.Flags().StringVar(&assigneeProperty, "assignee-prop", "Assignee", "Property name to use for --assignee")
	cmd.Flags().StringVar(&priorityEquals, "priority", "", "Shorthand: filter Priority equals value (type select/status; requires schema lookup)")
	cmd.Flags().StringVar(&priorityProperty, "priority-prop", "Priority", "Property name to use for --priority")

	// Flag aliases
	flagAlias(cmd.Flags(), "filter", "fi")
	flagAlias(cmd.Flags(), "filter-file", "ff")
	flagAlias(cmd.Flags(), "datasource", "ds")
	flagAlias(cmd.Flags(), "status-prop", "sp")

	return cmd
}
