package output

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

type fieldSpec struct {
	Key    string
	Tokens []pathToken
}

type pathToken struct {
	Key   *string
	Index *int
}

// ValidateFields validates --fields/--pick syntax.
func ValidateFields(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	_, err := parseFieldSpecs(raw)
	return err
}

func applyOutputTransforms(ctx context.Context, data interface{}, format Format) (interface{}, error) {
	fieldsRaw := strings.TrimSpace(FieldsFromContext(ctx))
	jsonPathRaw := strings.TrimSpace(JSONPathFromContext(ctx))
	if fieldsRaw == "" && jsonPathRaw == "" {
		return data, nil
	}

	if format == FormatTable {
		return nil, clierrors.NewUserError(
			"--fields/--jsonpath are not supported with table output",
			"Use --output json|ndjson|jsonl|yaml|text instead",
		)
	}

	if fieldsRaw != "" {
		projected, err := projectFields(data, fieldsRaw)
		if err != nil {
			return nil, err
		}
		data = projected
	}

	if jsonPathRaw != "" {
		extracted, err := applyJSONPath(data, jsonPathRaw)
		if err != nil {
			return nil, err
		}
		data = extracted
	}

	return data, nil
}

func projectFields(data interface{}, raw string) (interface{}, error) {
	specs, err := parseFieldSpecs(raw)
	if err != nil {
		return nil, clierrors.WrapUserError(err, "invalid --fields value", "Example: --fields id,name=properties.Name.title[0].plain_text")
	}

	normalized, err := normalizeToInterface(data)
	if err != nil {
		return nil, err
	}

	switch v := normalized.(type) {
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			out = append(out, projectOne(item, specs))
		}
		return out, nil
	default:
		return projectOne(v, specs), nil
	}
}

func projectOne(item interface{}, specs []fieldSpec) map[string]interface{} {
	out := make(map[string]interface{}, len(specs))
	for _, spec := range specs {
		if val, ok := extractValue(item, spec.Tokens); ok {
			out[spec.Key] = val
		} else {
			out[spec.Key] = nil
		}
	}
	return out
}

func parseFieldSpecs(raw string) ([]fieldSpec, error) {
	parts := strings.Split(raw, ",")
	specs := make([]fieldSpec, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key := part
		path := part
		if eq := strings.Index(part, "="); eq >= 0 {
			key = strings.TrimSpace(part[:eq])
			path = strings.TrimSpace(part[eq+1:])
		}
		if key == "" || path == "" {
			return nil, fmt.Errorf("invalid field spec %q", part)
		}

		tokens, err := parsePathTokens(path)
		if err != nil {
			return nil, fmt.Errorf("invalid field path %q: %w", path, err)
		}

		specs = append(specs, fieldSpec{
			Key:    key,
			Tokens: tokens,
		})
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no fields provided")
	}
	return specs, nil
}

func parsePathTokens(path string) ([]pathToken, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty path")
	}

	tokens := []pathToken{}
	for i := 0; i < len(path); {
		switch path[i] {
		case '.':
			i++
			continue
		case '[':
			// Parse index or quoted key
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("missing closing ]")
			}
			end += i
			content := strings.TrimSpace(path[i+1 : end])
			if content == "" {
				return nil, fmt.Errorf("empty bracket")
			}
			if content[0] == '"' || content[0] == '\'' {
				key, err := parseQuoted(content)
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, pathToken{Key: &key})
			} else {
				idx, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("invalid index %q", content)
				}
				tokens = append(tokens, pathToken{Index: &idx})
			}
			i = end + 1
		default:
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			key := strings.TrimSpace(path[start:i])
			if key == "" {
				return nil, fmt.Errorf("empty segment")
			}
			// Pure integer segments are treated as array indices,
			// allowing dot notation (title.0) as alternative to brackets (title[0]).
			if idx, err := strconv.Atoi(key); err == nil {
				tokens = append(tokens, pathToken{Index: &idx})
			} else {
				// Apply shorthand aliases only to dot-path segments.
				// Quoted bracket keys remain literal by design.
				key = canonicalizeAliasToken(key)
				tokens = append(tokens, pathToken{Key: &key})
			}
		}
	}

	return tokens, nil
}

func parseQuoted(content string) (string, error) {
	if len(content) < 2 {
		return "", fmt.Errorf("invalid quoted segment")
	}
	quote := content[0]
	if content[len(content)-1] != quote {
		return "", fmt.Errorf("unterminated quoted segment")
	}
	body := content[1 : len(content)-1]
	var b strings.Builder
	b.Grow(len(body))
	escaped := false
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if escaped {
			escaped = false
			b.WriteByte(ch)
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		b.WriteByte(ch)
	}
	if escaped {
		return "", fmt.Errorf("unterminated escape")
	}
	return b.String(), nil
}

func extractValue(data interface{}, tokens []pathToken) (interface{}, bool) {
	cur := data
	for _, tok := range tokens {
		switch {
		case tok.Key != nil:
			m, ok := cur.(map[string]interface{})
			if !ok {
				return nil, false
			}
			val, ok := m[*tok.Key]
			if !ok {
				return nil, false
			}
			cur = val
		case tok.Index != nil:
			arr, ok := cur.([]interface{})
			if !ok {
				return nil, false
			}
			idx := *tok.Index
			if idx < 0 || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func normalizeToInterface(data interface{}) (interface{}, error) {
	switch data.(type) {
	case map[string]interface{}, []interface{}:
		return data, nil
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode data: %w", err)
	}
	var out interface{}
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}
	return out, nil
}

func applyJSONPath(data interface{}, raw string) (interface{}, error) {
	normalized := normalizeJSONPath(raw)
	if normalized == "" {
		return nil, clierrors.NewUserError("invalid --jsonpath value", "Example: --jsonpath '$.results[0].id'")
	}
	normalizedData, err := normalizeToInterface(data)
	if err != nil {
		return nil, err
	}
	value, err := jsonpath.Get(normalized, normalizedData)
	if err != nil {
		return nil, clierrors.WrapUserError(err, "invalid --jsonpath value", "Example: --jsonpath '$.results[0].id'")
	}
	return value, nil
}

func normalizeJSONPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(trimmed, "$"), strings.HasPrefix(trimmed, "@"):
		// keep as-is
	case strings.HasPrefix(trimmed, "."), strings.HasPrefix(trimmed, "["):
		trimmed = "$" + trimmed
	default:
		trimmed = "$." + trimmed
	}

	rewritten, _ := expandDotPathAliases(trimmed)
	return rewritten
}

func isEmptyResult(data interface{}) bool {
	if data == nil {
		return true
	}

	switch v := data.(type) {
	case Table:
		return len(v.Rows) == 0
	case map[string]interface{}:
		if len(v) == 0 {
			return true
		}
		if results, ok := v["results"].([]interface{}); ok {
			return len(results) == 0
		}
		if items, ok := v["items"].([]interface{}); ok {
			return len(items) == 0
		}
		return false
	}

	rv := reflect.ValueOf(data)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len() == 0
	case reflect.Map:
		if rv.Len() == 0 {
			return true
		}
		results := rv.MapIndex(reflect.ValueOf("results"))
		if results.IsValid() {
			for results.Kind() == reflect.Interface || results.Kind() == reflect.Ptr {
				results = results.Elem()
			}
			if results.Kind() == reflect.Slice || results.Kind() == reflect.Array {
				return results.Len() == 0
			}
		}
		items := rv.MapIndex(reflect.ValueOf("items"))
		if items.IsValid() {
			for items.Kind() == reflect.Interface || items.Kind() == reflect.Ptr {
				items = items.Elem()
			}
			if items.Kind() == reflect.Slice || items.Kind() == reflect.Array {
				return items.Len() == 0
			}
		}
	case reflect.Struct:
		results := rv.FieldByName("Results")
		if results.IsValid() && (results.Kind() == reflect.Slice || results.Kind() == reflect.Array) {
			return results.Len() == 0
		}
		items := rv.FieldByName("Items")
		if items.IsValid() && (items.Kind() == reflect.Slice || items.Kind() == reflect.Array) {
			return items.Len() == 0
		}
	}

	return false
}
