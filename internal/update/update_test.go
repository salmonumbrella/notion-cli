// internal/update/update_test.go
package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeHTTPClient struct {
	status int
	body   string
	err    error
}

func (f fakeHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.2.3", "1.2.3"},
		{"v2.0.0-beta", "2.0.0-beta"},
	}

	for _, tt := range tests {
		result := parseVersion(tt.input)
		if result != tt.expected {
			t.Errorf("parseVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "2.0.0", true},
		{"1.9.0", "1.10.0", true}, // integer comparison, not string
		{"dev", "1.0.0", false},   // dev version, don't prompt
		{"", "1.0.0", false},      // empty version, don't prompt
	}

	for _, tt := range tests {
		result := isNewer(tt.current, tt.latest)
		if result != tt.expected {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
		}
	}
}

func TestShouldCheck(t *testing.T) {
	now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	checker := NewChecker(
		WithNow(func() time.Time { return now }),
		WithCheckInterval(24*time.Hour),
	)

	// First check should always succeed
	if !checker.shouldCheck("", time.Time{}) {
		t.Error("first check should always be allowed")
	}

	// Recent check should be skipped
	recent := now.Add(-1 * time.Hour)
	if checker.shouldCheck("1.0.0", recent) {
		t.Error("recent check should be skipped")
	}

	// Old check should be allowed
	old := now.Add(-25 * time.Hour)
	if !checker.shouldCheck("1.0.0", old) {
		t.Error("old check should be allowed")
	}
}

func TestCheckerCheck_FetchError(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	_, err := CheckWithOptions(
		context.Background(),
		"1.0.0",
		WithCachePath(cachePath),
		WithHTTPClient(fakeHTTPClient{err: errors.New("boom")}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var updateErr *UpdateError
	if !errors.As(err, &updateErr) {
		t.Fatalf("expected UpdateError, got %T", err)
	}
}

func TestCheckerCheck_SaveCacheErrorReturnsMessage(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	msg, err := CheckWithOptions(
		context.Background(),
		"1.0.0",
		WithCachePath(cachePath),
		WithHTTPClient(fakeHTTPClient{
			status: http.StatusOK,
			body:   `{"tag_name":"v9.9.9"}`,
		}),
		func(c *Checker) {
			c.writeFile = func(string, []byte, os.FileMode) error { return errors.New("write failed") }
		},
	)
	if msg == "" {
		t.Fatal("expected update message despite cache write failure")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
