package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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

			client := NewNotionClient(token)

			ds, err := client.GetDataSource(ctx, dataSourceID)
			if err != nil {
				return fmt.Errorf("failed to get data source: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, ds)
		},
	}
}

func newDataSourceCreateCmd() *cobra.Command {
	var parentID string
	var propertiesJSON string

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
			if propertiesJSON == "" {
				return fmt.Errorf("--properties is required")
			}

			normalizedParent, err := normalizeNotionID(parentID)
			if err != nil {
				return err
			}
			parentID = normalizedParent

			var properties map[string]interface{}
			resolved, err := readJSONInput(propertiesJSON)
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

			client := NewNotionClient(token)

			req := &notion.CreateDataSourceRequest{
				Parent:     map[string]interface{}{"database_id": parentID},
				Properties: properties,
			}

			ds, err := client.CreateDataSource(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create data source: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent database ID (required)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON (required)")

	return cmd
}

func newDataSourceUpdateCmd() *cobra.Command {
	var propertiesJSON string

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
			if propertiesJSON != "" {
				resolved, err := readJSONInput(propertiesJSON)
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

			client := NewNotionClient(token)

			req := &notion.UpdateDataSourceRequest{
				Properties: properties,
			}

			ds, err := client.UpdateDataSource(ctx, dataSourceID, req)
			if err != nil {
				return fmt.Errorf("failed to update data source: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, ds)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Properties JSON")

	return cmd
}

func newDataSourceQueryCmd() *cobra.Command {
	var filterJSON string
	var pageSize int
	var resultsOnly bool

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
			if pageSize > 100 {
				return fmt.Errorf("page-size must be between 1 and 100")
			}
			if limit > 0 && (pageSize == 0 || pageSize > limit) {
				if limit > 100 {
					pageSize = 100
				} else {
					pageSize = limit
				}
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w", err)
			}

			client := NewNotionClient(token)

			req := &notion.QueryDataSourceRequest{
				Filter:   filter,
				PageSize: pageSize,
			}

			result, err := client.QueryDataSource(ctx, dataSourceID, req)
			if err != nil {
				return fmt.Errorf("failed to query data source: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			if resultsOnly || format == output.FormatTable {
				return printer.Print(ctx, result.Results)
			}
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterJSON, "filter", "", "Filter JSON")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Results per page")
	cmd.Flags().BoolVar(&resultsOnly, "results-only", false, "Output only the results array")

	return cmd
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

			client := NewNotionClient(token)

			list, err := client.ListDataSourceTemplates(ctx)
			if err != nil {
				return fmt.Errorf("failed to list templates: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, list)
		},
	}
}
