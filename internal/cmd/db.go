package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage Notion databases",
		Long:  `Retrieve, query, create, and update Notion databases.`,
	}

	cmd.AddCommand(newDBGetCmd())
	cmd.AddCommand(newDBQueryCmd())
	cmd.AddCommand(newDBCreateCmd())
	cmd.AddCommand(newDBUpdateCmd())

	return cmd
}

func newDBGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <database-id>",
		Short: "Get a database by ID",
		Long: `Retrieve a Notion database by its ID.

Example:
  notion db get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseID := args[0]

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Get database
			ctx := context.Background()
			database, err := client.GetDatabase(ctx, databaseID)
			if err != nil {
				return fmt.Errorf("failed to get database: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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

	cmd := &cobra.Command{
		Use:   "query <database-id>",
		Short: "Query a database",
		Long: `Query a Notion database with optional filters and sorts.

The --filter flag accepts a JSON object representing the filter.
The --filter-file flag reads filter JSON from a file (useful for complex filters).
The --sorts flag accepts a JSON array of sort objects.
The --sorts-file flag reads sorts JSON from a file.
Use --page-size to control the number of results per page (max 100).
Use --start-cursor for pagination.

Example - Query all pages:
  notion db query 12345678-1234-1234-1234-123456789012

Example - Query with filter (single line recommended):
  notion db query 12345678-1234-1234-1234-123456789012 --filter '{"property":"Status","select":{"equals":"Done"}}'

Example - Query with filter from file (avoids shell escaping issues):
  notion db query 12345678-1234-1234-1234-123456789012 --filter-file filter.json

Example - Query with sorts:
  notion db query 12345678-1234-1234-1234-123456789012 --sorts '[{"property":"Created","direction":"descending"}]'

Example - Query with pagination:
  notion db query 12345678-1234-1234-1234-123456789012 --page-size 10 --start-cursor abc123

Note: When using multi-line commands with backslash (\), ensure there are no
trailing spaces after the backslash. Otherwise the shell may split the command
incorrectly, causing "accepts 1 arg(s), received N" errors.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseID := args[0]

			// Read filter from file if specified
			if filterFile != "" {
				data, err := os.ReadFile(filterFile)
				if err != nil {
					return fmt.Errorf("failed to read filter file: %w", err)
				}
				filterJSON = string(data)
			}

			// Parse filter if provided
			var filter map[string]interface{}
			if filterJSON != "" {
				if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
					return fmt.Errorf("failed to parse filter JSON: %w", err)
				}
			}

			// Read sorts from file if specified
			if sortsFile != "" {
				data, err := os.ReadFile(sortsFile)
				if err != nil {
					return fmt.Errorf("failed to read sorts file: %w", err)
				}
				sortsJSON = string(data)
			}

			// Parse sorts if provided
			var sorts []map[string]interface{}
			if sortsJSON != "" {
				if err := json.Unmarshal([]byte(sortsJSON), &sorts); err != nil {
					return fmt.Errorf("failed to parse sorts JSON: %w", err)
				}
			}

			// Validate page size
			if pageSize > 100 {
				return fmt.Errorf("page-size must be between 1 and 100")
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Build query request
			req := &notion.DatabaseQueryRequest{
				Filter:      filter,
				Sorts:       sorts,
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			// Query database
			ctx := context.Background()
			result, err := client.QueryDatabase(ctx, databaseID, req)
			if err != nil {
				return fmt.Errorf("failed to query database: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filterJSON, "filter", "", "Filter as JSON object")
	cmd.Flags().StringVar(&filterFile, "filter-file", "", "Read filter JSON from file")
	cmd.Flags().StringVar(&sortsJSON, "sorts", "", "Sorts as JSON array")
	cmd.Flags().StringVar(&sortsFile, "sorts-file", "", "Read sorts JSON from file")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page (max 100)")

	return cmd
}

func newDBCreateCmd() *cobra.Command {
	var parentID string
	var titleText string
	var propertiesJSON string
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
			// Validate required flags
			if parentID == "" {
				return fmt.Errorf("--parent flag is required")
			}
			if propertiesJSON == "" {
				return fmt.Errorf("--properties flag is required")
			}

			// Parse properties JSON
			var properties map[string]map[string]interface{}
			if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
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

			// Parse optional fields
			var description []map[string]interface{}
			if descriptionJSON != "" {
				if err := json.Unmarshal([]byte(descriptionJSON), &description); err != nil {
					return fmt.Errorf("failed to parse description JSON: %w", err)
				}
			}

			var icon map[string]interface{}
			if iconJSON != "" {
				if err := json.Unmarshal([]byte(iconJSON), &icon); err != nil {
					return fmt.Errorf("failed to parse icon JSON: %w", err)
				}
			}

			var cover map[string]interface{}
			if coverJSON != "" {
				if err := json.Unmarshal([]byte(coverJSON), &cover); err != nil {
					return fmt.Errorf("failed to parse cover JSON: %w", err)
				}
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Build request
			req := &notion.CreateDatabaseRequest{
				Parent:      parent,
				Title:       title,
				Description: description,
				Properties:  properties,
				Icon:        icon,
				Cover:       cover,
				IsInline:    isInline,
			}

			// Create database
			ctx := context.Background()
			database, err := client.CreateDatabase(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, database)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page ID (required)")
	cmd.Flags().StringVar(&titleText, "title", "", "Database title as plain text")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Database properties as JSON object (required)")
	cmd.Flags().StringVar(&descriptionJSON, "description", "", "Database description as JSON array")
	cmd.Flags().StringVar(&iconJSON, "icon", "", "Database icon as JSON object")
	cmd.Flags().StringVar(&coverJSON, "cover", "", "Database cover as JSON object")
	cmd.Flags().BoolVar(&isInline, "inline", false, "Create as inline database")

	return cmd
}

func newDBUpdateCmd() *cobra.Command {
	var titleText string
	var propertiesJSON string
	var descriptionJSON string
	var iconJSON string
	var coverJSON string
	var archived bool
	var setArchived bool

	cmd := &cobra.Command{
		Use:   "update <database-id>",
		Short: "Update a database",
		Long: `Update a Notion database's metadata and properties.

The --title flag updates the database title.
The --properties flag accepts a JSON object to update the database schema.
The --description flag updates the database description.
The --archived flag archives or unarchives the database.

Example - Update title:
  notion db update 12345678-1234-1234-1234-123456789012 --title "Updated Tasks"

Example - Add a new property:
  notion db update 12345678-1234-1234-1234-123456789012 \
    --properties '{"Priority":{"select":{"options":[{"name":"High","color":"red"},{"name":"Low","color":"blue"}]}}}'

Example - Archive database:
  notion db update 12345678-1234-1234-1234-123456789012 --archived true`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseID := args[0]

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

			// Parse properties if provided
			var properties map[string]map[string]interface{}
			if propertiesJSON != "" {
				if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Parse optional fields
			var description []map[string]interface{}
			if descriptionJSON != "" {
				if err := json.Unmarshal([]byte(descriptionJSON), &description); err != nil {
					return fmt.Errorf("failed to parse description JSON: %w", err)
				}
			}

			var icon map[string]interface{}
			if iconJSON != "" {
				if err := json.Unmarshal([]byte(iconJSON), &icon); err != nil {
					return fmt.Errorf("failed to parse icon JSON: %w", err)
				}
			}

			var cover map[string]interface{}
			if coverJSON != "" {
				if err := json.Unmarshal([]byte(coverJSON), &cover); err != nil {
					return fmt.Errorf("failed to parse cover JSON: %w", err)
				}
			}

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Build request
			req := &notion.UpdateDatabaseRequest{
				Title:       title,
				Description: description,
				Properties:  properties,
				Icon:        icon,
				Cover:       cover,
			}

			// Set archived flag if specified
			if setArchived {
				req.Archived = &archived
			}

			// Update database
			ctx := context.Background()
			database, err := client.UpdateDatabase(ctx, databaseID, req)
			if err != nil {
				return fmt.Errorf("failed to update database: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, database)
		},
	}

	cmd.Flags().StringVar(&titleText, "title", "", "Database title as plain text")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Database properties as JSON object")
	cmd.Flags().StringVar(&descriptionJSON, "description", "", "Database description as JSON array")
	cmd.Flags().StringVar(&iconJSON, "icon", "", "Database icon as JSON object")
	cmd.Flags().StringVar(&coverJSON, "cover", "", "Database cover as JSON object")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the database")

	// Track if archived flag was explicitly set
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	return cmd
}
