package output

import (
	"context"
	"reflect"
)

// ApplyResultsOnly extracts the "results" slice from common Notion list envelopes
// when --results-only is enabled OR when the output format is table (since table
// format requires a flat slice, not a list envelope).
func ApplyResultsOnly(ctx context.Context, data interface{}) interface{} {
	if !ResultsOnlyFromContext(ctx) && FormatFromContext(ctx) != FormatTable {
		return data
	}
	results, ok := extractResults(data)
	if !ok {
		return data
	}
	return results
}

func extractResults(data interface{}) (interface{}, bool) {
	if data == nil {
		return nil, false
	}

	// Fast-path for the most common case.
	if m, ok := data.(map[string]interface{}); ok {
		if v, ok := m["results"]; ok {
			return v, true
		}
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil, false
	}

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		f := v.FieldByName("Results")
		if f.IsValid() && (f.Kind() == reflect.Slice || f.Kind() == reflect.Array) {
			return f.Interface(), true
		}
	}

	// Allow map[any]any shapes (e.g. YAML decode or generic map).
	if v.Kind() == reflect.Map {
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String && k.String() == "results" {
				return iter.Value().Interface(), true
			}
		}
	}

	return nil, false
}
