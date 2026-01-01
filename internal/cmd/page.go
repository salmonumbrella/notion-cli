package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Notion pages",
		Long:  `Create, retrieve, and update Notion pages.`,
	}

	cmd.AddCommand(newPageGetCmd())
	cmd.AddCommand(newPageCreateCmd())
	cmd.AddCommand(newPageUpdateCmd())
	cmd.AddCommand(newPageCreateBatchCmd())
	cmd.AddCommand(newPageDuplicateCmd())
	cmd.AddCommand(newPageExportCmd())
	cmd.AddCommand(newPagePropertyCmd())
	cmd.AddCommand(newPageMoveCmd())

	return cmd
}

func newPageGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <page-id>",
		Short: "Get a page by ID",
		Long: `Retrieve a Notion page by its ID.

Example:
  notion page get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get page
			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return fmt.Errorf("failed to get page: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}
}

func newPageCreateCmd() *cobra.Command {
	var parentID string
	var parentType string
	var propertiesJSON string
	var dataSourceID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Notion page with the specified parent and properties.

The --properties flag accepts a JSON string with the page properties.
The simplest format for a basic page is:
  {"title": [{"text": {"content": "Page Title"}}]}

The --parent-type flag specifies whether the parent is a "page" (default), "database", or "data-source".
Use --data-source to target a specific data source (overrides --parent-type database).

Examples:
  # Create page under another page
  notion page create --parent 12345678-1234-1234-1234-123456789012 \
    --properties '{"title": [{"text": {"content": "My New Page"}}]}'

  # Create page under a database
  notion page create --parent 87654321-4321-4321-4321-210987654321 \
    --parent-type database \
    --properties '{"Name": {"title": [{"text": {"content": "Database Entry"}}]}}'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate required flags
			if parentID == "" && dataSourceID == "" {
				return fmt.Errorf("--parent flag is required (or use --data-source)")
			}
			if propertiesJSON == "" {
				return fmt.Errorf("--properties flag is required")
			}

			// Parse properties JSON
			var properties map[string]interface{}
			if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
				return fmt.Errorf("failed to parse properties JSON: %w", err)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			parent, err := resolvePageParent(ctx, client, parentID, parentType, dataSourceID)
			if err != nil {
				return err
			}

			// Build request
			req := &notion.CreatePageRequest{
				Parent:     parent,
				Properties: properties,
			}

			// Create page
			page, err := client.CreatePage(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create page: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'data-source'")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON (required)")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID (optional, overrides --parent-type database)")

	return cmd
}

func newPageUpdateCmd() *cobra.Command {
	var propertiesJSON string
	var archived bool
	var setArchived bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update <page-id>",
		Short: "Update a page",
		Long: `Update a Notion page's properties.

The --properties flag accepts a JSON string with the properties to update.
Only the properties specified will be updated; others remain unchanged.

Example:
  notion page update 12345678-1234-1234-1234-123456789012 \
    --properties '{"title": [{"text": {"content": "Updated Title"}}]}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]

			// Parse properties JSON if provided
			var properties map[string]interface{}
			if propertiesJSON != "" {
				if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			if dryRun {
				// Fetch current page to show what would be updated
				currentPage, err := client.GetPage(ctx, pageID)
				if err != nil {
					return fmt.Errorf("failed to fetch page: %w", err)
				}

				printer := NewDryRunPrinter(os.Stderr)
				printer.Header("update", "page", pageID)

				// Show archived status change if applicable
				if setArchived {
					if archived != currentPage.Archived {
						printer.Change("Archived", fmt.Sprintf("%t", currentPage.Archived), fmt.Sprintf("%t", archived))
					} else {
						printer.Unchanged("Archived")
					}
				}

				// Show properties to update
				if propertiesJSON != "" {
					printer.Section("Properties to update:")
					for propName := range properties {
						fmt.Fprintf(os.Stderr, "  - %s\n", propName)
					}

					// Show current values for properties being updated
					printer.Section("\nCurrent property values:")
					for propName := range properties {
						if currentVal, ok := currentPage.Properties[propName]; ok {
							currentBytes, _ := json.Marshal(currentVal)
							fmt.Fprintf(os.Stderr, "  %s: %s\n", propName, string(currentBytes))
						} else {
							fmt.Fprintf(os.Stderr, "  %s: (not set)\n", propName)
						}
					}

					printer.Section("New property values:")
					for propName, propVal := range properties {
						newBytes, _ := json.Marshal(propVal)
						fmt.Fprintf(os.Stderr, "  %s: %s\n", propName, string(newBytes))
					}
				}

				printer.Footer()
				return nil
			}

			// Build request
			req := &notion.UpdatePageRequest{
				Properties: properties,
			}

			// Set archived flag if specified
			if setArchived {
				req.Archived = &archived
			}

			// Update page
			page, err := client.UpdatePage(ctx, pageID, req)
			if err != nil {
				return fmt.Errorf("failed to update page: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the page")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	// Track if archived flag was explicitly set
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	return cmd
}

func newPagePropertyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "property <page-id> <property-id>",
		Short: "Get a specific page property",
		Long: `Retrieve a specific property value from a page.

This is useful for retrieving paginated properties or getting
just a specific property without the entire page.

Example:
  notion page property 12345678-1234-1234-1234-123456789012 title`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			propertyID := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get property
			property, err := client.GetPageProperty(ctx, pageID, propertyID)
			if err != nil {
				return fmt.Errorf("failed to get page property: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, property)
		},
	}
}

func newPageMoveCmd() *cobra.Command {
	var parentID string
	var parentType string
	var after string

	cmd := &cobra.Command{
		Use:   "move <page-id>",
		Short: "Move a page to a new parent",
		Long: `Move a page to a different parent page or database.

Example - Move page under another page:
  notion page move abc123 --parent def456

Example - Move page to database:
  notion page move abc123 --parent db789 --parent-type database`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]

			if parentID == "" {
				return fmt.Errorf("--parent is required")
			}

			var parentKey string
			switch parentType {
			case "page":
				parentKey = "page_id"
			case "database":
				parentKey = "database_id"
			default:
				return fmt.Errorf("invalid --parent-type: %s (expected 'page' or 'database')", parentType)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)

			req := &notion.MovePageRequest{
				Parent: map[string]interface{}{parentKey: parentID},
				After:  after,
			}

			page, err := client.MovePage(ctx, pageID, req)
			if err != nil {
				return fmt.Errorf("failed to move page: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "New parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type: 'page' or 'database'")
	cmd.Flags().StringVar(&after, "after", "", "Block ID to position after")

	return cmd
}
