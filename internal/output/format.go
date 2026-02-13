package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

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
	case FormatNDJSON, "jsonl":
		return FormatNDJSON, nil
	case FormatTable:
		return FormatTable, nil
	case FormatYAML:
		return FormatYAML, nil
	default:
		return "", errors.New("invalid --output format (expected text|json|ndjson|jsonl|table|yaml)")
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

	// Inject _meta for list envelopes (JSON/NDJSON/YAML only)
	if p.format == FormatJSON || p.format == FormatNDJSON || p.format == FormatYAML {
		data = injectMeta(data)
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
		return p.printText(ctx, data)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printYAML outputs data as YAML.
func (p *Printer) printYAML(data interface{}) error {
	enc := yaml.NewEncoder(p.w)
	enc.SetIndent(2)
	defer func() { _ = enc.Close() }()
	return enc.Encode(data)
}

// printText outputs data as human-readable text.
// If a --query jq filter is present, it applies the filter first, then renders
// the filtered result. For list envelopes (struct/map with Results slice):
// renders items as a table. For bare slices of structs/maps: renders as a table.
// For single structs: key-value pairs with indented nested values.
// For primitives: direct output.
func (p *Printer) printText(ctx context.Context, data interface{}) error {
	// Apply --query filter if present
	if query := QueryFromContext(ctx); query != "" {
		results, err := runQueryRaw(query, data)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
		if len(results) == 1 {
			data = results[0]
		} else {
			data = results
		}
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

	switch v.Kind() {
	case reflect.Map:
		// Check for list envelope (map with "results" key containing a slice)
		if results, meta, ok := p.extractListEnvelope(v); ok {
			return p.printTextListEnvelope(results, meta)
		}
		return p.printTextMap(v)
	case reflect.Struct:
		// Check for list envelope (struct with Results field containing a slice)
		if results, meta, ok := p.extractStructListEnvelope(v); ok {
			return p.printTextListEnvelope(results, meta)
		}
		return p.printTextStruct(v, "")
	case reflect.Slice, reflect.Array:
		return p.printTextSlice(v)
	default:
		_, err := fmt.Fprintf(p.w, "%v\n", data)
		return err
	}
}

// extractListEnvelope checks if a map has a "results" key with a slice value.
// Returns the results slice, metadata key-values, and whether it matched.
func (p *Printer) extractListEnvelope(v reflect.Value) (reflect.Value, []keyValue, bool) {
	var resultsVal reflect.Value
	var meta []keyValue

	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		val := iter.Value()
		keyStr := fmt.Sprintf("%v", k)

		// Unwrap interface
		for val.Kind() == reflect.Interface {
			val = val.Elem()
		}

		if keyStr == "results" && (val.Kind() == reflect.Slice || val.Kind() == reflect.Array) {
			resultsVal = val
		} else {
			meta = append(meta, keyValue{key: keyStr, val: p.formatScalar(val)})
		}
	}

	if !resultsVal.IsValid() {
		return reflect.Value{}, nil, false
	}

	sort.Slice(meta, func(i, j int) bool { return meta[i].key < meta[j].key })
	return resultsVal, meta, true
}

// extractStructListEnvelope checks if a struct has a Results field with a slice value.
func (p *Printer) extractStructListEnvelope(v reflect.Value) (reflect.Value, []keyValue, bool) {
	t := v.Type()
	resultsIdx := -1
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := fieldJSONName(f)
		if name == "results" {
			fv := v.Field(i)
			for fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					return reflect.Value{}, nil, false
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array {
				resultsIdx = i
			}
			break
		}
	}
	if resultsIdx < 0 {
		return reflect.Value{}, nil, false
	}

	var meta []keyValue
	for i := 0; i < t.NumField(); i++ {
		if i == resultsIdx {
			continue
		}
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := fieldJSONName(f)
		meta = append(meta, keyValue{key: name, val: p.formatScalar(v.Field(i))})
	}

	resultsVal := v.Field(resultsIdx)
	for resultsVal.Kind() == reflect.Ptr {
		if resultsVal.IsNil() {
			return reflect.Value{}, nil, false
		}
		resultsVal = resultsVal.Elem()
	}
	return resultsVal, meta, true
}

type keyValue struct {
	key string
	val string
}

// printTextListEnvelope prints metadata, then renders results as a table.
func (p *Printer) printTextListEnvelope(results reflect.Value, meta []keyValue) error {
	if results.Len() == 0 {
		// Print metadata even for empty results
		for _, kv := range meta {
			if _, err := fmt.Fprintf(p.w, "%s: %s\n", kv.key, kv.val); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(p.w, "results: (none)\n")
		return err
	}

	return p.printTextSlice(results)
}

// printTextMap outputs a map as key-value pairs sorted by key.
func (p *Printer) printTextMap(v reflect.Value) error {
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})

	for _, key := range keys {
		val := v.MapIndex(key)
		formatted := p.formatValue(val)
		if _, err := fmt.Fprintf(p.w, "%v: %v\n", key, formatted); err != nil {
			return err
		}
	}
	return nil
}

// printTextStruct outputs a struct as key-value pairs with indented nested values.
func (p *Printer) printTextStruct(v reflect.Value, indent string) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := fieldJSONName(field)
		value := v.Field(i)

		// Dereference pointers
		for value.Kind() == reflect.Ptr {
			if value.IsNil() {
				break
			}
			value = value.Elem()
		}
		// Unwrap interface
		if value.Kind() == reflect.Interface {
			if value.IsNil() {
				if _, err := fmt.Fprintf(p.w, "%s%s: <nil>\n", indent, name); err != nil {
					return err
				}
				continue
			}
			value = value.Elem()
			for value.Kind() == reflect.Ptr {
				if value.IsNil() {
					break
				}
				value = value.Elem()
			}
		}

		if !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) {
			if _, err := fmt.Fprintf(p.w, "%s%s: <nil>\n", indent, name); err != nil {
				return err
			}
			continue
		}

		switch value.Kind() {
		case reflect.Struct:
			if _, err := fmt.Fprintf(p.w, "%s%s:\n", indent, name); err != nil {
				return err
			}
			if err := p.printTextStruct(value, indent+"  "); err != nil {
				return err
			}
		case reflect.Map:
			if value.Len() == 0 {
				if _, err := fmt.Fprintf(p.w, "%s%s: {}\n", indent, name); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(p.w, "%s%s:\n", indent, name); err != nil {
					return err
				}
				if err := p.printIndentedMap(value, indent+"  "); err != nil {
					return err
				}
			}
		case reflect.Slice, reflect.Array:
			if value.Len() == 0 {
				if _, err := fmt.Fprintf(p.w, "%s%s: []\n", indent, name); err != nil {
					return err
				}
			} else if p.isScalarSlice(value) {
				if _, err := fmt.Fprintf(p.w, "%s%s: %s\n", indent, name, p.formatValue(value)); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(p.w, "%s%s:\n", indent, name); err != nil {
					return err
				}
				for j := 0; j < value.Len(); j++ {
					item := value.Index(j)
					for item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface {
						if (item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface) && item.IsNil() {
							break
						}
						item = item.Elem()
					}
					if _, err := fmt.Fprintf(p.w, "%s  - %s\n", indent, p.formatCompact(item)); err != nil {
						return err
					}
				}
			}
		default:
			if _, err := fmt.Fprintf(p.w, "%s%s: %v\n", indent, name, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// printIndentedMap prints a map with indentation.
func (p *Printer) printIndentedMap(v reflect.Value, indent string) error {
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})
	for _, key := range keys {
		val := v.MapIndex(key)
		for val.Kind() == reflect.Interface {
			val = val.Elem()
		}
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				break
			}
			val = val.Elem()
		}
		if val.Kind() == reflect.Map || val.Kind() == reflect.Struct {
			if _, err := fmt.Fprintf(p.w, "%s%v:\n", indent, key); err != nil {
				return err
			}
			if val.Kind() == reflect.Map {
				if err := p.printIndentedMap(val, indent+"  "); err != nil {
					return err
				}
			} else {
				if err := p.printTextStruct(val, indent+"  "); err != nil {
					return err
				}
			}
		} else {
			formatted := p.formatValue(val)
			if _, err := fmt.Fprintf(p.w, "%s%v: %s\n", indent, key, formatted); err != nil {
				return err
			}
		}
	}
	return nil
}

// printTextSlice outputs a slice as a table when items are structs/maps,
// or one item per line for scalars.
func (p *Printer) printTextSlice(v reflect.Value) error {
	if v.Len() == 0 {
		return nil
	}

	// Check if items are structs or maps — render as table
	first := v.Index(0)
	for first.Kind() == reflect.Ptr || first.Kind() == reflect.Interface {
		if (first.Kind() == reflect.Ptr || first.Kind() == reflect.Interface) && first.IsNil() {
			break
		}
		first = first.Elem()
	}

	switch first.Kind() {
	case reflect.Struct:
		return p.printTextTableFromStructs(v)
	case reflect.Map:
		return p.printTextTableFromMaps(v)
	}

	// Scalar slice — one per line
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		formatted := p.formatValue(item)
		if _, err := fmt.Fprintf(p.w, "%v\n", formatted); err != nil {
			return err
		}
	}
	return nil
}

// textFieldInfo describes a field to include in text table output.
type textFieldInfo struct {
	index int
	name  string
}

// selectTextFields picks which struct fields to show in text table output.
// Includes scalar fields (string, number, bool) and skips complex nested types,
// long URL fields (avatar_url), and internal metadata.
func (p *Printer) selectTextFields(t reflect.Type) []textFieldInfo {
	var fields []textFieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := fieldJSONName(f)
		if name == "-" {
			continue
		}
		// Skip noisy fields that aren't useful in compact table view
		if name == "avatar_url" || name == "request_id" {
			continue
		}
		// Include scalars and pointers-to-scalars (e.g. *string, *Person)
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			fields = append(fields, textFieldInfo{index: i, name: name})
		case reflect.Struct:
			// Include small structs (like Person{Email}) as flattened value
			fields = append(fields, textFieldInfo{index: i, name: name})
		case reflect.Interface:
			// Include interface{} fields (like bot) for type info
			fields = append(fields, textFieldInfo{index: i, name: name})
		}
	}
	return fields
}

// printTextTableFromStructs renders a slice of structs as an aligned table
// with only text-friendly fields.
func (p *Printer) printTextTableFromStructs(v reflect.Value) error {
	if v.Len() == 0 {
		return nil
	}

	first := v.Index(0)
	for first.Kind() == reflect.Ptr || first.Kind() == reflect.Interface {
		if (first.Kind() == reflect.Ptr || first.Kind() == reflect.Interface) && first.IsNil() {
			break
		}
		first = first.Elem()
	}

	fields := p.selectTextFields(first.Type())
	if len(fields) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)

	// Header
	for i, fi := range fields {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, strings.ToUpper(fi.name))
	}
	_, _ = fmt.Fprintln(tw)

	// Rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		for item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface {
			if (item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface) && item.IsNil() {
				break
			}
			item = item.Elem()
		}
		if !item.IsValid() || item.Kind() == reflect.Ptr {
			continue
		}

		for j, fi := range fields {
			if j > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			val := item.Field(fi.index)
			_, _ = fmt.Fprint(tw, p.formatCompact(val))
		}
		_, _ = fmt.Fprintln(tw)
	}

	return tw.Flush()
}

// printTextTableFromMaps renders a slice of maps as an aligned table.
// Only includes keys whose values are scalar across all items (no nested maps/slices).
func (p *Printer) printTextTableFromMaps(v reflect.Value) error {
	if v.Len() == 0 {
		return nil
	}

	// Collect all unique keys and track which ones have only scalar values
	type keyInfo struct {
		seen      bool
		hasNested bool
	}
	keysInfo := make(map[string]*keyInfo)

	for i := 0; i < v.Len(); i++ {
		m := derefValue(v.Index(i))
		if m.Kind() != reflect.Map {
			continue
		}
		iter := m.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key())
			// Skip noisy keys
			if key == "avatar_url" || key == "request_id" {
				continue
			}
			ki, ok := keysInfo[key]
			if !ok {
				ki = &keyInfo{}
				keysInfo[key] = ki
			}
			ki.seen = true
			val := derefValue(iter.Value())
			if val.IsValid() {
				switch val.Kind() {
				case reflect.Map, reflect.Slice, reflect.Array:
					if val.Len() > 0 {
						ki.hasNested = true
					}
				case reflect.Struct:
					ki.hasNested = true
				}
			}
		}
	}

	// Include scalar keys, plus "title" and "name" (which may be rich text arrays
	// but are always useful and will be flattened by formatCompact/extractPlainText)
	alwaysInclude := map[string]bool{"title": true, "name": true}
	var keys []string
	for k, ki := range keysInfo {
		if ki.seen && (!ki.hasNested || alwaysInclude[k]) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		// Fallback: if everything is nested, show id/name/object/title/type at minimum
		fallback := []string{"id", "object", "title", "name", "type", "url"}
		for _, fb := range fallback {
			if ki, ok := keysInfo[fb]; ok && ki.seen {
				keys = append(keys, fb)
			}
		}
	}

	if len(keys) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)

	// Header
	for i, key := range keys {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, strings.ToUpper(key))
	}
	_, _ = fmt.Fprintln(tw)

	// Rows
	for i := 0; i < v.Len(); i++ {
		m := derefValue(v.Index(i))
		if m.Kind() != reflect.Map {
			continue
		}

		for j, key := range keys {
			if j > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			val := m.MapIndex(reflect.ValueOf(key))
			if val.IsValid() {
				_, _ = fmt.Fprint(tw, p.formatCompact(val))
			} else {
				_, _ = fmt.Fprint(tw, "-")
			}
		}
		_, _ = fmt.Fprintln(tw)
	}

	return tw.Flush()
}

// derefValue dereferences pointers and interfaces to the underlying value.
func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

// fieldJSONName returns the json tag name for a struct field, or the field name.
func fieldJSONName(f reflect.StructField) string {
	if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
		if idx := strings.Index(tag, ","); idx > 0 {
			return tag[:idx]
		}
		return tag
	}
	return f.Name
}

// formatScalar formats a simple value as a string for metadata display.
func (p *Printer) formatScalar(v reflect.Value) string {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}
	return fmt.Sprintf("%v", v)
}

// formatCompact formats a value for table cell display — keeps it short.
// Dereferences pointers, flattens small structs, extracts plain_text from rich text arrays.
func (p *Printer) formatCompact(v reflect.Value) string {
	if !v.IsValid() {
		return "<nil>"
	}
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		// Flatten small structs: Person{Email: "x"} → "x"
		t := v.Type()
		var parts []string
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			fv := v.Field(i)
			val := p.formatCompact(fv)
			if val != "" && val != "<nil>" {
				parts = append(parts, val)
			}
		}
		if len(parts) == 1 {
			return parts[0]
		}
		return strings.Join(parts, ", ")
	case reflect.Map:
		if v.Len() == 0 {
			return "{}"
		}
		// Try to extract plain_text for compact display (Notion rich text maps)
		if pt := v.MapIndex(reflect.ValueOf("plain_text")); pt.IsValid() {
			return p.formatCompact(pt)
		}
		return p.formatMap(v)
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return "[]"
		}
		// Extract plain_text from rich text arrays: [{plain_text: "Issues"}, ...] → "Issues"
		if text := p.extractPlainText(v); text != "" {
			return text
		}
		return p.formatSlice(v)
	default:
		s := fmt.Sprintf("%v", v)
		return s
	}
}

// extractPlainText extracts concatenated plain_text from a Notion rich text array.
// Returns empty string if the slice doesn't look like rich text.
func (p *Printer) extractPlainText(v reflect.Value) string {
	if v.Len() == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < v.Len(); i++ {
		item := derefValue(v.Index(i))
		if item.Kind() != reflect.Map {
			return ""
		}
		pt := item.MapIndex(reflect.ValueOf("plain_text"))
		if !pt.IsValid() {
			return ""
		}
		ptv := derefValue(pt)
		parts = append(parts, fmt.Sprintf("%v", ptv))
	}
	return strings.Join(parts, "")
}

// isScalarSlice returns true if all elements are simple types (string, number, bool).
func (p *Printer) isScalarSlice(v reflect.Value) bool {
	if v.Len() == 0 {
		return true
	}
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		for item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface {
			if (item.Kind() == reflect.Ptr || item.Kind() == reflect.Interface) && item.IsNil() {
				break
			}
			item = item.Elem()
		}
		switch item.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			return false
		}
	}
	return true
}

// formatValue recursively formats a reflect.Value into a human-readable string.
// Handles pointers, slices, maps, and structs instead of falling through to Go's
// default %v which outputs pointer addresses and raw struct notation.
func (p *Printer) formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<nil>"
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	// Handle interfaces (e.g. map values typed as interface{})
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
		// Dereference again if interface held a pointer
		for v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return "<nil>"
			}
			v = v.Elem()
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		return p.formatStruct(v)
	case reflect.Map:
		return p.formatMap(v)
	case reflect.Slice, reflect.Array:
		return p.formatSlice(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatStruct formats a struct as {key: value, key: value}.
func (p *Printer) formatStruct(v reflect.Value) string {
	t := v.Type()
	var parts []string
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			if idx := strings.Index(tag, ","); idx > 0 {
				name = tag[:idx]
			} else {
				name = tag
			}
		}
		val := p.formatValue(v.Field(i))
		parts = append(parts, fmt.Sprintf("%s %s", name, val))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// formatMap formats a map as map[key:value key:value].
func (p *Printer) formatMap(v reflect.Value) string {
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})
	var parts []string
	for _, key := range keys {
		val := p.formatValue(v.MapIndex(key))
		parts = append(parts, fmt.Sprintf("%v:%v", key, val))
	}
	return "map[" + strings.Join(parts, " ") + "]"
}

// formatSlice formats a slice as [item1, item2, ...].
func (p *Printer) formatSlice(v reflect.Value) string {
	var parts []string
	for i := 0; i < v.Len(); i++ {
		parts = append(parts, p.formatValue(v.Index(i)))
	}
	return "[" + strings.Join(parts, ", ") + "]"
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
				_, _ = fmt.Fprint(tw, p.formatCompact(val))
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
			_, _ = fmt.Fprint(tw, p.formatCompact(val))
		}
		_, _ = fmt.Fprintln(tw)
	}

	return nil
}
