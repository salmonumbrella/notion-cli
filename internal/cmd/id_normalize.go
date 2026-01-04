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
