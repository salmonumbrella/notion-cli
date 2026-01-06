// Package richtext provides utilities for building Notion rich text objects
// with markdown formatting and @mentions.
package richtext

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// MentionPattern matches @Name patterns in text (alphanumeric, hyphens, underscores)
var MentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]+)`)

// MarkdownToken represents a parsed markdown segment with its formatting
type MarkdownToken struct {
	Content string
	Bold    bool
	Italic  bool
	Code    bool
}

// ParseMarkdown parses text for markdown patterns and returns tokens.
//
// Supported patterns:
//   - **bold** or __bold__
//   - *italic* or _italic_
//   - `code`
//   - ***bold italic*** or ___bold italic___
//
// Limitations:
//   - Nested or overlapping formatting (e.g., "**bold *italic** text*") may not
//     parse as intended. The parser matches patterns left-to-right without nesting.
//   - Unmatched markers are treated as literal text (graceful degradation).
func ParseMarkdown(text string) []MarkdownToken {
	if text == "" {
		return nil
	}

	var tokens []MarkdownToken
	remaining := text

	for len(remaining) > 0 {
		// Find the earliest markdown pattern
		earliest := -1
		var matched string
		var tokenContent string
		var bold, italic, code bool

		// Check for code first (highest priority, doesn't nest)
		if idx := strings.Index(remaining, "`"); idx != -1 {
			endIdx := strings.Index(remaining[idx+1:], "`")
			if endIdx != -1 {
				if earliest == -1 || idx < earliest {
					earliest = idx
					tokenContent = remaining[idx+1 : idx+1+endIdx]
					matched = remaining[idx : idx+1+endIdx+1]
					code = true
					bold = false
					italic = false
				}
			}
		}

		// Check for bold+italic (***text*** or ___text___)
		for _, marker := range []string{"***", "___"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+3:], marker)
				if endIdx != -1 {
					earliest = idx
					tokenContent = remaining[idx+3 : idx+3+endIdx]
					matched = remaining[idx : idx+3+endIdx+3]
					bold = true
					italic = true
					code = false
				}
			}
		}

		// Check for bold (**text** or __text__)
		for _, marker := range []string{"**", "__"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+2:], marker)
				if endIdx != -1 {
					earliest = idx
					tokenContent = remaining[idx+2 : idx+2+endIdx]
					matched = remaining[idx : idx+2+endIdx+2]
					bold = true
					italic = false
					code = false
				}
			}
		}

		// Check for italic (*text* or _text_) - must not be ** or __
		for _, marker := range []string{"*", "_"} {
			doubleMarker := marker + marker
			idx := strings.Index(remaining, marker)
			// Skip if this is actually a double marker
			if idx != -1 && strings.HasPrefix(remaining[idx:], doubleMarker) {
				// Find next single marker that isn't part of a double
				searchFrom := idx + 1
				for searchFrom < len(remaining) {
					nextIdx := strings.Index(remaining[searchFrom:], marker)
					if nextIdx == -1 {
						idx = -1
						break
					}
					actualIdx := searchFrom + nextIdx
					// Check if this is part of a double marker
					if actualIdx > 0 && string(remaining[actualIdx-1]) == marker {
						searchFrom = actualIdx + 1
						continue
					}
					if actualIdx+1 < len(remaining) && string(remaining[actualIdx+1]) == marker {
						searchFrom = actualIdx + 2
						continue
					}
					idx = actualIdx
					break
				}
				if idx == -1 {
					continue
				}
			}
			if idx != -1 && (earliest == -1 || idx < earliest) {
				// Find closing marker that isn't part of a double
				searchEnd := idx + 1
				endIdx := -1
				for searchEnd < len(remaining) {
					nextEnd := strings.Index(remaining[searchEnd:], marker)
					if nextEnd == -1 {
						break
					}
					actualEnd := searchEnd + nextEnd
					// Check if this closing marker is part of a double
					if actualEnd+1 < len(remaining) && string(remaining[actualEnd+1]) == marker {
						searchEnd = actualEnd + 2
						continue
					}
					if actualEnd > 0 && string(remaining[actualEnd-1]) == marker {
						searchEnd = actualEnd + 1
						continue
					}
					endIdx = actualEnd - idx - 1
					break
				}
				if endIdx > 0 {
					earliest = idx
					tokenContent = remaining[idx+1 : idx+1+endIdx]
					matched = remaining[idx : idx+1+endIdx+1]
					bold = false
					italic = true
					code = false
				}
			}
		}

		if earliest == -1 {
			// No more markdown patterns, add remaining as plain text
			tokens = append(tokens, MarkdownToken{Content: remaining})
			break
		}

		// Add text before the pattern
		if earliest > 0 {
			tokens = append(tokens, MarkdownToken{Content: remaining[:earliest]})
		}

		// Add the formatted token
		tokens = append(tokens, MarkdownToken{
			Content: tokenContent,
			Bold:    bold,
			Italic:  italic,
			Code:    code,
		})

		remaining = remaining[earliest+len(matched):]
	}

	return tokens
}

// CreateAnnotations creates a Notion Annotations object from formatting flags.
// Returns nil if all formatting is default (allows omitempty to work).
func CreateAnnotations(bold, italic, code bool) *notion.Annotations {
	if !bold && !italic && !code {
		return nil
	}
	return &notion.Annotations{
		Bold:          bold,
		Italic:        italic,
		Strikethrough: false,
		Underline:     false,
		Code:          code,
		Color:         "default",
	}
}

// BuildWithMentions parses text for markdown formatting and @Name patterns,
// replacing them with properly formatted rich text and mention objects.
// Supports: **bold**, *italic*, _italic_, `code`, ***bold italic***, and @mentions.
// Returns the rich_text array with interleaved text and mention objects.
func BuildWithMentions(text string, userIDs []string) []notion.RichText {
	if text == "" && len(userIDs) == 0 {
		return []notion.RichText{}
	}

	tokens := ParseMarkdown(text)
	return BuildWithMentionsFromTokens(tokens, userIDs)
}

// BuildWithMentionsFromTokens builds rich text from pre-parsed markdown tokens.
// Use this when you've already parsed markdown (e.g., for verbose output) to avoid
// parsing twice. The tokens should come from ParseMarkdown.
func BuildWithMentionsFromTokens(tokens []MarkdownToken, userIDs []string) []notion.RichText {
	if len(tokens) == 0 && len(userIDs) == 0 {
		return []notion.RichText{}
	}

	// Process each token, looking for @mentions within them
	var richText []notion.RichText
	userIDIndex := 0

	for _, token := range tokens {
		// Check for @mentions within this token's content
		matches := MentionPattern.FindAllStringIndex(token.Content, -1)

		if len(matches) == 0 {
			// No mentions in this token, add it directly with its formatting
			if token.Content != "" {
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.Content},
					Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
				})
			}
			continue
		}

		// Process mentions within this formatted token
		lastEnd := 0
		for _, match := range matches {
			start, end := match[0], match[1]

			// Add text before this mention (with the token's formatting)
			if start > lastEnd {
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.Content[lastEnd:start]},
					Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
				})
			}

			// Add mention if we have a user ID for it
			if userIDIndex < len(userIDs) {
				richText = append(richText, notion.RichText{
					Type: "mention",
					Mention: &notion.Mention{
						Type: "user",
						User: &notion.UserMention{ID: userIDs[userIDIndex]},
					},
				})
				userIDIndex++
			} else {
				// No more user IDs, keep the @Name as plain text with formatting
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.Content[start:end]},
					Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
				})
			}

			lastEnd = end
		}

		// Add remaining text after the last mention (with formatting)
		if lastEnd < len(token.Content) {
			richText = append(richText, notion.RichText{
				Type:        "text",
				Text:        &notion.TextContent{Content: token.Content[lastEnd:]},
				Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
			})
		}
	}

	// If there are extra user IDs (no matching @Name patterns), append them at the end
	for ; userIDIndex < len(userIDs); userIDIndex++ {
		richText = append(richText, notion.RichText{
			Type: "mention",
			Mention: &notion.Mention{
				Type: "user",
				User: &notion.UserMention{ID: userIDs[userIDIndex]},
			},
		})
	}

	return richText
}

// MarkdownSummary holds counts of detected markdown patterns
type MarkdownSummary struct {
	Bold       int
	Italic     int
	Code       int
	BoldItalic int
	Plain      int
}

// SummarizeTokens counts markdown patterns in parsed tokens
func SummarizeTokens(tokens []MarkdownToken) MarkdownSummary {
	var summary MarkdownSummary
	for _, token := range tokens {
		switch {
		case token.Bold && token.Italic:
			summary.BoldItalic++
		case token.Bold:
			summary.Bold++
		case token.Italic:
			summary.Italic++
		case token.Code:
			summary.Code++
		default:
			summary.Plain++
		}
	}
	return summary
}

// FormatSummary returns a human-readable summary of parsed markdown
func FormatSummary(summary MarkdownSummary) string {
	var parts []string
	if summary.Bold > 0 {
		parts = append(parts, formatCount(summary.Bold, "bold"))
	}
	if summary.Italic > 0 {
		parts = append(parts, formatCount(summary.Italic, "italic"))
	}
	if summary.Code > 0 {
		parts = append(parts, formatCount(summary.Code, "code"))
	}
	if summary.BoldItalic > 0 {
		parts = append(parts, formatCount(summary.BoldItalic, "bold+italic"))
	}
	if len(parts) == 0 {
		return "Parsed markdown: no formatting detected"
	}
	return "Parsed markdown: " + strings.Join(parts, ", ")
}

func formatCount(count int, label string) string {
	return fmt.Sprintf("%d %s", count, label)
}
