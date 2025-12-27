package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
)

func newMCPSearchCmd() *cobra.Command {
	var (
		aiFlag        bool
		queryType     string
		pageURL       string
		dataSourceURL string
		teamspaceID   string
		filtersJSON   string
	)

	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"s"},
		Short:   "Search Notion workspace via MCP",
		Long: `Search your Notion workspace using the MCP notion-search tool.

By default uses workspace search. Use --ai for semantic search that
also searches connected apps.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			mode := mcp.SearchModeWorkspace
			if aiFlag {
				mode = mcp.SearchModeAI
			}

			filters, err := parseMCPJSONObject(filtersJSON, "filters")
			if err != nil {
				return err
			}

			result, err := client.Search(ctx, mcp.SearchParams{
				Query:             args[0],
				ContentSearchMode: mode,
				QueryType:         queryType,
				PageURL:           pageURL,
				DataSourceURL:     dataSourceURL,
				TeamspaceID:       teamspaceID,
				Filters:           filters,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&aiFlag, "ai", "a", false, "Use AI-powered semantic search (searches connected apps too)")
	cmd.Flags().StringVar(&queryType, "query-type", "", "Search target type: internal or user")
	cmd.Flags().StringVar(&pageURL, "page-url", "", "Restrict search to a page by URL or ID")
	cmd.Flags().StringVar(&dataSourceURL, "data-source-url", "", "Restrict search to a data source URL (collection://...)")
	cmd.Flags().StringVar(&teamspaceID, "teamspace-id", "", "Restrict search to a teamspace ID")
	cmd.Flags().StringVar(&filtersJSON, "filters", "", "Search filters as a JSON object")

	return cmd
}

func newMCPFetchCmd() *cobra.Command {
	var includeDiscussions bool

	cmd := &cobra.Command{
		Use:     "fetch <id-or-url>",
		Aliases: []string{"f"},
		Short:   "Fetch a Notion entity as markdown via MCP",
		Long: `Retrieve a Notion page, database, or data source as markdown content using the
MCP notion-fetch tool.

Accepts an entity ID or full Notion URL (including notion.site pages and
collection:// data source URLs).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.Fetch(ctx, args[0], includeDiscussions)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&includeDiscussions, "include-discussions", false, "Include discussion markers and counts in fetched markdown")
	return cmd
}
