package cmd

import (
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// Helper to create annotations for tests
func boldAnnotation() *notion.Annotations {
	return &notion.Annotations{Bold: true, Color: "default"}
}

func italicAnnotation() *notion.Annotations {
	return &notion.Annotations{Italic: true, Color: "default"}
}

func codeAnnotation() *notion.Annotations {
	return &notion.Annotations{Code: true, Color: "default"}
}

func boldItalicAnnotation() *notion.Annotations {
	return &notion.Annotations{Bold: true, Italic: true, Color: "default"}
}

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []markdownToken
	}{
		{
			name:     "plain text",
			text:     "Hello world",
			expected: []markdownToken{{content: "Hello world"}},
		},
		{
			name:     "empty text",
			text:     "",
			expected: nil,
		},
		{
			name: "bold text with **",
			text: "This is **bold** text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "bold", bold: true},
				{content: " text"},
			},
		},
		{
			name: "italic text with *",
			text: "This is *italic* text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "italic", italic: true},
				{content: " text"},
			},
		},
		{
			name: "italic text with _",
			text: "This is _italic_ text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "italic", italic: true},
				{content: " text"},
			},
		},
		{
			name: "code text",
			text: "This is `code` text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "code", code: true},
				{content: " text"},
			},
		},
		{
			name: "bold and italic with ***",
			text: "This is ***bold italic*** text",
			expected: []markdownToken{
				{content: "This is "},
				{content: "bold italic", bold: true, italic: true},
				{content: " text"},
			},
		},
		{
			name: "multiple formats",
			text: "**bold** and *italic* and `code`",
			expected: []markdownToken{
				{content: "bold", bold: true},
				{content: " and "},
				{content: "italic", italic: true},
				{content: " and "},
				{content: "code", code: true},
			},
		},
		{
			name:     "unmatched bold marker treated as literal",
			text:     "This has **unmatched bold",
			expected: []markdownToken{{content: "This has **unmatched bold"}},
		},
		{
			name:     "unmatched italic marker treated as literal",
			text:     "This has *unmatched italic",
			expected: []markdownToken{{content: "This has *unmatched italic"}},
		},
		{
			name:     "unmatched code marker treated as literal",
			text:     "This has `unmatched code",
			expected: []markdownToken{{content: "This has `unmatched code"}},
		},
		{
			name: "bold at start",
			text: "**bold** at start",
			expected: []markdownToken{
				{content: "bold", bold: true},
				{content: " at start"},
			},
		},
		{
			name: "bold at end",
			text: "at end **bold**",
			expected: []markdownToken{
				{content: "at end "},
				{content: "bold", bold: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMarkdown(tt.text)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d\nexpected: %+v\ngot: %+v",
					len(tt.expected), len(result), tt.expected, result)
			}

			for i := range result {
				if result[i].content != tt.expected[i].content {
					t.Errorf("token %d: expected content %q, got %q", i, tt.expected[i].content, result[i].content)
				}
				if result[i].bold != tt.expected[i].bold {
					t.Errorf("token %d: expected bold=%v, got %v", i, tt.expected[i].bold, result[i].bold)
				}
				if result[i].italic != tt.expected[i].italic {
					t.Errorf("token %d: expected italic=%v, got %v", i, tt.expected[i].italic, result[i].italic)
				}
				if result[i].code != tt.expected[i].code {
					t.Errorf("token %d: expected code=%v, got %v", i, tt.expected[i].code, result[i].code)
				}
			}
		})
	}
}

func TestBuildRichTextWithMentions(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		userIDs  []string
		expected []notion.RichText
	}{
		{
			name:    "plain text without mentions",
			text:    "Hello world",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Hello world"}},
			},
		},
		{
			name:     "empty text and no mentions",
			text:     "",
			userIDs:  nil,
			expected: []notion.RichText{},
		},
		{
			name:    "text with @Name and matching user ID",
			text:    "Hey @Georges, can you review?",
			userIDs: []string{"georges-user-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Hey "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "georges-user-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: ", can you review?"}},
			},
		},
		{
			name:    "text with multiple @Names and matching user IDs",
			text:    "@Alice and @Bob please review",
			userIDs: []string{"alice-id", "bob-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "bob-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " please review"}},
			},
		},
		{
			name:    "more @Names than user IDs - extras kept as plain text",
			text:    "@Alice and @Bob and @Charlie",
			userIDs: []string{"alice-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "text", Text: &notion.TextContent{Content: "@Bob"}},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "text", Text: &notion.TextContent{Content: "@Charlie"}},
			},
		},
		{
			name:    "more user IDs than @Names - extras appended at end",
			text:    "Hey @Alice",
			userIDs: []string{"alice-id", "bob-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Hey "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "bob-id"}}},
			},
		},
		{
			name:    "user IDs without @Names in text - legacy behavior appends at end",
			text:    "Please review this",
			userIDs: []string{"user-id-1", "user-id-2"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Please review this"}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "user-id-1"}}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "user-id-2"}}},
			},
		},
		{
			name:    "@Name with hyphen and underscore",
			text:    "Hi @User-Name_123",
			userIDs: []string{"user-123-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Hi "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "user-123-id"}}},
			},
		},
		{
			name:    "@Name at end of text",
			text:    "Thanks @Georges",
			userIDs: []string{"georges-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Thanks "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "georges-id"}}},
			},
		},
		{
			name:    "@Name at start of text",
			text:    "@Georges please look",
			userIDs: []string{"georges-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "georges-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " please look"}},
			},
		},
		// Markdown formatting tests
		{
			name:    "bold text",
			text:    "This is **bold** text",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This is "}},
				{Type: "text", Text: &notion.TextContent{Content: "bold"}, Annotations: boldAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " text"}},
			},
		},
		{
			name:    "italic text with asterisk",
			text:    "This is *italic* text",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This is "}},
				{Type: "text", Text: &notion.TextContent{Content: "italic"}, Annotations: italicAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " text"}},
			},
		},
		{
			name:    "italic text with underscore",
			text:    "This is _italic_ text",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This is "}},
				{Type: "text", Text: &notion.TextContent{Content: "italic"}, Annotations: italicAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " text"}},
			},
		},
		{
			name:    "code text",
			text:    "This is `code` text",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This is "}},
				{Type: "text", Text: &notion.TextContent{Content: "code"}, Annotations: codeAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " text"}},
			},
		},
		{
			name:    "bold and italic combined",
			text:    "This is ***bold italic*** text",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This is "}},
				{Type: "text", Text: &notion.TextContent{Content: "bold italic"}, Annotations: boldItalicAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " text"}},
			},
		},
		{
			name:    "multiple markdown formats",
			text:    "**bold** and *italic* and `code`",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "bold"}, Annotations: boldAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "text", Text: &notion.TextContent{Content: "italic"}, Annotations: italicAnnotation()},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "text", Text: &notion.TextContent{Content: "code"}, Annotations: codeAnnotation()},
			},
		},
		{
			name:    "markdown with mention",
			text:    "Hey **@Georges**, check this",
			userIDs: []string{"georges-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Hey "}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "georges-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: ", check this"}},
			},
		},
		{
			name:    "unmatched bold markers treated as literal",
			text:    "This has **unmatched bold",
			userIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "This has **unmatched bold"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRichTextWithMentions(tt.text, tt.userIDs)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d rich text elements, got %d\nexpected: %+v\ngot: %+v",
					len(tt.expected), len(result), tt.expected, result)
			}

			for i := range result {
				if result[i].Type != tt.expected[i].Type {
					t.Errorf("element %d: expected type %q, got %q", i, tt.expected[i].Type, result[i].Type)
				}

				if tt.expected[i].Text != nil {
					if result[i].Text == nil {
						t.Errorf("element %d: expected text content, got nil", i)
					} else if result[i].Text.Content != tt.expected[i].Text.Content {
						t.Errorf("element %d: expected text content %q, got %q",
							i, tt.expected[i].Text.Content, result[i].Text.Content)
					}
				}

				if tt.expected[i].Mention != nil {
					if result[i].Mention == nil {
						t.Errorf("element %d: expected mention, got nil", i)
					} else if result[i].Mention.User == nil {
						t.Errorf("element %d: expected user mention, got nil", i)
					} else if result[i].Mention.User.ID != tt.expected[i].Mention.User.ID {
						t.Errorf("element %d: expected user ID %q, got %q",
							i, tt.expected[i].Mention.User.ID, result[i].Mention.User.ID)
					}
				}

				// Check annotations
				if tt.expected[i].Annotations != nil {
					if result[i].Annotations == nil {
						t.Errorf("element %d: expected annotations, got nil", i)
					} else {
						if result[i].Annotations.Bold != tt.expected[i].Annotations.Bold {
							t.Errorf("element %d: expected bold=%v, got %v",
								i, tt.expected[i].Annotations.Bold, result[i].Annotations.Bold)
						}
						if result[i].Annotations.Italic != tt.expected[i].Annotations.Italic {
							t.Errorf("element %d: expected italic=%v, got %v",
								i, tt.expected[i].Annotations.Italic, result[i].Annotations.Italic)
						}
						if result[i].Annotations.Code != tt.expected[i].Annotations.Code {
							t.Errorf("element %d: expected code=%v, got %v",
								i, tt.expected[i].Annotations.Code, result[i].Annotations.Code)
						}
					}
				} else if result[i].Annotations != nil {
					t.Errorf("element %d: expected no annotations, got %+v", i, result[i].Annotations)
				}
			}
		})
	}
}
