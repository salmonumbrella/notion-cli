package cmd

import (
	"context"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// PathEntry represents one ancestor in a page's breadcrumb path.
type PathEntry struct {
	Type  string `json:"type"` // "page" or "database"
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// EnrichedPage wraps a Page with additional metadata resolved via extra API calls.
// parent_title is the resolved title of the parent database or page.
// child_count is the number of immediate child blocks.
// path is the breadcrumb path from root to immediate parent.
type EnrichedPage struct {
	*notion.Page
	ParentTitle string      `json:"parent_title,omitempty"`
	ChildCount  int         `json:"child_count"`
	Path        []PathEntry `json:"path,omitempty"`
}

// enrichPage fetches additional metadata for a page: parent title and child count.
// This requires extra API calls, so it's opt-in via the --enrich flag.
func enrichPage(ctx context.Context, client *notion.Client, page *notion.Page) *EnrichedPage {
	enriched := &EnrichedPage{Page: page}

	// Build breadcrumb path (root-first) and derive parent title from it.
	// The last entry in the path is the immediate parent, so we extract
	// ParentTitle from there instead of making a separate API call.
	if page.Parent != nil {
		enriched.Path = buildBreadcrumbPath(ctx, client, page.Parent)
		if len(enriched.Path) > 0 {
			enriched.ParentTitle = enriched.Path[len(enriched.Path)-1].Title
		}
	}

	// Get child count (immediate children only)
	children, err := client.GetBlockChildren(ctx, page.ID, nil)
	if err == nil && children != nil {
		enriched.ChildCount = len(children.Results)
	}

	return enriched
}

// buildBreadcrumbPath walks the parent chain and returns the path from root to
// the immediate parent. Stops at workspace root or after maxDepth hops to avoid
// runaway chains. Returns entries in root-first order.
func buildBreadcrumbPath(ctx context.Context, client *notion.Client, parent map[string]interface{}) []PathEntry {
	const maxDepth = 10
	var entries []PathEntry

	current := parent
	for i := 0; i < maxDepth && current != nil; i++ {
		if dbID, ok := current["database_id"].(string); ok && dbID != "" {
			db, err := client.GetDatabase(ctx, dbID)
			if err != nil {
				break
			}
			entries = append(entries, PathEntry{
				Type:  "database",
				ID:    dbID,
				Title: extractDatabaseTitle(*db),
			})
			break // Stop at database boundary to limit API calls
		}

		if pageID, ok := current["page_id"].(string); ok && pageID != "" {
			parentPage, err := client.GetPage(ctx, pageID)
			if err != nil {
				break
			}
			entries = append(entries, PathEntry{
				Type:  "page",
				ID:    pageID,
				Title: extractPageTitleFromProperties(parentPage.Properties),
			})
			current = parentPage.Parent
			continue
		}

		break // workspace or unknown parent type
	}

	// Reverse: we collected child-first, but want root-first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

// extractPageTitleFromProperties extracts the plain text title from a page's properties.
func extractPageTitleFromProperties(properties map[string]interface{}) string {
	for _, propVal := range properties {
		prop, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		if prop["type"] == "title" {
			if titleArr, ok := prop["title"].([]interface{}); ok {
				return extractTitlePlainText(titleArr)
			}
		}
	}
	return ""
}
