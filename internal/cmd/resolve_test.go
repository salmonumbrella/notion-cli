package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

// mockSearcher implements the searcher interface for testing
type mockSearcher struct {
	results   []map[string]interface{}
	err       error
	called    bool
	callCount int
}

func (m *mockSearcher) Search(ctx context.Context, req *notion.SearchRequest) (*notion.SearchResult, error) {
	m.called = true
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return &notion.SearchResult{Results: m.results}, nil
}

func TestResolveID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Aliases: map[string]skill.CustomAlias{
			"standup": {TargetID: "page-789", Type: "page"},
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"me", "user-456"},
		{"standup", "page-789"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveID_NilSkillFile(t *testing.T) {
	got := resolveID(nil, "test-input")
	if got != "test-input" {
		t.Errorf("resolveID(nil, %q) = %q, want %q", "test-input", got, "test-input")
	}
}

func TestResolveUserID(t *testing.T) {
	sf := &skill.SkillFile{
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Databases: map[string]skill.DatabaseAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"me", "user-456"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveUserID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveUserID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveDatabaseID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users:   map[string]skill.UserAlias{},
		Aliases: map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveDatabaseID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveDatabaseID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWithSkillFile(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	ctx := context.Background()
	ctx = WithSkillFile(ctx, sf)

	got := SkillFileFromContext(ctx)
	if got != sf {
		t.Errorf("SkillFileFromContext() did not return the expected skill file")
	}
}

func TestSkillFileFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := SkillFileFromContext(ctx)
	if got != nil {
		t.Errorf("SkillFileFromContext() = %v, want nil", got)
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"12345678-1234-1234-1234-123456789012", true},
		{"12345678123412341234123456789012", true},     // Without dashes
		{"ABCDEFAB-1234-5678-9ABC-DEF012345678", true}, // Uppercase
		{"abcdefab-1234-5678-9abc-def012345678", true}, // Lowercase
		{"AbCdEfAb-1234-5678-9aBc-DeF012345678", true}, // Mixed case
		{"1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d", true}, // Typical Notion ID

		// Invalid - not UUIDs
		{"Meeting Notes", false},
		{"my-page-alias", false},
		{"12345", false},
		{"", false},
		{"123456781234123412341234567890", false},        // Too short
		{"123456781234123412341234567890123", false},     // Too long
		{"12345678-1234-1234-1234-12345678901g", false},  // Invalid char
		{"https://example.invalid/page-12345678", false}, // URL
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeUUID(tt.input)
			if got != tt.expected {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractPlainTextFromRichText(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected string
	}{
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "",
		},
		{
			name: "plain_text field",
			input: []interface{}{
				map[string]interface{}{
					"plain_text": "Hello World",
				},
			},
			expected: "Hello World",
		},
		{
			name: "text.content field",
			input: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"content": "Hello World",
					},
				},
			},
			expected: "Hello World",
		},
		{
			name: "multiple parts",
			input: []interface{}{
				map[string]interface{}{"plain_text": "Hello "},
				map[string]interface{}{"plain_text": "World"},
			},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainTextFromRichText(tt.input)
			if got != tt.expected {
				t.Errorf("extractPlainTextFromRichText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractResultTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "empty result",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name: "page with title property",
			input: map[string]interface{}{
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type": "title",
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "My Page Title",
							},
						},
					},
				},
			},
			expected: "My Page Title",
		},
		{
			name: "database with title array",
			input: map[string]interface{}{
				"title": []interface{}{
					map[string]interface{}{
						"plain_text": "My Database",
					},
				},
			},
			expected: "My Database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractResultTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFilterExactTitleMatches(t *testing.T) {
	results := []map[string]interface{}{
		{
			"id": "id-1",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes"}},
				},
			},
		},
		{
			"id": "id-2",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes - Q1"}},
				},
			},
		},
		{
			"id": "id-3",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "meeting notes"}},
				},
			},
		},
	}

	// Should match "Meeting Notes" exactly (case-insensitive)
	matches := filterExactTitleMatches(results, "Meeting Notes")
	if len(matches) != 2 {
		t.Errorf("filterExactTitleMatches() returned %d matches, want 2", len(matches))
	}

	// Should not match partial title
	matches = filterExactTitleMatches(results, "Meeting")
	if len(matches) != 0 {
		t.Errorf("filterExactTitleMatches() returned %d matches for partial title, want 0", len(matches))
	}
}

func TestResolveBySearch_UUIDInput(t *testing.T) {
	mock := &mockSearcher{}
	ctx := context.Background()

	// UUID input should be returned as-is without search
	result, err := resolveBySearch(ctx, mock, "12345678-1234-1234-1234-123456789012", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("got %q, want UUID unchanged", result)
	}
	if mock.called {
		t.Error("search should not be called for UUID input")
	}
}

func TestResolveBySearch_ShortInput(t *testing.T) {
	mock := &mockSearcher{}
	ctx := context.Background()

	// Short input (< 2 chars) should be returned as-is
	result, err := resolveBySearch(ctx, mock, "x", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "x" {
		t.Errorf("got %q, want short input unchanged", result)
	}
	if mock.called {
		t.Error("search should not be called for short input")
	}

	// Empty input
	result, err = resolveBySearch(ctx, mock, "", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty input unchanged", result)
	}
}

func TestResolveBySearch_SingleExactMatch(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-id-123",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes"}},
					},
				},
			},
		},
	}
	ctx := context.Background()

	result, err := resolveBySearch(ctx, mock, "Meeting Notes", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "page-id-123" {
		t.Errorf("got %q, want %q", result, "page-id-123")
	}
	if !mock.called {
		t.Error("search should be called")
	}
}

func TestResolveBySearch_MultipleExactMatches(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-id-1",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes"}},
					},
				},
			},
			{
				"id":     "page-id-2",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes"}},
					},
				},
			},
		},
	}
	ctx := context.Background()

	result, err := resolveBySearch(ctx, mock, "Meeting Notes", "page")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if result != "" {
		t.Errorf("got result %q, want empty string on ambiguous", result)
	}
	// Check error message contains "ambiguous"
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error should mention 'ambiguous', got: %v", err)
	}
}

func TestResolveBySearch_SinglePartialMatch(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-id-456",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes - Q1 2024"}},
					},
				},
			},
		},
	}
	ctx := context.Background()

	// Searching for "Meeting" should find the partial match
	result, err := resolveBySearch(ctx, mock, "Meeting", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "page-id-456" {
		t.Errorf("got %q, want %q", result, "page-id-456")
	}
}

func TestResolveBySearch_MultiplePartialMatches(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-id-1",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes - Q1"}},
					},
				},
			},
			{
				"id":     "page-id-2",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Meeting Notes - Q2"}},
					},
				},
			},
		},
	}
	ctx := context.Background()

	// Searching for "Meeting" with multiple partial matches should return ambiguous error
	result, err := resolveBySearch(ctx, mock, "Meeting", "page")
	if err == nil {
		t.Fatal("expected ambiguous error for multiple partial matches, got nil")
	}
	if result != "" {
		t.Errorf("got result %q, want empty string on ambiguous", result)
	}
}

func TestResolveBySearch_NoMatches(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{},
	}
	ctx := context.Background()

	// No matches should return original input
	result, err := resolveBySearch(ctx, mock, "Nonexistent Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Nonexistent Page" {
		t.Errorf("got %q, want original input unchanged", result)
	}
}

func TestResolveBySearch_SearchError(t *testing.T) {
	mock := &mockSearcher{
		err: errors.New("API error"),
	}
	ctx := context.Background()

	// Search error should return original input (not error)
	result, err := resolveBySearch(ctx, mock, "Some Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Some Page" {
		t.Errorf("got %q, want original input on search error", result)
	}
}

func TestResolveIDWithSearch_SkillFileMatch(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"tasks": {ID: "db-tasks-123"},
		},
		Users:   map[string]skill.UserAlias{},
		Aliases: map[string]skill.CustomAlias{},
	}

	mock := &mockSearcher{}
	ctx := context.Background()

	// Skill file alias should be resolved without search
	result, err := resolveIDWithSearch(ctx, mock, sf, "tasks", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "db-tasks-123" {
		t.Errorf("got %q, want %q", result, "db-tasks-123")
	}
	if mock.called {
		t.Error("search should not be called when skill file has a match")
	}
}

func TestResolveIDWithSearch_NoSkillFileMatch_SearchMatch(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "db-search-456",
				"object": "data_source",
				"title":  []interface{}{map[string]interface{}{"plain_text": "Projects Database"}},
			},
		},
	}
	ctx := context.Background()

	// No skill file match, should fall back to search
	result, err := resolveIDWithSearch(ctx, mock, sf, "Projects Database", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "db-search-456" {
		t.Errorf("got %q, want %q", result, "db-search-456")
	}
	if !mock.called {
		t.Error("search should be called when skill file has no match")
	}
}

func TestResolveIDWithSearch_NoMatchAnywhere(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	mock := &mockSearcher{
		results: []map[string]interface{}{},
	}
	ctx := context.Background()

	// No match in skill file or search - returns original
	result, err := resolveIDWithSearch(ctx, mock, sf, "Unknown Thing", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Unknown Thing" {
		t.Errorf("got %q, want original input unchanged", result)
	}
}

func TestResolveIDWithSearch_NilSkillFile(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-id-789",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "My Page"}},
					},
				},
			},
		},
	}
	ctx := context.Background()

	// Nil skill file should fall through to search
	result, err := resolveIDWithSearch(ctx, mock, nil, "My Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "page-id-789" {
		t.Errorf("got %q, want %q", result, "page-id-789")
	}
	if !mock.called {
		t.Error("search should be called with nil skill file")
	}
}

func TestSearchFilterValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"page", "page"},
		{"database", "data_source"},
		{"", ""},
		{"other", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := searchFilterValue(tt.input)
			if got != tt.expected {
				t.Errorf("searchFilterValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildSearchFilter(t *testing.T) {
	tests := []struct {
		name        string
		filterType  string
		expectNil   bool
		expectedVal string
	}{
		{
			name:       "empty filter type",
			filterType: "",
			expectNil:  true,
		},
		{
			name:        "page filter",
			filterType:  "page",
			expectNil:   false,
			expectedVal: "page",
		},
		{
			name:        "database filter mapped to data_source",
			filterType:  "database",
			expectNil:   false,
			expectedVal: "data_source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSearchFilter(tt.filterType)

			if tt.expectNil {
				if got != nil {
					t.Errorf("buildSearchFilter(%q) = %v, want nil", tt.filterType, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("buildSearchFilter(%q) = nil, want non-nil", tt.filterType)
			}

			if got["property"] != "object" {
				t.Errorf("buildSearchFilter(%q)[\"property\"] = %v, want \"object\"", tt.filterType, got["property"])
			}

			if got["value"] != tt.expectedVal {
				t.Errorf("buildSearchFilter(%q)[\"value\"] = %v, want %q", tt.filterType, got["value"], tt.expectedVal)
			}
		})
	}
}

// --- Search Cache Tests ---

func TestSearchCache_Basic(t *testing.T) {
	cache := NewSearchCache()

	// Initially empty
	if cache.Len() != 0 {
		t.Errorf("new cache should be empty, got %d entries", cache.Len())
	}

	// Get from empty cache returns nil
	result := cache.Get("query", "page")
	if result != nil {
		t.Errorf("Get from empty cache should return nil, got %v", result)
	}

	// Set and get
	expected := &notion.SearchResult{
		Results: []map[string]interface{}{{"id": "abc"}},
	}
	cache.Set("query", "page", expected)

	if cache.Len() != 1 {
		t.Errorf("cache should have 1 entry, got %d", cache.Len())
	}

	got := cache.Get("query", "page")
	if got != expected {
		t.Errorf("Get() = %v, want %v", got, expected)
	}
}

func TestSearchCache_DifferentKeys(t *testing.T) {
	cache := NewSearchCache()

	result1 := &notion.SearchResult{Results: []map[string]interface{}{{"id": "1"}}}
	result2 := &notion.SearchResult{Results: []map[string]interface{}{{"id": "2"}}}
	result3 := &notion.SearchResult{Results: []map[string]interface{}{{"id": "3"}}}

	// Same query, different filter type
	cache.Set("query", "page", result1)
	cache.Set("query", "database", result2)

	// Different query, same filter type
	cache.Set("other", "page", result3)

	if cache.Len() != 3 {
		t.Errorf("cache should have 3 entries, got %d", cache.Len())
	}

	if got := cache.Get("query", "page"); got != result1 {
		t.Errorf("Get(query, page) = %v, want %v", got, result1)
	}
	if got := cache.Get("query", "database"); got != result2 {
		t.Errorf("Get(query, database) = %v, want %v", got, result2)
	}
	if got := cache.Get("other", "page"); got != result3 {
		t.Errorf("Get(other, page) = %v, want %v", got, result3)
	}
}

func TestSearchCacheFromContext(t *testing.T) {
	// Without cache in context
	ctx := context.Background()
	if cache := SearchCacheFromContext(ctx); cache != nil {
		t.Errorf("SearchCacheFromContext() without cache should return nil, got %v", cache)
	}

	// With cache in context
	expectedCache := NewSearchCache()
	ctx = WithSearchCache(ctx, expectedCache)

	if got := SearchCacheFromContext(ctx); got != expectedCache {
		t.Errorf("SearchCacheFromContext() = %v, want %v", got, expectedCache)
	}
}

func TestResolveBySearch_WithCache(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-cached-123",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Test Page"}},
					},
				},
			},
		},
	}

	cache := NewSearchCache()
	ctx := WithSearchCache(context.Background(), cache)

	// First call - should hit API
	result1, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != "page-cached-123" {
		t.Errorf("first call: got %q, want %q", result1, "page-cached-123")
	}
	if mock.callCount != 1 {
		t.Errorf("first call: expected 1 API call, got %d", mock.callCount)
	}

	// Second call with same query - should use cache
	result2, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != "page-cached-123" {
		t.Errorf("second call: got %q, want %q", result2, "page-cached-123")
	}
	if mock.callCount != 1 {
		t.Errorf("second call: expected still 1 API call (cached), got %d", mock.callCount)
	}

	// Third call with different filter type - should hit API again
	_, err = resolveBySearch(ctx, mock, "Test Page", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have made new API call since filter type is different
	if mock.callCount != 2 {
		t.Errorf("third call: expected 2 API calls, got %d", mock.callCount)
	}
}

func TestResolveBySearch_WithoutCache(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "page-123",
				"object": "page",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []interface{}{map[string]interface{}{"plain_text": "Test Page"}},
					},
				},
			},
		},
	}

	// No cache in context
	ctx := context.Background()

	// First call
	result1, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != "page-123" {
		t.Errorf("first call: got %q, want %q", result1, "page-123")
	}
	if mock.callCount != 1 {
		t.Errorf("first call: expected 1 API call, got %d", mock.callCount)
	}

	// Second call without cache - should hit API again
	result2, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != "page-123" {
		t.Errorf("second call: got %q, want %q", result2, "page-123")
	}
	if mock.callCount != 2 {
		t.Errorf("second call: expected 2 API calls (no cache), got %d", mock.callCount)
	}
}

func TestResolveBySearch_CacheUUIDSkip(t *testing.T) {
	mock := &mockSearcher{
		results: []map[string]interface{}{},
	}

	cache := NewSearchCache()
	ctx := WithSearchCache(context.Background(), cache)

	// UUID input should skip search entirely (no cache interaction)
	uuid := "12345678-1234-1234-1234-123456789012"
	result, err := resolveBySearch(ctx, mock, uuid, "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != uuid {
		t.Errorf("got %q, want UUID unchanged", result)
	}
	if mock.callCount != 0 {
		t.Error("search should not be called for UUID input")
	}
	if cache.Len() != 0 {
		t.Error("cache should remain empty for UUID input")
	}
}

func TestResolveBySearch_CacheWithError(t *testing.T) {
	mock := &mockSearcher{
		err: errors.New("API error"),
	}

	cache := NewSearchCache()
	ctx := WithSearchCache(context.Background(), cache)

	// First call fails - should not cache the failure
	result1, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != "Test Page" {
		t.Errorf("got %q, want original input on error", result1)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 API call, got %d", mock.callCount)
	}
	// Failed searches should not be cached
	if cache.Len() != 0 {
		t.Errorf("cache should not store failed searches, got %d entries", cache.Len())
	}

	// Second call should try again
	result2, err := resolveBySearch(ctx, mock, "Test Page", "page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != "Test Page" {
		t.Errorf("got %q, want original input on error", result2)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 API calls (errors not cached), got %d", mock.callCount)
	}
}

func TestResolveIDWithSearch_CacheIntegration(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	mock := &mockSearcher{
		results: []map[string]interface{}{
			{
				"id":     "db-search-789",
				"object": "data_source",
				"title":  []interface{}{map[string]interface{}{"plain_text": "Projects"}},
			},
		},
	}

	cache := NewSearchCache()
	ctx := WithSearchCache(context.Background(), cache)

	// First call - should search and cache
	result1, err := resolveIDWithSearch(ctx, mock, sf, "Projects", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != "db-search-789" {
		t.Errorf("first call: got %q, want %q", result1, "db-search-789")
	}
	if mock.callCount != 1 {
		t.Errorf("first call: expected 1 API call, got %d", mock.callCount)
	}

	// Second call - should use cache
	result2, err := resolveIDWithSearch(ctx, mock, sf, "Projects", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != "db-search-789" {
		t.Errorf("second call: got %q, want %q", result2, "db-search-789")
	}
	if mock.callCount != 1 {
		t.Errorf("second call: expected still 1 API call (cached), got %d", mock.callCount)
	}
}
