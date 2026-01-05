package cmd

import (
	"fmt"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func normalizeNotionID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("id is required")
	}

	if looksLikeURL(trimmed) {
		id, err := notion.ExtractIDFromNotionURL(trimmed)
		if err != nil {
			return "", err
		}
		return id, nil
	}

	return trimmed, nil
}

// looksLikeURL determines whether the given value appears to be a URL rather than
// a raw Notion ID. It uses several heuristics:
//
//  1. Explicit URL schemes: Matches "http://" or "https://" prefixes
//  2. Notion domains: Matches strings containing "notion.so" or "notion.site"
//  3. Path separators: Matches strings containing "/" to catch URL fragments
//     (e.g., "notion.so/Page-Title-abc123" pasted without the scheme)
//
// The "/" check is intentionally broad to catch partial URLs that users might copy
// from their browser. This is unlikely to cause false positives since Notion IDs
// are UUIDs (32 hex characters with optional dashes), which never contain slashes.
func looksLikeURL(value string) bool {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	if strings.Contains(lower, "notion.so") || strings.Contains(lower, "notion.site") {
		return true
	}
	return strings.Contains(value, "/")
}
