package errors

import (
	"errors"
	"fmt"
	"time"
)

// ValidationError represents an input validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// UserError represents an error caused by user input or configuration.
// Suggestion can provide a concrete fix for the user.
type UserError struct {
	Message    string
	Suggestion string
	Err        error
}

func (e *UserError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *UserError) Unwrap() error {
	return e.Err
}

// NewUserError creates a UserError with a message and optional suggestion.
func NewUserError(message, suggestion string) *UserError {
	return &UserError{Message: message, Suggestion: suggestion}
}

// WrapUserError wraps an underlying error with a user-facing message and suggestion.
func WrapUserError(err error, message, suggestion string) *UserError {
	return &UserError{Message: message, Suggestion: suggestion, Err: err}
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

// AuthRequiredError wraps an error with authentication required message and suggestion.
func AuthRequiredError(err error) error {
	return WrapUserError(err, "authentication required", "Run 'notion auth login' or 'notion auth add-token' to configure")
}

// CircuitBreakerError indicates the circuit is open
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "service temporarily unavailable (circuit breaker open)"
}

// Type checkers
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

func IsUserError(err error) bool {
	var e *UserError
	return errors.As(err, &e)
}

// UserSuggestion returns a suggestion string if err is a UserError.
func UserSuggestion(err error) string {
	var e *UserError
	if errors.As(err, &e) {
		return e.Suggestion
	}
	return ""
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
