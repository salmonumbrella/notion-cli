package errors

import (
	"errors"
	"fmt"
	"strings"
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
	Reason     string
	Suggestion string
	Err        error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("authentication error: %s: %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("authentication error: %s", e.Reason)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// AuthRequiredError wraps an error with authentication required message and suggestion.
func AuthRequiredError(err error) error {
	return &AuthError{
		Reason:     "authentication required",
		Suggestion: "Run 'ntn auth login' or 'ntn auth add-token' to configure",
		Err:        err,
	}
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

// UserSuggestion returns a suggestion string if err is a UserError or AuthError.
func UserSuggestion(err error) string {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Suggestion
	}
	var ae *AuthError
	if errors.As(err, &ae) {
		return ae.Suggestion
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

// NotFoundError creates a user-friendly error for when a resource is not found.
// entityType is the type of entity (e.g., "page", "database", "block").
// identifier is the ID or name that was searched for.
func NotFoundError(entityType, identifier string) error {
	suggestion := fmt.Sprintf("Run 'ntn search %s' to find matching %ss\n  • Check the ID or name is correct\n  • Verify your integration has access to this %s", identifier, entityType, entityType)
	return NewUserError(
		fmt.Sprintf("%s %q not found", entityType, identifier),
		suggestion,
	)
}

// NotFoundWithSearchError creates a user-friendly error for when a resource is not found,
// including search results that might be helpful.
func NotFoundWithSearchError(entityType, identifier string, suggestions []string) error {
	msg := fmt.Sprintf("%s %q not found", entityType, identifier)
	var suggestion string
	if len(suggestions) > 0 {
		suggestion = fmt.Sprintf("Did you mean one of these?\n%s\n\nOr run 'ntn search %s' to find more matches", formatSuggestionList(suggestions), identifier)
	} else {
		suggestion = fmt.Sprintf("Run 'ntn search %s' to find matching %ss\n  • Check the ID or name is correct\n  • Verify your integration has access to this %s", identifier, entityType, entityType)
	}
	return NewUserError(msg, suggestion)
}

// NoDatabaseConfiguredError creates a user-friendly error when no database is configured for page creation.
func NoDatabaseConfiguredError() error {
	msg := "no database configured for page creation"
	suggestion := "To fix this:\n  1. Run 'ntn skill init' to configure your workspace\n  2. Or use 'ntn page create --parent <database-id> --parent-type database --properties '{...}' explicitly"
	return NewUserError(msg, suggestion)
}

// formatSuggestionList formats a list of suggestions as a bulleted list.
func formatSuggestionList(items []string) string {
	var result string
	for _, item := range items {
		result += fmt.Sprintf("  • %s\n", item)
	}
	return result
}

// APINotFoundError wraps an API error with helpful suggestions for "not found" responses.
// Returns the original error if it's not a 404-style error.
func APINotFoundError(err error, entityType, identifier string) error {
	if err == nil {
		return nil
	}
	// Check if the error message contains "not found" or similar
	errStr := err.Error()
	if contains404Indicators(errStr) {
		return WrapUserError(err, fmt.Sprintf("failed to get %s", entityType),
			fmt.Sprintf("The %s %q was not found.\n\nSuggestions:\n  • Run 'ntn search %s' to find matching %ss\n  • Check the ID or name is correct\n  • Verify your integration has access to this %s",
				entityType, identifier, identifier, entityType, entityType))
	}
	return err
}

// contains404Indicators checks if an error message indicates a "not found" error.
func contains404Indicators(errStr string) bool {
	indicators := []string{
		"object_not_found",
		"404",
		"not found",
		"could not find",
	}
	errLower := strings.ToLower(errStr)
	for _, indicator := range indicators {
		if strings.Contains(errLower, indicator) {
			return true
		}
	}
	return false
}
