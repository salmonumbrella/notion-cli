package notion

import "strings"

// NewSyncedBlock creates a synced_block structure.
// If sourceBlockID is nil, creates an original synced block with children.
// If sourceBlockID is provided, creates a duplicate that syncs from the source.
func NewSyncedBlock(sourceBlockID *string, children []map[string]interface{}) map[string]interface{} {
	syncedBlock := map[string]interface{}{}

	if sourceBlockID != nil {
		// Duplicate synced block
		syncedBlock["synced_from"] = map[string]interface{}{
			"type":     "block_id",
			"block_id": *sourceBlockID,
		}
	} else {
		// Original synced block
		syncedBlock["synced_from"] = nil
		syncedBlock["children"] = children
	}

	return map[string]interface{}{
		"type":         "synced_block",
		"synced_block": syncedBlock,
	}
}

// NewTableOfContents creates a table_of_contents block.
// Color can be: default, gray, brown, orange, yellow, green, blue, purple, pink, red,
// or their _background variants.
func NewTableOfContents(color string) map[string]interface{} {
	if color == "" {
		color = "default"
	}
	return map[string]interface{}{
		"type": "table_of_contents",
		"table_of_contents": map[string]interface{}{
			"color": color,
		},
	}
}

// NewBreadcrumb creates a breadcrumb block.
// Breadcrumbs automatically show the page hierarchy.
func NewBreadcrumb() map[string]interface{} {
	return map[string]interface{}{
		"type":       "breadcrumb",
		"breadcrumb": map[string]interface{}{},
	}
}

// NewColumn creates a column block with children.
func NewColumn(children []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "column",
		"column": map[string]interface{}{
			"children": children,
		},
	}
}

// NewColumnList creates a column_list block with the specified columns.
// Requires at least 2 columns, and each column must have at least one child.
func NewColumnList(columns ...[]map[string]interface{}) map[string]interface{} {
	columnBlocks := make([]map[string]interface{}, len(columns))
	for i, children := range columns {
		columnBlocks[i] = NewColumn(children)
	}

	return map[string]interface{}{
		"type": "column_list",
		"column_list": map[string]interface{}{
			"children": columnBlocks,
		},
	}
}

// NewLinkPreview creates a link_preview block.
func NewLinkPreview(url string) map[string]interface{} {
	return map[string]interface{}{
		"type": "link_preview",
		"link_preview": map[string]interface{}{
			"url": url,
		},
	}
}

// Common block helpers

// NewParagraph creates a paragraph block with text content.
func NewParagraph(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "paragraph",
		"paragraph": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewHeading1 creates a heading_1 block.
func NewHeading1(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_1",
		"heading_1": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewHeading2 creates a heading_2 block.
func NewHeading2(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_2",
		"heading_2": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewHeading3 creates a heading_3 block.
func NewHeading3(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_3",
		"heading_3": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewBulletedListItem creates a bulleted_list_item block.
func NewBulletedListItem(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "bulleted_list_item",
		"bulleted_list_item": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewNumberedListItem creates a numbered_list_item block.
func NewNumberedListItem(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "numbered_list_item",
		"numbered_list_item": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewToDo creates a to_do block.
func NewToDo(text string, checked bool) map[string]interface{} {
	return map[string]interface{}{
		"type": "to_do",
		"to_do": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
			"checked": checked,
		},
	}
}

// NewDivider creates a divider block.
func NewDivider() map[string]interface{} {
	return map[string]interface{}{
		"type":    "divider",
		"divider": map[string]interface{}{},
	}
}

// NewCallout creates a callout block with an emoji icon.
func NewCallout(text string, emoji string) map[string]interface{} {
	return map[string]interface{}{
		"type": "callout",
		"callout": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
			"icon": map[string]interface{}{
				"type":  "emoji",
				"emoji": emoji,
			},
		},
	}
}

// NewQuote creates a quote block.
func NewQuote(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "quote",
		"quote": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": text,
					},
				},
			},
		},
	}
}

// NewCode creates a code block.
func NewCode(code string, language string) map[string]interface{} {
	return map[string]interface{}{
		"type": "code",
		"code": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": code,
					},
				},
			},
			"language": language,
		},
	}
}

// markdownToken represents a parsed markdown segment with its formatting
type markdownToken struct {
	content string
	bold    bool
	italic  bool
	code    bool
}

// parseInlineMarkdown parses text for inline markdown patterns and returns tokens.
// Supports: **bold**, *italic*, `code`, ***bold italic***
func parseInlineMarkdown(text string) []markdownToken {
	if text == "" {
		return nil
	}

	var tokens []markdownToken
	remaining := text

	for len(remaining) > 0 {
		earliest := -1
		var matched string
		var tokenContent string
		var bold, italic, code bool

		// Check for code first (highest priority)
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

		// Check for italic (*text* or _text_)
		for _, marker := range []string{"*", "_"} {
			doubleMarker := marker + marker
			idx := strings.Index(remaining, marker)
			if idx != -1 && strings.HasPrefix(remaining[idx:], doubleMarker) {
				searchFrom := idx + 1
				for searchFrom < len(remaining) {
					nextIdx := strings.Index(remaining[searchFrom:], marker)
					if nextIdx == -1 {
						idx = -1
						break
					}
					actualIdx := searchFrom + nextIdx
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
				searchEnd := idx + 1
				endIdx := -1
				for searchEnd < len(remaining) {
					nextEnd := strings.Index(remaining[searchEnd:], marker)
					if nextEnd == -1 {
						break
					}
					actualEnd := searchEnd + nextEnd
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
			tokens = append(tokens, markdownToken{content: remaining})
			break
		}

		if earliest > 0 {
			tokens = append(tokens, markdownToken{content: remaining[:earliest]})
		}

		tokens = append(tokens, markdownToken{
			content: tokenContent,
			bold:    bold,
			italic:  italic,
			code:    code,
		})

		remaining = remaining[earliest+len(matched):]
	}

	return tokens
}

// ParseMarkdownToRichText converts markdown-formatted text to Notion rich_text array.
// Supports: **bold**, *italic*, `code`, ***bold italic***
func ParseMarkdownToRichText(text string) []map[string]interface{} {
	tokens := parseInlineMarkdown(text)
	if len(tokens) == 0 {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, 0, len(tokens))
	for _, token := range tokens {
		rt := map[string]interface{}{
			"type": "text",
			"text": map[string]interface{}{
				"content": token.content,
			},
		}

		// Only add annotations if there's formatting
		if token.bold || token.italic || token.code {
			rt["annotations"] = map[string]interface{}{
				"bold":          token.bold,
				"italic":        token.italic,
				"strikethrough": false,
				"underline":     false,
				"code":          token.code,
				"color":         "default",
			}
		}

		result = append(result, rt)
	}
	return result
}

// NewParagraphWithMarkdown creates a paragraph block with markdown parsing.
func NewParagraphWithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "paragraph",
		"paragraph": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewHeading1WithMarkdown creates a heading_1 block with markdown parsing.
func NewHeading1WithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_1",
		"heading_1": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewHeading2WithMarkdown creates a heading_2 block with markdown parsing.
func NewHeading2WithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_2",
		"heading_2": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewHeading3WithMarkdown creates a heading_3 block with markdown parsing.
func NewHeading3WithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "heading_3",
		"heading_3": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewBulletedListItemWithMarkdown creates a bulleted_list_item block with markdown parsing.
func NewBulletedListItemWithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "bulleted_list_item",
		"bulleted_list_item": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewNumberedListItemWithMarkdown creates a numbered_list_item block with markdown parsing.
func NewNumberedListItemWithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "numbered_list_item",
		"numbered_list_item": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewToDoWithMarkdown creates a to_do block with markdown parsing.
func NewToDoWithMarkdown(text string, checked bool) map[string]interface{} {
	return map[string]interface{}{
		"type": "to_do",
		"to_do": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
			"checked":   checked,
		},
	}
}

// NewCalloutWithMarkdown creates a callout block with markdown parsing.
func NewCalloutWithMarkdown(text string, emoji string) map[string]interface{} {
	return map[string]interface{}{
		"type": "callout",
		"callout": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
			"icon": map[string]interface{}{
				"type":  "emoji",
				"emoji": emoji,
			},
		},
	}
}

// NewQuoteWithMarkdown creates a quote block with markdown parsing.
func NewQuoteWithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "quote",
		"quote": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewToggleWithMarkdown creates a toggle block with markdown parsing.
func NewToggleWithMarkdown(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "toggle",
		"toggle": map[string]interface{}{
			"rich_text": ParseMarkdownToRichText(text),
		},
	}
}

// NewTableRow creates a table_row block.
// Each cell is a rich_text array ([]map[string]interface{}).
func NewTableRow(cells [][]map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "table_row",
		"table_row": map[string]interface{}{
			"cells": cells,
		},
	}
}

// NewTable creates a table block with nested table_row children.
// The Notion API requires table_row blocks to be nested inside the table's children array.
func NewTable(width int, hasColumnHeader bool, rows []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "table",
		"table": map[string]interface{}{
			"table_width":       width,
			"has_column_header": hasColumnHeader,
			"has_row_header":    false,
			"children":          rows,
		},
	}
}

// NewTableWithMarkdown creates a table block from raw cell strings.
// Each cell string is parsed for inline markdown (bold, italic, code).
func NewTableWithMarkdown(rows [][]string, hasColumnHeader bool) map[string]interface{} {
	if len(rows) == 0 {
		return NewTable(0, false, nil)
	}

	width := len(rows[0])
	var tableRows []map[string]interface{}
	for _, row := range rows {
		cells := make([][]map[string]interface{}, width)
		for j := 0; j < width; j++ {
			if j < len(row) {
				cells[j] = ParseMarkdownToRichText(row[j])
			} else {
				cells[j] = ParseMarkdownToRichText("")
			}
		}
		tableRows = append(tableRows, NewTableRow(cells))
	}

	return NewTable(width, hasColumnHeader, tableRows)
}
