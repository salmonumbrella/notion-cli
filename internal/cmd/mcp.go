package cmd

import (
	"context"
	"fmt"
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
	cmd.AddCommand(newMCPMoveCmd())
	cmd.AddCommand(newMCPDuplicateCmd())
	cmd.AddCommand(newMCPTeamsCmd())
	cmd.AddCommand(newMCPUsersCmd())
	cmd.AddCommand(newMCPToolsCmd())
	cmd.AddCommand(newMCPCallCmd())
	cmd.AddCommand(newMCPDBCmd())
	cmd.AddCommand(newMCPQueryCmd())
	cmd.AddCommand(newMCPMeetingNotesCmd())

	return cmd
}

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
