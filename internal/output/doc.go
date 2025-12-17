// Package output provides output formatting functionality for the notion CLI.
//
// It supports three output formats:
//   - text: Human-readable key-value pairs (default)
//   - json: Pretty-printed JSON
//   - table: Tabular format for lists with aligned columns
//
// Usage:
//
//	// Parse format from CLI flag
//	format, err := output.ParseFormat(outputFlag)
//	if err != nil {
//	    return err
//	}
//
//	// Create printer
//	printer := output.NewPrinter(os.Stdout, format)
//
//	// Print data
//	if err := printer.Print(ctx, data); err != nil {
//	    return err
//	}
//
// The Printer automatically handles different data types:
//   - Structs: Field names and values (uses json tags if present)
//   - Maps: Key-value pairs
//   - Slices: For text format, one item per line; for table format, tabular output
//   - Primitives: Direct output
//
// Table format requires slices of structs or maps. The table will use:
//   - For structs: Field names from json tags or struct field names
//   - For maps: All unique keys found across all maps in the slice
package output
