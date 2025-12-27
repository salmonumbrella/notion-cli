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
		{"| A | B |", true},
		{"| --- | --- |", true},
		{"| single cell |", true},
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

func TestParseMarkdownToBlocks_SimpleTable(t *testing.T) {
	input := "| Name | Role |\n| --- | --- |\n| Alice | Engineer |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 table block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "table" {
		t.Errorf("expected type 'table', got %v", blocks[0]["type"])
	}

	tbl := blocks[0]["table"].(map[string]interface{})
	if tbl["table_width"] != 2 {
		t.Errorf("expected table_width 2, got %v", tbl["table_width"])
	}
	if tbl["has_column_header"] != true {
		t.Errorf("expected has_column_header true")
	}

	children := tbl["children"].([]map[string]interface{})
	if len(children) != 2 {
		t.Errorf("expected 2 rows (header + 1 data), got %d", len(children))
	}
}

func TestParseMarkdownToBlocks_TableWithSurroundingContent(t *testing.T) {
	input := "# Title\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\nParagraph after."
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks (heading, table, paragraph), got %d", len(blocks))
	}

	expectedTypes := []string{"heading_1", "table", "paragraph"}
	for i, b := range blocks {
		if b["type"] != expectedTypes[i] {
			t.Errorf("block %d: expected %q, got %q", i, expectedTypes[i], b["type"])
		}
	}
}

func TestParseMarkdownToBlocks_TableInlineFormatting(t *testing.T) {
	input := "| Name | Status |\n| --- | --- |\n| **Alice** | `active` |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	tbl := blocks[0]["table"].(map[string]interface{})
	children := tbl["children"].([]map[string]interface{})

	// Check data row (index 1) has inline formatting
	dataRow := children[1]["table_row"].(map[string]interface{})
	cells := dataRow["cells"].([][]map[string]interface{})

	// First cell: **Alice** should have bold annotation
	if cells[0][0]["text"].(map[string]interface{})["content"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", cells[0][0]["text"])
	}
	ann := cells[0][0]["annotations"].(map[string]interface{})
	if ann["bold"] != true {
		t.Error("expected bold on 'Alice'")
	}

	// Second cell: `active` should have code annotation
	if cells[1][0]["text"].(map[string]interface{})["content"] != "active" {
		t.Errorf("expected 'active', got %v", cells[1][0]["text"])
	}
	ann2 := cells[1][0]["annotations"].(map[string]interface{})
	if ann2["code"] != true {
		t.Error("expected code on 'active'")
	}
}

func TestParseMarkdownToBlocks_TableMultipleDataRows(t *testing.T) {
	input := "| Endpoint | Method | Status |\n| --- | --- | --- |\n| /api/orders | GET | PASS |\n| /api/users | POST | FAIL |\n| /api/items | GET | PASS |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 table block, got %d", len(blocks))
	}

	tbl := blocks[0]["table"].(map[string]interface{})
	if tbl["table_width"] != 3 {
		t.Errorf("expected table_width 3, got %v", tbl["table_width"])
	}

	children := tbl["children"].([]map[string]interface{})
	if len(children) != 4 {
		t.Errorf("expected 4 rows (1 header + 3 data), got %d", len(children))
	}
}

func TestParseMarkdownToBlocks_TableEmptyCells(t *testing.T) {
	input := "| A | B | C |\n| --- | --- | --- |\n|  | data |  |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	tbl := blocks[0]["table"].(map[string]interface{})
	children := tbl["children"].([]map[string]interface{})
	dataRow := children[1]["table_row"].(map[string]interface{})
	cells := dataRow["cells"].([][]map[string]interface{})
	if len(cells) != 3 {
		t.Errorf("expected 3 cells, got %d", len(cells))
	}
}

func TestParseMarkdownToBlocks_TableAlignmentMarkers(t *testing.T) {
	// Alignment colons in separator should be recognized and ignored
	input := "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 table block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "table" {
		t.Errorf("expected type 'table', got %v", blocks[0]["type"])
	}
}

func TestParseMarkdownToBlocks_PipeInParagraph(t *testing.T) {
	// A single pipe line that doesn't end with | should NOT be a table
	input := "This has a | pipe in it"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "paragraph" {
		t.Errorf("expected paragraph, got %v", blocks[0]["type"])
	}
}

func TestParseMarkdownToBlocks_TableHeaderOnly(t *testing.T) {
	// Header + separator with no data rows is still a valid table
	input := "| Name | Role |\n| --- | --- |"
	blocks := parseMarkdownToBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "table" {
		t.Errorf("expected table, got %v", blocks[0]["type"])
	}

	tbl := blocks[0]["table"].(map[string]interface{})
	children := tbl["children"].([]map[string]interface{})
	if len(children) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(children))
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
