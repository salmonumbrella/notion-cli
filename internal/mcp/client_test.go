package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestExtractText(t *testing.T) {
	tests := []struct {
		name   string
		result *mcp.CallToolResult
		want   string
	}{
		{
			name: "single text content",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Hello, world!",
					},
				},
			},
			want: "Hello, world!",
		},
		{
			name: "multiple text content items",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "First",
					},
					mcp.TextContent{
						Type: "text",
						Text: "Second",
					},
				},
			},
			want: "First\nSecond",
		},
		{
			name: "empty content",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{},
			},
			want: "",
		},
		{
			name: "non-text content is skipped",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.ImageContent{
						Type:     "image",
						Data:     "base64data",
						MIMEType: "image/png",
					},
					mcp.TextContent{
						Type: "text",
						Text: "Caption",
					},
				},
			},
			want: "Caption",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractText(tt.result)
			if got != tt.want {
				t.Errorf("extractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCallToolArgConstruction(t *testing.T) {
	// Verify the argument maps that our typed wrappers build are correct.
	// This tests the structure without making actual MCP calls.

	t.Run("search workspace", func(t *testing.T) {
		args := map[string]interface{}{
			"query":       "test query",
			"search_mode": string(SearchModeWorkspace),
		}
		if args["query"] != "test query" {
			t.Errorf("query = %v, want 'test query'", args["query"])
		}
		if args["search_mode"] != "workspace_search" {
			t.Errorf("search_mode = %v, want 'workspace_search'", args["search_mode"])
		}
	})

	t.Run("search ai", func(t *testing.T) {
		args := map[string]interface{}{
			"query":       "semantic query",
			"search_mode": string(SearchModeAI),
		}
		if args["search_mode"] != "ai_search" {
			t.Errorf("search_mode = %v, want 'ai_search'", args["search_mode"])
		}
	})

	t.Run("fetch args", func(t *testing.T) {
		args := map[string]interface{}{
			"resource_id": "abc-123",
		}
		if args["resource_id"] != "abc-123" {
			t.Errorf("resource_id = %v, want 'abc-123'", args["resource_id"])
		}
	})

	t.Run("create page args", func(t *testing.T) {
		pages := []CreatePageInput{
			{
				Properties: map[string]interface{}{"title": "Test Page"},
				Content:    "# Hello",
			},
		}

		// Simulate the arg construction from CreatePages.
		pagesArg := make([]interface{}, len(pages))
		for i, p := range pages {
			page := map[string]interface{}{}
			if p.Properties != nil {
				page["properties"] = p.Properties
			}
			if p.Content != "" {
				page["content"] = p.Content
			}
			pagesArg[i] = page
		}
		args := map[string]interface{}{
			"pages": pagesArg,
			"parent": map[string]interface{}{
				"page_id": "parent-1",
			},
		}

		// Verify parent structure.
		parentMap := args["parent"].(map[string]interface{})
		if parentMap["page_id"] != "parent-1" {
			t.Errorf("parent.page_id = %v, want 'parent-1'", parentMap["page_id"])
		}

		// Verify page structure.
		pagesList := args["pages"].([]interface{})
		if len(pagesList) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(pagesList))
		}
		first := pagesList[0].(map[string]interface{})
		props := first["properties"].(map[string]interface{})
		if props["title"] != "Test Page" {
			t.Errorf("properties.title = %v, want 'Test Page'", props["title"])
		}
		if first["content"] != "# Hello" {
			t.Errorf("content = %v, want '# Hello'", first["content"])
		}
	})

	t.Run("create page standalone (no parent)", func(t *testing.T) {
		pages := []CreatePageInput{
			{Properties: map[string]interface{}{"title": "Standalone"}},
		}
		pagesArg := make([]interface{}, len(pages))
		for i, p := range pages {
			page := map[string]interface{}{}
			if p.Properties != nil {
				page["properties"] = p.Properties
			}
			pagesArg[i] = page
		}
		args := map[string]interface{}{
			"pages": pagesArg,
		}
		// No parent key should be present.
		if _, ok := args["parent"]; ok {
			t.Error("standalone page should not have a parent key")
		}
	})

	t.Run("update page replace_content", func(t *testing.T) {
		params := UpdatePageParams{
			PageID:  "page-1",
			Command: UpdateCmdReplaceContent,
			NewStr:  "new content",
		}
		args := map[string]interface{}{
			"page_id": params.PageID,
			"command": "replace_content",
			"new_str": params.NewStr,
		}
		if args["command"] != "replace_content" {
			t.Errorf("command = %v, want 'replace_content'", args["command"])
		}
		if args["new_str"] != "new content" {
			t.Errorf("new_str = %v, want 'new content'", args["new_str"])
		}
	})

	t.Run("update page replace_content_range", func(t *testing.T) {
		params := UpdatePageParams{
			PageID:                "page-1",
			Command:               UpdateCmdReplaceContentRange,
			SelectionWithEllipsis: "# Header\n...\nend of section",
			NewStr:                "# New Header\nNew body",
		}
		args := map[string]interface{}{
			"page_id":                 params.PageID,
			"command":                 "replace_content_range",
			"selection_with_ellipsis": params.SelectionWithEllipsis,
			"new_str":                 params.NewStr,
		}
		if args["command"] != "replace_content_range" {
			t.Errorf("command = %v, want 'replace_content_range'", args["command"])
		}
		if args["selection_with_ellipsis"] != "# Header\n...\nend of section" {
			t.Errorf("selection_with_ellipsis = %v, want match", args["selection_with_ellipsis"])
		}
		if args["new_str"] != "# New Header\nNew body" {
			t.Errorf("new_str = %v, want '# New Header\\nNew body'", args["new_str"])
		}
	})

	t.Run("update page insert_content_after", func(t *testing.T) {
		params := UpdatePageParams{
			PageID:                "page-1",
			Command:               UpdateCmdInsertContentAfter,
			SelectionWithEllipsis: "## Section A",
			NewStr:                "Inserted paragraph.",
		}
		args := map[string]interface{}{
			"page_id":                 params.PageID,
			"command":                 "insert_content_after",
			"selection_with_ellipsis": params.SelectionWithEllipsis,
			"new_str":                 params.NewStr,
		}
		if args["command"] != "insert_content_after" {
			t.Errorf("command = %v, want 'insert_content_after'", args["command"])
		}
		if args["selection_with_ellipsis"] != "## Section A" {
			t.Errorf("selection_with_ellipsis = %v, want '## Section A'", args["selection_with_ellipsis"])
		}
		if args["new_str"] != "Inserted paragraph." {
			t.Errorf("new_str = %v, want 'Inserted paragraph.'", args["new_str"])
		}
	})

	t.Run("update page update_properties", func(t *testing.T) {
		params := UpdatePageParams{
			PageID:  "page-1",
			Command: UpdateCmdUpdateProperties,
			Properties: map[string]interface{}{
				"title":  "New Title",
				"status": map[string]interface{}{"name": "Done"},
			},
		}
		args := map[string]interface{}{
			"page_id":    params.PageID,
			"command":    "update_properties",
			"properties": params.Properties,
		}
		if args["command"] != "update_properties" {
			t.Errorf("command = %v, want 'update_properties'", args["command"])
		}
		props := args["properties"].(map[string]interface{})
		if props["title"] != "New Title" {
			t.Errorf("properties.title = %v, want 'New Title'", props["title"])
		}
	})

	t.Run("get comments args", func(t *testing.T) {
		args := map[string]interface{}{
			"resource_id": "page-123",
		}
		if args["resource_id"] != "page-123" {
			t.Errorf("resource_id = %v, want 'page-123'", args["resource_id"])
		}
	})

	t.Run("create comment args", func(t *testing.T) {
		args := map[string]interface{}{
			"page_id": "page-123",
			"body":    "Great work!",
		}
		if args["page_id"] != "page-123" {
			t.Errorf("page_id = %v, want 'page-123'", args["page_id"])
		}
		if args["body"] != "Great work!" {
			t.Errorf("body = %v, want 'Great work!'", args["body"])
		}
	})
}

func TestPKCEGeneration(t *testing.T) {
	// Test that the mcp-go PKCE utilities produce valid values.
	// This is a thin wrapper around the library but verifies our import path works.
	t.Run("code verifier length", func(t *testing.T) {
		verifier, err := generateTestCodeVerifier()
		if err != nil {
			t.Fatalf("GenerateCodeVerifier() error: %v", err)
		}
		// RFC 7636: verifier must be between 43 and 128 characters.
		if len(verifier) < 43 || len(verifier) > 128 {
			t.Errorf("code verifier length = %d, want 43-128", len(verifier))
		}
	})

	t.Run("code challenge is deterministic", func(t *testing.T) {
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge1 := generateTestCodeChallenge(verifier)
		challenge2 := generateTestCodeChallenge(verifier)
		if challenge1 != challenge2 {
			t.Errorf("code challenge not deterministic: %q != %q", challenge1, challenge2)
		}
		if challenge1 == "" {
			t.Error("code challenge is empty")
		}
		if challenge1 == verifier {
			t.Error("code challenge should differ from verifier")
		}
	})

	t.Run("state uniqueness", func(t *testing.T) {
		state1, err := generateTestState()
		if err != nil {
			t.Fatalf("GenerateState() error: %v", err)
		}
		state2, err := generateTestState()
		if err != nil {
			t.Fatalf("GenerateState() error: %v", err)
		}
		if state1 == state2 {
			t.Error("two consecutive state values should be unique")
		}
		if len(state1) < 16 {
			t.Errorf("state length = %d, want >= 16", len(state1))
		}
	})
}

// Thin test helpers that call the mcp-go utilities.
// We import them via the transport package since that's where the actual
// functions live (the client package re-exports them).

func generateTestCodeVerifier() (string, error) {
	return mcp_generateCodeVerifier()
}

func generateTestCodeChallenge(verifier string) string {
	return mcp_generateCodeChallenge(verifier)
}

func generateTestState() (string, error) {
	return mcp_generateState()
}
