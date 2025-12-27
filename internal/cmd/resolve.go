// Package cmd provides CLI commands for the notion-cli tool
package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

// searchCacheKey is the context key for the search result cache
type searchCacheKey struct{}

// SearchCache stores search results for the duration of a single command execution.
// It reduces API calls when the same search is performed multiple times (e.g., batch operations).
type SearchCache struct {
	mu    sync.RWMutex
	cache map[string]*notion.SearchResult
}

// NewSearchCache creates a new empty search cache.
func NewSearchCache() *SearchCache {
	return &SearchCache{
		cache: make(map[string]*notion.SearchResult),
	}
}

// cacheKey generates a cache key from query and filter type.
func cacheKey(query, filterType string) string {
	return query + "\x00" + filterType
}

// Get retrieves a cached search result, returning nil if not found.
func (c *SearchCache) Get(query, filterType string) *notion.SearchResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[cacheKey(query, filterType)]
}

// Set stores a search result in the cache.
func (c *SearchCache) Set(query, filterType string, result *notion.SearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[cacheKey(query, filterType)] = result
}

// Len returns the number of cached entries.
func (c *SearchCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// WithSearchCache adds the search cache to context.
func WithSearchCache(ctx context.Context, cache *SearchCache) context.Context {
	return context.WithValue(ctx, searchCacheKey{}, cache)
}

// SearchCacheFromContext retrieves the search cache from context, or nil if not present.
func SearchCacheFromContext(ctx context.Context) *SearchCache {
	if cache, ok := ctx.Value(searchCacheKey{}).(*SearchCache); ok {
		return cache
	}
	return nil
}

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

// searchFilterValue converts a user-facing filter type to the Notion API value.
// The Notion API (2025-09-03+) uses "data_source" instead of "database".
// This function centralizes that mapping to ensure consistency.
func searchFilterValue(filterType string) string {
	if filterType == "database" {
		return "data_source"
	}
	return filterType
}

// buildSearchFilter creates a search filter for the given object type.
// Accepts "page" or "database" as filterType. Returns nil if filterType is empty.
func buildSearchFilter(filterType string) map[string]interface{} {
	if filterType == "" {
		return nil
	}
	return map[string]interface{}{
		"property": "object",
		"value":    searchFilterValue(filterType),
	}
}

// looksLikeUUID checks if input looks like a Notion UUID (with or without dashes)
func looksLikeUUID(input string) bool {
	return uuidPattern.MatchString(input)
}

// resolveBySearch attempts to resolve an input to a Notion ID via search.
// If input looks like a UUID, it returns the input unchanged.
// If search finds exactly one match, it returns that ID.
// If search finds multiple matches, it returns an error with suggestions.
// If search finds no matches, it returns the original input to let API fail with clear error.
//
// When a SearchCache is present in the context, results are cached to avoid
// duplicate API calls for identical queries within the same command execution.
func resolveBySearch(ctx context.Context, client searcher, input string, filterType string) (string, error) {
	// Skip search if input already looks like a UUID
	if looksLikeUUID(input) {
		return input, nil
	}

	// Skip search for empty or very short inputs (likely accidental)
	if len(strings.TrimSpace(input)) < 2 {
		return input, nil
	}

	// Check cache first
	cache := SearchCacheFromContext(ctx)
	var result *notion.SearchResult
	if cache != nil {
		result = cache.Get(input, filterType)
	}

	// If not in cache, perform the search
	if result == nil {
		req := &notion.SearchRequest{
			Query:    input,
			PageSize: 10, // Limit results for performance
			Filter:   buildSearchFilter(filterType),
		}

		var err error
		result, err = client.Search(ctx, req)
		if err != nil {
			// Search failed - return original input and let API fail with specific error
			return input, nil
		}

		// Store in cache for future lookups
		if cache != nil {
			cache.Set(input, filterType, result)
		}
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
