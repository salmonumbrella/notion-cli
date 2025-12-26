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
type yesKey struct{}
type limitKey struct{}
type sortFieldKey struct{}
type sortDescKey struct{}

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
