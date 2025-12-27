package cmd

import (
	"context"
	"errors"
	"net/http"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

const (
	ExitOK        = 0
	ExitSystem    = 1
	ExitUser      = 2
	ExitAuth      = 3
	ExitNotFound  = 4
	ExitRateLimit = 5
	ExitTemp      = 6
	ExitCanceled  = 130
)

// ExitCode maps a command error to a stable process exit code for automation.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if errors.Is(err, context.Canceled) {
		return ExitCanceled
	}
	if code, ok := proxiedCommandExitStatus(err); ok {
		return code
	}

	var apiErr *notion.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusNotFound {
			return ExitNotFound
		}
		if apiErr.StatusCode == http.StatusTooManyRequests {
			return ExitRateLimit
		}
		if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
			return ExitAuth
		}
		// Notion API errors caused by invalid input are still "user" in most cases.
		// We conservatively treat them as user errors unless there is a clear server
		// class status.
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return ExitUser
		}
		return ExitSystem
	}

	if clierrors.IsRateLimitError(err) {
		return ExitRateLimit
	}
	if clierrors.IsAuthError(err) {
		return ExitAuth
	}
	if clierrors.IsCircuitBreakerError(err) {
		return ExitTemp
	}
	if clierrors.IsValidationError(err) || clierrors.IsUserError(err) {
		return ExitUser
	}

	return ExitSystem
}
