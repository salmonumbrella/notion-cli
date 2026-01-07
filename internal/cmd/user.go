package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
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
			userID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get user
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
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Long: `List all users in the Notion workspace.

Supports pagination with --start-cursor and --page-size flags.
Use --all to fetch all pages of results automatically.

Example:
  notion user list
  notion user list --page-size 50
  notion user list --start-cursor abc123
  notion user list --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			format := output.FormatFromContext(ctx)

			pageSize = capPageSize(pageSize, limit)

			// Validate page size
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// If --all flag is set, fetch all pages
			if all {
				var allUsers []*notion.User
				cursor := startCursor
				hasMore := false
				var nextCursor *string

				for {
					opts := &notion.ListUsersOptions{
						StartCursor: cursor,
						PageSize:    pageSize,
					}

					userList, err := client.ListUsers(ctx, opts)
					if err != nil {
						return fmt.Errorf("failed to list users: %w", err)
					}

					allUsers = append(allUsers, userList.Results...)
					hasMore = userList.HasMore
					nextCursor = userList.NextCursor

					if limit > 0 && len(allUsers) >= limit {
						allUsers = allUsers[:limit]
						break
					}

					if !userList.HasMore || userList.NextCursor == nil || *userList.NextCursor == "" {
						break
					}
					cursor = *userList.NextCursor
				}

				// Print all results
				printer := output.NewPrinter(os.Stdout, GetOutputFormat())
				if format == output.FormatTable {
					return printer.Print(ctx, allUsers)
				}
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     allUsers,
					"has_more":    hasMore,
					"next_cursor": nextCursor,
				})
			}

			// Single page request
			opts := &notion.ListUsersOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			userList, err := client.ListUsers(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			if limit > 0 && len(userList.Results) > limit {
				userList.Results = userList.Results[:limit]
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			if format == output.FormatTable {
				return printer.Print(ctx, userList.Results)
			}
			return printer.Print(ctx, userList)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor from previous response")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of items per page (max 100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large datasets)")

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
			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get self
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
