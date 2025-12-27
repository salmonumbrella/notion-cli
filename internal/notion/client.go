package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/salmonumbrella/notion-cli/internal/debug"
	ctxerrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

const (
	defaultBaseURL = "https://api.notion.com/v1"
	apiVersion     = "2025-09-03"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	baseDelay      = 1 * time.Second

	// Circuit breaker defaults
	defaultCircuitBreakerThreshold       = 5
	defaultCircuitBreakerRecoveryTimeout = 30 * time.Second
)

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("circuit breaker is open - too many consecutive API failures")

// circuitBreaker implements a circuit breaker pattern to prevent hammering a failing API
type circuitBreaker struct {
	mu              sync.Mutex
	failures        int
	lastFailure     time.Time
	open            bool
	threshold       int
	recoveryTimeout time.Duration
	enabled         bool
}

// recordSuccess clears the failure counter and closes the circuit
func (cb *circuitBreaker) recordSuccess() {
	if !cb.enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	wasOpen := cb.open
	cb.failures = 0
	cb.open = false

	if wasOpen {
		slog.Info("circuit breaker recovered", "component", "circuit_breaker")
	}
}

// recordFailure increments the failure counter and opens circuit if threshold reached
// Returns true if the circuit just opened
func (cb *circuitBreaker) recordFailure() bool {
	if !cb.enabled {
		return false
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold && !cb.open {
		cb.open = true
		slog.Warn("circuit breaker opened", "component", "circuit_breaker", "failures", cb.failures)
		return true
	}

	return false
}

// isOpen checks if the circuit is currently open
// Auto-recovers if recovery timeout has passed
func (cb *circuitBreaker) isOpen() bool {
	if !cb.enabled {
		return false
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}

	// Check if recovery timeout has passed
	if time.Since(cb.lastFailure) > cb.recoveryTimeout {
		cb.open = false
		cb.failures = 0
		slog.Debug("circuit breaker half-open, attempting recovery", "component", "circuit_breaker")
		return false
	}

	return true
}

// Client is the Notion API client
type Client struct {
	httpClient     *http.Client
	token          string
	baseURL        string
	version        string
	disableAuth    bool
	maxRetries     int
	circuitBreaker *circuitBreaker
	rateLimiter    *RateLimitTracker
}

// NewClient creates a new Notion API client with the given token
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		token:       token,
		baseURL:     defaultBaseURL,
		version:     apiVersion,
		disableAuth: false,
		maxRetries:  maxRetries,
		circuitBreaker: &circuitBreaker{
			threshold:       defaultCircuitBreakerThreshold,
			recoveryTimeout: defaultCircuitBreakerRecoveryTimeout,
			enabled:         false, // Disabled by default
		},
		rateLimiter: NewRateLimitTracker(),
	}
}

// WithHTTPClient sets a custom HTTP client
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.httpClient = client
	return c
}

// WithBaseURL sets a custom base URL (useful for testing)
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithAuthHeaderDisabled disables sending the default Authorization header.
func (c *Client) WithAuthHeaderDisabled() *Client {
	c.disableAuth = true
	return c
}

// WithMaxRetries sets the maximum number of retries for transient errors.
func (c *Client) WithMaxRetries(n int) *Client {
	c.maxRetries = n
	return c
}

// WithCircuitBreaker enables circuit breaker with custom threshold and recovery timeout
func (c *Client) WithCircuitBreaker(threshold int, recoveryTimeout time.Duration) *Client {
	c.circuitBreaker.enabled = true
	c.circuitBreaker.threshold = threshold
	c.circuitBreaker.recoveryTimeout = recoveryTimeout
	return c
}

// EnableCircuitBreaker enables circuit breaker with default settings
func (c *Client) EnableCircuitBreaker() *Client {
	c.circuitBreaker.enabled = true
	return c
}

// WithDebug enables debug mode for HTTP request/response logging
func (c *Client) WithDebug() *Client {
	return c.WithDebugOutput(os.Stderr)
}

// WithDebugOutput enables debug mode for HTTP request/response logging to the provided writer.
func (c *Client) WithDebugOutput(w io.Writer) *Client {
	// Wrap the existing transport with the debug transport
	baseTransport := c.httpClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	c.httpClient.Transport = debug.NewDebugTransport(baseTransport, w)
	return c
}

// doRequest performs an HTTP request with retry logic for rate limits and transient errors
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	// Check if circuit breaker is open
	if c.circuitBreaker.isOpen() {
		return nil, ctxerrors.WrapContext(method, url, 0, ErrCircuitOpen)
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt, lastErr)

			// Log retry with rate limit info if applicable
			if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode == http.StatusTooManyRequests {
				slog.Debug("rate limited, waiting before retry",
					"method", method,
					"path", path,
					"attempt", attempt,
					"delay", delay.String(),
					"retry_after", apiErr.RetryAfter.String())
			} else {
				slog.Debug("retrying request",
					"method", method,
					"path", path,
					"attempt", attempt,
					"delay", delay.String())
			}

			select {
			case <-ctx.Done():
				return nil, ctxerrors.WrapContext(method, url, 0, ctx.Err())
			case <-time.After(delay):
			}
		}

		resp, err := c.doRequestOnce(ctx, method, path, body)
		if err != nil {
			lastErr = err

			// Check if error is retryable
			if apiErr, ok := err.(*APIError); ok {
				if isRetryable(apiErr.StatusCode) {
					continue
				}
			}

			// Non-retryable error, return immediately
			return nil, ctxerrors.WrapContext(method, url, getStatusCode(err), err)
		}

		// Success - record it to reset circuit breaker
		c.circuitBreaker.recordSuccess()
		return resp, nil
	}

	// All retries exhausted - record as a single failure for circuit breaker
	if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode >= 500 {
		c.circuitBreaker.recordFailure()
	}

	return nil, ctxerrors.WrapContext(method, url, getStatusCode(lastErr), lastErr)
}

// doRequestOnce performs a single HTTP request attempt with proper headers and error handling
func (c *Client) doRequestOnce(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	contentType := ""
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
		contentType = "application/json"
	}

	return c.doRequestOnceWithReader(ctx, method, c.baseURL+path, reqBody, contentType, nil)
}

// doRequestOnceWithReader performs a single HTTP request attempt with a raw reader body.
// It applies shared auth/version headers and decodes API errors consistently.
func (c *Client) doRequestOnceWithReader(ctx context.Context, method, requestURL string, body io.Reader, contentType string, extraHeaders map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	if !c.disableAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Notion-Version", c.version)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range extraHeaders {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Update rate limit tracker with response headers
	c.rateLimiter.Update(resp)

	// Check for error responses
	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, &APIError{StatusCode: resp.StatusCode}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Response:   &errResp,
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	return resp, nil
}

// doMultipartRequest performs a multipart/form-data POST request with retry logic
func (c *Client) doMultipartRequest(ctx context.Context, url string, fieldName string, file io.Reader, filename, contentType string, result interface{}) error {
	// Check if circuit breaker is open
	if c.circuitBreaker.isOpen() {
		return ctxerrors.WrapContext(http.MethodPost, url, 0, ErrCircuitOpen)
	}

	// Read the entire file into memory so we can retry if needed
	fileData, err := io.ReadAll(file)
	if err != nil {
		return ctxerrors.WrapContext(http.MethodPost, url, 0, fmt.Errorf("failed to read file: %w", err))
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt, lastErr)

			// Log retry with rate limit info if applicable
			if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode == http.StatusTooManyRequests {
				slog.Debug("rate limited, waiting before retry",
					"url", url,
					"attempt", attempt,
					"delay", delay.String(),
					"retry_after", apiErr.RetryAfter.String())
			} else {
				slog.Debug("retrying multipart request",
					"url", url,
					"attempt", attempt,
					"delay", delay.String())
			}

			select {
			case <-ctx.Done():
				return ctxerrors.WrapContext(http.MethodPost, url, 0, ctx.Err())
			case <-time.After(delay):
			}
		}

		err := c.doMultipartRequestOnce(ctx, url, fieldName, bytes.NewReader(fileData), filename, contentType, result)
		if err != nil {
			lastErr = err

			// Check if error is retryable
			if apiErr, ok := err.(*APIError); ok {
				if isRetryable(apiErr.StatusCode) {
					continue
				}
			}

			// Non-retryable error, return immediately
			return ctxerrors.WrapContext(http.MethodPost, url, getStatusCode(err), err)
		}

		// Success - record it to reset circuit breaker
		c.circuitBreaker.recordSuccess()
		return nil
	}

	// All retries exhausted - record as a single failure for circuit breaker
	if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode >= 500 {
		c.circuitBreaker.recordFailure()
	}

	return ctxerrors.WrapContext(http.MethodPost, url, getStatusCode(lastErr), lastErr)
}

// doMultipartRequestOnce performs a single multipart/form-data POST request
func (c *Client) doMultipartRequestOnce(ctx context.Context, url string, fieldName string, file io.Reader, filename, contentType string, result interface{}) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form part with correct content type (not application/octet-stream default)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	if contentType != "" {
		h.Set("Content-Type", contentType)
	} else {
		h.Set("Content-Type", "application/octet-stream")
	}
	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create form part: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	resp, err := c.doRequestOnceWithReader(ctx, http.MethodPost, url, &buf, writer.FormDataContentType(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// calculateRetryDelay calculates the delay before the next retry attempt
func (c *Client) calculateRetryDelay(attempt int, lastErr error) time.Duration {
	// Check if the error has a Retry-After header
	if apiErr, ok := lastErr.(*APIError); ok && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	// Exponential backoff: 1s, 2s, 4s
	delay := baseDelay * time.Duration(1<<(attempt-1))

	// Add jitter (0-25% of delay)
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	delay += jitter

	return delay
}

// isRetryable returns true if the HTTP status code indicates a retryable error
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

// parseRetryAfter parses the Retry-After header value
// Returns the duration to wait, or 0 if not parseable
func parseRetryAfter(retryAfter string) time.Duration {
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date (not commonly used by Notion API, but part of HTTP spec)
	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

// getStatusCode extracts the HTTP status code from an error if it's an APIError
func getStatusCode(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode
	}
	return 0
}

// doGet performs a GET request with optional query parameters
func (c *Client) doGet(ctx context.Context, path string, query url.Values, result interface{}) error {
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doPost performs a POST request
func (c *Client) doPost(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doPatch performs a PATCH request
func (c *Client) doPatch(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doDelete performs a DELETE request
func (c *Client) doDelete(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// GetRateLimitInfo returns the current rate limit information
// Returns nil if no API requests have been made yet
func (c *Client) GetRateLimitInfo() *RateLimitInfo {
	return c.rateLimiter.Get()
}
