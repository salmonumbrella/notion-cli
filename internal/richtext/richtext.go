// Package richtext provides utilities for building Notion rich text objects
// with markdown formatting and @mentions.
package richtext

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// mentionPattern matches @Name patterns in text (alphanumeric, hyphens, underscores)
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]+)`)

// pageMentionPattern matches @@Name patterns for page mentions
var pageMentionPattern = regexp.MustCompile(`@@([A-Za-z0-9_-]+)`)

// linkPattern matches markdown links [text](url)
var linkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// CountMentions returns the number of @Name patterns found in the text.
func CountMentions(text string) int {
	return len(mentionPattern.FindAllStringIndex(text, -1))
}

// CountUserMentionsOnly returns the number of @Name patterns that are NOT part of @@Name patterns.
// This provides an accurate count of user mentions by filtering out the @Name portion of page mentions.
func CountUserMentionsOnly(text string) int {
	userMatches := mentionPattern.FindAllStringIndex(text, -1)
	pageMatches := pageMentionPattern.FindAllStringIndex(text, -1)
	filtered := filterUserMentionsFromPageMentions(userMatches, pageMatches)
	return len(filtered)
}

// FindMentions returns all @Name patterns found in the text (e.g., "@John", "@Jane-Doe").
func FindMentions(text string) []string {
	return mentionPattern.FindAllString(text, -1)
}

// CountPageMentions returns the number of @@Name patterns found in the text.
func CountPageMentions(text string) int {
	return len(pageMentionPattern.FindAllStringIndex(text, -1))
}

// FindPageMentions returns all @@Name patterns found in the text (e.g., "@@ProjectPlan").
func FindPageMentions(text string) []string {
	return pageMentionPattern.FindAllString(text, -1)
}

// MarkdownToken represents a parsed markdown segment with its formatting
type MarkdownToken struct {
	Content string
	Bold    bool
	Italic  bool
	Code    bool
	IsLink  bool
	LinkURL string
}

// ParseMarkdown parses text for markdown patterns and returns tokens.
//
// Supported patterns:
//   - **bold** or __bold__
//   - *italic* or _italic_
//   - `code`
//   - ***bold italic*** or ___bold italic___
//   - [text](url) for links
//   - **[text](url)** for formatted links
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
		var bold, italic, code, isLink bool
		var linkURL string

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
					isLink = false
					linkURL = ""
				}
			}
		}

		// Check for links [text](url) - can be inside formatting
		if linkMatch := linkPattern.FindStringIndex(remaining); linkMatch != nil {
			idx := linkMatch[0]
			if earliest == -1 || idx < earliest {
				// Extract link text and URL
				fullMatch := remaining[linkMatch[0]:linkMatch[1]]
				submatches := linkPattern.FindStringSubmatch(fullMatch)
				if len(submatches) == 3 {
					earliest = idx
					tokenContent = submatches[1] // link text
					linkURL = submatches[2]      // URL
					matched = fullMatch
					isLink = true
					bold = false
					italic = false
					code = false
				}
			}
		}

		// Check for bold+italic (***text*** or ___text___)
		for _, marker := range []string{"***", "___"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+3:], marker)
				if endIdx != -1 {
					innerContent := remaining[idx+3 : idx+3+endIdx]
					// Check if inner content is a link
					if linkMatch := linkPattern.FindStringSubmatch(innerContent); linkMatch != nil && linkMatch[0] == innerContent {
						earliest = idx
						tokenContent = linkMatch[1]
						linkURL = linkMatch[2]
						matched = remaining[idx : idx+3+endIdx+3]
						bold = true
						italic = true
						code = false
						isLink = true
					} else {
						earliest = idx
						tokenContent = innerContent
						matched = remaining[idx : idx+3+endIdx+3]
						bold = true
						italic = true
						code = false
						isLink = false
						linkURL = ""
					}
				}
			}
		}

		// Check for bold (**text** or __text__)
		for _, marker := range []string{"**", "__"} {
			if idx := strings.Index(remaining, marker); idx != -1 && (earliest == -1 || idx < earliest) {
				endIdx := strings.Index(remaining[idx+2:], marker)
				if endIdx != -1 {
					innerContent := remaining[idx+2 : idx+2+endIdx]
					// Check if inner content is a link
					if linkMatch := linkPattern.FindStringSubmatch(innerContent); linkMatch != nil && linkMatch[0] == innerContent {
						earliest = idx
						tokenContent = linkMatch[1]
						linkURL = linkMatch[2]
						matched = remaining[idx : idx+2+endIdx+2]
						bold = true
						italic = false
						code = false
						isLink = true
					} else {
						earliest = idx
						tokenContent = innerContent
						matched = remaining[idx : idx+2+endIdx+2]
						bold = true
						italic = false
						code = false
						isLink = false
						linkURL = ""
					}
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
					innerContent := remaining[idx+1 : idx+1+endIdx]
					// Check if inner content is a link
					if linkMatch := linkPattern.FindStringSubmatch(innerContent); linkMatch != nil && linkMatch[0] == innerContent {
						earliest = idx
						tokenContent = linkMatch[1]
						linkURL = linkMatch[2]
						matched = remaining[idx : idx+1+endIdx+1]
						bold = false
						italic = true
						code = false
						isLink = true
					} else {
						earliest = idx
						tokenContent = innerContent
						matched = remaining[idx : idx+1+endIdx+1]
						bold = false
						italic = true
						code = false
						isLink = false
						linkURL = ""
					}
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
			IsLink:  isLink,
			LinkURL: linkURL,
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
// Supports: **bold**, *italic*, _italic_, `code`, ***bold italic***, [links](url), and @mentions.
// Returns the rich_text array with interleaved text and mention objects.
func BuildWithMentions(text string, userIDs []string) []notion.RichText {
	return BuildWithMentionsAndPages(text, userIDs, nil)
}

// BuildWithMentionsAndPages parses text for markdown formatting, @Name patterns (user mentions),
// and @@Name patterns (page mentions), replacing them with properly formatted rich text.
// Supports: **bold**, *italic*, _italic_, `code`, ***bold italic***, [links](url), @mentions, and @@page mentions.
// Returns the rich_text array with interleaved text, link, and mention objects.
func BuildWithMentionsAndPages(text string, userIDs []string, pageIDs []string) []notion.RichText {
	if text == "" && len(userIDs) == 0 && len(pageIDs) == 0 {
		return []notion.RichText{}
	}

	tokens := ParseMarkdown(text)
	return BuildWithMentionsFromTokens(tokens, userIDs, pageIDs)
}

// BuildWithMentionsFromTokens builds rich text from pre-parsed markdown tokens.
// Use this when you've already parsed markdown (e.g., for verbose output) to avoid
// parsing twice. The tokens should come from ParseMarkdown.
// userIDs are matched to @Name patterns, pageIDs are matched to @@Name patterns.
func BuildWithMentionsFromTokens(tokens []MarkdownToken, userIDs []string, pageIDs []string) []notion.RichText {
	if len(tokens) == 0 && len(userIDs) == 0 && len(pageIDs) == 0 {
		return []notion.RichText{}
	}

	// Process each token, looking for @mentions and @@page mentions within them
	var richText []notion.RichText
	userIDIndex := 0
	pageIDIndex := 0

	for _, token := range tokens {
		// If this token is a link, handle it specially
		if token.IsLink {
			richText = append(richText, notion.RichText{
				Type: "text",
				Text: &notion.TextContent{
					Content: token.Content,
					Link:    &notion.Link{URL: token.LinkURL},
				},
				Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
			})
			continue
		}

		// Find all mention patterns (both @ and @@) within this token's content
		userMatches := mentionPattern.FindAllStringIndex(token.Content, -1)
		pageMatches := pageMentionPattern.FindAllStringIndex(token.Content, -1)

		// Filter out user mentions that are actually part of page mentions (@@Name contains @Name)
		filteredUserMatches := filterUserMentionsFromPageMentions(userMatches, pageMatches)

		if len(filteredUserMatches) == 0 && len(pageMatches) == 0 {
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

		// Merge and sort all matches by position
		allMatches := mergeAndSortMatches(filteredUserMatches, pageMatches)

		// Process mentions within this formatted token
		lastEnd := 0
		for _, match := range allMatches {
			start, end := match.start, match.end

			// Add text before this mention (with the token's formatting)
			if start > lastEnd {
				richText = append(richText, notion.RichText{
					Type:        "text",
					Text:        &notion.TextContent{Content: token.Content[lastEnd:start]},
					Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
				})
			}

			if match.isPageMention {
				// Add page mention if we have a page ID for it
				if pageIDIndex < len(pageIDs) {
					richText = append(richText, notion.RichText{
						Type: "mention",
						Mention: &notion.Mention{
							Type: "page",
							Page: &notion.PageMention{ID: pageIDs[pageIDIndex]},
						},
					})
					pageIDIndex++
				} else {
					// No more page IDs, keep the @@Name as plain text with formatting
					richText = append(richText, notion.RichText{
						Type:        "text",
						Text:        &notion.TextContent{Content: token.Content[start:end]},
						Annotations: CreateAnnotations(token.Bold, token.Italic, token.Code),
					})
				}
			} else {
				// Add user mention if we have a user ID for it
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

	// If there are extra page IDs (no matching @@Name patterns), append them at the end
	for ; pageIDIndex < len(pageIDs); pageIDIndex++ {
		richText = append(richText, notion.RichText{
			Type: "mention",
			Mention: &notion.Mention{
				Type: "page",
				Page: &notion.PageMention{ID: pageIDs[pageIDIndex]},
			},
		})
	}

	return richText
}

// mentionMatch represents a found mention with its position and type
type mentionMatch struct {
	start         int
	end           int
	isPageMention bool
}

// filterUserMentionsFromPageMentions removes user mention matches that overlap with page mentions.
// Since @@Name contains @Name, we need to filter out the @Name match when there's a @@Name.
func filterUserMentionsFromPageMentions(userMatches, pageMatches [][]int) [][]int {
	if len(pageMatches) == 0 {
		return userMatches
	}

	var filtered [][]int
	for _, um := range userMatches {
		overlaps := false
		for _, pm := range pageMatches {
			// Check if user match is inside page match (page match is @@ + Name, user match is @ + Name)
			// Page match starts at @@, user match at @ inside the @@Name would be at pm[0]+1
			if um[0] >= pm[0] && um[1] <= pm[1] {
				overlaps = true
				break
			}
		}
		if !overlaps {
			filtered = append(filtered, um)
		}
	}
	return filtered
}

// mergeAndSortMatches combines user and page mention matches into a sorted slice
func mergeAndSortMatches(userMatches, pageMatches [][]int) []mentionMatch {
	var all []mentionMatch
	for _, m := range userMatches {
		all = append(all, mentionMatch{start: m[0], end: m[1], isPageMention: false})
	}
	for _, m := range pageMatches {
		all = append(all, mentionMatch{start: m[0], end: m[1], isPageMention: true})
	}

	// Sort by start position
	slices.SortFunc(all, func(a, b mentionMatch) int {
		return a.start - b.start
	})

	return all
}

// MarkdownSummary holds counts of detected markdown patterns
type MarkdownSummary struct {
	Bold       int
	Italic     int
	Code       int
	BoldItalic int
	Links      int
	Plain      int
}

// SummarizeTokens counts markdown patterns in parsed tokens
func SummarizeTokens(tokens []MarkdownToken) MarkdownSummary {
	var summary MarkdownSummary
	for _, token := range tokens {
		if token.IsLink {
			summary.Links++
			// Also count the formatting on the link
			if token.Bold && token.Italic {
				summary.BoldItalic++
			} else if token.Bold {
				summary.Bold++
			} else if token.Italic {
				summary.Italic++
			}
			continue
		}
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
	if summary.Links > 0 {
		parts = append(parts, formatCount(summary.Links, "link"))
	}
	if len(parts) == 0 {
		return "Parsed markdown: no formatting detected"
	}
	return "Parsed markdown: " + strings.Join(parts, ", ")
}

func formatCount(count int, label string) string {
	return fmt.Sprintf("%d %s", count, label)
}

// ValidateLinkURLs checks link tokens for malformed URLs and returns warnings.
// It warns for:
//   - Empty URLs
//   - URLs containing spaces
//   - URLs that look like web URLs but are missing protocol (e.g., "example.com")
//
// It does NOT warn for:
//   - Relative URLs (starting with / or ./)
//   - Valid schemes (http://, https://, mailto:, tel:, etc.)
//   - Anchor links (#section)
func ValidateLinkURLs(tokens []MarkdownToken) []string {
	var warnings []string
	for _, token := range tokens {
		if !token.IsLink {
			continue
		}
		url := token.LinkURL
		if warning := validateSingleURL(url, token.Content); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings
}

// validateSingleURL checks a single URL and returns a warning message if malformed.
// Returns empty string if URL is valid.
func validateSingleURL(url, linkText string) string {
	// Empty URL
	if url == "" {
		return fmt.Sprintf("link [%s] has empty URL", linkText)
	}

	// URL with spaces
	if strings.Contains(url, " ") {
		return fmt.Sprintf("link [%s](%s) contains spaces in URL", linkText, url)
	}

	// Allow relative URLs
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") {
		return ""
	}

	// Allow anchor links
	if strings.HasPrefix(url, "#") {
		return ""
	}

	// Allow known schemes
	knownSchemes := []string{"http://", "https://", "mailto:", "tel:", "ftp://", "file://", "data:"}
	for _, scheme := range knownSchemes {
		if strings.HasPrefix(strings.ToLower(url), scheme) {
			return ""
		}
	}

	// Check if it looks like a web URL missing protocol (contains dot, looks like domain)
	// e.g., "example.com", "www.example.com/path"
	if looksLikeWebURL(url) {
		return fmt.Sprintf("link [%s](%s) may be missing protocol (http:// or https://)", linkText, url)
	}

	return ""
}

// looksLikeWebURL returns true if the URL looks like it should be a web URL
// but is missing the protocol. Checks for common patterns like domain.tld.
func looksLikeWebURL(url string) bool {
	// Must contain a dot to look like a domain
	if !strings.Contains(url, ".") {
		return false
	}

	// Get the potential domain part (before any path)
	domain := url
	if idx := strings.Index(url, "/"); idx != -1 {
		domain = url[:idx]
	}

	// Check for common TLDs or www prefix
	domain = strings.ToLower(domain)
	if strings.HasPrefix(domain, "www.") {
		return true
	}

	// Common TLDs that suggest this should be a full URL
	commonTLDs := []string{".com", ".org", ".net", ".io", ".co", ".edu", ".gov", ".dev", ".app", ".so"}
	for _, tld := range commonTLDs {
		if strings.HasSuffix(domain, tld) {
			return true
		}
	}

	return false
}
