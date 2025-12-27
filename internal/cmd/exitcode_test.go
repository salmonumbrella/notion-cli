package cmd

import (
	"context"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"canceled", context.Canceled, ExitCanceled},
		{"user", clierrors.NewUserError("bad", "hint"), ExitUser},
		{"validation", &clierrors.ValidationError{Field: "x", Message: "bad"}, ExitUser},
		{"auth", &clierrors.AuthError{Reason: "no token"}, ExitAuth},
		{"rate_limit", &clierrors.RateLimitError{}, ExitRateLimit},
		{"circuit_breaker", &clierrors.CircuitBreakerError{}, ExitTemp},
		{"api_404", &notion.APIError{StatusCode: 404}, ExitNotFound},
		{"api_429", &notion.APIError{StatusCode: 429}, ExitRateLimit},
		{"api_401", &notion.APIError{StatusCode: 401}, ExitAuth},
		{"api_403", &notion.APIError{StatusCode: 403}, ExitAuth},
		{"api_400", &notion.APIError{StatusCode: 400}, ExitUser},
		{"api_500", &notion.APIError{StatusCode: 500}, ExitSystem},
		{"auth_required", clierrors.AuthRequiredError(nil), ExitAuth},
		{"proxied_exit", &proxiedCommandExitError{Code: 42}, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
