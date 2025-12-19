package errors

import (
	"errors"
	"testing"
	"time"
)

func TestAPIError(t *testing.T) {
	err := &APIError{
		Status:  400,
		Code:    "validation_error",
		Message: "Invalid request",
	}

	expected := "notion api error (status 400, code validation_error): Invalid request"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	if !IsAPIError(err) {
		t.Error("IsAPIError should return true for APIError")
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "page_id",
		Message: "must be a valid UUID",
	}

	expected := "validation error for page_id: must be a valid UUID"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	if !IsValidationError(err) {
		t.Error("IsValidationError should return true for ValidationError")
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{
		RetryAfter: 30 * time.Second,
	}

	expected := "rate limited, retry after 30s"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	if !IsRateLimitError(err) {
		t.Error("IsRateLimitError should return true for RateLimitError")
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{
		Reason: "invalid API key",
	}

	expected := "authentication error: invalid API key"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	if !IsAuthError(err) {
		t.Error("IsAuthError should return true for AuthError")
	}
}

func TestCircuitBreakerError(t *testing.T) {
	err := &CircuitBreakerError{}

	expected := "service temporarily unavailable (circuit breaker open)"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	if !IsCircuitBreakerError(err) {
		t.Error("IsCircuitBreakerError should return true for CircuitBreakerError")
	}
}

func TestTypeCheckers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{
			name:    "APIError with wrapped error",
			err:     errors.New("wrapped: " + (&APIError{Status: 500, Code: "internal_error", Message: "test"}).Error()),
			checker: IsAPIError,
			want:    false,
		},
		{
			name:    "nil error",
			err:     nil,
			checker: IsAPIError,
			want:    false,
		},
		{
			name:    "generic error",
			err:     errors.New("generic error"),
			checker: IsValidationError,
			want:    false,
		},
		{
			name:    "ValidationError is not APIError",
			err:     &ValidationError{Field: "test", Message: "test"},
			checker: IsAPIError,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checker(tt.err); got != tt.want {
				t.Errorf("checker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := &APIError{
		Status:  404,
		Code:    "object_not_found",
		Message: "Page not found",
	}

	wrappedErr := errors.New("failed to fetch page: " + baseErr.Error())

	// Wrapped errors should not match type checks (demonstrating proper use of errors.As)
	if IsAPIError(wrappedErr) {
		t.Error("String-wrapped APIError should not match IsAPIError")
	}

	// But errors.As-based wrapping would work
	properlyWrapped := &APIError{
		Status:  404,
		Code:    "object_not_found",
		Message: "Page not found",
	}

	if !IsAPIError(properlyWrapped) {
		t.Error("Properly typed error should match IsAPIError")
	}
}
