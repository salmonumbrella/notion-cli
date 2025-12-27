package richtext

import (
	"testing"
)

func TestSanitizeForComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "inline code preserved",
			input:    "Use `fmt.Println` here",
			expected: "Use `fmt.Println` here",
		},
		{
			name:     "bold and italic preserved",
			input:    "This is **bold** and *italic*",
			expected: "This is **bold** and *italic*",
		},
		{
			name:     "single-line code block to inline code",
			input:    "Before\n```\nbrew install something\n```\nAfter",
			expected: "Before\n`brew install something`\nAfter",
		},
		{
			name:     "code block with language specifier",
			input:    "Before\n```bash\nbrew install something\n```\nAfter",
			expected: "Before\n`brew install something`\nAfter",
		},
		{
			name:     "multi-line code block",
			input:    "Install:\n```bash\nexport FOO=\"bar\"\nexport BAZ=\"qux\"\n```\nDone",
			expected: "Install:\n`export FOO=\"bar\"`\n`export BAZ=\"qux\"`\nDone",
		},
		{
			name:     "code block with empty lines skipped",
			input:    "Code:\n```\nline1\n\nline2\n```\nEnd",
			expected: "Code:\n`line1`\n`line2`\nEnd",
		},
		{
			name:     "code block line containing backticks left as plain text",
			input:    "Code:\n```bash\necho \"hello `world`\"\n```\nEnd",
			expected: "Code:\necho \"hello `world`\"\nEnd",
		},
		{
			name:     "multiple code blocks converted independently",
			input:    "First:\n```\ncmd1\n```\nMiddle\n```\ncmd2\n```\nEnd",
			expected: "First:\n`cmd1`\nMiddle\n`cmd2`\nEnd",
		},
		{
			name:     "empty code block removed",
			input:    "Before\n```\n\n```\nAfter",
			expected: "Before\n\nAfter",
		},
		{
			name:     "bold text around code blocks preserved",
			input:    "**How to install:**\n```bash\nbrew install foo\n```\n**Important:** done",
			expected: "**How to install:**\n`brew install foo`\n**Important:** done",
		},
		{
			name:     "mentions inside code blocks preserved",
			input:    "Hey @Isaac\n```\nnotion comment add --text \"test\"\n```\nDone",
			expected: "Hey @Isaac\n`notion comment add --text \"test\"`\nDone",
		},
		{
			name: "realistic agent comment with code blocks and bold",
			input: `@Isaac — The shopline-cli now has full admin API integration.

**How to install and test the CLI yourself:**

` + "```" + `
brew install salmonumbrella/tap/shopline-cli
` + "```" + `

Then create a ` + "`" + `.env` + "`" + ` file wherever you work:

` + "```" + `
export SHOPLINEADMINBASEURL="https://api.example.com"
export SHOPLINEADMINTOKEN="abc123"
` + "```" + `

**Important design note:** The CLI repo is public.`,
			expected: `@Isaac — The shopline-cli now has full admin API integration.

**How to install and test the CLI yourself:**

` + "`" + `brew install salmonumbrella/tap/shopline-cli` + "`" + `

Then create a ` + "`" + `.env` + "`" + ` file wherever you work:

` + "`" + `export SHOPLINEADMINBASEURL="https://api.example.com"` + "`" + `
` + "`" + `export SHOPLINEADMINTOKEN="abc123"` + "`" + `

**Important design note:** The CLI repo is public.`,
		},
		// Edge cases from code review
		{
			name:     "stray fence on its own line removed",
			input:    "Before\n```bash\nAfter",
			expected: "Before\n\nAfter",
		},
		{
			name:     "inline triple backticks in prose NOT stripped",
			input:    "run ```echo hello``` in terminal",
			expected: "run ```echo hello``` in terminal",
		},
		{
			name:     "triple backticks inside inline code preserved",
			input:    "Use ` ``` ` to start a code block",
			expected: "Use ` ``` ` to start a code block",
		},
		{
			name:     "four-backtick fence handled",
			input:    "Before\n````python\ncode here\n````\nAfter",
			expected: "Before\n`code here`\nAfter",
		},
		{
			name:     "unclosed fence line removed",
			input:    "Before\n```bash\ncode here\nmore code",
			expected: "Before\n\ncode here\nmore code",
		},
		{
			name:     "CRLF line endings normalized",
			input:    "Before\r\n```\r\ncmd\r\n```\r\nAfter",
			expected: "Before\n`cmd`\nAfter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForComments(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForComments():\n  input:    %q\n  expected: %q\n  got:      %q",
					tt.input, tt.expected, result)
			}
		})
	}
}

func TestSanitizeForCommentsThenParse(t *testing.T) {
	// Integration test: sanitize → ParseMarkdown should correctly detect bold/italic/code
	tests := []struct {
		name           string
		input          string
		expectedTokens []MarkdownToken
	}{
		{
			name:  "bold preserved after sanitizing code block",
			input: "**Important:**\n```\ncmd\n```\nDone",
			expectedTokens: []MarkdownToken{
				{Content: "Important:", Bold: true},
				{Content: "\n"},
				{Content: "cmd", Code: true},
				{Content: "\nDone"},
			},
		},
		{
			name:  "inline code from sanitized block parsed correctly",
			input: "Install:\n```bash\nbrew install foo\n```",
			expectedTokens: []MarkdownToken{
				{Content: "Install:\n"},
				{Content: "brew install foo", Code: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := SanitizeForComments(tt.input)
			tokens := ParseMarkdown(sanitized)

			if len(tokens) != len(tt.expectedTokens) {
				t.Fatalf("expected %d tokens, got %d\nsanitized: %q\ntokens: %+v",
					len(tt.expectedTokens), len(tokens), sanitized, tokens)
			}

			for i := range tokens {
				if tokens[i].Content != tt.expectedTokens[i].Content {
					t.Errorf("token %d: expected content %q, got %q", i, tt.expectedTokens[i].Content, tokens[i].Content)
				}
				if tokens[i].Bold != tt.expectedTokens[i].Bold {
					t.Errorf("token %d: expected bold=%v, got %v", i, tt.expectedTokens[i].Bold, tokens[i].Bold)
				}
				if tokens[i].Code != tt.expectedTokens[i].Code {
					t.Errorf("token %d: expected code=%v, got %v", i, tt.expectedTokens[i].Code, tokens[i].Code)
				}
			}
		})
	}
}
