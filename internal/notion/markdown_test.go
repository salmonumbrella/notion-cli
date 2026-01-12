package notion

import (
	"testing"
)

func TestParseInlineMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []markdownToken
	}{
		{
			name:  "plain text",
			input: "Hello world",
			expected: []markdownToken{
				{content: "Hello world"},
			},
		},
		{
			name:  "bold text",
			input: "Hello **world**",
			expected: []markdownToken{
				{content: "Hello "},
				{content: "world", bold: true},
			},
		},
		{
			name:  "italic text",
			input: "Hello *world*",
			expected: []markdownToken{
				{content: "Hello "},
				{content: "world", italic: true},
			},
		},
		{
			name:  "code text",
			input: "Use the `fmt.Println` function",
			expected: []markdownToken{
				{content: "Use the "},
				{content: "fmt.Println", code: true},
				{content: " function"},
			},
		},
		{
			name:  "mixed formatting",
			input: "This is **bold** and *italic* text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "bold", bold: true},
				{content: " and "},
				{content: "italic", italic: true},
				{content: " text"},
			},
		},
		{
			name:  "bold and italic",
			input: "This is ***bold and italic***",
			expected: []markdownToken{
				{content: "This is "},
				{content: "bold and italic", bold: true, italic: true},
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInlineMarkdown(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(result))
				t.Errorf("result: %+v", result)
				return
			}

			for i, token := range result {
				if token.content != tt.expected[i].content {
					t.Errorf("token %d: expected content %q, got %q", i, tt.expected[i].content, token.content)
				}
				if token.bold != tt.expected[i].bold {
					t.Errorf("token %d: expected bold=%v, got %v", i, tt.expected[i].bold, token.bold)
				}
				if token.italic != tt.expected[i].italic {
					t.Errorf("token %d: expected italic=%v, got %v", i, tt.expected[i].italic, token.italic)
				}
				if token.code != tt.expected[i].code {
					t.Errorf("token %d: expected code=%v, got %v", i, tt.expected[i].code, token.code)
				}
			}
		})
	}
}

func TestParseMarkdownToRichText(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"plain text", "Hello world"},
		{"bold text", "Hello **world**"},
		{"italic text", "Hello *world*"},
		{"code text", "Use `code` here"},
		{"mixed", "**Bold** and *italic* and `code`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMarkdownToRichText(tt.input)

			if len(result) == 0 && tt.input != "" {
				t.Error("expected non-empty result for non-empty input")
			}

			// Verify structure
			for _, rt := range result {
				if rt["type"] != "text" {
					t.Errorf("expected type 'text', got %v", rt["type"])
				}
				textContent, ok := rt["text"].(map[string]interface{})
				if !ok {
					t.Error("expected text to be map[string]interface{}")
				}
				if _, ok := textContent["content"]; !ok {
					t.Error("expected text.content to exist")
				}
			}
		})
	}
}

func TestNewParagraphWithMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"plain", "Hello world"},
		{"with bold", "Hello **world**"},
		{"with multiple formats", "**Bold** and *italic*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewParagraphWithMarkdown(tt.input)

			if result["type"] != "paragraph" {
				t.Errorf("expected type 'paragraph', got %v", result["type"])
			}

			paragraph, ok := result["paragraph"].(map[string]interface{})
			if !ok {
				t.Error("expected paragraph to be map[string]interface{}")
			}

			richText, ok := paragraph["rich_text"].([]map[string]interface{})
			if !ok {
				t.Error("expected rich_text to be []map[string]interface{}")
			}

			if len(richText) == 0 {
				t.Error("expected non-empty rich_text")
			}
		})
	}
}

func TestNewHeadingWithMarkdown(t *testing.T) {
	text := "**Bold** heading"

	// Test heading 1
	h1 := NewHeading1WithMarkdown(text)
	if h1["type"] != "heading_1" {
		t.Errorf("expected type 'heading_1', got %v", h1["type"])
	}

	// Test heading 2
	h2 := NewHeading2WithMarkdown(text)
	if h2["type"] != "heading_2" {
		t.Errorf("expected type 'heading_2', got %v", h2["type"])
	}

	// Test heading 3
	h3 := NewHeading3WithMarkdown(text)
	if h3["type"] != "heading_3" {
		t.Errorf("expected type 'heading_3', got %v", h3["type"])
	}
}

func TestNewListItemsWithMarkdown(t *testing.T) {
	text := "Item with **bold** text"

	// Test bullet
	bullet := NewBulletedListItemWithMarkdown(text)
	if bullet["type"] != "bulleted_list_item" {
		t.Errorf("expected type 'bulleted_list_item', got %v", bullet["type"])
	}

	// Test numbered
	numbered := NewNumberedListItemWithMarkdown(text)
	if numbered["type"] != "numbered_list_item" {
		t.Errorf("expected type 'numbered_list_item', got %v", numbered["type"])
	}
}

func TestNewToDoWithMarkdown(t *testing.T) {
	text := "Task with **bold** text"

	// Test unchecked
	unchecked := NewToDoWithMarkdown(text, false)
	if unchecked["type"] != "to_do" {
		t.Errorf("expected type 'to_do', got %v", unchecked["type"])
	}
	todo := unchecked["to_do"].(map[string]interface{})
	if todo["checked"] != false {
		t.Error("expected checked=false")
	}

	// Test checked
	checked := NewToDoWithMarkdown(text, true)
	todo = checked["to_do"].(map[string]interface{})
	if todo["checked"] != true {
		t.Error("expected checked=true")
	}
}
