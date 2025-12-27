package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newMCPToolsCmd() *cobra.Command {
	var includeSchema bool

	cmd := &cobra.Command{
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
				if includeSchema {
					// Prefer raw schemas when provided so we preserve unknown/extended schema fields.
					if len(t.RawInputSchema) > 0 {
						var inputSchema interface{}
						if err := json.Unmarshal(t.RawInputSchema, &inputSchema); err == nil {
							entry["input_schema"] = inputSchema
						} else {
							entry["input_schema"] = t.InputSchema
						}
					} else {
						entry["input_schema"] = t.InputSchema
					}

					if len(t.RawOutputSchema) > 0 {
						var outputSchema interface{}
						if err := json.Unmarshal(t.RawOutputSchema, &outputSchema); err == nil {
							entry["output_schema"] = outputSchema
						} else {
							entry["output_schema"] = t.OutputSchema
						}
					} else if t.OutputSchema.Type != "" || len(t.OutputSchema.Properties) > 0 || len(t.OutputSchema.Required) > 0 || len(t.OutputSchema.Defs) > 0 {
						entry["output_schema"] = t.OutputSchema
					}

					if t.Annotations.Title != "" ||
						t.Annotations.ReadOnlyHint != nil ||
						t.Annotations.DestructiveHint != nil ||
						t.Annotations.IdempotentHint != nil ||
						t.Annotations.OpenWorldHint != nil {
						entry["annotations"] = t.Annotations
					}
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

	cmd.Flags().BoolVar(&includeSchema, "schema", false, "Include input/output JSON schemas and annotations")

	return cmd
}

func newMCPMoveCmd() *cobra.Command {
	var parentID string

	cmd := &cobra.Command{
		Use:     "move <page-id>...",
		Aliases: []string{"mv"},
		Short:   "Move pages to a new parent",
		Long: `Move one or more Notion pages or databases to a new parent page
using the MCP notion-move-pages tool.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.MovePages(ctx, args, parentID)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "Destination parent page ID")
	_ = cmd.MarkFlagRequired("parent")

	return cmd
}

func newMCPDuplicateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "duplicate <page-id>",
		Aliases: []string{"dup"},
		Short:   "Duplicate a Notion page",
		Long: `Duplicate a Notion page using the MCP notion-duplicate-page tool.

The duplication is performed asynchronously by Notion. The command returns
immediately with a confirmation; the duplicated page may take a moment to
appear in your workspace.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := client.DuplicatePage(ctx, args[0])
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}
}

func newMCPTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "teams [query]",
		Aliases: []string{"tm"},
		Short:   "List workspace teams (teamspaces)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			var query string
			if len(args) > 0 {
				query = args[0]
			}

			result, err := client.GetTeams(ctx, query)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}
}

func newMCPUsersCmd() *cobra.Command {
	var (
		userID   string
		cursor   string
		pageSize int
	)

	cmd := &cobra.Command{
		Use:     "users [query]",
		Aliases: []string{"u"},
		Short:   "List workspace users",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			var query string
			if len(args) > 0 {
				query = args[0]
			}

			result, err := client.GetUsers(ctx, query, userID, cursor, pageSize)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "Fetch a specific user (\"self\" for current)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page")

	return cmd
}
