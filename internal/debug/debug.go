package debug

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type contextKey struct{}

// WithDebug injects the debug flag into the context
func WithDebug(ctx context.Context, debug bool) context.Context {
	return context.WithValue(ctx, contextKey{}, debug)
}

// IsDebug returns true if debug mode is enabled in the context
func IsDebug(ctx context.Context) bool {
	if v, ok := ctx.Value(contextKey{}).(bool); ok {
		return v
	}
	return false
}

// DebugTransport wraps http.RoundTripper to log requests/responses when debug mode is enabled
type DebugTransport struct {
	Transport http.RoundTripper
	Output    io.Writer
}

// NewDebugTransport creates a new DebugTransport with the given base transport
// If output is nil, it defaults to os.Stderr
func NewDebugTransport(base http.RoundTripper, output io.Writer) *DebugTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if output == nil {
		output = os.Stderr
	}
	return &DebugTransport{
		Transport: base,
		Output:    output,
	}
}

// RoundTrip implements http.RoundTripper
func (t *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request
	_, _ = fmt.Fprintf(t.Output, "\n--> %s %s\n", req.Method, req.URL)

	// Log headers with token redaction
	for key, values := range req.Header {
		if key == "Authorization" {
			// Redact token, show only last 4 chars
			val := values[0]
			if strings.HasPrefix(val, "Bearer ") {
				token := val[7:] // Remove "Bearer " prefix
				if len(token) > 10 {
					val = "Bearer ..." + token[len(token)-4:]
				}
			}
			_, _ = fmt.Fprintf(t.Output, "    %s: %s\n", key, val)
		} else {
			_, _ = fmt.Fprintf(t.Output, "    %s: %s\n", key, strings.Join(values, ", "))
		}
	}

	// Log request body if present
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			_, _ = fmt.Fprintf(t.Output, "    [ERROR reading request body: %v]\n", err)
		} else {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Restore body for actual request
			if len(bodyBytes) > 0 {
				bodyStr := string(bodyBytes)
				if len(bodyStr) > 500 {
					bodyStr = bodyStr[:500] + "... [truncated]"
				}
				_, _ = fmt.Fprintf(t.Output, "    Body: %s\n", bodyStr)
			}
		}
	}

	// Execute the request
	resp, err := t.Transport.RoundTrip(req)

	duration := time.Since(start)

	// Log error if request failed
	if err != nil {
		_, _ = fmt.Fprintf(t.Output, "<-- ERROR: %v (%s)\n\n", err, duration)
		return resp, err
	}

	// Log response
	_, _ = fmt.Fprintf(t.Output, "<-- %d %s (%s)\n", resp.StatusCode, resp.Status, duration)

	// Show rate limit info if present (before showing all headers)
	if rl := resp.Header.Get("X-RateLimit-Remaining"); rl != "" {
		limit := resp.Header.Get("X-RateLimit-Limit")
		reset := resp.Header.Get("X-RateLimit-Reset")

		// Calculate seconds until reset
		resetStr := ""
		if reset != "" {
			// Notion API returns Unix timestamp as string
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				resetTime := time.Unix(ts, 0)
				remaining := time.Until(resetTime)
				if remaining > 0 {
					resetStr = fmt.Sprintf(" (resets in %ds)", int(remaining.Seconds()))
				}
			}
		}

		_, _ = fmt.Fprintf(t.Output, "    Rate-Limit: %s/%s remaining%s\n", rl, limit, resetStr)
	}

	// Log response headers
	for key, values := range resp.Header {
		_, _ = fmt.Fprintf(t.Output, "    %s: %s\n", key, strings.Join(values, ", "))
	}

	// Log response body if present
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			_, _ = fmt.Fprintf(t.Output, "    [ERROR reading response body: %v]\n\n", err)
		} else {
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Restore body for caller
			if len(bodyBytes) > 0 {
				bodyStr := string(bodyBytes)
				if len(bodyStr) > 1000 {
					bodyStr = bodyStr[:1000] + "... [truncated]"
				}
				_, _ = fmt.Fprintf(t.Output, "    Body: %s\n", bodyStr)
			}
		}
	}

	_, _ = fmt.Fprintln(t.Output)

	return resp, err
}
