package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.notion.com/v1"
	apiVersion     = "2025-09-03"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	baseDelay      = 1 * time.Second
)

// Client is the Notion API client
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
	version    string
}

// NewClient creates a new Notion API client with the given token
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		token:   token,
		baseURL: defaultBaseURL,
		version: apiVersion,
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

// doRequest performs an HTTP request with retry logic for rate limits and transient errors
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt, lastErr)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
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
			return nil, err
		}

		// Success
		return resp, nil
	}

	// All retries exhausted
	return nil, lastErr
}

// doRequestOnce performs a single HTTP request attempt with proper headers and error handling
func (c *Client) doRequestOnce(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", c.version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for error responses
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()

		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			// If we can't decode the error response, return a generic error
			return nil, &APIError{
				StatusCode: resp.StatusCode,
			}
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
func (c *Client) doMultipartRequest(ctx context.Context, url string, fieldName string, file io.Reader, filename string, result interface{}) error {
	// Read the entire file into memory so we can retry if needed
	fileData, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			delay := c.calculateRetryDelay(attempt, lastErr)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doMultipartRequestOnce(ctx, url, fieldName, bytes.NewReader(fileData), filename, result)
		if err != nil {
			lastErr = err

			// Check if error is retryable
			if apiErr, ok := err.(*APIError); ok {
				if isRetryable(apiErr.StatusCode) {
					continue
				}
			}

			// Non-retryable error, return immediately
			return err
		}

		// Success
		return nil
	}

	// All retries exhausted
	return lastErr
}

// doMultipartRequestOnce performs a single multipart/form-data POST request
func (c *Client) doMultipartRequestOnce(ctx context.Context, url string, fieldName string, file io.Reader, filename string, result interface{}) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return &APIError{StatusCode: resp.StatusCode}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Response:   &errResp,
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

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

// doGet performs a GET request with optional query parameters
func (c *Client) doGet(ctx context.Context, path string, query url.Values, result interface{}) error {
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
