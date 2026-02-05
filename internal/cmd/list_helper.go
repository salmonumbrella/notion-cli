// internal/cmd/list_helper.go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
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
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)

			result, err := config.Fetch(ctx, pageSize)
			if err != nil {
				return err
			}

			if limit > 0 && len(result.Items) > limit {
				result.Items = result.Items[:limit]
			}

			if len(result.Items) == 0 {
				// Early fail-empty check before reaching Printer.Print, which has its
				// own centralized check for non-list-helper paths (e.g., db query).
				if output.FailEmptyFromContext(ctx) {
					return clierrors.NewUserError("no results", "Remove --fail-empty to allow empty output")
				}
				if !output.QuietFromContext(ctx) {
					msg := config.EmptyMessage
					if msg == "" {
						msg = "No items found"
					}
					_, _ = fmt.Fprintln(stderrFromContext(ctx), msg)
				}
				return nil
			}

			if updated, ok := output.ApplyAgentOptions(ctx, result.Items).([]T); ok {
				result.Items = updated
			}

			// Use Printer for JSON, YAML, and text formats
			if format == output.FormatJSON || format == output.FormatYAML || format == output.FormatText {
				printer := output.NewPrinter(stdoutFromContext(ctx), format)
				return printer.Print(ctx, result.Items)
			}

			// Table output (default for FormatTable or unrecognized)
			rows := make([][]string, 0, len(result.Items))
			for _, item := range result.Items {
				rows = append(rows, config.RowFunc(item))
			}

			table := output.Table{
				Headers: config.Headers,
				Rows:    rows,
			}

			printer := output.NewPrinter(stdoutFromContext(ctx), format)
			if err := printer.Print(ctx, table); err != nil {
				return err
			}

			if result.HasMore && (limit == 0 || len(result.Items) < limit) && !output.QuietFromContext(ctx) {
				_, _ = fmt.Fprintln(stderrFromContext(ctx), "\n(more results available, use --page-size to fetch more)")
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&pageSize, "page-size", 50, "Number of items per page")

	return cmd
}
