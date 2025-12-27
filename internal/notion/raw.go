package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ctxerrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

// RawResponse represents a low-level API response.
type RawResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// DoRawRequest performs a raw HTTP request against the Notion API.
func (c *Client) DoRawRequest(ctx context.Context, method, path string, body []byte, headers http.Header) (*RawResponse, error) {
	url := buildRawURL(c.baseURL, path)

	if c.circuitBreaker.isOpen() {
		return nil, ctxerrors.WrapContext(method, url, 0, ErrCircuitOpen)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt, lastErr)
			if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode == http.StatusTooManyRequests {
				slog.Debug("rate limited, waiting before retry", "method", method, "url", url, "attempt", attempt, "delay", delay.String(), "retry_after", apiErr.RetryAfter.String())
			} else {
				slog.Debug("retrying request", "method", method, "url", url, "attempt", attempt, "delay", delay.String())
			}

			select {
			case <-ctx.Done():
				return nil, ctxerrors.WrapContext(method, url, 0, ctx.Err())
			case <-time.After(delay):
			}
		}

		resp, err := c.doRawRequestOnce(ctx, method, url, body, headers)
		if err != nil {
			lastErr = err
			if apiErr, ok := err.(*APIError); ok {
				if isRetryable(apiErr.StatusCode) {
					continue
				}
			}
			return nil, ctxerrors.WrapContext(method, url, getStatusCode(err), err)
		}

		c.circuitBreaker.recordSuccess()
		return resp, nil
	}

	if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode >= 500 {
		c.circuitBreaker.recordFailure()
	}

	return nil, ctxerrors.WrapContext(method, url, getStatusCode(lastErr), lastErr)
}

func buildRawURL(baseURL, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

func (c *Client) doRawRequestOnce(ctx context.Context, method, url string, body []byte, headers http.Header) (*RawResponse, error) {
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if !c.disableAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Notion-Version", c.version)
	for key, values := range headers {
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Update rate limit tracker with response headers
	c.rateLimiter.Update(resp)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(data, &errResp); err != nil {
			return nil, &APIError{StatusCode: resp.StatusCode}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Response:   &errResp,
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	return &RawResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       data,
	}, nil
}
