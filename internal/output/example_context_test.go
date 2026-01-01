package output_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

// ExampleWithFormat demonstrates how to use context-based dependency injection
// for output formatting in commands.
func ExampleWithFormat() {
	// Simulate what root.go does: inject format into context
	ctx := context.Background()
	ctx = output.WithFormat(ctx, output.FormatJSON)

	// In your command, retrieve format from context
	format := output.FormatFromContext(ctx)

	// Create printer with the format from context
	var buf bytes.Buffer
	printer := output.NewPrinter(&buf, format)

	// Print some data
	data := map[string]string{
		"status":  "success",
		"message": "Using context-based dependency injection",
	}
	_ = printer.Print(ctx, data)

	fmt.Print(buf.String())
	// Output:
	// {
	//   "message": "Using context-based dependency injection",
	//   "status": "success"
	// }
}

// ExampleFormatFromContext_command shows the typical usage pattern in a cobra command.
func ExampleFormatFromContext_command() {
	// This is what you would do in your cobra command's RunE function:
	//
	// RunE: func(cmd *cobra.Command, args []string) error {
	//     // Get format from context (injected by root.PersistentPreRunE)
	//     format := output.FormatFromContext(cmd.Context())
	//
	//     // Create printer
	//     printer := output.NewPrinter(os.Stdout, format)
	//
	//     // Print your data
	//     return printer.Print(cmd.Context(), data)
	// }

	// Simulate the pattern
	ctx := output.WithFormat(context.Background(), output.FormatText)
	format := output.FormatFromContext(ctx)
	printer := output.NewPrinter(os.Stdout, format)

	data := map[string]string{
		"name":  "notion-cli",
		"theme": "context-based DI",
	}
	_ = printer.Print(ctx, data)

	// Output:
	// name: notion-cli
	// theme: context-based DI
}

// ExampleFormatFromContext_fallback demonstrates the fallback behavior
// when no format is set in context.
func ExampleFormatFromContext_fallback() {
	// Empty context returns default format (FormatText)
	ctx := context.Background()
	format := output.FormatFromContext(ctx)

	fmt.Printf("Default format: %s\n", format)
	// Output:
	// Default format: text
}
