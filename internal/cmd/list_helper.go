// internal/cmd/list_helper.go
package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

// ListResult represents a paginated list response.
type ListResult[T any] struct {
	Items   []T
	HasMore bool
}

// ListConfig configures a list command using generics.
type ListConfig[T any] struct {
	Use          string
	Short        string
	Long         string
	Example      string
	Headers      []string
	RowFunc      func(T) []string
	Fetch        func(ctx context.Context, pageSize int) (ListResult[T], error)
	EmptyMessage string
}

// NewListCommand creates a Cobra command from a ListConfig.
func NewListCommand[T any](config ListConfig[T]) *cobra.Command {
	var pageSize int

	cmd := &cobra.Command{
		Use:     config.Use,
		Short:   config.Short,
		Long:    config.Long,
		Example: config.Example,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			format := output.FormatFromContext(ctx)

			result, err := config.Fetch(ctx, pageSize)
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				msg := config.EmptyMessage
				if msg == "" {
					msg = "No items found"
				}
				fmt.Fprintln(os.Stderr, msg)
				return nil
			}

			// Use Printer for JSON, YAML, and text formats
			if format == output.FormatJSON || format == output.FormatYAML || format == output.FormatText {
				printer := output.NewPrinter(os.Stdout, format)
				return printer.Print(ctx, result.Items)
			}

			// Table output (default for FormatTable or unrecognized)
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// Print headers
			for i, h := range config.Headers {
				if i > 0 {
					_, _ = fmt.Fprint(tw, "\t")
				}
				_, _ = fmt.Fprint(tw, h)
			}
			_, _ = fmt.Fprintln(tw)

			// Print rows
			for _, item := range result.Items {
				row := config.RowFunc(item)
				for i, cell := range row {
					if i > 0 {
						_, _ = fmt.Fprint(tw, "\t")
					}
					_, _ = fmt.Fprint(tw, cell)
				}
				_, _ = fmt.Fprintln(tw)
			}

			if err := tw.Flush(); err != nil {
				return fmt.Errorf("failed to flush output: %w", err)
			}

			if result.HasMore {
				fmt.Fprintln(os.Stderr, "\n(more results available, use --page-size to fetch more)")
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&pageSize, "page-size", 50, "Number of items per page")

	return cmd
}
