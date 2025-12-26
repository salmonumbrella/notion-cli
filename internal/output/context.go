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
