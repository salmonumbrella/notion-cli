package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

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
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a new database",
		Long: `Create a new Notion database.

The --parent flag specifies the parent page ID (required).
The --title flag specifies the database title as plain text.
The --properties flag accepts a JSON object defining the database schema (required).
The --datasource-title flag sets the title of the initial data source (optional).

Example - Create a simple task database:
  ntn db create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --title "Tasks" \
    --properties '{"Name":{"title":{}},"Status":{"select":{"options":[{"name":"Todo","color":"red"},{"name":"Done","color":"green"}]}}}'

Example - Create with description:
  ntn db create \
    --parent 12345678-1234-1234-1234-123456789012 \
    --title "Projects" \
    --description '[{"type":"text","text":{"content":"My projects database"}}]' \
    --properties '{"Name":{"title":{}}}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			if parentID == "" {
				return fmt.Errorf("--parent flag is required")
			}
			if !hasJSONInput(propertiesJSON, propertiesFile) {
				return fmt.Errorf("--properties or --properties-file is required")
			}

			normalizedParent, err := cmdutil.NormalizeNotionID(resolveID(sf, parentID))
			if err != nil {
				return err
			}
			parentID = normalizedParent

			properties, resolvedProperties, err := resolveAndDecodeJSON[map[string]map[string]interface{}](propertiesJSON, propertiesFile, "failed to parse properties JSON")
			if err != nil {
				return err
			}
			propertiesJSON = resolvedProperties

			parent := map[string]interface{}{
				"type":    "page_id",
				"page_id": parentID,
			}

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

			var dataSourceTitleRT []notion.RichText
			if dataSourceTitle != "" {
				dataSourceTitleRT = []notion.RichText{
					{Type: "text", Text: &notion.TextContent{Content: dataSourceTitle}},
				}
			}

			var description []map[string]interface{}
			if descriptionJSON != "" {
				parsed, resolved, err := readAndDecodeJSON[[]map[string]interface{}](descriptionJSON, "failed to parse description JSON")
				if err != nil {
					return err
				}
				descriptionJSON = resolved
				description = parsed
			}

			var icon map[string]interface{}
			if iconJSON != "" {
				parsed, resolved, err := readAndDecodeJSON[map[string]interface{}](iconJSON, "failed to parse icon JSON")
				if err != nil {
					return err
				}
				iconJSON = resolved
				icon = parsed
			}

			var cover map[string]interface{}
			if coverJSON != "" {
				parsed, resolved, err := readAndDecodeJSON[map[string]interface{}](coverJSON, "failed to parse cover JSON")
				if err != nil {
					return err
				}
				coverJSON = resolved
				cover = parsed
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

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

			database, err := client.CreateDatabase(ctx, req)
			if err != nil {
				return wrapAPIError(err, "create database", "database", parentID)
			}

			return printerForContext(ctx).Print(ctx, database)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page ID (required)")
	cmd.Flags().StringVar(&titleText, "title", "", "Database title as plain text")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Database properties as JSON object (required, @file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().StringVar(&dataSourceTitle, "datasource-title", "", "Initial data source title (optional)")
	cmd.Flags().StringVar(&descriptionJSON, "description", "", "Database description as JSON array")
	cmd.Flags().StringVar(&iconJSON, "icon", "", "Database icon as JSON object")
	cmd.Flags().StringVar(&coverJSON, "cover", "", "Database cover as JSON object")
	cmd.Flags().BoolVar(&isInline, "inline", false, "Create as inline database")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")

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
		Use:     "update <database-id>",
		Aliases: []string{"u"},
		Short:   "Update a database",
		Long: `Update a Notion database's metadata and properties.

The --title flag updates the database title.
The --properties flag accepts a JSON object to update the data source schema.
Use --datasource to target a specific data source in a multi-source database.
The --description flag updates the database description.
The --archived flag archives or unarchives the database.

Example - Update title:
  ntn db update 12345678-1234-1234-1234-123456789012 --title "Updated Tasks"

Example - Add a new property (data source schema):
  ntn db update 12345678-1234-1234-1234-123456789012 \
    --properties '{"Priority":{"select":{"options":[{"name":"High","color":"red"},{"name":"Low","color":"blue"}]}}}'

Example - Archive database:
  ntn db update 12345678-1234-1234-1234-123456789012 --archived true`,
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

			propertiesProvided := hasJSONInput(propertiesJSON, propertiesFile)
			var properties map[string]map[string]interface{}
			if propertiesProvided {
				parsed, resolved, err := resolveAndDecodeJSON[map[string]map[string]interface{}](propertiesJSON, propertiesFile, "failed to parse properties JSON")
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				properties = parsed
			}

			descriptionProvided := descriptionJSON != ""
			var description []map[string]interface{}
			if descriptionProvided {
				parsed, resolved, err := readAndDecodeJSON[[]map[string]interface{}](descriptionJSON, "failed to parse description JSON")
				if err != nil {
					return err
				}
				descriptionJSON = resolved
				description = parsed
			}

			iconProvided := iconJSON != ""
			var icon map[string]interface{}
			if iconProvided {
				parsed, resolved, err := readAndDecodeJSON[map[string]interface{}](iconJSON, "failed to parse icon JSON")
				if err != nil {
					return err
				}
				iconJSON = resolved
				icon = parsed
			}

			coverProvided := coverJSON != ""
			var cover map[string]interface{}
			if coverProvided {
				parsed, resolved, err := readAndDecodeJSON[map[string]interface{}](coverJSON, "failed to parse cover JSON")
				if err != nil {
					return err
				}
				coverJSON = resolved
				cover = parsed
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			var resolvedDataSourceID string
			if propertiesProvided {
				resolvedDataSourceID, err = resolveDataSourceID(ctx, client, databaseID, dataSourceID)
				if err != nil {
					return err
				}
			}

			if dryRun {
				currentDB, err := client.GetDatabase(ctx, databaseID)
				if err != nil {
					return wrapAPIError(err, "get database", "database", args[0])
				}

				printer := NewDryRunPrinter(stderrFromContext(ctx))
				printer.Header("update", "database", databaseID)

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

				if setArchived {
					if archived != currentDB.Archived {
						printer.Change("Archived", fmt.Sprintf("%t", currentDB.Archived), fmt.Sprintf("%t", archived))
					} else {
						printer.Unchanged("Archived")
					}
				}

				if propertiesProvided {
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

				if descriptionProvided {
					printer.Section("Description:")
					_, _ = fmt.Fprintf(stderrFromContext(ctx), "  Updating description\n")
				}

				printer.Footer()
				return nil
			}

			var updatedDB *notion.Database
			var updatedDataSource *notion.DataSource

			if propertiesProvided {
				propsForDS := make(map[string]interface{}, len(properties))
				for k, v := range properties {
					propsForDS[k] = v
				}
				ds, err := client.UpdateDataSource(ctx, resolvedDataSourceID, &notion.UpdateDataSourceRequest{Properties: propsForDS})
				if err != nil {
					return wrapAPIError(err, "update data source", "database", args[0])
				}
				updatedDataSource = ds
			}

			if titleText != "" || descriptionProvided || iconProvided || coverProvided || setArchived {
				req := &notion.UpdateDatabaseRequest{
					Title:       title,
					Description: description,
					Icon:        icon,
					Cover:       cover,
				}
				if setArchived {
					req.Archived = &archived
				}

				db, err := client.UpdateDatabase(ctx, databaseID, req)
				if err != nil {
					return wrapAPIError(err, "update database", "database", args[0])
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
	cmd.Flags().StringVar(&dataSourceID, "datasource", "", "Data source ID for schema updates (optional)")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	// Flag aliases
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")
	flagAlias(cmd.Flags(), "datasource", "ds")
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}
