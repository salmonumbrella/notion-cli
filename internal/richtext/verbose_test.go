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
				"User mentions:",
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

	if !strings.Contains(output, "  User mentions:") {
		t.Errorf("expected indented 'User mentions:', got: %q", output)
	}
	if !strings.Contains(output, "    @Alice → alice-id") {
		t.Errorf("expected double-indented mapping, got: %q", output)
	}
}

func TestFormatAllMentionMappings(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		userIDs        []string
		pageIDs        []string
		expectContains []string
	}{
		{
			name:    "user mentions only",
			text:    "@Alice says hello",
			userIDs: []string{"alice-id"},
			pageIDs: nil,
			expectContains: []string{
				"User mentions:",
				"@Alice → alice-id",
			},
		},
		{
			name:    "page mentions only",
			text:    "See @@ProjectPlan",
			userIDs: nil,
			pageIDs: []string{"project-plan-id"},
			expectContains: []string{
				"Page mentions:",
				"@@ProjectPlan → project-plan-id",
			},
		},
		{
			name:    "mixed user and page mentions",
			text:    "@Alice see @@ProjectPlan",
			userIDs: []string{"alice-id"},
			pageIDs: []string{"project-plan-id"},
			expectContains: []string{
				"User mentions:",
				"@Alice → alice-id",
				"Page mentions:",
				"@@ProjectPlan → project-plan-id",
			},
		},
		{
			name:    "page mention without ID",
			text:    "See @@ProjectPlan",
			userIDs: nil,
			pageIDs: nil,
			expectContains: []string{
				"Page mentions:",
				"@@ProjectPlan → (no page ID available)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			FormatAllMentionMappings(&buf, tt.text, tt.userIDs, tt.pageIDs)
			output := buf.String()

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got: %q", expected, output)
				}
			}
		})
	}
}
