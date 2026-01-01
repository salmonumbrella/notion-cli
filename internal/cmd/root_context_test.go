package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

// TestRootCmd_ContextInjection verifies that the root command correctly
// injects the output format into the context for subcommands to use.
func TestRootCmd_ContextInjection(t *testing.T) {
	tests := []struct {
		name       string
		outputFlag string
		want       output.Format
	}{
		{
			name:       "text format",
			outputFlag: "text",
			want:       output.FormatText,
		},
		{
			name:       "json format",
			outputFlag: "json",
			want:       output.FormatJSON,
		},
		{
			name:       "table format",
			outputFlag: "table",
			want:       output.FormatTable,
		},
		{
			name:       "default format (no flag)",
			outputFlag: "",
			want:       output.FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command that checks the context
			var gotFormat output.Format
			testCmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Retrieve format from context
					gotFormat = output.FormatFromContext(cmd.Context())
					return nil
				},
			}

			// Create a fresh root command for this test
			root := &cobra.Command{
				Use: "notion",
				PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
					// Parse and validate output format from flag
					formatStr, _ := cmd.Flags().GetString("output")
					format, err := output.ParseFormat(formatStr)
					if err != nil {
						return err
					}

					// Inject format into context
					ctx := output.WithFormat(cmd.Context(), format)
					cmd.SetContext(ctx)

					return nil
				},
			}

			// Add output flag
			if tt.outputFlag != "" {
				root.PersistentFlags().String("output", tt.outputFlag, "Output format")
			} else {
				root.PersistentFlags().String("output", "text", "Output format")
			}

			// Add test command as subcommand
			root.AddCommand(testCmd)

			// Execute the test command
			args := []string{"test"}
			if tt.outputFlag != "" {
				args = []string{"--output", tt.outputFlag, "test"}
			}
			root.SetArgs(args)

			ctx := context.Background()
			if err := root.ExecuteContext(ctx); err != nil {
				t.Fatalf("ExecuteContext() error = %v", err)
			}

			// Verify the format was correctly injected
			if gotFormat != tt.want {
				t.Errorf("format from context = %v, want %v", gotFormat, tt.want)
			}
		})
	}
}

// TestRootCmd_ContextFallback verifies that FormatFromContext
// returns a default value when the context doesn't have a format.
func TestRootCmd_ContextFallback(t *testing.T) {
	ctx := context.Background()
	got := output.FormatFromContext(ctx)
	want := output.FormatText

	if got != want {
		t.Errorf("FormatFromContext() with empty context = %v, want %v", got, want)
	}
}

// TestRootCmd_BackwardsCompatibility verifies that the global
// GetOutputFormat() function still works alongside context injection.
func TestRootCmd_BackwardsCompatibility(t *testing.T) {
	// Track if formats were set correctly
	var globalFormat, contextFormat output.Format

	// Create a test command that checks both methods
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			globalFormat = GetOutputFormat()
			contextFormat = output.FormatFromContext(cmd.Context())
			return nil
		},
	}

	// Create a fresh root command
	root := &cobra.Command{
		Use: "notion",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			formatStr, _ := cmd.Flags().GetString("output")
			format, err := output.ParseFormat(formatStr)
			if err != nil {
				return err
			}

			// Set both global var and context (for backwards compatibility)
			outputFormat = format
			ctx := output.WithFormat(cmd.Context(), format)
			cmd.SetContext(ctx)

			return nil
		},
	}

	root.PersistentFlags().String("output", "json", "Output format")
	root.AddCommand(testCmd)
	root.SetArgs([]string{"--output", "json", "test"})

	ctx := context.Background()
	if err := root.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	// Verify both methods return the same format
	if globalFormat != output.FormatJSON {
		t.Errorf("GetOutputFormat() = %v, want %v", globalFormat, output.FormatJSON)
	}

	if contextFormat != output.FormatJSON {
		t.Errorf("FormatFromContext() = %v, want %v", contextFormat, output.FormatJSON)
	}

	// Verify both methods agree
	if globalFormat != contextFormat {
		t.Errorf("global format %v != context format %v", globalFormat, contextFormat)
	}
}
