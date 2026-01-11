package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newDataSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "datasource",
		Aliases: []string{"ds"},
		Short:   "Manage Notion data sources",
		Long:    `Create, retrieve, query, and update Notion data sources (API v2025-09-03).`,
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
		Use:   "get <datasource-id>",
		Short: "Get a data source by ID",
		Long: `Retrieve a Notion data source by its ID.

Example:
  notion datasource get 12345678-1234-1234-1234-123456789012
  notion ds get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataSourceID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			ds, err := client.GetDataSource(ctx, dataSourceID)
			if err != nil {
				return fmt.Errorf("failed to get data source: %w", err)
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
		Use:   "create",
		Short: "Create a new data source",
		Long: `Create a new Notion data source.

The --parent flag specifies the parent database ID (required).
The --properties flag accepts a JSON object defining the data source schema (required).

Example:
  notion datasource create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --properties '{"Name":{"title":{}},"Status":{"select":{}}}'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if parentID == "" {
				return fmt.Errorf("--parent is required")
			}
			if propertiesJSON == "" && propertiesFile == "" {
				return fmt.Errorf("--properties or --properties-file is required")
			}

			normalizedParent, err := normalizeNotionID(parentID)
			if err != nil {
				return err
			}
			parentID = normalizedParent

			var properties map[string]interface{}
			resolved, err := resolveJSONInput(propertiesJSON, propertiesFile)
			if err != nil {
				return err
			}
			propertiesJSON = resolved
			if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
				return fmt.Errorf("invalid properties JSON: %w", err)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			req := &notion.CreateDataSourceRequest{
				Parent:     map[string]interface{}{"database_id": parentID},
				Properties: properties,
			}

			ds, err := client.CreateDataSource(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create data source: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent database ID (required)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON (required, @file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")

	return cmd
}

func newDataSourceUpdateCmd() *cobra.Command {
	var propertiesJSON string
	var propertiesFile string

	cmd := &cobra.Command{
		Use:   "update <datasource-id>",
		Short: "Update a data source",
		Long: `Update a Notion data source's properties.

The --properties flag accepts a JSON object to update the data source schema.

Example:
  notion datasource update 12345678-1234-1234-1234-123456789012 \
    --properties '{"Priority":{"select":{}}}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataSourceID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			var properties map[string]interface{}
			if propertiesJSON != "" || propertiesFile != "" {
				resolved, err := resolveJSONInput(propertiesJSON, propertiesFile)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
					return fmt.Errorf("invalid properties JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			req := &notion.UpdateDataSourceRequest{
				Properties: properties,
			}

			ds, err := client.UpdateDataSource(ctx, dataSourceID, req)
			if err != nil {
				return fmt.Errorf("failed to update data source: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")

	return cmd
}

func newDataSourceQueryCmd() *cobra.Command {
	var filterJSON string
	var pageSize int
	var resultsOnly bool
	var selectProperty string
	var selectEquals string
	var selectMatch string

	cmd := &cobra.Command{
		Use:   "query <datasource-id>",
		Short: "Query a data source",
		Long: `Query a Notion data source with optional filters and pagination.

The --filter flag accepts a JSON object representing the filter.
Use --page-size to control the number of results per page.

Example - Query all pages:
  notion datasource query 12345678-1234-1234-1234-123456789012

Example - Query with filter:
  notion datasource query 12345678-1234-1234-1234-123456789012 \
    --filter '{"property":"Status","select":{"equals":"Active"}}' \
    --page-size 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataSourceID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			format := output.FormatFromContext(ctx)

			if (selectEquals != "" || selectMatch != "") && selectProperty == "" {
				return fmt.Errorf("--select-property is required when using --select-equals or --select-match")
			}

			var filter map[string]interface{}
			if filterJSON != "" {
				resolved, err := readJSONInput(filterJSON)
				if err != nil {
					return err
				}
				filterJSON = resolved
				if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
					return fmt.Errorf("invalid filter JSON: %w", err)
				}
			}
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			req := &notion.QueryDataSourceRequest{
				Filter:   filter,
				PageSize: pageSize,
			}

			result, err := client.QueryDataSource(ctx, dataSourceID, req)
			if err != nil {
				return fmt.Errorf("failed to query data source: %w", err)
			}

			if selectProperty != "" && (selectEquals != "" || selectMatch != "") {
				filtered, err := filterResultsBySelect(result.Results, selectProperty, selectEquals, selectMatch)
				if err != nil {
					return err
				}
				result.Results = filtered
			}

			printer := printerForContext(ctx)
			if resultsOnly || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterJSON, "filter", "", "Filter JSON")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Results per page")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")
	cmd.Flags().StringVar(&selectProperty, "select-property", "", "Property name to match (select or multi_select)")
	cmd.Flags().StringVar(&selectEquals, "select-equals", "", "Match select name exactly")
	cmd.Flags().StringVar(&selectMatch, "select-match", "", "Match select name with regex (Go syntax, use (?i) for case-insensitive). Note: filtering is applied after fetching, so fewer results may be returned when combined with --limit")

	return cmd
}

func filterResultsBySelect(results []notion.Page, propName, equals, match string) ([]notion.Page, error) {
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
		Use:   "templates",
		Short: "List data source templates",
		Long: `List available Notion data source templates.

Example:
  notion datasource templates
  notion ds templates`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			list, err := client.ListDataSourceTemplates(ctx)
			if err != nil {
				return fmt.Errorf("failed to list templates: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, list)
		},
	}
}

func newDataSourceListCmd() *cobra.Command {
	var pageSize int
	var all bool
	var resultsOnly bool
	var startCursor string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all data sources in the workspace",
		Long: `List all data sources (databases) in the Notion workspace.

This command searches for all databases accessible to the integration.
Use --page-size to control results per page (max 100).
Use --all to fetch all pages of results automatically.
Use --results-only to output just the results array (useful for piping to jq).

Example:
  notion datasource list
  notion ds list --all --results-only
  notion ds list -o json | jq '.results[].id'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			format := output.FormatFromContext(ctx)

			pageSize = capPageSize(pageSize, limit)

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(ctx, token)

			// Use search with database filter
			filter := map[string]interface{}{
				"property": "object",
				"value":    "data_source",
			}

			if all {
				var allResults []map[string]interface{}
				cursor := startCursor
				hasMore := false
				var nextCursor *string

				for {
					req := &notion.SearchRequest{
						Filter:      filter,
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					result, err := client.Search(ctx, req)
					if err != nil {
						return fmt.Errorf("failed to list data sources: %w", err)
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

				printer := printerForContext(ctx)
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
				Filter:      filter,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			result, err := client.Search(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to list data sources: %w", err)
			}

			printer := printerForContext(ctx)
			if resultsOnly || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")

	return cmd
}
