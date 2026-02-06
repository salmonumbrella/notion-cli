package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "db",
		Aliases: []string{"database", "databases"},
		Short:   "Manage Notion databases",
		Long:    `Retrieve, query, create, and update Notion databases.`,
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

	return cmd
}

func newDBListCmd() *cobra.Command {
	var startCursor string
	var pageSize int
	var all bool
	var titleMatch string

	cmd := &cobra.Command{
		Use:   "list [query]",
		Short: "List databases",
		Long: `List Notion databases (data sources) with optional title search.

Example - List databases:
  notion db list

Example - Search by title:
  notion db list "Vendor"

Example - Fetch all results:
  notion db list --all`,
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

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)
			filter := map[string]interface{}{
				"property": "object",
				"value":    "data_source",
			}

			if all {
				allResults, _, _, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]map[string]interface{}, *string, bool, error) {
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
					return fmt.Errorf("failed to list databases: %w", err)
				}

				if titleRE != nil {
					allResults = filterDataSourcesByTitle(allResults, titleRE)
				}

				printer := printerForContext(ctx)
				return printer.Print(ctx, allResults)
			}

			req := &notion.SearchRequest{
				Query:       query,
				Filter:      filter,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.Search(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to list databases: %w", err)
			}

			if titleRE != nil {
				result.Results = filterDataSourcesByTitle(result.Results, titleRE)
			}
			if limit > 0 && len(result.Results) > limit {
				result.Results = result.Results[:limit]
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result.Results)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")
	cmd.Flags().StringVar(&titleMatch, "title-match", "", "Regex to match database title (Go syntax, use (?i) for case-insensitive). Note: filtering is applied after fetching, so fewer results may be returned when combined with --limit")

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
		Use:   "get <database-id-or-name>",
		Short: "Get a database by ID or name",
		Long: `Retrieve a Notion database by its ID or name.

If you provide a name instead of an ID, the CLI will search for matching databases.

Example:
  notion db get 12345678-1234-1234-1234-123456789012
  notion db get "Projects"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Resolve ID with search fallback
			databaseID, err := resolveIDWithSearch(ctx, client, sf, args[0], "database")
			if err != nil {
				return err
			}
			databaseID, err = cmdutil.NormalizeNotionID(databaseID)
			if err != nil {
				return err
			}

			// Get database
			database, err := client.GetDatabase(ctx, databaseID)
			if err != nil {
				return errors.APINotFoundError(err, "database", args[0])
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
		Use:   "query <database-id-or-name>",
		Short: "Query a database",
		Long: `Query a Notion database with optional filters and sorts.

If you provide a name instead of an ID, the CLI will search for matching databases.

The --filter flag accepts a JSON object representing the filter (supports @file or - for stdin).
The --filter-file flag reads filter JSON from a file (useful for complex filters; - for stdin).
The --sorts flag accepts a JSON array of sort objects (supports @file or - for stdin).
The --sorts-file flag reads sorts JSON from a file (- for stdin).
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.
Use --all to fetch all pages of results automatically.
Use --data-source to query a specific data source in a multi-source database.
Use global --results-only to output just the results array.

AGENT-FRIENDLY FILTER SHORTHANDS:
You can avoid writing JSON filters for common fields:
  --status "In Progress"         Filter Status equals value (property type: status or select)
  --assignee me                  Filter Assignee contains user (property type: people)
  --priority High                Filter Priority equals value (property type: select or status)

These shorthands require fetching the data source schema once to determine the
correct filter shape. They combine with --filter using AND.

Example - Query all pages:
  notion db query 12345678-1234-1234-1234-123456789012

Example - Query by name:
  notion db query "Projects"

Example - Query with filter (single line recommended):
  notion db query 12345678-1234-1234-1234-123456789012 --filter '{"property":"Status","select":{"equals":"Done"}}'

Example - Query with filter from file (avoids shell escaping issues):
  notion db query 12345678-1234-1234-1234-123456789012 --filter @filter.json

Example - Query with sorts:
  notion db query 12345678-1234-1234-1234-123456789012 --sorts '[{"property":"Created","direction":"descending"}]'

Example - Query with pagination:
  notion db query 12345678-1234-1234-1234-123456789012 --page-size 10 --start-cursor abc123

Example - Fetch all results:
  notion db query 12345678-1234-1234-1234-123456789012 --all

Note: When using multi-line commands with backslash (\), ensure there are no
trailing spaces after the backslash. Otherwise the shell may split the command
incorrectly, causing "accepts 1 arg(s), received N" errors.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Get token early so we can use client for search resolution
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}
			client := NewNotionClient(ctx, token)

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
			format := output.FormatFromContext(ctx)

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

			// Resolve and parse filter if provided
			var filter map[string]interface{}
			if filterJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(filterJSON)
				if err != nil {
					return err
				}
				filterJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(filterJSON, &filter); err != nil {
					return fmt.Errorf("failed to parse filter JSON: %w", err)
				}
			}

			// Back-compat: if --sorts-file is set, treat it as the source of --sorts.
			// Also enables stdin via --sorts-file -.
			if sortsFile != "" {
				sortsJSON = "@" + sortsFile
				if strings.TrimSpace(sortsFile) == "-" {
					sortsJSON = "-"
				}
			}

			// Resolve and parse sorts if provided
			var sorts []map[string]interface{}
			if sortsJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(sortsJSON)
				if err != nil {
					return err
				}
				sortsJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(sortsJSON, &sorts); err != nil {
					return fmt.Errorf("failed to parse sorts JSON: %w", err)
				}
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
					return errors.APINotFoundError(err, "database", args[0])
				}

				if selectProperty != "" && (selectEquals != "" || selectNot != "" || selectMatch != "") {
					filtered, err := filterResultsBySelect(allPages, selectProperty, selectEquals, selectNot, selectMatch)
					if err != nil {
						return err
					}
					allPages = filtered
				}

				printer := printerForContext(ctx)
				if output.ResultsOnlyFromContext(ctx) || format == output.FormatTable {
					return printer.Print(ctx, allPages)
				}
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
				return errors.APINotFoundError(err, "database", args[0])
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
			if output.ResultsOnlyFromContext(ctx) || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
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
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID to query (optional)")
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

	return cmd
}

func newDBCreateCmd() *cobra.Command {
	var parentID string
	var titleText string
	var propertiesJSON string
	var propertiesFile string
	var dataSourceTitle string
	var descriptionJSON string
	var iconJSON string
	var coverJSON string
	var isInline bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new database",
		Long: `Create a new Notion database.

The --parent flag specifies the parent page ID (required).
The --title flag specifies the database title as plain text.
The --properties flag accepts a JSON object defining the database schema (required).
The --data-source-title flag sets the title of the initial data source (optional).

Example - Create a simple task database:
  notion db create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --title "Tasks" \
    --properties '{"Name":{"title":{}},"Status":{"select":{"options":[{"name":"Todo","color":"red"},{"name":"Done","color":"green"}]}}}'

Example - Create with description:
  notion db create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --title "Projects" \
    --description '[{"type":"text","text":{"content":"My projects database"}}]' \
    --properties '{"Name":{"title":{}}}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Validate required flags
			if parentID == "" {
				return fmt.Errorf("--parent flag is required")
			}
			if propertiesJSON == "" && propertiesFile == "" {
				return fmt.Errorf("--properties or --properties-file is required")
			}

			normalizedParent, err := cmdutil.NormalizeNotionID(resolveID(sf, parentID))
			if err != nil {
				return err
			}
			parentID = normalizedParent

			// Resolve and parse properties JSON
			var properties map[string]map[string]interface{}
			resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
			if err != nil {
				return err
			}
			propertiesJSON = resolved
			if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
				return fmt.Errorf("failed to parse properties JSON: %w", err)
			}

			// Build parent object
			parent := map[string]interface{}{
				"type":    "page_id",
				"page_id": parentID,
			}

			// Build title if provided
			var title []map[string]interface{}
			if titleText != "" {
				title = []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": titleText,
						},
					},
				}
			}

			// Build initial data source title if provided
			var dataSourceTitleRT []notion.RichText
			if dataSourceTitle != "" {
				dataSourceTitleRT = []notion.RichText{
					{
						Type: "text",
						Text: &notion.TextContent{Content: dataSourceTitle},
					},
				}
			}

			// Resolve and parse optional fields
			var description []map[string]interface{}
			if descriptionJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(descriptionJSON)
				if err != nil {
					return err
				}
				descriptionJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(descriptionJSON, &description); err != nil {
					return fmt.Errorf("failed to parse description JSON: %w", err)
				}
			}

			var icon map[string]interface{}
			if iconJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(iconJSON)
				if err != nil {
					return err
				}
				iconJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(iconJSON, &icon); err != nil {
					return fmt.Errorf("failed to parse icon JSON: %w", err)
				}
			}

			var cover map[string]interface{}
			if coverJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(coverJSON)
				if err != nil {
					return err
				}
				coverJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(coverJSON, &cover); err != nil {
					return fmt.Errorf("failed to parse cover JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Build request
			req := &notion.CreateDatabaseRequest{
				Parent:      parent,
				Title:       title,
				Description: description,
				Icon:        icon,
				Cover:       cover,
				IsInline:    isInline,
				InitialDataSource: &notion.InitialDataSource{
					Title:      dataSourceTitleRT,
					Properties: properties,
				},
			}

			// Create database
			database, err := client.CreateDatabase(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, database)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page ID (required)")
	cmd.Flags().StringVar(&titleText, "title", "", "Database title as plain text")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Database properties as JSON object (required, @file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().StringVar(&dataSourceTitle, "data-source-title", "", "Initial data source title (optional)")
	cmd.Flags().StringVar(&descriptionJSON, "description", "", "Database description as JSON array")
	cmd.Flags().StringVar(&iconJSON, "icon", "", "Database icon as JSON object")
	cmd.Flags().StringVar(&coverJSON, "cover", "", "Database cover as JSON object")
	cmd.Flags().BoolVar(&isInline, "inline", false, "Create as inline database")

	return cmd
}

func newDBUpdateCmd() *cobra.Command {
	var titleText string
	var propertiesJSON string
	var propertiesFile string
	var descriptionJSON string
	var iconJSON string
	var coverJSON string
	var archived bool
	var setArchived bool
	var dryRun bool
	var dataSourceID string

	cmd := &cobra.Command{
		Use:   "update <database-id>",
		Short: "Update a database",
		Long: `Update a Notion database's metadata and properties.

The --title flag updates the database title.
The --properties flag accepts a JSON object to update the data source schema.
Use --data-source to target a specific data source in a multi-source database.
The --description flag updates the database description.
The --archived flag archives or unarchives the database.

Example - Update title:
  notion db update 12345678-1234-1234-1234-123456789012 --title "Updated Tasks"

Example - Add a new property (data source schema):
  notion db update 12345678-1234-1234-1234-123456789012 \
    --properties '{"Priority":{"select":{"options":[{"name":"High","color":"red"},{"name":"Low","color":"blue"}]}}}'

Example - Archive database:
  notion db update 12345678-1234-1234-1234-123456789012 --archived true`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			databaseID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
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

			// Build title if provided
			var title []map[string]interface{}
			if titleText != "" {
				title = []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": titleText,
						},
					},
				}
			}

			// Resolve and parse properties if provided
			var properties map[string]map[string]interface{}
			if propertiesJSON != "" || propertiesFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Resolve and parse optional fields
			var description []map[string]interface{}
			if descriptionJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(descriptionJSON)
				if err != nil {
					return err
				}
				descriptionJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(descriptionJSON, &description); err != nil {
					return fmt.Errorf("failed to parse description JSON: %w", err)
				}
			}

			var icon map[string]interface{}
			if iconJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(iconJSON)
				if err != nil {
					return err
				}
				iconJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(iconJSON, &icon); err != nil {
					return fmt.Errorf("failed to parse icon JSON: %w", err)
				}
			}

			var cover map[string]interface{}
			if coverJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(coverJSON)
				if err != nil {
					return err
				}
				coverJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(coverJSON, &cover); err != nil {
					return fmt.Errorf("failed to parse cover JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			var resolvedDataSourceID string
			if propertiesJSON != "" {
				resolvedDataSourceID, err = resolveDataSourceID(ctx, client, databaseID, dataSourceID)
				if err != nil {
					return err
				}
			}

			if dryRun {
				// Fetch current database to show what would be updated
				currentDB, err := client.GetDatabase(ctx, databaseID)
				if err != nil {
					return fmt.Errorf("failed to fetch database: %w", err)
				}

				printer := NewDryRunPrinter(stderrFromContext(ctx))
				printer.Header("update", "database", databaseID)

				// Show title change if applicable
				if titleText != "" {
					currentTitle := ""
					if len(currentDB.Title) > 0 {
						if textObj, ok := currentDB.Title[0]["text"].(map[string]interface{}); ok {
							if content, ok := textObj["content"].(string); ok {
								currentTitle = content
							}
						}
					}
					printer.Change("Title", currentTitle, titleText)
				}

				// Show archived status change if applicable
				if setArchived {
					if archived != currentDB.Archived {
						printer.Change("Archived", fmt.Sprintf("%t", currentDB.Archived), fmt.Sprintf("%t", archived))
					} else {
						printer.Unchanged("Archived")
					}
				}

				// Show properties to update
				if propertiesJSON != "" {
					label := "Properties to update:"
					if resolvedDataSourceID != "" {
						label = fmt.Sprintf("Properties to update (data source %s):", resolvedDataSourceID)
					}
					printer.Section(label)
					for propName := range properties {
						if _, exists := currentDB.Properties[propName]; exists {
							_, _ = fmt.Fprintf(stderrFromContext(ctx), "  - %s (updating existing)\n", propName)
						} else {
							_, _ = fmt.Fprintf(stderrFromContext(ctx), "  - %s (adding new)\n", propName)
						}
					}
				}

				// Show description change if applicable
				if descriptionJSON != "" {
					printer.Section("Description:")
					_, _ = fmt.Fprintf(stderrFromContext(ctx), "  Updating description\n")
				}

				printer.Footer()
				return nil
			}

			var updatedDB *notion.Database
			var updatedDataSource *notion.DataSource

			// Update data source schema if properties were provided
			if propertiesJSON != "" {
				// Convert map[string]map[string]interface{} to map[string]interface{}
				propsForDS := make(map[string]interface{})
				for k, v := range properties {
					propsForDS[k] = v
				}
				dsReq := &notion.UpdateDataSourceRequest{
					Properties: propsForDS,
				}
				ds, err := client.UpdateDataSource(ctx, resolvedDataSourceID, dsReq)
				if err != nil {
					return fmt.Errorf("failed to update data source: %w", err)
				}
				updatedDataSource = ds
			}

			// Update database metadata if needed
			if titleText != "" || descriptionJSON != "" || iconJSON != "" || coverJSON != "" || setArchived {
				req := &notion.UpdateDatabaseRequest{
					Title:       title,
					Description: description,
					Icon:        icon,
					Cover:       cover,
				}

				// Set archived flag if specified
				if setArchived {
					req.Archived = &archived
				}

				db, err := client.UpdateDatabase(ctx, databaseID, req)
				if err != nil {
					return fmt.Errorf("failed to update database: %w", err)
				}
				updatedDB = db
			}

			if updatedDB == nil && updatedDataSource == nil {
				return fmt.Errorf("no updates specified")
			}

			printer := printerForContext(ctx)
			if updatedDB != nil && updatedDataSource != nil {
				return printer.Print(ctx, map[string]interface{}{
					"database":    updatedDB,
					"data_source": updatedDataSource,
				})
			}
			if updatedDB != nil {
				return printer.Print(ctx, updatedDB)
			}
			return printer.Print(ctx, updatedDataSource)
		},
	}

	cmd.Flags().StringVar(&titleText, "title", "", "Database title as plain text")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Database properties as JSON object (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().StringVar(&descriptionJSON, "description", "", "Database description as JSON array")
	cmd.Flags().StringVar(&iconJSON, "icon", "", "Database icon as JSON object")
	cmd.Flags().StringVar(&coverJSON, "cover", "", "Database cover as JSON object")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the database")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID for schema updates (optional)")

	// Track if archived flag was explicitly set
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	return cmd
}
