package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Interact with Notion via the MCP protocol",
		Long: `Use Notion's Model Context Protocol (MCP) server for markdown-based
page operations, AI-powered search, and comment management.

Requires a separate OAuth login (ntn mcp login) which authenticates
directly with Notion's MCP server using your personal Notion account.`,
	}

	cmd.AddCommand(newMCPLoginCmd())
	cmd.AddCommand(newMCPLogoutCmd())
	cmd.AddCommand(newMCPStatusCmd())
	cmd.AddCommand(newMCPSearchCmd())
	cmd.AddCommand(newMCPFetchCmd())
	cmd.AddCommand(newMCPCreateCmd())
	cmd.AddCommand(newMCPEditCmd())
	cmd.AddCommand(newMCPCommentCmd())
	cmd.AddCommand(newMCPToolsCmd())

	return cmd
}

func newMCPLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Notion MCP via OAuth",
		Long: `Start an OAuth 2.0 PKCE flow to authenticate with Notion's MCP server.

This opens your browser to authorize notion-cli. The resulting token is
stored locally at ~/.config/ntn/mcp-token.json (mode 0600).

This is separate from 'ntn auth login' which uses the Notion REST API.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.Login(cmd.Context())
		},
	}
}

func newMCPLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove MCP OAuth token",
		Long:  `Remove the stored MCP OAuth token from disk.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mcp.Logout(); err != nil {
				return err
			}
			ctx := cmd.Context()
			printer := printerForContext(ctx)
			return printer.Print(ctx, map[string]interface{}{
				"status":  "success",
				"message": "MCP token removed",
			})
		},
	}
}

func newMCPStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show MCP authentication status",
		Long:  `Display whether an MCP OAuth token is configured, its expiry, and client ID.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			status, err := mcp.Status()
			if err != nil {
				return err
			}
			printer := printerForContext(ctx)
			return printer.Print(ctx, status)
		},
	}
}

func newMCPSearchCmd() *cobra.Command {
	var aiFlag bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Notion workspace via MCP",
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

			result, err := client.Search(ctx, args[0], mode)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&aiFlag, "ai", false, "Use AI-powered semantic search (searches connected apps too)")

	return cmd
}

func newMCPFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <page-id-or-url>",
		Short: "Fetch a Notion page as markdown via MCP",
		Long: `Retrieve a Notion page or database as markdown content using the
MCP notion-fetch tool.

Accepts a page ID or full Notion URL.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.Fetch(ctx, args[0])
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}
}

func newMCPCreateCmd() *cobra.Command {
	var (
		parentID       string
		dataSourceID   string
		title          string
		content        string
		contentFile    string
		propertiesJSON string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Notion page via MCP",
		Long: `Create a new Notion page with markdown content using the MCP
notion-create-pages tool.

Use --parent for a parent page ID, or --data-source for a data source ID.
If neither is provided, the page is created as a standalone workspace page.

Use --title as a shorthand for setting properties.title. For full control
over page properties, use --properties with a JSON object.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if title == "" && propertiesJSON == "" {
				return fmt.Errorf("--title or --properties is required")
			}

			// Read content from file if specified.
			body := content
			if contentFile != "" {
				data, err := os.ReadFile(contentFile)
				if err != nil {
					return fmt.Errorf("failed to read content file: %w", err)
				}
				body = string(data)
			}

			// Build properties map.
			props := map[string]interface{}{}
			if propertiesJSON != "" {
				if err := json.Unmarshal([]byte(propertiesJSON), &props); err != nil {
					return fmt.Errorf("invalid --properties JSON: %w", err)
				}
			}
			if title != "" {
				props["title"] = title
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			// Build parent if specified.
			var parent *mcp.CreatePagesParent
			if parentID != "" || dataSourceID != "" {
				parent = &mcp.CreatePagesParent{
					PageID:       parentID,
					DataSourceID: dataSourceID,
				}
			}

			page := mcp.CreatePageInput{
				Properties: props,
				Content:    body,
			}
			result, err := client.CreatePages(ctx, parent, []mcp.CreatePageInput{page})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page ID")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID for the parent")
	cmd.Flags().StringVar(&title, "title", "", "Page title (shorthand for properties.title)")
	cmd.Flags().StringVar(&content, "content", "", "Markdown content for the page body")
	cmd.Flags().StringVar(&contentFile, "file", "", "Read markdown content from a file path")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as a JSON object")

	return cmd
}

func newMCPEditCmd() *cobra.Command {
	var (
		replaceContent string
		replaceRange   string
		insertAfter    string
		newStr         string
		propertiesJSON string
	)

	cmd := &cobra.Command{
		Use:   "edit <page-id>",
		Short: "Edit a Notion page via MCP",
		Long: `Edit a Notion page using the MCP notion-update-page tool.

Supports four operations:
  --replace <markdown>                          Replace entire page content
  --replace-range <selection> --new <markdown>  Replace a range identified by selection_with_ellipsis
  --insert-after <selection> --new <markdown>   Insert content after the matched selection
  --properties <json>                           Update page properties (JSON object)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pageID := args[0]

			// Count operations to ensure exactly one is specified.
			ops := 0
			if replaceContent != "" {
				ops++
			}
			if replaceRange != "" {
				ops++
			}
			if insertAfter != "" {
				ops++
			}
			if propertiesJSON != "" {
				ops++
			}
			if ops == 0 {
				return fmt.Errorf("specify one of --replace, --replace-range, --insert-after, or --properties")
			}
			if ops > 1 {
				return fmt.Errorf("specify only one of --replace, --replace-range, --insert-after, or --properties")
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			var params mcp.UpdatePageParams
			params.PageID = pageID

			switch {
			case replaceContent != "":
				params.Command = mcp.UpdateCmdReplaceContent
				params.NewStr = replaceContent
			case replaceRange != "":
				if newStr == "" {
					return fmt.Errorf("--new is required when using --replace-range")
				}
				params.Command = mcp.UpdateCmdReplaceContentRange
				params.SelectionWithEllipsis = replaceRange
				params.NewStr = newStr
			case insertAfter != "":
				if newStr == "" {
					return fmt.Errorf("--new is required when using --insert-after")
				}
				params.Command = mcp.UpdateCmdInsertContentAfter
				params.SelectionWithEllipsis = insertAfter
				params.NewStr = newStr
			case propertiesJSON != "":
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(propertiesJSON), &props); err != nil {
					return fmt.Errorf("invalid --properties JSON: %w", err)
				}
				params.Command = mcp.UpdateCmdUpdateProperties
				params.Properties = props
			}

			result, err := client.UpdatePage(ctx, params)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&replaceContent, "replace", "", "Replace entire page content with markdown")
	cmd.Flags().StringVar(&replaceRange, "replace-range", "", "Selection with ellipsis to match a content range")
	cmd.Flags().StringVar(&insertAfter, "insert-after", "", "Selection with ellipsis to insert content after")
	cmd.Flags().StringVar(&newStr, "new", "", "New markdown content (used with --replace-range or --insert-after)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties to update (JSON object)")

	return cmd
}

func newMCPCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments on Notion pages via MCP",
	}

	cmd.AddCommand(newMCPCommentListCmd())
	cmd.AddCommand(newMCPCommentAddCmd())

	return cmd
}

func newMCPCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <page-id>",
		Short: "List comments on a Notion page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.GetComments(ctx, args[0])
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}
}

func newMCPCommentAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <page-id> <text>",
		Short: "Add a comment to a Notion page",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.CreateComment(ctx, args[0], args[1])
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}
}

func newMCPToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List available MCP tools",
		Long:  `List all tools advertised by the Notion MCP server.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			tools, err := client.ListTools(ctx)
			if err != nil {
				return err
			}

			// Build a serializable list.
			toolList := make([]map[string]interface{}, len(tools))
			for i, t := range tools {
				entry := map[string]interface{}{
					"name": t.Name,
				}
				if t.Description != "" {
					entry["description"] = t.Description
				}
				toolList[i] = entry
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, map[string]interface{}{
				"object":  "list",
				"results": toolList,
			})
		},
	}
}

// mcpClientFromToken loads the persisted MCP token and returns a connected
// MCP client plus a cleanup function that should be deferred.
func mcpClientFromToken(ctx context.Context) (*mcp.Client, func(), error) {
	tf, err := mcp.LoadToken()
	if err != nil {
		return nil, nil, err
	}

	client, err := mcp.NewClient(ctx, tf.AccessToken)
	if err != nil {
		// Provide a hint if the error looks auth-related.
		errStr := err.Error()
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "Unauthorized") {
			return nil, nil, fmt.Errorf("%w\n\nYour MCP token may have expired. Try: ntn mcp login", err)
		}
		return nil, nil, err
	}

	cleanup := func() { _ = client.Close() }
	return client, cleanup, nil
}
