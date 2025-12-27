package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestWrapAPIError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		action     string
		entityType string
		identifier string
		wantNil    bool
		wantMsg    string // substring to check in the error message
		wantType   string // "user" for *UserError, "wrapped" for fmt.Errorf wrap, "original" for unchanged
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			action:  "get page",
			wantNil: true,
		},
		{
			name:       "404 API error returns UserError",
			err:        &notion.APIError{StatusCode: 404, Response: &notion.ErrorResponse{Status: 404, Code: "object_not_found", Message: "not found"}},
			action:     "get page",
			entityType: "page",
			identifier: "abc-123",
			wantType:   "user",
			wantMsg:    "failed to get page",
		},
		{
			name:       "error containing not found text returns UserError",
			err:        fmt.Errorf("object_not_found: could not find resource"),
			action:     "get database",
			entityType: "database",
			identifier: "db-456",
			wantType:   "user",
			wantMsg:    "failed to get database",
		},
		{
			name:       "non-404 error with action wraps with failed to prefix",
			err:        fmt.Errorf("connection refused"),
			action:     "list pages",
			entityType: "page",
			identifier: "xyz",
			wantType:   "wrapped",
			wantMsg:    "failed to list pages",
		},
		{
			name:       "non-404 error with empty action returns original",
			err:        fmt.Errorf("connection refused"),
			action:     "",
			entityType: "page",
			identifier: "xyz",
			wantType:   "original",
			wantMsg:    "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapAPIError(tt.err, tt.action, tt.entityType, tt.identifier)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("wrapAPIError() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("wrapAPIError() = nil, want non-nil error")
			}

			switch tt.wantType {
			case "user":
				var ue *clierrors.UserError
				if !errors.As(got, &ue) {
					t.Fatalf("wrapAPIError() type = %T, want *errors.UserError", got)
				}
			case "wrapped":
				// Should wrap the original error
				if !errors.Is(got, tt.err) {
					t.Fatal("wrapAPIError() does not wrap the original error")
				}
			case "original":
				if got != tt.err {
					t.Fatalf("wrapAPIError() = %v, want original error %v", got, tt.err)
				}
			}

			if msg := got.Error(); !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("wrapAPIError().Error() = %q, want substring %q", msg, tt.wantMsg)
			}
		})
	}
}
