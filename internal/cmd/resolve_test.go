package cmd

import (
	"context"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func TestResolveID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Aliases: map[string]skill.CustomAlias{
			"standup": {TargetID: "page-789", Type: "page"},
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"me", "user-456"},
		{"standup", "page-789"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveID_NilSkillFile(t *testing.T) {
	got := resolveID(nil, "test-input")
	if got != "test-input" {
		t.Errorf("resolveID(nil, %q) = %q, want %q", "test-input", got, "test-input")
	}
}

func TestResolveUserID(t *testing.T) {
	sf := &skill.SkillFile{
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Databases: map[string]skill.DatabaseAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"me", "user-456"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveUserID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveUserID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveDatabaseID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users:   map[string]skill.UserAlias{},
		Aliases: map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveDatabaseID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveDatabaseID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWithSkillFile(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	ctx := context.Background()
	ctx = WithSkillFile(ctx, sf)

	got := SkillFileFromContext(ctx)
	if got != sf {
		t.Errorf("SkillFileFromContext() did not return the expected skill file")
	}
}

func TestSkillFileFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := SkillFileFromContext(ctx)
	if got != nil {
		t.Errorf("SkillFileFromContext() = %v, want nil", got)
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"12345678-1234-1234-1234-123456789012", true},
		{"12345678123412341234123456789012", true},     // Without dashes
		{"ABCDEFAB-1234-5678-9ABC-DEF012345678", true}, // Uppercase
		{"abcdefab-1234-5678-9abc-def012345678", true}, // Lowercase
		{"AbCdEfAb-1234-5678-9aBc-DeF012345678", true}, // Mixed case
		{"1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d", true}, // Typical Notion ID

		// Invalid - not UUIDs
		{"Meeting Notes", false},
		{"my-page-alias", false},
		{"12345", false},
		{"", false},
		{"123456781234123412341234567890", false},       // Too short
		{"123456781234123412341234567890123", false},    // Too long
		{"12345678-1234-1234-1234-12345678901g", false}, // Invalid char
		{"https://notion.so/page-12345678", false},      // URL
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeUUID(tt.input)
			if got != tt.expected {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractPlainTextFromRichText(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected string
	}{
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "",
		},
		{
			name: "plain_text field",
			input: []interface{}{
				map[string]interface{}{
					"plain_text": "Hello World",
				},
			},
			expected: "Hello World",
		},
		{
			name: "text.content field",
			input: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"content": "Hello World",
					},
				},
			},
			expected: "Hello World",
		},
		{
			name: "multiple parts",
			input: []interface{}{
				map[string]interface{}{"plain_text": "Hello "},
				map[string]interface{}{"plain_text": "World"},
			},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainTextFromRichText(tt.input)
			if got != tt.expected {
				t.Errorf("extractPlainTextFromRichText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractResultTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "empty result",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name: "page with title property",
			input: map[string]interface{}{
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type": "title",
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "My Page Title",
							},
						},
					},
				},
			},
			expected: "My Page Title",
		},
		{
			name: "database with title array",
			input: map[string]interface{}{
				"title": []interface{}{
					map[string]interface{}{
						"plain_text": "My Database",
					},
				},
			},
			expected: "My Database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractResultTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFilterExactTitleMatches(t *testing.T) {
	results := []map[string]interface{}{
		{
			"id": "id-1",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes"}},
				},
			},
		},
		{
			"id": "id-2",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes - Q1"}},
				},
			},
		},
		{
			"id": "id-3",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "meeting notes"}},
				},
			},
		},
	}

	// Should match "Meeting Notes" exactly (case-insensitive)
	matches := filterExactTitleMatches(results, "Meeting Notes")
	if len(matches) != 2 {
		t.Errorf("filterExactTitleMatches() returned %d matches, want 2", len(matches))
	}

	// Should not match partial title
	matches = filterExactTitleMatches(results, "Meeting")
	if len(matches) != 0 {
		t.Errorf("filterExactTitleMatches() returned %d matches for partial title, want 0", len(matches))
	}
}
