// Package cmd provides CLI commands for the notion-cli tool
package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

// skillFileKey is the context key for the loaded skill file
type skillFileKey struct{}

// WithSkillFile adds the skill file to context
func WithSkillFile(ctx context.Context, sf *skill.SkillFile) context.Context {
	return context.WithValue(ctx, skillFileKey{}, sf)
}

// SkillFileFromContext retrieves the skill file from context
func SkillFileFromContext(ctx context.Context) *skill.SkillFile {
	if sf, ok := ctx.Value(skillFileKey{}).(*skill.SkillFile); ok {
		return sf
	}
	return nil
}

// resolveID resolves an alias to its target ID, or returns the input unchanged
func resolveID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}

	// Try database alias
	if id, ok := sf.ResolveDatabase(input); ok {
		return id
	}

	// Try user alias
	if id, ok := sf.ResolveUser(input); ok {
		return id
	}

	// Try custom alias
	if id, _, ok := sf.ResolveAlias(input); ok {
		return id
	}

	// Return unchanged
	return input
}

// resolveUserID specifically resolves user aliases
func resolveUserID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}
	if id, ok := sf.ResolveUser(input); ok {
		return id
	}
	return input
}

// resolveDatabaseID specifically resolves database aliases
func resolveDatabaseID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}
	if id, ok := sf.ResolveDatabase(input); ok {
		return id
	}
	return input
}

// searcher is a minimal interface for search-based resolution
type searcher interface {
	Search(ctx context.Context, req *notion.SearchRequest) (*notion.SearchResult, error)
}

// uuidPattern matches Notion UUIDs (with or without dashes)
var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{12}$`)

// looksLikeUUID checks if input looks like a Notion UUID (with or without dashes)
func looksLikeUUID(input string) bool {
	return uuidPattern.MatchString(input)
}

// resolveBySearch attempts to resolve an input to a Notion ID via search.
// If input looks like a UUID, it returns the input unchanged.
// If search finds exactly one match, it returns that ID.
// If search finds multiple matches, it returns an error with suggestions.
// If search finds no matches, it returns the original input to let API fail with clear error.
func resolveBySearch(ctx context.Context, client searcher, input string, filterType string) (string, error) {
	// Skip search if input already looks like a UUID
	if looksLikeUUID(input) {
		return input, nil
	}

	// Skip search for empty or very short inputs (likely accidental)
	if len(strings.TrimSpace(input)) < 2 {
		return input, nil
	}

	// Build search request
	req := &notion.SearchRequest{
		Query:    input,
		PageSize: 10, // Limit results for performance
	}

	// Apply type filter if specified
	switch filterType {
	case "page":
		req.Filter = map[string]interface{}{
			"property": "object",
			"value":    "page",
		}
	case "database":
		req.Filter = map[string]interface{}{
			"property": "object",
			"value":    "data_source",
		}
	}

	result, err := client.Search(ctx, req)
	if err != nil {
		// Search failed - return original input and let API fail with specific error
		return input, nil
	}

	// Filter results to exact title matches (case-insensitive)
	exactMatches := filterExactTitleMatches(result.Results, input)

	// If exactly one exact match, return its ID
	if len(exactMatches) == 1 {
		if id, ok := exactMatches[0]["id"].(string); ok {
			return id, nil
		}
	}

	// If multiple exact matches, return error with suggestions
	if len(exactMatches) > 1 {
		return "", buildAmbiguousError(input, exactMatches)
	}

	// No exact matches - check if there are partial matches
	if len(result.Results) == 1 {
		// Single partial match - use it
		if id, ok := result.Results[0]["id"].(string); ok {
			return id, nil
		}
	}

	if len(result.Results) > 1 {
		// Multiple partial matches - return error with suggestions
		return "", buildAmbiguousError(input, result.Results)
	}

	// No matches at all - return original (let API fail with clear error)
	return input, nil
}

// filterExactTitleMatches filters search results to those with titles exactly matching the input
func filterExactTitleMatches(results []map[string]interface{}, input string) []map[string]interface{} {
	inputLower := strings.ToLower(strings.TrimSpace(input))
	var matches []map[string]interface{}

	for _, r := range results {
		title := extractResultTitle(r)
		if strings.ToLower(title) == inputLower {
			matches = append(matches, r)
		}
	}

	return matches
}

// extractResultTitle extracts the title from a search result (page or database)
func extractResultTitle(result map[string]interface{}) string {
	// Try page properties.title (for pages)
	if properties, ok := result["properties"].(map[string]interface{}); ok {
		// Look for the title property
		for _, v := range properties {
			prop, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			if prop["type"] == "title" {
				if titleArr, ok := prop["title"].([]interface{}); ok {
					return extractPlainTextFromRichText(titleArr)
				}
			}
		}
	}

	// Try database/data_source title array
	if titleArr, ok := result["title"].([]interface{}); ok {
		return extractPlainTextFromRichText(titleArr)
	}

	return ""
}

// extractPlainTextFromRichText extracts plain text from a rich_text array
func extractPlainTextFromRichText(richText []interface{}) string {
	var parts []string
	for _, item := range richText {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if plain, ok := entry["plain_text"].(string); ok && plain != "" {
			parts = append(parts, plain)
		} else if text, ok := entry["text"].(map[string]interface{}); ok {
			if content, ok := text["content"].(string); ok && content != "" {
				parts = append(parts, content)
			}
		}
	}
	return strings.Join(parts, "")
}

// buildAmbiguousError builds an error message for ambiguous name matches
func buildAmbiguousError(input string, results []map[string]interface{}) error {
	var suggestions []string
	maxSuggestions := 5
	for i, r := range results {
		if i >= maxSuggestions {
			break
		}
		id, _ := r["id"].(string)
		title := extractResultTitle(r)
		objType, _ := r["object"].(string)
		if objType == "data_source" {
			objType = "database"
		}

		// Format: ID (type): Title
		if title != "" {
			suggestions = append(suggestions, fmt.Sprintf("  %s (%s): %s", id, objType, title))
		} else {
			suggestions = append(suggestions, fmt.Sprintf("  %s (%s)", id, objType))
		}
	}

	count := len(results)
	if count > maxSuggestions {
		suggestions = append(suggestions, fmt.Sprintf("  ... and %d more", count-maxSuggestions))
	}

	return fmt.Errorf("ambiguous name %q matches %d results:\n%s\n\nUse the ID directly to specify which one you mean",
		input, count, strings.Join(suggestions, "\n"))
}

// resolveIDWithSearch resolves input using skill file first, then search fallback.
// This combines the fast skill file lookup with search-based name resolution.
func resolveIDWithSearch(ctx context.Context, client searcher, sf *skill.SkillFile, input string, filterType string) (string, error) {
	// First try skill file resolution (fast, no API call)
	resolved := resolveID(sf, input)
	if resolved != input {
		return resolved, nil // Found in skill file
	}

	// Then try search-based resolution
	return resolveBySearch(ctx, client, input, filterType)
}
