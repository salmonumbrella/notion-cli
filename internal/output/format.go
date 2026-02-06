package output

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

// Format represents the output format type.
type Format string

const (
	// FormatText is human-readable key-value format (default).
	FormatText Format = "text"
	// FormatJSON is pretty-printed JSON format.
	FormatJSON Format = "json"
	// FormatNDJSON is newline-delimited JSON format.
	FormatNDJSON Format = "ndjson"
	// FormatTable is tabular format for lists.
	FormatTable Format = "table"
	// FormatYAML is YAML format.
	FormatYAML Format = "yaml"
)

// ParseFormat converts a string to a Format type.
// Empty string defaults to FormatText.
// Returns error if the format is invalid.
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatText, "":
		return FormatText, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatNDJSON:
		return FormatNDJSON, nil
	case FormatTable:
		return FormatTable, nil
	case FormatYAML:
		return FormatYAML, nil
	default:
		return "", errors.New("invalid --output format (expected text|json|ndjson|table|yaml)")
	}
}

// Printer handles output formatting across different formats.
type Printer struct {
	w      io.Writer
	format Format
}

// NewPrinter creates a new Printer that writes to w in the given format.
func NewPrinter(w io.Writer, format Format) *Printer {
	return &Printer{
		w:      w,
		format: format,
	}
}

// Print outputs data in the configured format.
// For single objects: JSON or text key-value display.
// For slices: JSON array or table with headers.
func (p *Printer) Print(ctx context.Context, data interface{}) error {
	if data == nil {
		return nil
	}

	data = ApplyAgentOptions(ctx, data)
	data = ApplyResultsOnly(ctx, data)
	updated, err := applyOutputTransforms(ctx, data, p.format)
	if err != nil {
		return err
	}
	data = updated
	if FailEmptyFromContext(ctx) && isEmptyResult(data) {
		return clierrors.NewUserError("no results", "Remove --fail-empty to allow empty output")
	}

	switch p.format {
	case FormatJSON:
		return p.printJSON(ctx, data)
	case FormatNDJSON:
		return p.printNDJSON(ctx, data)
	case FormatYAML:
		return p.printYAML(data)
	case FormatTable:
		return p.printTable(data)
	case FormatText:
		return p.printText(data)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printJSON outputs data as pretty-printed JSON.
// If a jq query is present in the context, it filters the output.
func (p *Printer) printJSON(ctx context.Context, data interface{}) error {
	query := QueryFromContext(ctx)
	if query == "" {
		// Normal JSON output
		enc := json.NewEncoder(p.w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	// Parse and run jq query
	parsed, err := gojq.Parse(query)
	if err != nil {
		return fmt.Errorf("invalid --query: %w", err)
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		return fmt.Errorf("invalid --query: %w", err)
	}

	iter := code.Run(data)
	enc := json.NewEncoder(p.w)
	enc.SetEscapeHTML(false)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("query error: %w", err)
		}
		if err := enc.Encode(v); err != nil {
			return err
		}
	}

	return nil
}

// printNDJSON outputs data as newline-delimited JSON.
// If a jq query is present in the context, it filters the output.
func (p *Printer) printNDJSON(ctx context.Context, data interface{}) error {
	query := QueryFromContext(ctx)
	enc := json.NewEncoder(p.w)
	enc.SetEscapeHTML(false)

	if query != "" {
		parsed, err := gojq.Parse(query)
		if err != nil {
			return fmt.Errorf("invalid --query: %w", err)
		}

		code, err := gojq.Compile(parsed)
		if err != nil {
			return fmt.Errorf("invalid --query: %w", err)
		}

		iter := code.Run(data)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, isErr := v.(error); isErr {
				return fmt.Errorf("query error: %w", err)
			}
			if err := enc.Encode(v); err != nil {
				return err
			}
		}
		return nil
	}

	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		for i := 0; i < v.Len(); i++ {
			if err := enc.Encode(v.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}

	return enc.Encode(data)
}

// printYAML outputs data as YAML.
func (p *Printer) printYAML(data interface{}) error {
	enc := yaml.NewEncoder(p.w)
	enc.SetIndent(2)
	defer func() { _ = enc.Close() }()
	return enc.Encode(data)
}

// printText outputs data as human-readable text.
// For maps and structs: key-value pairs.
// For slices: one item per line.
// For primitives: direct output.
func (p *Printer) printText(data interface{}) error {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Map:
		return p.printTextMap(v)
	case reflect.Struct:
		return p.printTextStruct(v)
	case reflect.Slice, reflect.Array:
		return p.printTextSlice(v)
	default:
		_, err := fmt.Fprintf(p.w, "%v\n", data)
		return err
	}
}

// printTextMap outputs a map as key-value pairs sorted by key.
func (p *Printer) printTextMap(v reflect.Value) error {
	// Collect keys and sort for consistent output
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})

	for _, key := range keys {
		val := v.MapIndex(key)
		if _, err := fmt.Fprintf(p.w, "%v: %v\n", key, val); err != nil {
			return err
		}
	}
	return nil
}

// printTextStruct outputs a struct as key-value pairs.
func (p *Printer) printTextStruct(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Use json tag if available, otherwise use field name
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			// Handle "field,omitempty" format
			if idx := strings.Index(tag, ","); idx > 0 {
				name = tag[:idx]
			} else {
				name = tag
			}
		}

		value := v.Field(i)
		if _, err := fmt.Fprintf(p.w, "%s: %v\n", name, value); err != nil {
			return err
		}
	}
	return nil
}

// printTextSlice outputs a slice as one item per line.
func (p *Printer) printTextSlice(v reflect.Value) error {
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if _, err := fmt.Fprintf(p.w, "%v\n", item); err != nil {
			return err
		}
	}
	return nil
}

// printTable outputs data in tabular format using text/tabwriter.
// Only works with slices of maps or structs.
func (p *Printer) printTable(data interface{}) error {
	switch v := data.(type) {
	case Table:
		return p.printTableFromTable(v)
	case *Table:
		if v == nil {
			return nil
		}
		return p.printTableFromTable(*v)
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// Table format only makes sense for slices
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return errors.New("table format requires a slice or array")
	}

	if v.Len() == 0 {
		return nil
	}

	// Get first element to determine columns
	first := v.Index(0)
	for first.Kind() == reflect.Ptr {
		if first.IsNil() {
			return errors.New("table format cannot handle nil elements")
		}
		first = first.Elem()
	}

	switch first.Kind() {
	case reflect.Map:
		return p.printTableFromMaps(v)
	case reflect.Struct:
		return p.printTableFromStructs(v)
	default:
		return errors.New("table format requires slice of maps or structs")
	}
}

// printTableFromTable outputs a table from a pre-built Table struct.
func (p *Printer) printTableFromTable(t Table) error {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)

	if len(t.Headers) > 0 {
		for i, h := range t.Headers {
			if i > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			_, _ = fmt.Fprint(tw, h)
		}
		_, _ = fmt.Fprintln(tw)
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			_, _ = fmt.Fprint(tw, cell)
		}
		_, _ = fmt.Fprintln(tw)
	}

	return tw.Flush()
}

// printTableFromMaps outputs a table from a slice of maps.
func (p *Printer) printTableFromMaps(v reflect.Value) error {
	if v.Len() == 0 {
		return nil
	}

	// Collect all unique keys from all maps
	keysMap := make(map[string]bool)
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		for m.Kind() == reflect.Ptr {
			if m.IsNil() {
				break
			}
			m = m.Elem()
		}
		if m.Kind() == reflect.Ptr {
			continue
		}
		iter := m.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key())
			keysMap[key] = true
		}
	}

	// Convert to sorted slice for consistent column order
	keys := make([]string, 0, len(keysMap))
	for k := range keysMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Print header
	for i, key := range keys {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, strings.ToUpper(key))
	}
	_, _ = fmt.Fprintln(tw)

	// Print rows
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		for m.Kind() == reflect.Ptr {
			if m.IsNil() {
				break
			}
			m = m.Elem()
		}
		if m.Kind() == reflect.Ptr {
			continue
		}

		for j, key := range keys {
			if j > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			val := m.MapIndex(reflect.ValueOf(key))
			if val.IsValid() {
				_, _ = fmt.Fprintf(tw, "%v", val)
			} else {
				_, _ = fmt.Fprint(tw, "-")
			}
		}
		_, _ = fmt.Fprintln(tw)
	}

	return nil
}

// printTableFromStructs outputs a table from a slice of structs.
func (p *Printer) printTableFromStructs(v reflect.Value) error {
	if v.Len() == 0 {
		return nil
	}

	first := v.Index(0)
	for first.Kind() == reflect.Ptr {
		if first.IsNil() {
			return errors.New("table format cannot handle nil elements")
		}
		first = first.Elem()
	}

	t := first.Type()

	// Collect exported fields and their display names
	type fieldInfo struct {
		index int
		name  string
	}
	var fields []fieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Use json tag if available, otherwise use field name
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			// Handle "field,omitempty" format
			if idx := strings.Index(tag, ","); idx > 0 {
				name = tag[:idx]
			} else {
				name = tag
			}
		}

		fields = append(fields, fieldInfo{index: i, name: name})
	}

	if len(fields) == 0 {
		return errors.New("no exported fields in struct")
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Print header
	for i, fi := range fields {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, strings.ToUpper(fi.name))
	}
	_, _ = fmt.Fprintln(tw)

	// Print rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		for item.Kind() == reflect.Ptr {
			if item.IsNil() {
				break
			}
			item = item.Elem()
		}
		if item.Kind() == reflect.Ptr {
			continue
		}

		for j, fi := range fields {
			if j > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			val := item.Field(fi.index)
			_, _ = fmt.Fprintf(tw, "%v", val)
		}
		_, _ = fmt.Fprintln(tw)
	}

	return nil
}
