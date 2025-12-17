package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage Notion users",
		Long:  `Retrieve and list Notion users in the workspace.`,
	}

	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserMeCmd())

	return cmd
}

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get a user by ID",
		Long: `Retrieve a Notion user by their ID.

Example:
  notion user get 12345678-1234-1234-1234-123456789012`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID := args[0]

			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Get user
			ctx := context.Background()
			user, err := client.GetUser(ctx, userID)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, user)
		},
	}
}

func newUserListCmd() *cobra.Command {
	var startCursor string
	var pageSize int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Long: `List all users in the Notion workspace.

Supports pagination with --start-cursor and --page-size flags.

Example:
  notion user list
  notion user list --page-size 50
  notion user list --start-cursor abc123`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Prepare options
			opts := &notion.ListUsersOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			// List users
			ctx := context.Background()
			userList, err := client.ListUsers(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			// Print result based on output format
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())

			// For table/text format, just show the users list
			// For JSON format, show the full response with pagination info
			if GetOutputFormat() == output.FormatJSON {
				return printer.Print(ctx, userList)
			}

			// For table/text, just show the users array
			return printer.Print(ctx, userList.Results)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor from previous response")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of items per page (max 100)")

	return cmd
}

func newUserMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Get the current bot user",
		Long: `Retrieve the bot user associated with the API token.

This is useful for verifying your authentication and seeing
which bot user is associated with your token.

Example:
  notion user me`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get token
			token, err := auth.GetToken()
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth add' to configure your API token", err)
			}

			// Create client
			client := notion.NewClient(token)

			// Get self
			ctx := context.Background()
			user, err := client.GetSelf(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, user)
		},
	}
}
