package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "API information and diagnostics",
	}

	cmd.AddCommand(newAPIStatusCmd())
	cmd.AddCommand(newAPIRequestCmd())
	return cmd
}

func newAPIStatusCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show API rate limit status",
		Long: `Show the current Notion API rate limit status.

By default, shows cached rate limit info from the most recent API call.
Use --refresh to make a fresh API call and get updated rate limit info.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			if refresh {
				// Make a lightweight API call to get fresh rate limit info
				_, err := client.GetSelf(ctx)
				if err != nil {
					return fmt.Errorf("failed to fetch API status: %w", err)
				}
			}

			info := client.GetRateLimitInfo()
			format := output.FormatFromContext(ctx)
			if info == nil {
				// Check output format - for JSON/YAML, output empty object
				if format == output.FormatJSON || format == output.FormatYAML {
					return printerForContext(ctx).Print(ctx, map[string]interface{}{
						"available": false,
						"message":   "No rate limit information available. Make an API call first, or use --refresh to fetch fresh data.",
					})
				}
				out := stdoutFromContext(ctx)
				_, _ = fmt.Fprintln(out, "No rate limit information available.")
				_, _ = fmt.Fprintln(out, "Make an API call first, or use --refresh to fetch fresh data.")
				return nil
			}

			// Build rate limit data structure
			data := map[string]interface{}{
				"available":  true,
				"remaining":  info.Remaining,
				"limit":      info.Limit,
				"request_id": info.RequestID,
			}

			if !info.ResetAt.IsZero() {
				data["reset_at"] = info.ResetAt.Format(time.RFC3339)
				remaining := time.Until(info.ResetAt)
				if remaining > 0 {
					data["resets_in_seconds"] = int(remaining.Seconds())
				}
			}

			// Check output format
			if format == output.FormatJSON || format == output.FormatYAML {
				return printerForContext(ctx).Print(ctx, data)
			}

			// Display rate limit info in text format
			out := stdoutFromContext(ctx)
			_, _ = fmt.Fprintln(out, "Rate Limit Status")
			_, _ = fmt.Fprintln(out, "─────────────────")
			_, _ = fmt.Fprintf(out, "Remaining:  %d / %d requests\n", info.Remaining, info.Limit)

			if !info.ResetAt.IsZero() {
				remaining := time.Until(info.ResetAt)
				if remaining > 0 {
					_, _ = fmt.Fprintf(out, "Resets in:  %s\n", remaining.Round(time.Second))
				} else {
					_, _ = fmt.Fprintln(out, "Reset:      Already reset")
				}
			}

			if info.RequestID != "" {
				_, _ = fmt.Fprintf(out, "Request ID: %s\n", info.RequestID)
			}

			// Warn if low
			if info.Limit > 0 {
				pct := float64(info.Remaining) / float64(info.Limit) * 100
				if pct < 10 && !output.QuietFromContext(ctx) {
					_, _ = fmt.Fprintf(out, "\nWarning: Rate limit is low (%.1f%% remaining)\n", pct)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Make fresh API call to get updated rate limit info")

	return cmd
}
