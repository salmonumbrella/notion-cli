package notion

import (
	"fmt"
	"time"
)

// ErrorResponse represents a Notion API error response
type ErrorResponse struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("notion API error %d (%s): %s", e.Status, e.Code, e.Message)
}

// APIError wraps an ErrorResponse with additional context
type APIError struct {
	StatusCode int
	Response   *ErrorResponse
	RetryAfter time.Duration
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Response != nil {
		return e.Response.Error()
	}
	return fmt.Sprintf("notion API error %d", e.StatusCode)
}
