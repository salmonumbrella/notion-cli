// internal/update/update_test.go
package update

import (
	"testing"
	"time"
)

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
		{"dev", "1.0.0", false}, // dev version, don't prompt
		{"", "1.0.0", false},    // empty version, don't prompt
	}

	for _, tt := range tests {
		result := isNewer(tt.current, tt.latest)
		if result != tt.expected {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
		}
	}
}

func TestShouldCheck(t *testing.T) {
	// First check should always succeed
	if !shouldCheck("", time.Time{}) {
		t.Error("first check should always be allowed")
	}

	// Recent check should be skipped
	recent := time.Now().Add(-1 * time.Hour)
	if shouldCheck("1.0.0", recent) {
		t.Error("recent check should be skipped")
	}

	// Old check should be allowed
	old := time.Now().Add(-25 * time.Hour)
	if !shouldCheck("1.0.0", old) {
		t.Error("old check should be allowed")
	}
}
