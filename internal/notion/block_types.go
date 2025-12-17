package notion

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
