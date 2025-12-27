package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
)

func newMCPDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage databases via MCP",
	}

	cmd.AddCommand(newMCPDBCreateCmd())
	cmd.AddCommand(newMCPDBUpdateCmd())

	return cmd
}

func newMCPDBCreateCmd() *cobra.Command {
	var (
		parentID       string
		title          string
		description    string
		schema         string
		propertiesJSON string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a new database via MCP",
		Long: `Create a new Notion database using the MCP notion-create-database tool.

Use --parent to specify the parent page ID. Use --title for the database title.
Use --schema to define the database schema using SQL DDL.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if schema == "" && propertiesJSON != "" {
				return fmt.Errorf("--properties is no longer supported for MCP db create; use --schema with SQL DDL")
			}
			if schema == "" {
				return fmt.Errorf("--schema is required")
			}

			params := mcp.CreateDatabaseParams{
				Schema: schema,
			}
			if propertiesJSON != "" {
				props, err := parseMCPJSONObject(propertiesJSON, "properties")
				if err != nil {
					return err
				}
				params.Properties = props
			}

			if parentID != "" {
				params.Parent = &mcp.CreateDatabaseParent{PageID: parentID}
			}

			if title != "" {
				params.Title = title
			}
			if description != "" {
				params.Description = description
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.CreateDatabase(ctx, params)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Database title")
	cmd.Flags().StringVar(&description, "description", "", "Database description")
	cmd.Flags().StringVar(&schema, "schema", "", "SQL DDL schema (e.g. CREATE TABLE (\"Name\" TITLE))")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Legacy JSON schema properties (deprecated; use --schema SQL DDL)")
	_ = cmd.Flags().MarkDeprecated("properties", "use --schema instead")

	return cmd
}

func newMCPDBUpdateCmd() *cobra.Command {
	var (
		dataSourceID   string
		title          string
		description    string
		statements     string
		propertiesJSON string
		trash          bool
		inline         bool
	)

	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   "Update a database schema via MCP",
		Long: `Update a Notion database schema using the MCP notion-update-data-source tool.

Use --id to specify the data source ID (required). Optionally set --title,
--statements (SQL DDL), --description, --inline, or --trash to move the database to trash.

For legacy compatibility, --properties is still accepted and forwarded.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			params := mcp.UpdateDataSourceParams{
				DataSourceID: dataSourceID,
			}

			if title != "" {
				params.Title = title
			}
			if description != "" {
				params.Description = description
			}
			if statements != "" {
				params.Statements = statements
			}
			if propertiesJSON != "" {
				props, err := parseMCPJSONObject(propertiesJSON, "properties")
				if err != nil {
					return err
				}
				params.Properties = props
			}

			if cmd.Flags().Changed("trash") {
				params.InTrash = &trash
			}
			if cmd.Flags().Changed("inline") {
				params.IsInline = &inline
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.UpdateDataSource(ctx, params)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&dataSourceID, "id", "", "Data source ID (required)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "New database title")
	cmd.Flags().StringVar(&description, "description", "", "New database description")
	cmd.Flags().StringVar(&statements, "statements", "", "Semicolon-separated SQL DDL statements (ADD/DROP/RENAME/ALTER COLUMN)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Legacy JSON schema properties (deprecated; prefer --statements SQL DDL)")
	cmd.Flags().BoolVar(&trash, "trash", false, "Move database to trash")
	cmd.Flags().BoolVar(&inline, "inline", false, "Set whether the data source is inline")
	_ = cmd.Flags().MarkDeprecated("properties", "use --statements instead")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func newMCPQueryCmd() *cobra.Command {
	var (
		viewURL   string
		paramsStr string
	)

	cmd := &cobra.Command{
		Use:     "query <sql-or-data-source-url>...",
		Aliases: []string{"q"},
		Short:   "Query databases using SQL via MCP",
		Long: `Query Notion databases using SQL or execute a database view.

SQL MODE (default):
  Pass one or more data source URLs (from 'ntn mcp fetch <db>') and a SQL query.
  Use the data source URL as the table name in your query.

  ntn mcp query 'SELECT * FROM "collection://abc123" LIMIT 10' collection://abc123
  ntn mcp query 'SELECT * FROM "collection://abc123" WHERE Status = ?' collection://abc123 --params '["In Progress"]'

VIEW MODE:
  Execute a database view's existing filters and sorts.

  ntn mcp query --view "https://www.notion.so/workspace/Tasks-abc123?v=def456"

Notes:
  - Use 'ntn mcp fetch <database-url>' first to get data source URLs
  - Data source URLs are found in <data-source url="collection://..."> tags
  - Checkbox values: use "__YES__" for checked, "__NO__" for unchecked
  - Use parameterized queries (? placeholders + --params) for safety`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			// View mode.
			if viewURL != "" {
				result, err := client.QueryDataSourcesView(ctx, mcp.QueryDataSourcesViewParams{
					ViewURL: viewURL,
				})
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
				return nil
			}

			// SQL mode: first arg is the SQL query, remaining args are data source URLs.
			if len(args) < 2 {
				return fmt.Errorf("SQL mode requires at least 2 args: <sql-query> <data-source-url>")
			}

			sqlQuery := args[0]
			dataSourceURLs := args[1:]

			queryParams, err := parseMCPJSONArray(paramsStr, "params")
			if err != nil {
				return err
			}

			result, err := client.QueryDataSourcesSQL(ctx, mcp.QueryDataSourcesSQLParams{
				DataSourceURLs: dataSourceURLs,
				Query:          sqlQuery,
				Params:         queryParams,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&viewURL, "view", "v", "", "Execute a database view by URL (view mode)")
	cmd.Flags().StringVarP(&paramsStr, "params", "P", "", "JSON array of query parameters for ? placeholders")

	return cmd
}

func newMCPMeetingNotesCmd() *cobra.Command {
	var (
		filterJSON string
		filterFile string
	)

	cmd := &cobra.Command{
		Use:     "meeting-notes",
		Aliases: []string{"mn"},
		Short:   "Query meeting notes via MCP",
		Long: `Query your meeting notes using the notion-query-meeting-notes MCP tool.

Provide an optional filter JSON object with --filter or --filter-file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := parseMCPJSONObjectFromInlineOrFile(filterJSON, filterFile, "filter", "filter-file")
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.QueryMeetingNotes(ctx, mcp.QueryMeetingNotesParams{
				Filter: filter,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&filterJSON, "filter", "", "Meeting notes filter as a JSON object")
	cmd.Flags().StringVar(&filterFile, "filter-file", "", "Path to file containing meeting notes filter JSON")

	return cmd
}

// mcpClientFromToken loads the persisted MCP token and returns a connected
