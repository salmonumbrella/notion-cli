package richtext

import (
	"strings"
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
		expected []MarkdownToken
	}{
		{
			name:     "plain text",
			text:     "Hello world",
			expected: []MarkdownToken{{Content: "Hello world"}},
		},
		{
			name:     "empty text",
			text:     "",
			expected: nil,
		},
		{
			name: "bold text with **",
			text: "This is **bold** text",
			expected: []MarkdownToken{
				{Content: "This is "},
				{Content: "bold", Bold: true},
				{Content: " text"},
			},
		},
		{
			name: "italic text with *",
			text: "This is *italic* text",
			expected: []MarkdownToken{
				{Content: "This is "},
				{Content: "italic", Italic: true},
				{Content: " text"},
			},
		},
		{
			name: "italic text with _",
			text: "This is _italic_ text",
			expected: []MarkdownToken{
				{Content: "This is "},
				{Content: "italic", Italic: true},
				{Content: " text"},
			},
		},
		{
			name: "code text",
			text: "This is `code` text",
			expected: []MarkdownToken{
				{Content: "This is "},
				{Content: "code", Code: true},
				{Content: " text"},
			},
		},
		{
			name: "bold and italic with ***",
			text: "This is ***bold italic*** text",
			expected: []MarkdownToken{
				{Content: "This is "},
				{Content: "bold italic", Bold: true, Italic: true},
				{Content: " text"},
			},
		},
		{
			name: "multiple formats",
			text: "**bold** and *italic* and `code`",
			expected: []MarkdownToken{
				{Content: "bold", Bold: true},
				{Content: " and "},
				{Content: "italic", Italic: true},
				{Content: " and "},
				{Content: "code", Code: true},
			},
		},
		{
			name:     "unmatched bold marker treated as literal",
			text:     "This has **unmatched bold",
			expected: []MarkdownToken{{Content: "This has **unmatched bold"}},
		},
		{
			name:     "unmatched italic marker treated as literal",
			text:     "This has *unmatched italic",
			expected: []MarkdownToken{{Content: "This has *unmatched italic"}},
		},
		{
			name:     "unmatched code marker treated as literal",
			text:     "This has `unmatched code",
			expected: []MarkdownToken{{Content: "This has `unmatched code"}},
		},
		{
			name: "bold at start",
			text: "**bold** at start",
			expected: []MarkdownToken{
				{Content: "bold", Bold: true},
				{Content: " at start"},
			},
		},
		{
			name: "bold at end",
			text: "at end **bold**",
			expected: []MarkdownToken{
				{Content: "at end "},
				{Content: "bold", Bold: true},
			},
		},
		// Link tests
		{
			name: "simple link",
			text: "Check [this](https://example.com)",
			expected: []MarkdownToken{
				{Content: "Check "},
				{Content: "this", IsLink: true, LinkURL: "https://example.com"},
			},
		},
		{
			name: "link with text around",
			text: "Before [link](url) after",
			expected: []MarkdownToken{
				{Content: "Before "},
				{Content: "link", IsLink: true, LinkURL: "url"},
				{Content: " after"},
			},
		},
		{
			name: "multiple links",
			text: "[one](url1) and [two](url2)",
			expected: []MarkdownToken{
				{Content: "one", IsLink: true, LinkURL: "url1"},
				{Content: " and "},
				{Content: "two", IsLink: true, LinkURL: "url2"},
			},
		},
		{
			name: "bold link",
			text: "**[bold link](url)**",
			expected: []MarkdownToken{
				{Content: "bold link", Bold: true, IsLink: true, LinkURL: "url"},
			},
		},
		{
			name: "italic link",
			text: "*[italic link](url)*",
			expected: []MarkdownToken{
				{Content: "italic link", Italic: true, IsLink: true, LinkURL: "url"},
			},
		},
		{
			name: "bold italic link",
			text: "***[important](url)***",
			expected: []MarkdownToken{
				{Content: "important", Bold: true, Italic: true, IsLink: true, LinkURL: "url"},
			},
		},
		{
			name:     "malformed link - missing close bracket",
			text:     "[broken link(url)",
			expected: []MarkdownToken{{Content: "[broken link(url)"}},
		},
		{
			name:     "malformed link - missing close paren",
			text:     "[broken](url",
			expected: []MarkdownToken{{Content: "[broken](url"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMarkdown(tt.text)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d\nexpected: %+v\ngot: %+v",
					len(tt.expected), len(result), tt.expected, result)
			}

			for i := range result {
				if result[i].Content != tt.expected[i].Content {
					t.Errorf("token %d: expected content %q, got %q", i, tt.expected[i].Content, result[i].Content)
				}
				if result[i].Bold != tt.expected[i].Bold {
					t.Errorf("token %d: expected bold=%v, got %v", i, tt.expected[i].Bold, result[i].Bold)
				}
				if result[i].Italic != tt.expected[i].Italic {
					t.Errorf("token %d: expected italic=%v, got %v", i, tt.expected[i].Italic, result[i].Italic)
				}
				if result[i].Code != tt.expected[i].Code {
					t.Errorf("token %d: expected code=%v, got %v", i, tt.expected[i].Code, result[i].Code)
				}
				if result[i].IsLink != tt.expected[i].IsLink {
					t.Errorf("token %d: expected isLink=%v, got %v", i, tt.expected[i].IsLink, result[i].IsLink)
				}
				if result[i].LinkURL != tt.expected[i].LinkURL {
					t.Errorf("token %d: expected linkURL=%q, got %q", i, tt.expected[i].LinkURL, result[i].LinkURL)
				}
			}
		})
	}
}

func TestBuildWithMentions(t *testing.T) {
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
			result := BuildWithMentions(tt.text, tt.userIDs)

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

func TestSummarizeTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []MarkdownToken
		expected MarkdownSummary
	}{
		{
			name:     "empty tokens",
			tokens:   nil,
			expected: MarkdownSummary{},
		},
		{
			name: "plain text only",
			tokens: []MarkdownToken{
				{Content: "Hello world"},
			},
			expected: MarkdownSummary{Plain: 1},
		},
		{
			name: "single bold",
			tokens: []MarkdownToken{
				{Content: "bold", Bold: true},
			},
			expected: MarkdownSummary{Bold: 1},
		},
		{
			name: "single italic",
			tokens: []MarkdownToken{
				{Content: "italic", Italic: true},
			},
			expected: MarkdownSummary{Italic: 1},
		},
		{
			name: "single code",
			tokens: []MarkdownToken{
				{Content: "code", Code: true},
			},
			expected: MarkdownSummary{Code: 1},
		},
		{
			name: "bold and italic combined",
			tokens: []MarkdownToken{
				{Content: "bold italic", Bold: true, Italic: true},
			},
			expected: MarkdownSummary{BoldItalic: 1},
		},
		{
			name: "mixed formatting",
			tokens: []MarkdownToken{
				{Content: "Hello "},
				{Content: "bold", Bold: true},
				{Content: " and "},
				{Content: "italic", Italic: true},
				{Content: " and "},
				{Content: "code", Code: true},
			},
			expected: MarkdownSummary{Plain: 3, Bold: 1, Italic: 1, Code: 1},
		},
		{
			name: "multiple of same type",
			tokens: []MarkdownToken{
				{Content: "bold1", Bold: true},
				{Content: " "},
				{Content: "bold2", Bold: true},
			},
			expected: MarkdownSummary{Plain: 1, Bold: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeTokens(tt.tokens)
			if result != tt.expected {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestFormatSummary(t *testing.T) {
	tests := []struct {
		name     string
		summary  MarkdownSummary
		expected string
	}{
		{
			name:     "no formatting",
			summary:  MarkdownSummary{Plain: 1},
			expected: "Parsed markdown: no formatting detected",
		},
		{
			name:     "empty summary",
			summary:  MarkdownSummary{},
			expected: "Parsed markdown: no formatting detected",
		},
		{
			name:     "bold only",
			summary:  MarkdownSummary{Bold: 2},
			expected: "Parsed markdown: 2 bold",
		},
		{
			name:     "italic only",
			summary:  MarkdownSummary{Italic: 1},
			expected: "Parsed markdown: 1 italic",
		},
		{
			name:     "code only",
			summary:  MarkdownSummary{Code: 3},
			expected: "Parsed markdown: 3 code",
		},
		{
			name:     "bold+italic only",
			summary:  MarkdownSummary{BoldItalic: 1},
			expected: "Parsed markdown: 1 bold+italic",
		},
		{
			name:     "multiple types",
			summary:  MarkdownSummary{Bold: 2, Italic: 1, Code: 1},
			expected: "Parsed markdown: 2 bold, 1 italic, 1 code",
		},
		{
			name:     "all types",
			summary:  MarkdownSummary{Bold: 1, Italic: 2, Code: 1, BoldItalic: 1, Plain: 3},
			expected: "Parsed markdown: 1 bold, 2 italic, 1 code, 1 bold+italic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSummary(tt.summary)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestVerboseOutputIntegration(t *testing.T) {
	// Test that ParseMarkdown -> SummarizeTokens -> FormatSummary
	// produces the expected verbose output for realistic inputs
	tests := []struct {
		name          string
		text          string
		expectedParts []string // substrings that should appear in output
		notExpected   []string // substrings that should NOT appear
	}{
		{
			name:          "plain text produces no formatting message",
			text:          "Hello world",
			expectedParts: []string{"no formatting detected"},
		},
		{
			name:          "bold and italic",
			text:          "This is **bold** and *italic*",
			expectedParts: []string{"1 bold", "1 italic"},
			notExpected:   []string{"no formatting"},
		},
		{
			name:          "multiple bold segments",
			text:          "**one** and **two** and **three**",
			expectedParts: []string{"3 bold"},
		},
		{
			name:          "code segment",
			text:          "Use `fmt.Println` here",
			expectedParts: []string{"1 code"},
		},
		{
			name:          "bold+italic combined",
			text:          "This is ***very important***",
			expectedParts: []string{"1 bold+italic"},
		},
		{
			name:          "link detected",
			text:          "Check [docs](https://example.com)",
			expectedParts: []string{"1 link"},
		},
		{
			name:          "bold link detected",
			text:          "**[important](url)**",
			expectedParts: []string{"1 link", "1 bold"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseMarkdown(tt.text)
			summary := SummarizeTokens(tokens)
			output := FormatSummary(summary)

			for _, part := range tt.expectedParts {
				if !strings.Contains(output, part) {
					t.Errorf("expected output to contain %q, got %q", part, output)
				}
			}
			for _, part := range tt.notExpected {
				if strings.Contains(output, part) {
					t.Errorf("expected output to NOT contain %q, got %q", part, output)
				}
			}
		})
	}
}

func TestBuildWithMentionsAndPages(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		userIDs  []string
		pageIDs  []string
		expected []notion.RichText
	}{
		{
			name:    "simple link",
			text:    "Check [docs](https://notion.so)",
			userIDs: nil,
			pageIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "Check "}},
				{Type: "text", Text: &notion.TextContent{Content: "docs", Link: &notion.Link{URL: "https://notion.so"}}},
			},
		},
		{
			name:    "bold link",
			text:    "**[important](url)**",
			userIDs: nil,
			pageIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "important", Link: &notion.Link{URL: "url"}}, Annotations: boldAnnotation()},
			},
		},
		{
			name:    "simple page mention",
			text:    "See @@ProjectPlan",
			userIDs: nil,
			pageIDs: []string{"project-plan-id"},
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "See "}},
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "project-plan-id"}}},
			},
		},
		{
			name:    "page mention without ID - kept as plain text",
			text:    "See @@ProjectPlan",
			userIDs: nil,
			pageIDs: nil,
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "See "}},
				{Type: "text", Text: &notion.TextContent{Content: "@@ProjectPlan"}},
			},
		},
		{
			name:    "mixed user and page mentions",
			text:    "@Alice see @@ProjectPlan",
			userIDs: []string{"alice-id"},
			pageIDs: []string{"project-plan-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " see "}},
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "project-plan-id"}}},
			},
		},
		{
			name:    "link and user mention",
			text:    "@Alice check [docs](https://example.com)",
			userIDs: []string{"alice-id"},
			pageIDs: nil,
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " check "}},
				{Type: "text", Text: &notion.TextContent{Content: "docs", Link: &notion.Link{URL: "https://example.com"}}},
			},
		},
		{
			name:    "multiple page mentions",
			text:    "@@PageOne and @@PageTwo",
			userIDs: nil,
			pageIDs: []string{"page-one-id", "page-two-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "page-one-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " and "}},
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "page-two-id"}}},
			},
		},
		{
			name:    "extra page IDs appended at end",
			text:    "@@OnePage only",
			userIDs: nil,
			pageIDs: []string{"one-page-id", "extra-page-id"},
			expected: []notion.RichText{
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "one-page-id"}}},
				{Type: "text", Text: &notion.TextContent{Content: " only"}},
				{Type: "mention", Mention: &notion.Mention{Type: "page", Page: &notion.PageMention{ID: "extra-page-id"}}},
			},
		},
		{
			name:    "mention inside link text - kept as literal",
			text:    "[See @Alice's doc](url)",
			userIDs: []string{"alice-id"},
			pageIDs: nil,
			// Link with literal text "@Alice's" inside, alice-id appended at end since no @mention found in non-link text
			expected: []notion.RichText{
				{Type: "text", Text: &notion.TextContent{Content: "See @Alice's doc", Link: &notion.Link{URL: "url"}}},
				{Type: "mention", Mention: &notion.Mention{Type: "user", User: &notion.UserMention{ID: "alice-id"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildWithMentionsAndPages(tt.text, tt.userIDs, tt.pageIDs)

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
					} else {
						if result[i].Text.Content != tt.expected[i].Text.Content {
							t.Errorf("element %d: expected text content %q, got %q",
								i, tt.expected[i].Text.Content, result[i].Text.Content)
						}
						// Check link
						if tt.expected[i].Text.Link != nil {
							if result[i].Text.Link == nil {
								t.Errorf("element %d: expected link, got nil", i)
							} else if result[i].Text.Link.URL != tt.expected[i].Text.Link.URL {
								t.Errorf("element %d: expected link URL %q, got %q",
									i, tt.expected[i].Text.Link.URL, result[i].Text.Link.URL)
							}
						} else if result[i].Text.Link != nil {
							t.Errorf("element %d: expected no link, got %+v", i, result[i].Text.Link)
						}
					}
				}

				if tt.expected[i].Mention != nil {
					if result[i].Mention == nil {
						t.Errorf("element %d: expected mention, got nil", i)
					} else {
						if tt.expected[i].Mention.User != nil {
							if result[i].Mention.User == nil {
								t.Errorf("element %d: expected user mention, got nil", i)
							} else if result[i].Mention.User.ID != tt.expected[i].Mention.User.ID {
								t.Errorf("element %d: expected user ID %q, got %q",
									i, tt.expected[i].Mention.User.ID, result[i].Mention.User.ID)
							}
						}
						if tt.expected[i].Mention.Page != nil {
							if result[i].Mention.Page == nil {
								t.Errorf("element %d: expected page mention, got nil", i)
							} else if result[i].Mention.Page.ID != tt.expected[i].Mention.Page.ID {
								t.Errorf("element %d: expected page ID %q, got %q",
									i, tt.expected[i].Mention.Page.ID, result[i].Mention.Page.ID)
							}
						}
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
					}
				}
			}
		})
	}
}

func TestCountUserMentionsOnly(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"plain text", 0},
		{"@user mention", 1},
		{"@one and @two", 2},
		{"@@page mention only", 0},
		{"@user and @@page", 1},
		{"@@Page-One and @@Page-Two", 0},     // only page mentions, no standalone user mentions
		{"@Alice and @@Bob and @Charlie", 2}, // @Alice and @Charlie are user mentions, @Bob is part of @@Bob
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := CountUserMentionsOnly(tt.text)
			if result != tt.expected {
				t.Errorf("CountUserMentionsOnly(%q) = %d, expected %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestCountPageMentions(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"plain text", 0},
		{"@user mention only", 0},
		{"@@page mention", 1},
		{"@@one and @@two", 2},
		{"@user and @@page", 1},
		{"@@Page-Name_123", 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := CountPageMentions(tt.text)
			if result != tt.expected {
				t.Errorf("CountPageMentions(%q) = %d, expected %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestFindPageMentions(t *testing.T) {
	tests := []struct {
		text     string
		expected []string
	}{
		{"", nil},
		{"plain text", nil},
		{"@@ProjectPlan", []string{"@@ProjectPlan"}},
		{"@@One and @@Two", []string{"@@One", "@@Two"}},
		{"@user and @@page", []string{"@@page"}},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := FindPageMentions(tt.text)
			if len(result) != len(tt.expected) {
				t.Errorf("FindPageMentions(%q) returned %d results, expected %d", tt.text, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("FindPageMentions(%q)[%d] = %q, expected %q", tt.text, i, v, tt.expected[i])
				}
			}
		})
	}
}
