package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newDataSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "datasource",
		Aliases: []string{"ds"},
		Short:   "Manage Notion data sources",
		Long: `Create, retrieve, query, and update Notion data sources (API v2025-09-03).

Use 'datasource' when you have a data source ID directly, need to manage templates,
or work with multi-source databases. For single-source databases, 'datasource query'
is equivalent to 'db query'. Use 'db' instead when you need database-level metadata
(title, icon, cover, archived) or when querying by database name.`,
	}

	cmd.AddCommand(newDataSourceGetCmd())
	cmd.AddCommand(newDataSourceCreateCmd())
	cmd.AddCommand(newDataSourceUpdateCmd())
	cmd.AddCommand(newDataSourceQueryCmd())
	cmd.AddCommand(newDataSourceTemplatesCmd())
	cmd.AddCommand(newDataSourceListCmd())

	return cmd
}

func newDataSourceGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <datasource-id>",
		Aliases: []string{"g"},
		Short:   "Get a data source by ID",
		Long: `Retrieve a Notion data source by its ID.

Example:
  ntn datasource get 12345678-1234-1234-1234-123456789012
  ntn ds get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			ds, err := client.GetDataSource(ctx, dataSourceID)
			if err != nil {
				return wrapAPIError(err, "get data source", "data source", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, ds)
		},
	}
}

func newDataSourceCreateCmd() *cobra.Command {
	var parentID string
	var propertiesJSON string
	var propertiesFile string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a new data source",
		Long: `Create a new Notion data source.

The --parent flag specifies the parent database ID (required).
The --properties flag accepts a JSON object defining the data source schema (required).

Example:
  ntn datasource create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --properties '{"Name":{"title":{}},"Status":{"select":{}}}'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			if parentID == "" {
				return fmt.Errorf("--parent is required")
			}
			if propertiesJSON == "" && propertiesFile == "" {
				return fmt.Errorf("--properties or --properties-file is required")
			}

			normalizedParent, err := cmdutil.NormalizeNotionID(resolveID(sf, parentID))
			if err != nil {
				return err
			}
			parentID = normalizedParent

			properties, resolved, err := resolveAndDecodeJSON[map[string]interface{}](propertiesJSON, propertiesFile, "invalid properties JSON")
			if err != nil {
				return err
			}
			propertiesJSON = resolved

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			req := &notion.CreateDataSourceRequest{
				Parent:     map[string]interface{}{"database_id": parentID},
				Properties: properties,
			}

			ds, err := client.CreateDataSource(ctx, req)
			if err != nil {
				return wrapAPIError(err, "create data source", "data source", parentID)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent database ID (required)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON (required, @file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")

	return cmd
}

func newDataSourceUpdateCmd() *cobra.Command {
	var propertiesJSON string
	var propertiesFile string

	cmd := &cobra.Command{
		Use:     "update <datasource-id>",
		Aliases: []string{"u"},
		Short:   "Update a data source",
		Long: `Update a Notion data source's properties.

The --properties flag accepts a JSON object to update the data source schema.

Example:
  ntn datasource update 12345678-1234-1234-1234-123456789012 \
    --properties '{"Priority":{"select":{}}}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			var properties map[string]interface{}
			if hasJSONInput(propertiesJSON, propertiesFile) {
				parsed, resolved, err := resolveAndDecodeJSON[map[string]interface{}](propertiesJSON, propertiesFile, "invalid properties JSON")
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				properties = parsed
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			req := &notion.UpdateDataSourceRequest{
				Properties: properties,
			}

			ds, err := client.UpdateDataSource(ctx, dataSourceID, req)
			if err != nil {
				return wrapAPIError(err, "update data source", "data source", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")

	// Flag aliases
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")

	return cmd
}

func newDataSourceQueryCmd() *cobra.Command {
	var filterJSON string
	var filterFile string
	var sortsJSON string
	var sortsFile string
	var startCursor string
	var pageSize int
	var all bool
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
		Use:     "query <datasource-id>",
		Aliases: []string{"q"},
		Short:   "Query a data source",
		Long: `Query a Notion data source with optional filters, sorts, and pagination.

The --filter flag accepts a JSON object representing the filter (supports @file or - for stdin).
The --filter-file flag reads filter JSON from a file (useful for complex filters; - for stdin).
The --sorts flag accepts a JSON array of sort objects (supports @file or - for stdin).
The --sorts-file flag reads sorts JSON from a file (- for stdin).
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use global --results-only to output just the results array.

AGENT-FRIENDLY FILTER SHORTHANDS:
You can avoid writing JSON filters for common fields:
  --status "In Progress"         Filter Status equals value (property type: status or select)
  --assignee me                  Filter Assignee contains user (property type: people)
  --priority High                Filter Priority equals value (property type: select or status)

These shorthands require fetching the data source schema once to determine the
correct filter shape. They combine with --filter using AND.

Example - Query all pages:
  ntn datasource query 12345678-1234-1234-1234-123456789012

Example - Query with filter:
  ntn datasource query 12345678-1234-1234-1234-123456789012 \
    --filter '{"property":"Status","select":{"equals":"Active"}}' \
    --page-size 10

Example - Query with filter from file (avoids shell escaping issues):
  ntn datasource query 12345678-1234-1234-1234-123456789012 --filter @filter.json

Example - Query with sorts:
  ntn datasource query 12345678-1234-1234-1234-123456789012 \
    --sorts '[{"property":"Created","direction":"descending"}]'

Example - Sort by created time (shorthand):
  ntn datasource query 12345678-1234-1234-1234-123456789012 \
    --sort-by created_time --desc

Example - Fetch all results:
  ntn datasource query 12345678-1234-1234-1234-123456789012 --all

Example - Query with pagination:
  ntn datasource query 12345678-1234-1234-1234-123456789012 --page-size 10 --start-cursor abc123

Note: When using multi-line commands with backslash (\), ensure there are no
trailing spaces after the backslash. Otherwise the shell may split the command
incorrectly, causing "accepts 1 arg(s), received N" errors.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
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

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Build server-side shorthand filters (optional).
			shorthandFilters, err := buildDBQueryShorthandFilters(ctx, client, sf, dataSourceID,
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

					result, err := client.QueryDataSource(ctx, dataSourceID, req)
					if err != nil {
						return nil, nil, false, err
					}

					return result.Results, result.NextCursor, result.HasMore, nil
				})
				if err != nil {
					return wrapAPIError(err, "query data source", "data source", args[0])
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

			result, err := client.QueryDataSource(ctx, dataSourceID, req)
			if err != nil {
				return wrapAPIError(err, "query data source", "data source", args[0])
			}

			if selectProperty != "" && (selectEquals != "" || selectNot != "" || selectMatch != "") {
				filtered, err := filterResultsBySelect(result.Results, selectProperty, selectEquals, selectNot, selectMatch)
				if err != nil {
					return err
				}
				result.Results = filtered
			}

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
	flagAlias(cmd.Flags(), "status-prop", "sp")

	return cmd
}

func filterResultsBySelect(results []notion.Page, propName, equals, notEquals, match string) ([]notion.Page, error) {
	var re *regexp.Regexp
	var err error
	if match != "" {
		re, err = regexp.Compile(match)
		if err != nil {
			return nil, fmt.Errorf("invalid --select-match regex: %w", err)
		}
	}

	filtered := make([]notion.Page, 0, len(results))
	for _, item := range results {
		prop, ok := item.Properties[propName].(map[string]interface{})
		if !ok {
			continue
		}

		names := extractSelectNames(prop)
		if len(names) == 0 {
			continue
		}

		if notEquals != "" {
			excluded := false
			for _, name := range names {
				if name == notEquals {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, item)
			}
			continue
		}

		matched := false
		for _, name := range names {
			if equals != "" && name == equals {
				matched = true
				break
			}
			if re != nil && re.MatchString(name) {
				matched = true
				break
			}
		}

		if matched {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

func extractSelectNames(prop map[string]interface{}) []string {
	if prop == nil {
		return nil
	}

	if sel, ok := prop["select"].(map[string]interface{}); ok {
		if name, ok := sel["name"].(string); ok && name != "" {
			return []string{name}
		}
	}

	if status, ok := prop["status"].(map[string]interface{}); ok {
		if name, ok := status["name"].(string); ok && name != "" {
			return []string{name}
		}
	}

	if multi, ok := prop["multi_select"].([]interface{}); ok {
		out := make([]string, 0, len(multi))
		for _, item := range multi {
			if m, ok := item.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok && name != "" {
					out = append(out, name)
				}
			}
		}
		return out
	}

	return nil
}

func newDataSourceTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "templates",
		Aliases: []string{"t"},
		Short:   "List data source templates",
		Long: `List available Notion data source templates.

Example:
  ntn datasource templates
  ntn ds templates`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			list, err := client.ListDataSourceTemplates(ctx)
			if err != nil {
				return wrapAPIError(err, "list data source templates", "data source", "templates")
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, list)
		},
	}
}

func newDataSourceListCmd() *cobra.Command {
	var pageSize int
	var all bool
	var startCursor string
	var light bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all data sources in the workspace",
		Long: `List all data sources (databases) in the Notion workspace.

This command searches for all databases accessible to the integration.
Use --page-size to control results per page (max 100).
Use --all to fetch all pages of results automatically.
Use --light (or --li) for compact output (id, object, title, url).
Use global --results-only to output just the results array (useful for piping to jq).

Example:
  ntn datasource list
  ntn ds list --li
  ntn ds list --all --results-only
  ntn ds list -o json | jq '.results[].id'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)

			pageSize = capPageSize(pageSize, limit)

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Use search with database filter
			filter := map[string]interface{}{
				"property": "object",
				"value":    "data_source",
			}

			if all {
				allResults, nextCursor, hasMore, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]map[string]interface{}, *string, bool, error) {
					req := &notion.SearchRequest{
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
					return wrapAPIError(err, "list data sources", "data source", "workspace")
				}
				allResults = normalizeDataSourceSearchResults(allResults)
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

			// Single page request
			req := &notion.SearchRequest{
				Filter:      filter,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.Search(ctx, req)
			if err != nil {
				return wrapAPIError(err, "list data sources", "data source", "workspace")
			}
			result.Results = normalizeDataSourceSearchResults(result.Results)

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

	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, object, title, url)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

// normalizeDataSourceSearchResults makes ds list JSON output easier to consume:
// - title is always an array (never null/missing)
// - name is a plain-text title fallback for scripts
// - title_plain_text mirrors name explicitly
func normalizeDataSourceSearchResults(results []map[string]interface{}) []map[string]interface{} {
	for _, item := range results {
		if item == nil {
			continue
		}

		titleValue, hasTitle := item["title"]
		if !hasTitle || titleValue == nil {
			titleValue = []interface{}{}
			item["title"] = titleValue
		}

		titleText := extractTitlePlainText(titleValue)
		item["title_plain_text"] = titleText

		if _, hasName := item["name"]; !hasName {
			item["name"] = titleText
		}
	}
	return results
}
