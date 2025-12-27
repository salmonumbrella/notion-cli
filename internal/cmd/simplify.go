package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func simplifyBlocks(blocks []notion.Block) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, simplifyBlock(b))
	}
	return out
}

func simplifyBlock(b notion.Block) map[string]interface{} {
	m := map[string]interface{}{
		"id":           b.ID,
		"type":         b.Type,
		"has_children": b.HasChildren,
	}

	if txt := blockPlainText(b); txt != "" {
		m["text"] = txt
	}

	// A few high-signal fields that agents often want without spelunking.
	switch b.Type {
	case "to_do":
		if checked, ok := b.Content["checked"].(bool); ok {
			m["checked"] = checked
		}
	case "code":
		if lang, ok := b.Content["language"].(string); ok && strings.TrimSpace(lang) != "" {
			m["language"] = lang
		}
	case "bookmark":
		if u, ok := b.Content["url"].(string); ok && strings.TrimSpace(u) != "" {
			m["url"] = u
		}
	}

	if len(b.Children) > 0 {
		m["children"] = simplifyBlocks(b.Children)
	}

	return m
}

func blockPlainText(b notion.Block) string {
	// child_page / child_database use "title" as a plain string.
	if title, ok := b.Content["title"].(string); ok {
		if strings.TrimSpace(title) != "" {
			return title
		}
	}

	// Most text-bearing blocks expose rich_text.
	if rt, ok := b.Content["rich_text"]; ok {
		if txt := plainTextFromRichTextArray(rt); strings.TrimSpace(txt) != "" {
			return txt
		}
	}

	// Some blocks carry useful text in caption.
	if capRT, ok := b.Content["caption"]; ok {
		if txt := plainTextFromRichTextArray(capRT); strings.TrimSpace(txt) != "" {
			return txt
		}
	}

	return ""
}

// simplifyPropertyValue converts a Notion property value (prop[propType]) into a best-effort scalar.
// It intentionally returns interface{} to preserve types (bool/number/string/arrays/maps).
func simplifyPropertyValue(propType string, value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch propType {
	case "title", "rich_text":
		return plainTextFromRichTextArray(value)
	case "number", "checkbox", "url", "email", "phone_number":
		return value
	case "select", "status":
		if m, ok := value.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				return name
			}
		}
		return nil
	case "multi_select":
		arr, ok := value.([]interface{})
		if !ok {
			return nil
		}
		names := make([]string, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := m["name"].(string); ok && strings.TrimSpace(name) != "" {
				names = append(names, name)
			}
		}
		return names
	case "relation":
		arr, ok := value.([]interface{})
		if !ok {
			return nil
		}
		ids := make([]string, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
				ids = append(ids, id)
			}
		}
		return ids
	case "people":
		arr, ok := value.([]interface{})
		if !ok {
			return nil
		}
		out := make([]map[string]interface{}, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			entry := map[string]interface{}{}
			if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
				entry["id"] = id
			}
			if name, ok := m["name"].(string); ok && strings.TrimSpace(name) != "" {
				entry["name"] = name
			}
			if len(entry) > 0 {
				out = append(out, entry)
			}
		}
		return out
	case "files":
		arr, ok := value.([]interface{})
		if !ok {
			return nil
		}
		names := make([]string, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := m["name"].(string); ok && strings.TrimSpace(name) != "" {
				names = append(names, name)
			}
		}
		return names
	case "date":
		// Keep the Notion date object; it's compact and includes end/time_zone when present.
		if m, ok := value.(map[string]interface{}); ok {
			return m
		}
		return value
	case "formula", "rollup":
		m, ok := value.(map[string]interface{})
		if !ok {
			return value
		}
		t, _ := m["type"].(string)
		if t == "" {
			return value
		}
		// Return the computed value directly (could be nil).
		return m[t]
	case "unique_id":
		m, ok := value.(map[string]interface{})
		if !ok {
			return value
		}
		prefix, _ := m["prefix"].(string)
		var numStr string
		switch n := m["number"].(type) {
		case float64:
			numStr = strconv.Itoa(int(n))
		case int:
			numStr = strconv.Itoa(n)
		case string:
			numStr = n
		}
		if strings.TrimSpace(prefix) != "" && strings.TrimSpace(numStr) != "" {
			return fmt.Sprintf("%s%s", prefix, numStr)
		}
		return value
	default:
		return value
	}
}

func plainTextFromRichTextArray(v interface{}) string {
	arr, ok := v.([]interface{})
	if !ok {
		return ""
	}
	if len(arr) == 0 {
		return ""
	}

	var b strings.Builder
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if s, ok := m["plain_text"].(string); ok {
			b.WriteString(s)
		} else if text, ok := m["text"].(map[string]interface{}); ok {
			// Fallback: some rich text may not carry plain_text in edge cases.
			if content, ok := text["content"].(string); ok {
				b.WriteString(content)
			}
		}
	}

	return b.String()
}
