package errors

import (
	"errors"
	"testing"
	"time"
)

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
			name:    "generic error",
			err:     errors.New("generic error"),
			checker: IsValidationError,
			want:    false,
		},
		{
			name:    "nil error",
			err:     nil,
			checker: IsValidationError,
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

func TestContextualError(t *testing.T) {
	inner := errors.New("connection refused")
	err := WrapContext("POST", "https://api.notion.com/v1/pages", 0, inner)

	ctxErr, ok := err.(*ContextualError)
	if !ok {
		t.Fatalf("expected *ContextualError, got %T", err)
	}

	if ctxErr.Method != "POST" {
		t.Errorf("expected method POST, got %s", ctxErr.Method)
	}
	if ctxErr.URL != "https://api.notion.com/v1/pages" {
		t.Errorf("expected URL, got %s", ctxErr.URL)
	}
	if !errors.Is(err, inner) {
		t.Errorf("expected Unwrap to return inner error")
	}

	expected := "POST https://api.notion.com/v1/pages: connection refused"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestContextualError_WithStatusCode(t *testing.T) {
	inner := errors.New("not found")
	err := WrapContext("GET", "/pages/123", 404, inner)

	expected := "GET /pages/123 (404): not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestContextualError_NilError(t *testing.T) {
	err := WrapContext("GET", "/test", 200, nil)
	if err != nil {
		t.Errorf("expected nil when wrapping nil error, got %v", err)
	}
}

func TestIsContextualError(t *testing.T) {
	inner := errors.New("test error")
	err := WrapContext("GET", "/test", 500, inner)

	if !IsContextualError(err) {
		t.Error("expected IsContextualError to return true")
	}

	if IsContextualError(inner) {
		t.Error("expected IsContextualError to return false for non-contextual error")
	}
}

func TestUserError(t *testing.T) {
	base := errors.New("missing token")
	err := WrapUserError(base, "authentication required", "Run 'notion auth login'")

	if !IsUserError(err) {
		t.Error("IsUserError should return true for UserError")
	}

	if got := UserSuggestion(err); got != "Run 'notion auth login'" {
		t.Errorf("UserSuggestion() = %q, want %q", got, "Run 'notion auth login'")
	}

	expected := "authentication required: missing token"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}
