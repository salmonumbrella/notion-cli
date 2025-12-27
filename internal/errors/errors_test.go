package errors

import (
	"errors"
	"strings"
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
	err := WrapUserError(base, "authentication required", "Run 'ntn auth login'")

	if !IsUserError(err) {
		t.Error("IsUserError should return true for UserError")
	}

	if got := UserSuggestion(err); got != "Run 'ntn auth login'" {
		t.Errorf("UserSuggestion() = %q, want %q", got, "Run 'ntn auth login'")
	}

	expected := "authentication required: missing token"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("page", "abc123")

	if !IsUserError(err) {
		t.Error("NotFoundError should be a UserError")
	}

	if !strings.Contains(err.Error(), "page") {
		t.Errorf("Error should mention entity type, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "abc123") {
		t.Errorf("Error should mention identifier, got: %s", err.Error())
	}

	suggestion := UserSuggestion(err)
	if !strings.Contains(suggestion, "ntn search") {
		t.Errorf("Suggestion should include search command, got: %s", suggestion)
	}
}

func TestNoDatabaseConfiguredError(t *testing.T) {
	err := NoDatabaseConfiguredError()
	if !IsUserError(err) {
		t.Error("NoDatabaseConfiguredError should be a UserError")
	}

	suggestion := UserSuggestion(err)
	if !strings.Contains(suggestion, "ntn skill init") {
		t.Errorf("Suggestion should include skill init, got: %s", suggestion)
	}
	if !strings.Contains(suggestion, "ntn page create --parent") {
		t.Errorf("Suggestion should include explicit page create command, got: %s", suggestion)
	}
}

func TestAPINotFoundError(t *testing.T) {
	// Test with a 404-style error
	baseErr := errors.New("notion API error 404 (object_not_found): Page not found")
	err := APINotFoundError(baseErr, "page", "abc123")

	if !IsUserError(err) {
		t.Error("APINotFoundError should return a UserError for 404")
	}

	suggestion := UserSuggestion(err)
	if !strings.Contains(suggestion, "ntn search") {
		t.Errorf("Suggestion should include search command, got: %s", suggestion)
	}

	// Test with a non-404 error
	baseErr = errors.New("rate limited")
	err = APINotFoundError(baseErr, "page", "abc123")

	// Should return original error unchanged
	if err.Error() != baseErr.Error() {
		t.Errorf("Expected original error for non-404, got: %s", err.Error())
	}
}

func TestContains404Indicators(t *testing.T) {
	tests := []struct {
		errStr   string
		expected bool
	}{
		{"notion API error 404 (object_not_found): Page not found", true},
		{"object_not_found", true},
		{"Could not find page", true},
		{"page not found", true},
		{"rate limited", false},
		{"validation error", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.errStr, func(t *testing.T) {
			if got := contains404Indicators(tt.errStr); got != tt.expected {
				t.Errorf("contains404Indicators(%q) = %v, want %v", tt.errStr, got, tt.expected)
			}
		})
	}
}
