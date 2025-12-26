package errors

import (
	"errors"
	"fmt"
	"time"
)

// APIError represents a Notion API error response
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("notion api error (status %d, code %s): %s", e.Status, e.Code, e.Message)
}

// ValidationError represents an input validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// RateLimitError represents a 429 response
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %v", e.RetryAfter)
}

// AuthError represents authentication failures
type AuthError struct {
	Reason string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %s", e.Reason)
}

// CircuitBreakerError indicates the circuit is open
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "service temporarily unavailable (circuit breaker open)"
}

// Type checkers
func IsAPIError(err error) bool {
	var e *APIError
	return errors.As(err, &e)
}

func IsRateLimitError(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

func IsAuthError(err error) bool {
	var e *AuthError
	return errors.As(err, &e)
}

func IsCircuitBreakerError(err error) bool {
	var e *CircuitBreakerError
	return errors.As(err, &e)
}

func IsValidationError(err error) bool {
	var e *ValidationError
	return errors.As(err, &e)
}

// ContextualError wraps an error with HTTP request context for debugging.
type ContextualError struct {
	Method     string
	URL        string
	StatusCode int
	Err        error
}

// WrapContext wraps an error with HTTP request context.
// StatusCode can be 0 if the request never completed.
// Returns nil if err is nil.
func WrapContext(method, url string, statusCode int, err error) error {
	if err == nil {
		return nil
	}
	return &ContextualError{
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
		Err:        err,
	}
}

func (e *ContextualError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s %s (%d): %s", e.Method, e.URL, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s %s: %s", e.Method, e.URL, e.Err)
}

func (e *ContextualError) Unwrap() error {
	return e.Err
}

// IsContextualError checks if an error is a ContextualError.
func IsContextualError(err error) bool {
	var ce *ContextualError
	return errors.As(err, &ce)
}
