package notion

import (
	"fmt"
	"regexp"
	"strings"
)

var notionIDRegex = regexp.MustCompile(`(?i)([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)

// ExtractIDFromNotionURL attempts to extract a Notion page/database ID from a URL or string.
func ExtractIDFromNotionURL(urlStr string) (string, error) {
	if strings.TrimSpace(urlStr) == "" {
		return "", fmt.Errorf("url is required")
	}

	match := notionIDRegex.FindString(urlStr)
	if match == "" {
		return "", fmt.Errorf("no Notion ID found in URL")
	}

	return strings.ToLower(match), nil
}
