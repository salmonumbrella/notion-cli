// Package output provides output formatting functionality for the notion CLI.
//
// It supports output formats:
//   - text: Human-readable key-value pairs (default)
//   - json: Pretty-printed JSON
//   - ndjson: Newline-delimited JSON
//   - table: Tabular format for lists with aligned columns
//   - yaml: YAML format for structured data
//
// # Context-Based Dependency Injection
//
// The recommended pattern is to use context-based dependency injection,
// which allows the output format to be set once in root.go and accessed
// throughout the command chain without passing it as a parameter.
//
// In root.go (PersistentPreRunE):
//
//	format, err := output.ParseFormat(formatFlag)
//	if err != nil {
//	    return err
//	}
//	ctx := output.WithFormat(cmd.Context(), format)
//	cmd.SetContext(ctx)
//
// In commands:
//
//	// Get format from context (injected by root.PersistentPreRunE)
//	format := output.FormatFromContext(cmd.Context())
//	printer := output.NewPrinter(os.Stdout, format)
//	return printer.Print(cmd.Context(), data)
//
// # Direct Construction
//
// You can also create a printer directly if you already have a format value:
//
//	printer := output.NewPrinter(os.Stdout, output.FormatJSON)
//	return printer.Print(ctx, data)
//
// # Data Type Handling
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
