package mcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// NotionMCPURL is the endpoint for Notion's remote MCP server.
	NotionMCPURL = "https://mcp.notion.com/mcp"

	clientName    = "notion-cli"
	clientVersion = "0.1.0"

	defaultCallToolTimeout     = 45 * time.Second
	defaultCallToolMaxAttempts = 3
	defaultCallToolBaseBackoff = 250 * time.Millisecond
	defaultCallToolMaxBackoff  = 2 * time.Second
)

// CallToolErrorKind categorizes a failed MCP tool call.
type CallToolErrorKind string

const (
	CallToolErrorKindUnknown        CallToolErrorKind = "unknown"
	CallToolErrorKindCanceled       CallToolErrorKind = "canceled"
	CallToolErrorKindTimeout        CallToolErrorKind = "timeout"
	CallToolErrorKindRateLimited    CallToolErrorKind = "rate_limited"
	CallToolErrorKindTransient      CallToolErrorKind = "transient"
	CallToolErrorKindAuthentication CallToolErrorKind = "authentication"
	CallToolErrorKindPermanent      CallToolErrorKind = "permanent"
)

// CallToolError wraps tool-call failures with a normalized error kind.
type CallToolError struct {
	ToolName string
	Kind     CallToolErrorKind
	Attempts int
	Err      error
}

func (e *CallToolError) Error() string {
	return fmt.Sprintf("MCP tool %q failed (kind=%s, attempts=%d): %v", e.ToolName, e.Kind, e.Attempts, e.Err)
}

func (e *CallToolError) Unwrap() error {
	return e.Err
}

type callToolPolicy struct {
	attemptTimeout time.Duration
	maxAttempts    int
	baseBackoff    time.Duration
	maxBackoff     time.Duration
}

func defaultCallToolPolicy() callToolPolicy {
	return callToolPolicy{
		attemptTimeout: defaultCallToolTimeout,
		maxAttempts:    defaultCallToolMaxAttempts,
		baseBackoff:    defaultCallToolBaseBackoff,
		maxBackoff:     defaultCallToolMaxBackoff,
	}
}

func (p callToolPolicy) normalized() callToolPolicy {
	if p.attemptTimeout <= 0 {
		p.attemptTimeout = defaultCallToolTimeout
	}
	if p.maxAttempts < 1 {
		p.maxAttempts = defaultCallToolMaxAttempts
	}
	if p.baseBackoff <= 0 {
		p.baseBackoff = defaultCallToolBaseBackoff
	}
	if p.maxBackoff <= 0 {
		p.maxBackoff = defaultCallToolMaxBackoff
	}
	if p.maxBackoff < p.baseBackoff {
		p.maxBackoff = p.baseBackoff
	}
	return p
}

func (p callToolPolicy) retryDelay(attempt int, kind CallToolErrorKind) time.Duration {
	delay := p.baseBackoff
	for i := 1; i < attempt; i++ {
		if delay >= p.maxBackoff/2 {
			delay = p.maxBackoff
			break
		}
		delay *= 2
	}
	if kind == CallToolErrorKindRateLimited && delay < time.Second {
		delay = time.Second
	}
	if delay > p.maxBackoff {
		delay = p.maxBackoff
	}
	return delay
}

// Client wraps the mcp-go client for Notion's MCP server.
type Client struct {
	inner            *mcpclient.Client
	callToolOverride func(ctx context.Context, name string, args map[string]interface{}) (string, error)
	callToolPolicy   callToolPolicy
}

// NewClient connects to Notion's MCP server using the provided Bearer token.
// The caller must call Close when done.
func NewClient(ctx context.Context, token string) (*Client, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	inner, err := mcpclient.NewStreamableHttpClient(
		NotionMCPURL,
		transport.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP transport: %w", err)
	}

	if err := inner.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize the MCP session.
	_, err = inner.Initialize(ctx, mcp.InitializeRequest{
		Params: struct {
			ProtocolVersion string                 `json:"protocolVersion"`
			Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			ClientInfo      mcp.Implementation     `json:"clientInfo"`
		}{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    clientName,
				Version: clientVersion,
			},
		},
	})
	if err != nil {
		_ = inner.Close()
		return nil, fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	return &Client{
		inner:          inner,
		callToolPolicy: defaultCallToolPolicy(),
	}, nil
}

// CallTool invokes an MCP tool by name with the given arguments and returns
// the concatenated text content from the result.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	result, attempts, err := c.callToolWithRetry(ctx, name, args)
	if err != nil {
		kind := classifyCallToolError(err)
		return "", &CallToolError{
			ToolName: name,
			Kind:     kind,
			Attempts: attempts,
			Err:      err,
		}
	}
	return extractText(result), nil
}

func (c *Client) callToolWithRetry(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, int, error) {
	policy := c.callToolPolicy.normalized()
	attempts := 0
	var lastErr error

	for attempt := 1; attempt <= policy.maxAttempts; attempt++ {
		attempts = attempt

		attemptCtx, cancel := withAttemptTimeout(ctx, policy.attemptTimeout)
		result, err := c.callToolOnce(attemptCtx, name, args)
		cancel()

		if err == nil && result != nil && !result.IsError {
			return result, attempt, nil
		}

		if err == nil && result != nil && result.IsError {
			err = fmt.Errorf("MCP tool %q returned error: %s", name, extractText(result))
		}
		if err == nil {
			err = fmt.Errorf("MCP tool %q returned an empty response", name)
		}
		lastErr = err

		kind := classifyCallToolError(err)
		if !isRetryableCallToolErrorKind(kind) || attempt == policy.maxAttempts {
			return nil, attempt, err
		}

		if err := sleepWithContext(ctx, policy.retryDelay(attempt, kind)); err != nil {
			return nil, attempt, err
		}
	}

	if lastErr != nil {
		return nil, attempts, lastErr
	}
	return nil, attempts, fmt.Errorf("MCP tool %q failed after retries", name)
}

func (c *Client) callToolOnce(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if c.callToolOverride != nil {
		text, err := c.callToolOverride(ctx, name, args)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: text,
				},
			},
		}, nil
	}

	result, err := c.inner.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("MCP tool %q failed: %w", name, err)
	}

	return result, nil
}

// ListTools returns the tools advertised by the Notion MCP server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	result, err := c.inner.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}
	return result.Tools, nil
}

// Close terminates the MCP connection.
func (c *Client) Close() error {
	if c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

// extractText concatenates all text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	var text string
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}
	return text
}

func withAttemptTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= timeout {
			return ctx, func() {}
		}
	}

	return context.WithTimeout(ctx, timeout)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableCallToolErrorKind(kind CallToolErrorKind) bool {
	switch kind {
	case CallToolErrorKindTimeout, CallToolErrorKindRateLimited, CallToolErrorKindTransient:
		return true
	default:
		return false
	}
}

func classifyCallToolError(err error) CallToolErrorKind {
	if err == nil {
		return CallToolErrorKindUnknown
	}

	if errors.Is(err, context.Canceled) {
		return CallToolErrorKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return CallToolErrorKindTimeout
	}

	msg := strings.ToLower(err.Error())

	if containsAny(msg, "unauthorized", "forbidden") || errorContainsHTTPStatus(msg, 401, 403) {
		return CallToolErrorKindAuthentication
	}

	if containsAny(msg, "rate limit", "too many requests") || errorContainsHTTPStatus(msg, 429) {
		return CallToolErrorKindRateLimited
	}

	if containsAny(
		msg,
		"timeout",
		"i/o timeout",
		"tls handshake timeout",
		"deadline exceeded",
	) {
		return CallToolErrorKindTimeout
	}

	if errorContainsHTTPStatus(msg, 500, 502, 503, 504) || containsAny(
		msg,
		"temporarily unavailable",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"connection reset",
		"connection refused",
		"broken pipe",
		"no such host",
		"eof",
	) {
		return CallToolErrorKindTransient
	}

	return CallToolErrorKindPermanent
}

func errorContainsHTTPStatus(msg string, codes ...int) bool {
	for _, code := range codes {
		s := strconv.Itoa(code)
		patterns := []string{
			"status " + s,
			"status: " + s,
			"status code " + s,
			"status code: " + s,
			"http " + s,
		}
		for _, p := range patterns {
			if strings.Contains(msg, p) {
				return true
			}
		}
	}
	return false
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
