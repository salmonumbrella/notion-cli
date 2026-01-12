package cmd

import (
	"testing"
)

func TestParseMarkdownToBlocks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []string
	}{
		{
			name:          "simple paragraph",
			input:         "Hello world",
			expectedCount: 1,
			expectedTypes: []string{"paragraph"},
		},
		{
			name:          "heading 1",
			input:         "# Main Title",
			expectedCount: 1,
			expectedTypes: []string{"heading_1"},
		},
		{
			name:          "heading 2",
			input:         "## Section",
			expectedCount: 1,
			expectedTypes: []string{"heading_2"},
		},
		{
			name:          "heading 3",
			input:         "### Subsection",
			expectedCount: 1,
			expectedTypes: []string{"heading_3"},
		},
		{
			name:          "divider with dashes",
			input:         "---",
			expectedCount: 1,
			expectedTypes: []string{"divider"},
		},
		{
			name:          "divider with asterisks",
			input:         "***",
			expectedCount: 1,
			expectedTypes: []string{"divider"},
		},
		{
			name:          "bullet list",
			input:         "- First item\n- Second item",
			expectedCount: 2,
			expectedTypes: []string{"bulleted_list_item", "bulleted_list_item"},
		},
		{
			name:          "numbered list",
			input:         "1. First\n2. Second",
			expectedCount: 2,
			expectedTypes: []string{"numbered_list_item", "numbered_list_item"},
		},
		{
			name:          "todo unchecked",
			input:         "- [ ] Task",
			expectedCount: 1,
			expectedTypes: []string{"to_do"},
		},
		{
			name:          "todo checked",
			input:         "- [x] Done task",
			expectedCount: 1,
			expectedTypes: []string{"to_do"},
		},
		{
			name:          "quote",
			input:         "> This is a quote",
			expectedCount: 1,
			expectedTypes: []string{"quote"},
		},
		{
			name:          "code block",
			input:         "```go\nfunc main() {}\n```",
			expectedCount: 1,
			expectedTypes: []string{"code"},
		},
		{
			name:          "mixed content",
			input:         "# Title\n\nParagraph text.\n\n- Bullet 1\n- Bullet 2\n\n---\n\n> Quote",
			expectedCount: 6,
			expectedTypes: []string{"heading_1", "paragraph", "bulleted_list_item", "bulleted_list_item", "divider", "quote"},
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name:          "only whitespace",
			input:         "   \n\n   \n",
			expectedCount: 0,
			expectedTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := parseMarkdownToBlocks(tt.input)

			if len(blocks) != tt.expectedCount {
				t.Errorf("expected %d blocks, got %d", tt.expectedCount, len(blocks))
				for i, b := range blocks {
					t.Logf("block %d: %v", i, b["type"])
				}
				return
			}

			for i, block := range blocks {
				if block["type"] != tt.expectedTypes[i] {
					t.Errorf("block %d: expected type %q, got %q", i, tt.expectedTypes[i], block["type"])
				}
			}
		})
	}
}

func TestIsBlockStart(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"# Heading", true},
		{"## Heading", true},
		{"### Heading", true},
		{"---", true},
		{"***", true},
		{"___", true},
		{"- Bullet", true},
		{"* Bullet", true},
		{"+ Bullet", true},
		{"1. Numbered", true},
		{"23. Numbered", true},
		{"> Quote", true},
		{"```", true},
		{"```go", true},
		{"- [ ] Todo", true},
		{"- [x] Todo", true},
		{"Normal text", false},
		{"", false},
		{"    indented", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isBlockStart(tt.line)
			if result != tt.expected {
				t.Errorf("isBlockStart(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestCountBlockTypes(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "paragraph"},
		{"type": "paragraph"},
		{"type": "heading_1"},
		{"type": "bulleted_list_item"},
		{"type": "bulleted_list_item"},
		{"type": "bulleted_list_item"},
	}

	counts := countBlockTypes(blocks)

	if counts["paragraph"] != 2 {
		t.Errorf("expected 2 paragraphs, got %d", counts["paragraph"])
	}
	if counts["heading_1"] != 1 {
		t.Errorf("expected 1 heading_1, got %d", counts["heading_1"])
	}
	if counts["bulleted_list_item"] != 3 {
		t.Errorf("expected 3 bulleted_list_item, got %d", counts["bulleted_list_item"])
	}
}

func TestParseMarkdownToBlocks_MultilineQuote(t *testing.T) {
	input := "> First line\n> Second line\n> Third line"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Errorf("expected 1 quote block for multi-line quote, got %d", len(blocks))
		return
	}

	if blocks[0]["type"] != "quote" {
		t.Errorf("expected type 'quote', got %v", blocks[0]["type"])
	}
}

func TestParseMarkdownToBlocks_CodeBlockWithLanguage(t *testing.T) {
	input := "```python\ndef hello():\n    print('hi')\n```"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Errorf("expected 1 code block, got %d", len(blocks))
		return
	}

	if blocks[0]["type"] != "code" {
		t.Errorf("expected type 'code', got %v", blocks[0]["type"])
	}

	code := blocks[0]["code"].(map[string]interface{})
	if code["language"] != "python" {
		t.Errorf("expected language 'python', got %v", code["language"])
	}
}
