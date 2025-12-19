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
