package output

import "context"

// contextKey is a private type for storing values in context
// to avoid collisions with other packages.
type contextKey struct{}

// queryKey is a private type for storing jq query in context.
type queryKey struct{}

// WithFormat returns a new context with the output format attached.
// This allows the format to be passed down through the command chain
// without needing to pass it as a parameter to every function.
func WithFormat(ctx context.Context, format Format) context.Context {
	return context.WithValue(ctx, contextKey{}, format)
}

// FormatFromContext retrieves the output format from the context.
// If no format is set in the context, it returns FormatText as the default.
func FormatFromContext(ctx context.Context) Format {
	if v, ok := ctx.Value(contextKey{}).(Format); ok {
		return v
	}
	return FormatText // default fallback
}

// WithQuery adds a jq query string to context.
func WithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, queryKey{}, query)
}

// QueryFromContext retrieves the jq query from context.
func QueryFromContext(ctx context.Context) string {
	if q, ok := ctx.Value(queryKey{}).(string); ok {
		return q
	}
	return ""
}

// Agent-friendly flag context keys
type (
	yesKey         struct{}
	limitKey       struct{}
	sortFieldKey   struct{}
	sortDescKey    struct{}
	quietKey       struct{}
	fieldsKey      struct{}
	jsonPathKey    struct{}
	failEmptyKey   struct{}
	resultsOnlyKey struct{}
	lightKey       struct{}
	compactJSONKey struct{}
)

// WithYes sets the --yes flag in context.
func WithYes(ctx context.Context, yes bool) context.Context {
	return context.WithValue(ctx, yesKey{}, yes)
}

// YesFromContext returns true if --yes flag is set.
func YesFromContext(ctx context.Context) bool {
	if y, ok := ctx.Value(yesKey{}).(bool); ok {
		return y
	}
	return false
}

// WithLimit sets the --limit value in context.
func WithLimit(ctx context.Context, limit int) context.Context {
	return context.WithValue(ctx, limitKey{}, limit)
}

// LimitFromContext returns the --limit value (0 = unlimited).
func LimitFromContext(ctx context.Context) int {
	if l, ok := ctx.Value(limitKey{}).(int); ok {
		return l
	}
	return 0
}

// WithSort sets sort field and direction in context.
func WithSort(ctx context.Context, field string, desc bool) context.Context {
	ctx = context.WithValue(ctx, sortFieldKey{}, field)
	ctx = context.WithValue(ctx, sortDescKey{}, desc)
	return ctx
}

// SortFromContext returns sort field and direction.
func SortFromContext(ctx context.Context) (field string, desc bool) {
	if f, ok := ctx.Value(sortFieldKey{}).(string); ok {
		field = f
	}
	if d, ok := ctx.Value(sortDescKey{}).(bool); ok {
		desc = d
	}
	return
}

// WithQuiet sets the --quiet flag in context.
func WithQuiet(ctx context.Context, quiet bool) context.Context {
	return context.WithValue(ctx, quietKey{}, quiet)
}

// QuietFromContext returns true if --quiet flag is set.
func QuietFromContext(ctx context.Context) bool {
	if q, ok := ctx.Value(quietKey{}).(bool); ok {
		return q
	}
	return false
}

// WithFields stores raw --fields/--pick input in context.
func WithFields(ctx context.Context, fields string) context.Context {
	return context.WithValue(ctx, fieldsKey{}, fields)
}

// FieldsFromContext returns raw --fields/--pick input.
func FieldsFromContext(ctx context.Context) string {
	if f, ok := ctx.Value(fieldsKey{}).(string); ok {
		return f
	}
	return ""
}

// WithJSONPath stores a JSONPath expression in context.
func WithJSONPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, jsonPathKey{}, path)
}

// JSONPathFromContext returns the JSONPath expression.
func JSONPathFromContext(ctx context.Context) string {
	if p, ok := ctx.Value(jsonPathKey{}).(string); ok {
		return p
	}
	return ""
}

// WithFailEmpty stores the --fail-empty flag in context.
func WithFailEmpty(ctx context.Context, fail bool) context.Context {
	return context.WithValue(ctx, failEmptyKey{}, fail)
}

// FailEmptyFromContext returns true if --fail-empty is set.
func FailEmptyFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(failEmptyKey{}).(bool); ok {
		return v
	}
	return false
}

// WithResultsOnly sets the global --results-only flag in context.
func WithResultsOnly(ctx context.Context, resultsOnly bool) context.Context {
	return context.WithValue(ctx, resultsOnlyKey{}, resultsOnly)
}

// ResultsOnlyFromContext returns true if global --results-only is set.
func ResultsOnlyFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(resultsOnlyKey{}).(bool); ok {
		return v
	}
	return false
}

// WithLight stores whether command-level light mode is enabled.
func WithLight(ctx context.Context, light bool) context.Context {
	return context.WithValue(ctx, lightKey{}, light)
}

// LightFromContext returns true when command-level light mode is enabled.
func LightFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(lightKey{}).(bool); ok {
		return v
	}
	return false
}

// WithCompactJSON stores whether JSON output should be compact.
func WithCompactJSON(ctx context.Context, compact bool) context.Context {
	return context.WithValue(ctx, compactJSONKey{}, compact)
}

// CompactJSONFromContext returns true when JSON output should be compact.
func CompactJSONFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(compactJSONKey{}).(bool); ok {
		return v
	}
	return false
}
