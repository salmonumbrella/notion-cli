package richtext

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatMentionMappings(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		userIDs        []string
		expectContains []string
	}{
		{
			name:    "single mention with user ID",
			text:    "@Alice says hello",
			userIDs: []string{"alice-id"},
			expectContains: []string{
				"Mentions:",
				"@Alice → alice-id",
			},
		},
		{
			name:    "multiple mentions",
			text:    "@Alice and @Bob",
			userIDs: []string{"alice-id", "bob-id"},
			expectContains: []string{
				"@Alice → alice-id",
				"@Bob → bob-id",
			},
		},
		{
			name:    "mention without user ID",
			text:    "@Alice and @Bob",
			userIDs: []string{"alice-id"},
			expectContains: []string{
				"@Alice → alice-id",
				"@Bob → (no user ID available)",
			},
		},
		{
			name:           "no mentions",
			text:           "plain text",
			userIDs:        []string{},
			expectContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			FormatMentionMappings(&buf, tt.text, tt.userIDs)
			output := buf.String()

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got: %q", expected, output)
				}
			}

			if len(tt.expectContains) == 0 && output != "" {
				t.Errorf("expected empty output for no mentions, got: %q", output)
			}
		})
	}
}

func TestFormatMentionMappingsIndented(t *testing.T) {
	var buf bytes.Buffer
	FormatMentionMappingsIndented(&buf, "@Alice says hi", []string{"alice-id"}, "  ")
	output := buf.String()

	if !strings.Contains(output, "  Mentions:") {
		t.Errorf("expected indented 'Mentions:', got: %q", output)
	}
	if !strings.Contains(output, "    @Alice → alice-id") {
		t.Errorf("expected double-indented mapping, got: %q", output)
	}
}
