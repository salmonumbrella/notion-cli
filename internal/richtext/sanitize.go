package richtext

import (
	"regexp"
	"strings"
)

// fencedCodeBlockPattern matches fenced code blocks: ```[lang]\n...\n```
// Uses `{3,} to also handle extended fences (````, ```“, etc.).
// The (?s) flag makes . match newlines; (.*?) is non-greedy.
var fencedCodeBlockPattern = regexp.MustCompile("(?s)`{3,}[^\n]*\n(.*?\n)`{3,}")

// strayFenceLine matches a ``` fence sitting on its own line (with optional language tag).
// Only matches at line-start to avoid stripping ``` inside inline code or prose.
var strayFenceLine = regexp.MustCompile("(?m)^[ \t]*`{3,}[a-zA-Z]*[ \t]*$")

// SanitizeForComments converts block-level markdown to comment-safe inline formatting.
// Notion comments only support inline formatting (bold, italic, inline code, links, mentions).
//
// This function:
//   - Converts fenced code blocks (```...```) to inline `code` spans (one per non-empty line)
//   - Lines containing backticks are kept as plain text to avoid nesting issues
//   - Removes stray triple-backtick fence lines that don't form complete blocks
//
// Limitations:
//   - Fenced blocks where the closing ``` is on the same line as content won't match.
//   - Content inside inline `code` spans is never modified.
func SanitizeForComments(text string) string {
	// Normalize CRLF to LF so regex line boundaries work consistently.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// First pass: convert complete fenced code blocks to inline code
	result := fencedCodeBlockPattern.ReplaceAllStringFunc(text, func(match string) string {
		lines := strings.Split(match, "\n")
		if len(lines) < 3 {
			return ""
		}

		// Remove first line (```lang) and last line (```)
		contentLines := lines[1 : len(lines)-1]

		var out []string
		for _, line := range contentLines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.Contains(trimmed, "`") {
				// Can't wrap in backticks — leave as plain text
				out = append(out, trimmed)
			} else {
				out = append(out, "`"+trimmed+"`")
			}
		}
		if len(out) == 0 {
			return ""
		}
		return strings.Join(out, "\n")
	})

	// Second pass: remove stray fence lines (unclosed blocks, etc.)
	// Only matches ``` at the start of a line to avoid corrupting inline content.
	result = strayFenceLine.ReplaceAllString(result, "")

	return result
}
