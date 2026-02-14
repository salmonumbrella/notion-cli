package cmd

import (
	"strings"
	"testing"
)

func TestRichTextToMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		items []interface{}
		want  string
	}{
		{
			name: "plain text only",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "Hello world",
					"text":       map[string]interface{}{"content": "Hello world"},
				},
			},
			want: "Hello world",
		},
		{
			name: "bold text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "bold",
					"text":       map[string]interface{}{"content": "bold"},
					"annotations": map[string]interface{}{
						"bold": true, "italic": false, "code": false,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
			},
			want: "**bold**",
		},
		{
			name: "italic text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "italic",
					"text":       map[string]interface{}{"content": "italic"},
					"annotations": map[string]interface{}{
						"bold": false, "italic": true, "code": false,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
			},
			want: "*italic*",
		},
		{
			name: "code text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "code",
					"text":       map[string]interface{}{"content": "code"},
					"annotations": map[string]interface{}{
						"bold": false, "italic": false, "code": true,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
			},
			want: "`code`",
		},
		{
			name: "bold italic text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "both",
					"text":       map[string]interface{}{"content": "both"},
					"annotations": map[string]interface{}{
						"bold": true, "italic": true, "code": false,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
			},
			want: "***both***",
		},
		{
			name: "strikethrough text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "deleted",
					"text":       map[string]interface{}{"content": "deleted"},
					"annotations": map[string]interface{}{
						"bold": false, "italic": false, "code": false,
						"strikethrough": true, "underline": false, "color": "default",
					},
				},
			},
			want: "~~deleted~~",
		},
		{
			name: "link with href",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "click here",
					"text": map[string]interface{}{
						"content": "click here",
						"link":    map[string]interface{}{"url": "https://example.com"},
					},
					"href": "https://example.com",
				},
			},
			want: "[click here](https://example.com)",
		},
		{
			name: "bold link",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "important",
					"text": map[string]interface{}{
						"content": "important",
						"link":    map[string]interface{}{"url": "https://example.com"},
					},
					"href": "https://example.com",
					"annotations": map[string]interface{}{
						"bold": true, "italic": false, "code": false,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
			},
			want: "**[important](https://example.com)**",
		},
		{
			name: "mixed inline segments",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "Hello ",
					"text":       map[string]interface{}{"content": "Hello "},
				},
				map[string]interface{}{
					"type":       "text",
					"plain_text": "world",
					"text":       map[string]interface{}{"content": "world"},
					"annotations": map[string]interface{}{
						"bold": true, "italic": false, "code": false,
						"strikethrough": false, "underline": false, "color": "default",
					},
				},
				map[string]interface{}{
					"type":       "text",
					"plain_text": " and ",
					"text":       map[string]interface{}{"content": " and "},
				},
				map[string]interface{}{
					"type":       "text",
					"plain_text": "link",
					"text": map[string]interface{}{
						"content": "link",
						"link":    map[string]interface{}{"url": "https://x.com"},
					},
					"href": "https://x.com",
				},
			},
			want: "Hello **world** and [link](https://x.com)",
		},
		{
			name:  "nil items",
			items: nil,
			want:  "",
		},
		{
			name:  "empty items",
			items: []interface{}{},
			want:  "",
		},
		{
			name: "no annotations key means plain text",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "plain",
					"text":       map[string]interface{}{"content": "plain"},
				},
			},
			want: "plain",
		},
		{
			name: "link via text.link fallback when href absent",
			items: []interface{}{
				map[string]interface{}{
					"type":       "text",
					"plain_text": "docs",
					"text": map[string]interface{}{
						"content": "docs",
						"link":    map[string]interface{}{"url": "https://example.com/docs"},
					},
				},
			},
			want: "[docs](https://example.com/docs)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := richTextToMarkdown(tt.items)
			if got != tt.want {
				t.Errorf("richTextToMarkdown:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockMarkdown_Table(t *testing.T) {
	// A 2-column, 3-row table with header row
	table := exportBlock{
		Type: "table",
		Content: map[string]interface{}{
			"table_width":       float64(2),
			"has_column_header": true,
			"has_row_header":    false,
		},
		Children: []exportBlock{
			{
				Type: "table_row",
				Content: map[string]interface{}{
					"cells": []interface{}{
						[]interface{}{map[string]interface{}{"plain_text": "Name", "text": map[string]interface{}{"content": "Name"}}},
						[]interface{}{map[string]interface{}{"plain_text": "Role", "text": map[string]interface{}{"content": "Role"}}},
					},
				},
			},
			{
				Type: "table_row",
				Content: map[string]interface{}{
					"cells": []interface{}{
						[]interface{}{map[string]interface{}{"plain_text": "Alice", "text": map[string]interface{}{"content": "Alice"}}},
						[]interface{}{map[string]interface{}{"plain_text": "Engineer", "text": map[string]interface{}{"content": "Engineer"}}},
					},
				},
			},
			{
				Type: "table_row",
				Content: map[string]interface{}{
					"cells": []interface{}{
						[]interface{}{map[string]interface{}{
							"plain_text": "Bob",
							"text":       map[string]interface{}{"content": "Bob"},
							"annotations": map[string]interface{}{
								"bold": true, "italic": false, "code": false,
								"strikethrough": false, "underline": false, "color": "default",
							},
						}},
						[]interface{}{map[string]interface{}{"plain_text": "Designer", "text": map[string]interface{}{"content": "Designer"}}},
					},
				},
			},
		},
	}

	lines := renderBlockMarkdown(table, 0)
	got := strings.Join(lines, "\n")
	want := "| Name | Role |\n| --- | --- |\n| Alice | Engineer |\n| **Bob** | Designer |"
	if got != want {
		t.Errorf("table render:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderBlockMarkdown_TableNoHeader(t *testing.T) {
	table := exportBlock{
		Type: "table",
		Content: map[string]interface{}{
			"table_width":       float64(2),
			"has_column_header": false,
			"has_row_header":    false,
		},
		Children: []exportBlock{
			{
				Type: "table_row",
				Content: map[string]interface{}{
					"cells": []interface{}{
						[]interface{}{map[string]interface{}{"plain_text": "A", "text": map[string]interface{}{"content": "A"}}},
						[]interface{}{map[string]interface{}{"plain_text": "B", "text": map[string]interface{}{"content": "B"}}},
					},
				},
			},
			{
				Type: "table_row",
				Content: map[string]interface{}{
					"cells": []interface{}{
						[]interface{}{map[string]interface{}{"plain_text": "C", "text": map[string]interface{}{"content": "C"}}},
						[]interface{}{map[string]interface{}{"plain_text": "D", "text": map[string]interface{}{"content": "D"}}},
					},
				},
			},
		},
	}

	lines := renderBlockMarkdown(table, 0)
	got := strings.Join(lines, "\n")
	want := "| A | B |\n| --- | --- |\n| C | D |"
	if got != want {
		t.Errorf("table no-header render:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderBlockMarkdown_QuoteWithChildren(t *testing.T) {
	block := exportBlock{
		Type: "quote",
		Content: map[string]interface{}{
			"rich_text": []interface{}{
				map[string]interface{}{"plain_text": "Main quote", "text": map[string]interface{}{"content": "Main quote"}},
			},
		},
		Children: []exportBlock{
			{
				Type: "paragraph",
				Content: map[string]interface{}{
					"rich_text": []interface{}{
						map[string]interface{}{"plain_text": "Nested paragraph", "text": map[string]interface{}{"content": "Nested paragraph"}},
					},
				},
			},
		},
	}

	lines := renderBlockMarkdown(block, 0)
	got := strings.Join(lines, "\n")
	want := "> Main quote\n> Nested paragraph"
	if got != want {
		t.Errorf("quote with children:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderBlockMarkdown_TableEmpty(t *testing.T) {
	table := exportBlock{
		Type: "table",
		Content: map[string]interface{}{
			"table_width":       float64(2),
			"has_column_header": false,
			"has_row_header":    false,
		},
		Children: []exportBlock{},
	}

	lines := renderBlockMarkdown(table, 0)
	if len(lines) != 0 {
		t.Errorf("empty table should produce no lines, got %d", len(lines))
	}
}
