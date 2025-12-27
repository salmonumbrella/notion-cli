package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newFetchCmd() *cobra.Command {
	var fetchType string

	cmd := &cobra.Command{
		Use:   "fetch <notion-url>",
		Short: "Fetch a page or database by URL",
		Long: `Fetch a Notion page or database using a Notion URL.

Examples:
  ntn fetch https://www.notion.so/My-Page-1234567890abcdef1234567890abcdef
  ntn fetch https://www.notion.so/My-Database-1234567890abcdef1234567890abcdef --type database`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urlStr := args[0]

			id, err := notion.ExtractIDFromNotionURL(urlStr)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(fetchType)) {
			case "", "auto":
				// Try page first, then database
				page, err := client.GetPage(ctx, id)
				if err == nil {
					printer := printerForContext(ctx)
					return printer.Print(ctx, page)
				}
				if apiErr, ok := err.(*notion.APIError); !ok || apiErr.StatusCode != http.StatusNotFound {
					return fmt.Errorf("failed to fetch page: %w", err)
				}

				db, err := client.GetDatabase(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to fetch database: %w", err)
				}
				printer := printerForContext(ctx)
				return printer.Print(ctx, db)
			case "page":
				page, err := client.GetPage(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to fetch page: %w", err)
				}
				printer := printerForContext(ctx)
				return printer.Print(ctx, page)
			case "database":
				db, err := client.GetDatabase(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to fetch database: %w", err)
				}
				printer := printerForContext(ctx)
				return printer.Print(ctx, db)
			default:
				return fmt.Errorf("invalid --type %q (expected page, database, or auto)", fetchType)
			}
		},
	}

	cmd.Flags().StringVar(&fetchType, "type", "auto", "Object type to fetch (page, database, auto)")

	return cmd
}
