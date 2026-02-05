package cmd

import (
	"context"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
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
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
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

			var properties map[string]interface{}
			resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
			if err != nil {
				return err
			}
			propertiesJSON = resolved
			if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
				return fmt.Errorf("invalid properties JSON: %w", err)
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
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
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			var properties map[string]interface{}
			if propertiesJSON != "" || propertiesFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
					return fmt.Errorf("invalid properties JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
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
	var filterFile string
	var sortsJSON string
	var sortsFile string
	var pageSize int
	var resultsOnly bool
	var selectProperty string
	var selectEquals string
	var selectNot string
	var selectMatch string

	cmd := &cobra.Command{
		Use:   "query <datasource-id>",
		Short: "Query a data source",
		Long: `Query a Notion data source with optional filters, sorts, and pagination.

The --filter flag accepts a JSON object representing the filter.
The --filter-file flag reads filter JSON from a file (useful for complex filters).
The --sorts flag accepts a JSON array of sort objects.
The --sorts-file flag reads sorts JSON from a file.
Use --page-size to control the number of results per page.

Example - Query all pages:
  notion datasource query 12345678-1234-1234-1234-123456789012

Example - Query with filter:
  notion datasource query 12345678-1234-1234-1234-123456789012 \
    --filter '{"property":"Status","select":{"equals":"Active"}}' \
    --page-size 10

Example - Query with filter from file (avoids shell escaping issues):
  notion datasource query 12345678-1234-1234-1234-123456789012 --filter-file filter.json

Example - Query with sorts:
  notion datasource query 12345678-1234-1234-1234-123456789012 \
    --sorts '[{"property":"Created","direction":"descending"}]'

Example - Sort by created time (shorthand):
  notion datasource query 12345678-1234-1234-1234-123456789012 \
    --sort-by created_time --desc`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			dataSourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}
			limit := output.LimitFromContext(ctx)
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

			var filter map[string]interface{}
			if filterJSON != "" || filterFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(filterJSON, filterFile)
				if err != nil {
					return err
				}
				filterJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(filterJSON, &filter); err != nil {
					return fmt.Errorf("invalid filter JSON: %w", err)
				}
			}

			// Resolve and parse sorts if provided
			sortField, sortDesc := output.SortFromContext(ctx)
			var sorts []map[string]interface{}
			if sortsJSON != "" || sortsFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(sortsJSON, sortsFile)
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

			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			pageSize = capPageSize(pageSize, limit)

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			req := &notion.QueryDataSourceRequest{
				Filter:   filter,
				Sorts:    sorts,
				PageSize: pageSize,
			}

			result, err := client.QueryDataSource(ctx, dataSourceID, req)
			if err != nil {
				return fmt.Errorf("failed to query data source: %w", err)
			}

			if selectProperty != "" && (selectEquals != "" || selectNot != "" || selectMatch != "") {
				filtered, err := filterResultsBySelect(result.Results, selectProperty, selectEquals, selectNot, selectMatch)
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
	cmd.Flags().StringVar(&filterFile, "filter-file", "", "Read filter JSON from file")
	cmd.Flags().StringVar(&sortsJSON, "sorts", "", "Sorts as JSON array")
	cmd.Flags().StringVar(&sortsFile, "sorts-file", "", "Read sorts JSON from file")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Results per page")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")
	cmd.Flags().StringVar(&selectProperty, "select-property", "", "Property name to match (select, multi_select, or status)")
	cmd.Flags().StringVar(&selectEquals, "select-equals", "", "Match select name exactly")
	cmd.Flags().StringVar(&selectNot, "select-not", "", "Exclude items where select name matches exactly")
	cmd.Flags().StringVar(&selectMatch, "select-match", "", "Match select name with regex (Go syntax, use (?i) for case-insensitive). Note: filtering is applied after fetching, so fewer results may be returned when combined with --limit")

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
				return errors.AuthRequiredError(err)
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
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

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
					return fmt.Errorf("failed to list data sources: %w", err)
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
