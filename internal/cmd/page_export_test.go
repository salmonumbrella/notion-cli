package cmd

import "testing"

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
