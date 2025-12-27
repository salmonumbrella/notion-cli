package cmd

import (
	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
)

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
