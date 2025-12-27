package cmd

import (
	"testing"
)

func TestAuthLoginCommand_HasNoBrowserFlag(t *testing.T) {
	cmd := newAuthLoginCmd()
	flag := cmd.Flags().Lookup("no-browser")
	if flag == nil {
		t.Fatalf("expected --no-browser flag to exist")
	}
}

func TestEnvTruthy(t *testing.T) {
	t.Setenv("NOTION_TEST_TRUTHY", "")
	if envTruthy("NOTION_TEST_TRUTHY") {
		t.Fatalf("expected empty env to be false")
	}

	t.Setenv("NOTION_TEST_TRUTHY", "true")
	if !envTruthy("NOTION_TEST_TRUTHY") {
		t.Fatalf("expected true env to be truthy")
	}

	t.Setenv("NOTION_TEST_TRUTHY", "1")
	if !envTruthy("NOTION_TEST_TRUTHY") {
		t.Fatalf("expected 1 env to be truthy")
	}

	t.Setenv("NOTION_TEST_TRUTHY", "false")
	if envTruthy("NOTION_TEST_TRUTHY") {
		t.Fatalf("expected false env to be false")
	}
}

func TestIsValidNotionTokenFormat(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		// Valid tokens
		{"secret_ prefix", "secret_abc123xyz", true},
		{"ntn_ prefix", "ntn_abc123xyz", true},
		{"long secret token", "secret_" + string(make([]byte, 100)), true},
		{"long ntn token", "ntn_" + string(make([]byte, 100)), true},

		// Invalid tokens
		{"empty", "", false},
		{"no prefix", "abc123xyz", false},
		{"wrong prefix", "token_abc123", false},
		{"partial secret prefix", "secre_abc123", false},
		{"partial ntn prefix", "nt_abc123", false},
		{"uppercase SECRET", "SECRET_abc123", false},
		{"uppercase NTN", "NTN_abc123", false},
		{"secret without underscore", "secretabc123", false},
		{"ntn without underscore", "ntnabc123", false},
		{"only prefix secret_", "secret_", true},   // technically valid format
		{"only prefix ntn_", "ntn_", true},         // technically valid format
		{"space in token", "secret_ abc123", true}, // format check only
		{"special chars", "secret_!@#$%", true},    // format check only
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidNotionTokenFormat(tt.token)
			if result != tt.expected {
				t.Errorf("isValidNotionTokenFormat(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestGenerateState(t *testing.T) {
	// Generate multiple states and ensure they're unique and well-formed
	states := make(map[string]bool)

	for i := 0; i < 100; i++ {
		state, err := generateState()
		if err != nil {
			t.Fatalf("generateState() error = %v", err)
		}

		// Check length (16 bytes = 32 hex chars)
		if len(state) != 32 {
			t.Errorf("generateState() length = %d, want 32", len(state))
		}

		// Check for hex characters only
		for _, c := range state {
			isHexDigit := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
			if !isHexDigit {
				t.Errorf("generateState() contains non-hex character: %c", c)
			}
		}

		// Check uniqueness
		if states[state] {
			t.Errorf("generateState() produced duplicate: %s", state)
		}
		states[state] = true
	}
}
