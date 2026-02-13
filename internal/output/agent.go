package output

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ApplyAgentOptions applies --limit/--sort-by/--desc to output data when possible.
func ApplyAgentOptions(ctx context.Context, data interface{}) interface{} {
	if data == nil {
		return data
	}

	limit := LimitFromContext(ctx)
	sortBy, desc := SortFromContext(ctx)
	sortBy, _ = NormalizeSortPath(sortBy)
	if limit == 0 && sortBy == "" {
		return data
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return data
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return data
		}
		elem := v.Elem()
		if elem.Kind() == reflect.Struct {
			if updated := applyToResultsField(elem, limit, sortBy, desc); updated != nil {
				return data
			}
		} else if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array {
			updated := applyToSlice(elem, limit, sortBy, desc)
			if updated.IsValid() {
				// Return a new slice value (not a pointer)
				return updated.Interface()
			}
		}
		return data
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		updated := applyToSlice(v, limit, sortBy, desc)
		if updated.IsValid() {
			return updated.Interface()
		}
	case reflect.Struct:
		if updated := applyToResultsField(v, limit, sortBy, desc); updated != nil {
			return updated
		}
	}

	return data
}

// applyToResultsField applies sort/limit to a struct field named Results (if present).
func applyToResultsField(v reflect.Value, limit int, sortBy string, desc bool) interface{} {
	if v.Kind() != reflect.Struct {
		return nil
	}

	resultsField := v.FieldByName("Results")
	if !resultsField.IsValid() || (resultsField.Kind() != reflect.Slice && resultsField.Kind() != reflect.Array) {
		return nil
	}

	updated := applyToSlice(resultsField, limit, sortBy, desc)
	if !updated.IsValid() {
		return nil
	}

	if resultsField.CanSet() {
		resultsField.Set(updated)
		return v.Interface()
	}

	// If struct is not settable (value copy), create a new copy and set Results.
	copyVal := reflect.New(v.Type()).Elem()
	copyVal.Set(v)
	copyResults := copyVal.FieldByName("Results")
	if copyResults.IsValid() && copyResults.CanSet() {
		copyResults.Set(updated)
		return copyVal.Interface()
	}

	return nil
}

// applyToSlice copies, sorts, and limits a slice value.
func applyToSlice(v reflect.Value, limit int, sortBy string, desc bool) reflect.Value {
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return reflect.Value{}
	}

	length := v.Len()
	if length == 0 {
		return v
	}

	// Copy to avoid mutating original
	sliceType := v.Type()
	if v.Kind() == reflect.Array {
		sliceType = reflect.SliceOf(v.Type().Elem())
	}
	copySlice := reflect.MakeSlice(sliceType, length, length)
	reflect.Copy(copySlice, v)

	if sortBy != "" {
		sortPath := strings.Split(sortBy, ".")
		sort.Slice(copySlice.Interface(), func(i, j int) bool {
			a := copySlice.Index(i)
			b := copySlice.Index(j)
			av, aok := extractSortableValue(a, sortPath)
			bv, bok := extractSortableValue(b, sortPath)
			if !aok && !bok {
				return false
			}
			if !aok {
				return false
			}
			if !bok {
				return true
			}
			cmp := compareValues(av, bv)
			if desc {
				return cmp > 0
			}
			return cmp < 0
		})
	}

	if limit > 0 && limit < copySlice.Len() {
		return copySlice.Slice(0, limit)
	}

	return copySlice
}

func extractSortableValue(v reflect.Value, path []string) (interface{}, bool) {
	if !v.IsValid() {
		return nil, false
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}

	if len(path) == 0 {
		return nil, false
	}

	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		key, ok := findMapKey(v, path[0])
		if !ok {
			return nil, false
		}
		val := v.MapIndex(key)
		if len(path) == 1 {
			return val.Interface(), true
		}
		return extractSortableValue(val, path[1:])
	case reflect.Struct:
		field, ok := findStructField(v, path[0])
		if !ok {
			return nil, false
		}
		if len(path) == 1 {
			return field.Interface(), true
		}
		return extractSortableValue(field, path[1:])
	default:
		if len(path) == 1 {
			return v.Interface(), true
		}
	}

	return nil, false
}

func findMapKey(v reflect.Value, name string) (reflect.Value, bool) {
	norm := normalizeName(name)
	for _, key := range v.MapKeys() {
		keyStr, ok := key.Interface().(string)
		if !ok {
			continue
		}
		if normalizeName(keyStr) == norm {
			return key, true
		}
	}
	return reflect.Value{}, false
}

func findStructField(v reflect.Value, name string) (reflect.Value, bool) {
	norm := normalizeName(name)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldName := normalizeName(field.Name)
		if fieldName == norm {
			return v.Field(i), true
		}
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			tagName := tag
			if idx := strings.Index(tag, ","); idx >= 0 {
				tagName = tag[:idx]
			}
			if normalizeName(tagName) == norm {
				return v.Field(i), true
			}
		}
	}
	return reflect.Value{}, false
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			// Try RFC3339 time comparison
			at, aerr := time.Parse(time.RFC3339, av)
			bt, berr := time.Parse(time.RFC3339, bv)
			if aerr == nil && berr == nil {
				switch {
				case at.Before(bt):
					return -1
				case at.After(bt):
					return 1
				default:
					return 0
				}
			}
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case bool:
		if bv, ok := b.(bool); ok {
			switch {
			case !av && bv:
				return -1
			case av && !bv:
				return 1
			default:
				return 0
			}
		}
	case int:
		if bv, ok := toFloat(b); ok {
			return compareFloat(float64(av), bv)
		}
	case int64:
		if bv, ok := toFloat(b); ok {
			return compareFloat(float64(av), bv)
		}
	case float64:
		if bv, ok := toFloat(b); ok {
			return compareFloat(av, bv)
		}
	case float32:
		if bv, ok := toFloat(b); ok {
			return compareFloat(float64(av), bv)
		}
	}

	as := fmt.Sprint(a)
	bs := fmt.Sprint(b)
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case uint32:
		return float64(n), true
	}
	return 0, false
}

func compareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
