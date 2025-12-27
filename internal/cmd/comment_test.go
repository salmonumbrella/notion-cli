package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildCommentRichTextVerbose_Output(t *testing.T) {
	var buf bytes.Buffer
	_ = buildCommentRichTextVerbose(&buf, "@Reviewer says hello", []string{"reviewer-id"}, nil, true, false)

	output := buf.String()

	// Should contain markdown summary line
	if !strings.Contains(output, "Parsed markdown:") {
		t.Errorf("verbose output should contain 'Parsed markdown:' summary, got: %s", output)
	}

	// Should contain mention mappings
	if !strings.Contains(output, "User mentions:") {
		t.Errorf("verbose output should contain 'User mentions:' header, got: %s", output)
	}
	if !strings.Contains(output, "@Reviewer → reviewer-id") {
		t.Errorf("verbose output should show mention mapping, got: %s", output)
	}
}

func TestBuildCommentRichTextVerbose_NoUserID(t *testing.T) {
	var buf bytes.Buffer
	_ = buildCommentRichTextVerbose(&buf, "@Alice and @Bob", []string{"alice-id"}, nil, true, false)

	output := buf.String()

	if !strings.Contains(output, "@Alice → alice-id") {
		t.Errorf("verbose output should show @Alice mapped, got: %s", output)
	}
	if !strings.Contains(output, "@Bob → (no user ID available)") {
		t.Errorf("verbose output should show @Bob without user ID, got: %s", output)
	}
}

func TestBuildCommentRichTextVerbose_Warnings(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		userIDs        []string
		pageIDs        []string
		emitWarnings   bool
		expectContains []string
		expectMissing  []string
	}{
		{
			name:         "no warning when emitWarnings is false",
			text:         "plain text",
			userIDs:      []string{"unused-id"},
			emitWarnings: false,
			expectMissing: []string{
				"warning:",
			},
		},
		{
			name:         "warning when all user mentions unused",
			text:         "plain text with no mentions",
			userIDs:      []string{"unused-id"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 1 --mention flag(s) provided but no @Name patterns found",
			},
		},
		{
			name:         "warning when some user mentions unused",
			text:         "@Alice is here",
			userIDs:      []string{"alice-id", "extra-id", "another-extra"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 2 of 3 --mention flag(s) unused",
			},
		},
		{
			name:          "no warning when all user mentions used",
			text:          "@Alice and @Bob",
			userIDs:       []string{"alice-id", "bob-id"},
			emitWarnings:  true,
			expectMissing: []string{"warning:"},
		},
		{
			name:          "no warning when no userIDs provided",
			text:          "plain text",
			userIDs:       nil,
			emitWarnings:  true,
			expectMissing: []string{"warning:"},
		},
		{
			name:         "warning when all page mentions unused",
			text:         "plain text with no page mentions",
			pageIDs:      []string{"unused-page-id"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 1 --page-mention flag(s) provided but no @@Name patterns found",
			},
		},
		{
			name:         "warning when some page mentions unused",
			text:         "@@PageOne is here",
			pageIDs:      []string{"page-one-id", "extra-page-id"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 1 of 2 --page-mention flag(s) unused",
			},
		},
		{
			name:          "no warning when all page mentions used",
			text:          "@@PageOne and @@PageTwo",
			pageIDs:       []string{"page-one-id", "page-two-id"},
			emitWarnings:  true,
			expectMissing: []string{"warning:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_ = buildCommentRichTextVerbose(&buf, tt.text, tt.userIDs, tt.pageIDs, false, tt.emitWarnings)

			output := buf.String()

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got: %s", expected, output)
				}
			}

			for _, notExpected := range tt.expectMissing {
				if strings.Contains(output, notExpected) {
					t.Errorf("expected output to NOT contain %q, got: %s", notExpected, output)
				}
			}
		})
	}
}

func TestBuildCommentRichTextVerbose_NoVerbose(t *testing.T) {
	var buf bytes.Buffer
	_ = buildCommentRichTextVerbose(&buf, "@Reviewer says hello", []string{"reviewer-id"}, nil, false, false)

	output := buf.String()

	// Should be empty when verbose is false and no warnings
	if output != "" {
		t.Errorf("expected no output when verbose=false, got: %s", output)
	}
}
