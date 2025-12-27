package output

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
)

// printJSON outputs data as pretty-printed JSON.
// If a jq query is present in the context, it filters the output.
func (p *Printer) printJSON(ctx context.Context, data interface{}) error {
	query := QueryFromContext(ctx)
	compact := CompactJSONFromContext(ctx)
	if query == "" {
		enc := json.NewEncoder(p.w)
		enc.SetEscapeHTML(false)
		if !compact {
			enc.SetIndent("", "  ")
		}
		return enc.Encode(data)
	}

	return p.runQuery(query, data, !compact)
}

// printNDJSON outputs data as newline-delimited JSON.
// If a jq query is present in the context, it filters the output.
func (p *Printer) printNDJSON(ctx context.Context, data interface{}) error {
	query := QueryFromContext(ctx)
	enc := json.NewEncoder(p.w)
	enc.SetEscapeHTML(false)

	if query != "" {
		return p.runQuery(query, data, false)
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

// runQuery normalizes data to map/slice form, runs a gojq query, and writes
// results as JSON. When prettyPrint is true, output is indented.
func (p *Printer) runQuery(query string, data interface{}, prettyPrint bool) error {
	// Normalize is idempotent; the cobra prerun hook already normalized,
	// but we re-apply here so the output layer works standalone.
	query, _ = NormalizeQuery(query)

	normalized, err := normalizeToInterface(data)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	parsed, err := gojq.Parse(query)
	if err != nil {
		return formatInvalidQueryErr(err)
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		return formatInvalidQueryErr(err)
	}

	enc := json.NewEncoder(p.w)
	enc.SetEscapeHTML(false)
	if prettyPrint {
		enc.SetIndent("", "  ")
	}

	iter := code.Run(normalized)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if queryErr, isErr := v.(error); isErr {
			return fmt.Errorf("query error: %s", safeErrorMessage(queryErr))
		}
		if err := enc.Encode(v); err != nil {
			return err
		}
	}

	return nil
}

// runQueryRaw normalizes data, runs a gojq query, and returns the results as
// a slice of interface{} values. Used by non-JSON formatters (text).
func runQueryRaw(query string, data interface{}) ([]interface{}, error) {
	// Normalize is idempotent; see comment in runQuery.
	query, _ = NormalizeQuery(query)

	normalized, err := normalizeToInterface(data)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	parsed, err := gojq.Parse(query)
	if err != nil {
		return nil, formatInvalidQueryErr(err)
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		return nil, formatInvalidQueryErr(err)
	}

	var results []interface{}
	iter := code.Run(normalized)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if queryErr, isErr := v.(error); isErr {
			return nil, fmt.Errorf("query error: %s", safeErrorMessage(queryErr))
		}
		results = append(results, v)
	}

	return results, nil
}

func formatInvalidQueryErr(err error) error {
	if err == nil {
		return fmt.Errorf("invalid --query")
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(msg, "unexpected eof") {
		return fmt.Errorf("invalid --query: %w\nHint: query looks incomplete; quote it fully or use --query-file", err)
	}

	return fmt.Errorf("invalid --query: %w", err)
}

// safeErrorMessage returns a best-effort string representation for errors whose
// Error method may panic (seen with some gojq runtime errors on typed values).
func safeErrorMessage(err error) (msg string) {
	if err == nil {
		return "unknown error"
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			msg = formatRecoveredErrorMessage(err, recovered)
		}
	}()

	msg = strings.TrimSpace(err.Error())
	if msg == "" {
		return fmt.Sprintf("%T", err)
	}
	return msg
}

func formatRecoveredErrorMessage(err error, recovered interface{}) string {
	var raw string
	switch v := recovered.(type) {
	case string:
		raw = v
	case error:
		raw = v.Error()
	default:
		return fmt.Sprintf("%T", err)
	}

	raw = strings.TrimSpace(raw)
	// gojq panic payloads often append the full offending value in parentheses.
	// Keep only the stable prefix to avoid dumping huge payloads.
	if idx := strings.Index(raw, " ("); idx > 0 {
		raw = raw[:idx]
	}
	if raw == "" {
		return fmt.Sprintf("%T", err)
	}
	return raw
}
