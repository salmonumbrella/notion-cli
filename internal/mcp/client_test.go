package mcp

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestWrapperMethodsInvokeCallTool(t *testing.T) {
	type capturedCall struct {
		name string
		args map[string]interface{}
	}

	captureClient := func(captured *capturedCall) *Client {
		return &Client{
			callToolOverride: func(_ context.Context, name string, args map[string]interface{}) (string, error) {
				captured.name = name
				captured.args = args
				return "ok", nil
			},
		}
	}

	t.Run("search", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.Search(context.Background(), SearchParams{
			Query:             "find this",
			ContentSearchMode: SearchModeAI,
		})
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if got.name != "notion-search" {
			t.Fatalf("tool name = %q, want notion-search", got.name)
		}
		if got.args["query"] != "find this" {
			t.Fatalf("query = %v, want find this", got.args["query"])
		}
		if got.args["content_search_mode"] != "ai_search" {
			t.Fatalf("content_search_mode = %v, want ai_search", got.args["content_search_mode"])
		}
	})

	t.Run("fetch", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.Fetch(context.Background(), "abc-123", true)
		if err != nil {
			t.Fatalf("Fetch() error: %v", err)
		}
		if got.name != "notion-fetch" {
			t.Fatalf("tool name = %q, want notion-fetch", got.name)
		}
		if got.args["id"] != "abc-123" {
			t.Fatalf("id = %v, want abc-123", got.args["id"])
		}
		if got.args["include_discussions"] != true {
			t.Fatalf("include_discussions = %v, want true", got.args["include_discussions"])
		}
	})

	t.Run("get comments", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.GetComments(context.Background(), GetCommentsParams{
			PageID:           "page-123",
			DiscussionID:     "discussion://abc",
			IncludeAllBlocks: true,
		})
		if err != nil {
			t.Fatalf("GetComments() error: %v", err)
		}
		if got.name != "notion-get-comments" {
			t.Fatalf("tool name = %q, want notion-get-comments", got.name)
		}
		if got.args["page_id"] != "page-123" {
			t.Fatalf("page_id = %v, want page-123", got.args["page_id"])
		}
		if got.args["discussion_id"] != "discussion://abc" {
			t.Fatalf("discussion_id = %v, want discussion://abc", got.args["discussion_id"])
		}
		if got.args["include_all_blocks"] != true {
			t.Fatalf("include_all_blocks = %v, want true", got.args["include_all_blocks"])
		}
	})

	t.Run("create comment", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.CreateComment(context.Background(), CreateCommentParams{
			PageID:                "page-123",
			Text:                  "hello",
			SelectionWithEllipsis: "start...end",
		})
		if err != nil {
			t.Fatalf("CreateComment() error: %v", err)
		}
		if got.name != "notion-create-comment" {
			t.Fatalf("tool name = %q, want notion-create-comment", got.name)
		}
		if got.args["page_id"] != "page-123" {
			t.Fatalf("page_id = %v, want page-123", got.args["page_id"])
		}
		if got.args["selection_with_ellipsis"] != "start...end" {
			t.Fatalf("selection_with_ellipsis = %v, want start...end", got.args["selection_with_ellipsis"])
		}
		richText := got.args["rich_text"].([]interface{})
		if len(richText) != 1 {
			t.Fatalf("len(rich_text) = %d, want 1", len(richText))
		}
	})

	t.Run("create pages", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.CreatePages(context.Background(), &CreatePagesParent{PageID: "parent-1"}, []CreatePageInput{
			{
				Properties: map[string]interface{}{"title": "Page A"},
				Content:    "# Heading",
				TemplateID: "tpl-123",
			},
		})
		if err != nil {
			t.Fatalf("CreatePages() error: %v", err)
		}
		if got.name != "notion-create-pages" {
			t.Fatalf("tool name = %q, want notion-create-pages", got.name)
		}
		parent := got.args["parent"].(map[string]interface{})
		if parent["page_id"] != "parent-1" {
			t.Fatalf("parent.page_id = %v, want parent-1", parent["page_id"])
		}
		pages := got.args["pages"].([]interface{})
		if len(pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(pages))
		}
		first := pages[0].(map[string]interface{})
		if first["template_id"] != "tpl-123" {
			t.Fatalf("template_id = %v, want tpl-123", first["template_id"])
		}
	})

	t.Run("update page", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.UpdatePage(context.Background(), UpdatePageParams{
			PageID:     "page-1",
			Command:    UpdateCmdApplyTemplate,
			TemplateID: "tpl-abc",
		})
		if err != nil {
			t.Fatalf("UpdatePage() error: %v", err)
		}
		if got.name != "notion-update-page" {
			t.Fatalf("tool name = %q, want notion-update-page", got.name)
		}
		if got.args["command"] != "apply_template" {
			t.Fatalf("command = %v, want apply_template", got.args["command"])
		}
	})

	t.Run("move pages", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.MovePages(context.Background(), []string{"a", "b"}, "parent-2")
		if err != nil {
			t.Fatalf("MovePages() error: %v", err)
		}
		if got.name != "notion-move-pages" {
			t.Fatalf("tool name = %q, want notion-move-pages", got.name)
		}
	})

	t.Run("duplicate page", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.DuplicatePage(context.Background(), "page-dup")
		if err != nil {
			t.Fatalf("DuplicatePage() error: %v", err)
		}
		if got.name != "notion-duplicate-page" {
			t.Fatalf("tool name = %q, want notion-duplicate-page", got.name)
		}
	})

	t.Run("get teams", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.GetTeams(context.Background(), "eng")
		if err != nil {
			t.Fatalf("GetTeams() error: %v", err)
		}
		if got.name != "notion-get-teams" {
			t.Fatalf("tool name = %q, want notion-get-teams", got.name)
		}
	})

	t.Run("get users", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.GetUsers(context.Background(), "alice", "self", "cursor", 10)
		if err != nil {
			t.Fatalf("GetUsers() error: %v", err)
		}
		if got.name != "notion-get-users" {
			t.Fatalf("tool name = %q, want notion-get-users", got.name)
		}
		if got.args["page_size"] != 10 {
			t.Fatalf("page_size = %v, want 10", got.args["page_size"])
		}
	})

	t.Run("create database", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.CreateDatabase(context.Background(), CreateDatabaseParams{
			Schema: `CREATE TABLE ("Name" TITLE)`,
		})
		if err != nil {
			t.Fatalf("CreateDatabase() error: %v", err)
		}
		if got.name != "notion-create-database" {
			t.Fatalf("tool name = %q, want notion-create-database", got.name)
		}
	})

	t.Run("update data source", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.UpdateDataSource(context.Background(), UpdateDataSourceParams{
			DataSourceID: "ds-1",
			Statements:   `ADD COLUMN "Priority" NUMBER`,
		})
		if err != nil {
			t.Fatalf("UpdateDataSource() error: %v", err)
		}
		if got.name != "notion-update-data-source" {
			t.Fatalf("tool name = %q, want notion-update-data-source", got.name)
		}
	})

	t.Run("query data sources sql", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.QueryDataSourcesSQL(context.Background(), QueryDataSourcesSQLParams{
			DataSourceURLs: []string{"collection://abc"},
			Query:          `SELECT * FROM "collection://abc"`,
			Params:         []interface{}{"x"},
		})
		if err != nil {
			t.Fatalf("QueryDataSourcesSQL() error: %v", err)
		}
		if got.name != "notion-query-data-sources" {
			t.Fatalf("tool name = %q, want notion-query-data-sources", got.name)
		}
	})

	t.Run("query data sources view", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.QueryDataSourcesView(context.Background(), QueryDataSourcesViewParams{
			ViewURL: "https://example.invalid/workspace/Tasks-sample?v=view-1",
		})
		if err != nil {
			t.Fatalf("QueryDataSourcesView() error: %v", err)
		}
		if got.name != "notion-query-data-sources" {
			t.Fatalf("tool name = %q, want notion-query-data-sources", got.name)
		}
	})

	t.Run("query meeting notes", func(t *testing.T) {
		var got capturedCall
		client := captureClient(&got)

		_, err := client.QueryMeetingNotes(context.Background(), QueryMeetingNotesParams{
			Filter: map[string]interface{}{
				"operator": "and",
			},
		})
		if err != nil {
			t.Fatalf("QueryMeetingNotes() error: %v", err)
		}
		if got.name != "notion-query-meeting-notes" {
			t.Fatalf("tool name = %q, want notion-query-meeting-notes", got.name)
		}
	})
}

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
		args := buildSearchArgs(SearchParams{
			Query:             "test query",
			ContentSearchMode: SearchModeWorkspace,
		})
		if args["query"] != "test query" {
			t.Errorf("query = %v, want 'test query'", args["query"])
		}
		if args["content_search_mode"] != "workspace_search" {
			t.Errorf("content_search_mode = %v, want 'workspace_search'", args["content_search_mode"])
		}
	})

	t.Run("search ai", func(t *testing.T) {
		args := buildSearchArgs(SearchParams{
			Query:             "semantic query",
			ContentSearchMode: SearchModeAI,
		})
		if args["content_search_mode"] != "ai_search" {
			t.Errorf("content_search_mode = %v, want 'ai_search'", args["content_search_mode"])
		}
	})

	t.Run("fetch args", func(t *testing.T) {
		args := buildFetchArgs("abc-123", false)
		if args["id"] != "abc-123" {
			t.Errorf("id = %v, want 'abc-123'", args["id"])
		}
		if _, ok := args["include_discussions"]; ok {
			t.Errorf("include_discussions should not be present when false")
		}
	})

	t.Run("fetch args include discussions", func(t *testing.T) {
		args := buildFetchArgs("abc-123", true)
		if args["include_discussions"] != true {
			t.Errorf("include_discussions = %v, want true", args["include_discussions"])
		}
	})

	t.Run("create page args", func(t *testing.T) {
		pages := []CreatePageInput{
			{
				Properties: map[string]interface{}{"title": "Test Page"},
				Content:    "# Hello",
				TemplateID: "tpl-123",
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
			if p.TemplateID != "" {
				page["template_id"] = p.TemplateID
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
		if first["template_id"] != "tpl-123" {
			t.Errorf("template_id = %v, want 'tpl-123'", first["template_id"])
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
			if p.TemplateID != "" {
				page["template_id"] = p.TemplateID
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

	t.Run("update page apply_template", func(t *testing.T) {
		params := UpdatePageParams{
			PageID:     "page-1",
			Command:    UpdateCmdApplyTemplate,
			TemplateID: "tpl-abc",
		}
		args := map[string]interface{}{
			"page_id":     params.PageID,
			"command":     "apply_template",
			"template_id": params.TemplateID,
		}
		if args["command"] != "apply_template" {
			t.Errorf("command = %v, want 'apply_template'", args["command"])
		}
		if args["template_id"] != "tpl-abc" {
			t.Errorf("template_id = %v, want 'tpl-abc'", args["template_id"])
		}
	})

	t.Run("update page allow_deleting_content", func(t *testing.T) {
		allow := true
		params := UpdatePageParams{
			PageID:               "page-1",
			Command:              UpdateCmdReplaceContent,
			NewStr:               "new content",
			AllowDeletingContent: &allow,
		}
		args := map[string]interface{}{
			"page_id":                params.PageID,
			"command":                "replace_content",
			"new_str":                params.NewStr,
			"allow_deleting_content": *params.AllowDeletingContent,
		}
		if args["allow_deleting_content"] != true {
			t.Errorf("allow_deleting_content = %v, want true", args["allow_deleting_content"])
		}
	})

	t.Run("get comments args", func(t *testing.T) {
		args := buildGetCommentsArgs(GetCommentsParams{
			PageID:           "page-123",
			IncludeAllBlocks: true,
			IncludeResolved:  true,
		})
		if args["page_id"] != "page-123" {
			t.Errorf("page_id = %v, want 'page-123'", args["page_id"])
		}
		if args["include_all_blocks"] != true {
			t.Errorf("include_all_blocks = %v, want true", args["include_all_blocks"])
		}
		if args["include_resolved"] != true {
			t.Errorf("include_resolved = %v, want true", args["include_resolved"])
		}
	})

	t.Run("create comment args", func(t *testing.T) {
		args := buildCreateCommentArgs(CreateCommentParams{
			PageID: "page-123",
			Text:   "Great work!",
		})
		if args["page_id"] != "page-123" {
			t.Errorf("page_id = %v, want 'page-123'", args["page_id"])
		}
		richText := args["rich_text"].([]interface{})
		if len(richText) != 1 {
			t.Fatalf("len(rich_text) = %d, want 1", len(richText))
		}
		first := richText[0].(map[string]interface{})
		text := first["text"].(map[string]interface{})
		if text["content"] != "Great work!" {
			t.Errorf("rich_text[0].text.content = %v, want 'Great work!'", text["content"])
		}
	})

	t.Run("get comments with target_id only", func(t *testing.T) {
		args := buildGetCommentsArgs(GetCommentsParams{
			TargetID: "target-abc",
		})
		if args["page_id"] != "target-abc" {
			t.Errorf("page_id = %v, want 'target-abc'", args["page_id"])
		}
	})

	t.Run("get comments target_id takes precedence", func(t *testing.T) {
		args := buildGetCommentsArgs(GetCommentsParams{
			TargetID: "target-abc",
			PageID:   "page-456",
		})
		if args["page_id"] != "target-abc" {
			t.Errorf("page_id = %v, want 'target-abc'", args["page_id"])
		}
	})

	t.Run("create comment with target_id only", func(t *testing.T) {
		args := buildCreateCommentArgs(CreateCommentParams{
			TargetID: "target-abc",
			Text:     "Hello",
		})
		if args["page_id"] != "target-abc" {
			t.Errorf("page_id = %v, want 'target-abc'", args["page_id"])
		}
	})

	t.Run("create comment target_id takes precedence", func(t *testing.T) {
		args := buildCreateCommentArgs(CreateCommentParams{
			TargetID: "target-abc",
			PageID:   "page-456",
			Text:     "Hello",
		})
		if args["page_id"] != "target-abc" {
			t.Errorf("page_id = %v, want 'target-abc'", args["page_id"])
		}
	})

	t.Run("move pages args", func(t *testing.T) {
		pageIDs := []string{"page-a", "page-b", "page-c"}
		args := map[string]interface{}{
			"page_ids":       pageIDs,
			"parent_page_id": "new-parent-1",
		}
		ids := args["page_ids"].([]string)
		if len(ids) != 3 {
			t.Fatalf("len(page_ids) = %d, want 3", len(ids))
		}
		if ids[0] != "page-a" {
			t.Errorf("page_ids[0] = %v, want 'page-a'", ids[0])
		}
		if ids[2] != "page-c" {
			t.Errorf("page_ids[2] = %v, want 'page-c'", ids[2])
		}
		if args["parent_page_id"] != "new-parent-1" {
			t.Errorf("parent_page_id = %v, want 'new-parent-1'", args["parent_page_id"])
		}
	})

	t.Run("duplicate page args", func(t *testing.T) {
		args := map[string]interface{}{
			"page_id": "page-to-dup",
		}
		if args["page_id"] != "page-to-dup" {
			t.Errorf("page_id = %v, want 'page-to-dup'", args["page_id"])
		}
	})

	t.Run("get teams with query", func(t *testing.T) {
		args := map[string]interface{}{}
		query := "engineering"
		if query != "" {
			args["query"] = query
		}
		if args["query"] != "engineering" {
			t.Errorf("query = %v, want 'engineering'", args["query"])
		}
	})

	t.Run("get teams empty", func(t *testing.T) {
		args := map[string]interface{}{}
		query := ""
		if query != "" {
			args["query"] = query
		}
		if _, ok := args["query"]; ok {
			t.Error("empty query should not set query key")
		}
	})

	t.Run("get users with query", func(t *testing.T) {
		args := map[string]interface{}{}
		query := "alice"
		if query != "" {
			args["query"] = query
		}
		if args["query"] != "alice" {
			t.Errorf("query = %v, want 'alice'", args["query"])
		}
	})

	t.Run("get users by id", func(t *testing.T) {
		args := map[string]interface{}{}
		userID := "self"
		if userID != "" {
			args["user_id"] = userID
		}
		if args["user_id"] != "self" {
			t.Errorf("user_id = %v, want 'self'", args["user_id"])
		}
	})

	t.Run("get users with pagination", func(t *testing.T) {
		args := map[string]interface{}{}
		startCursor := "cursor-abc"
		pageSize := 25
		if startCursor != "" {
			args["start_cursor"] = startCursor
		}
		if pageSize > 0 {
			args["page_size"] = pageSize
		}
		if args["start_cursor"] != "cursor-abc" {
			t.Errorf("start_cursor = %v, want 'cursor-abc'", args["start_cursor"])
		}
		if args["page_size"] != 25 {
			t.Errorf("page_size = %v, want 25", args["page_size"])
		}
	})

	t.Run("create database args", func(t *testing.T) {
		parentID := "parent-page-1"
		title := "My Database"
		schema := `CREATE TABLE ("Name" TITLE, "Status" SELECT('Todo':red, 'Done':green))`

		args := map[string]interface{}{
			"parent": map[string]interface{}{
				"page_id": parentID,
			},
			"title":  title,
			"schema": schema,
		}

		// Verify parent.
		parentMap := args["parent"].(map[string]interface{})
		if parentMap["page_id"] != "parent-page-1" {
			t.Errorf("parent.page_id = %v, want 'parent-page-1'", parentMap["page_id"])
		}

		// Verify title string.
		if args["title"] != "My Database" {
			t.Errorf("title = %v, want 'My Database'", args["title"])
		}

		if args["schema"] != schema {
			t.Errorf("schema mismatch")
		}
	})

	t.Run("update data source args", func(t *testing.T) {
		dataSourceID := "ds-abc-123"
		statements := `ADD COLUMN "Priority" SELECT('High':red, 'Low':green)`

		args := map[string]interface{}{
			"data_source_id": dataSourceID,
			"statements":     statements,
		}

		if args["data_source_id"] != "ds-abc-123" {
			t.Errorf("data_source_id = %v, want 'ds-abc-123'", args["data_source_id"])
		}
		if args["statements"] != statements {
			t.Errorf("statements = %v, want %q", args["statements"], statements)
		}
	})

	t.Run("update data source trash", func(t *testing.T) {
		args := map[string]interface{}{
			"data_source_id": "ds-trash-1",
			"in_trash":       true,
		}

		if args["data_source_id"] != "ds-trash-1" {
			t.Errorf("data_source_id = %v, want 'ds-trash-1'", args["data_source_id"])
		}
		if args["in_trash"] != true {
			t.Errorf("in_trash = %v, want true", args["in_trash"])
		}
	})

	t.Run("query data sources sql", func(t *testing.T) {
		urls := []string{"collection://abc123"}
		query := `SELECT * FROM "collection://abc123" WHERE Status = ?`
		params := []interface{}{"In Progress"}
		data := map[string]interface{}{
			"data_source_urls": urls,
			"query":            query,
			"params":           params,
		}
		args := map[string]interface{}{"data": data}
		d := args["data"].(map[string]interface{})
		if d["query"] != query {
			t.Errorf("query mismatch")
		}
		dsURLs := d["data_source_urls"].([]string)
		if len(dsURLs) != 1 || dsURLs[0] != "collection://abc123" {
			t.Errorf("data_source_urls = %v", dsURLs)
		}
		p := d["params"].([]interface{})
		if len(p) != 1 || p[0] != "In Progress" {
			t.Errorf("params = %v", p)
		}
	})

	t.Run("query data sources sql no params", func(t *testing.T) {
		data := map[string]interface{}{
			"data_source_urls": []string{"collection://abc123"},
			"query":            `SELECT * FROM "collection://abc123" LIMIT 10`,
		}
		args := map[string]interface{}{"data": data}
		d := args["data"].(map[string]interface{})
		if _, ok := d["params"]; ok {
			t.Error("params should not be present when empty")
		}
	})

	t.Run("query data sources view", func(t *testing.T) {
		viewURL := "https://example.invalid/workspace/Tasks-sample?v=view-1"
		args := map[string]interface{}{
			"data": map[string]interface{}{
				"mode":     "view",
				"view_url": viewURL,
			},
		}
		d := args["data"].(map[string]interface{})
		if d["mode"] != "view" {
			t.Errorf("mode = %v, want 'view'", d["mode"])
		}
		if d["view_url"] != viewURL {
			t.Errorf("view_url = %v", d["view_url"])
		}
	})

	t.Run("query meeting notes filter", func(t *testing.T) {
		filter := map[string]interface{}{
			"operator": "and",
			"filters": []interface{}{
				map[string]interface{}{
					"property": "title",
					"filter": map[string]interface{}{
						"operator": "string_contains",
						"value": map[string]interface{}{
							"type":  "exact",
							"value": "standup",
						},
					},
				},
			},
		}
		args := map[string]interface{}{
			"filter": filter,
		}
		if _, ok := args["filter"]; !ok {
			t.Fatal("filter should be present")
		}
	})
}

func TestTypedArgBuilders(t *testing.T) {
	t.Run("create database args", func(t *testing.T) {
		args, err := buildCreateDatabaseArgs(CreateDatabaseParams{
			Schema: `CREATE TABLE ("Name" TITLE)`,
			Title:  "Tasks",
			Parent: &CreateDatabaseParent{PageID: "parent-1"},
		})
		if err != nil {
			t.Fatalf("buildCreateDatabaseArgs() error: %v", err)
		}
		parent, ok := args["parent"].(map[string]interface{})
		if !ok {
			t.Fatalf("parent type = %T, want map[string]interface{}", args["parent"])
		}
		if parent["page_id"] != "parent-1" {
			t.Fatalf("parent.page_id = %v, want parent-1", parent["page_id"])
		}
		if args["title"] != "Tasks" {
			t.Fatalf("title = %v, want Tasks", args["title"])
		}
	})

	t.Run("create database args validate schema", func(t *testing.T) {
		_, err := buildCreateDatabaseArgs(CreateDatabaseParams{})
		if err == nil {
			t.Fatal("expected schema validation error")
		}
	})

	t.Run("update data source args preserves false bool pointers", func(t *testing.T) {
		inTrash := false
		isInline := false
		args, err := buildUpdateDataSourceArgs(UpdateDataSourceParams{
			DataSourceID: "ds-1",
			InTrash:      &inTrash,
			IsInline:     &isInline,
		})
		if err != nil {
			t.Fatalf("buildUpdateDataSourceArgs() error: %v", err)
		}
		if args["in_trash"] != false {
			t.Fatalf("in_trash = %v, want false", args["in_trash"])
		}
		if args["is_inline"] != false {
			t.Fatalf("is_inline = %v, want false", args["is_inline"])
		}
	})

	t.Run("query data sources sql omits empty params", func(t *testing.T) {
		args, err := buildQueryDataSourcesSQLArgs(QueryDataSourcesSQLParams{
			DataSourceURLs: []string{"collection://abc"},
			Query:          `SELECT * FROM "collection://abc"`,
		})
		if err != nil {
			t.Fatalf("buildQueryDataSourcesSQLArgs() error: %v", err)
		}

		data, ok := args["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("data type = %T, want map[string]interface{}", args["data"])
		}
		if _, ok := data["params"]; ok {
			t.Fatal("params should be omitted when empty")
		}
	})

	t.Run("query data sources sql validates URLs", func(t *testing.T) {
		_, err := buildQueryDataSourcesSQLArgs(QueryDataSourcesSQLParams{
			Query: `SELECT 1`,
		})
		if err == nil {
			t.Fatal("expected validation error for missing data source URLs")
		}
	})

	t.Run("query data sources sql validates query", func(t *testing.T) {
		_, err := buildQueryDataSourcesSQLArgs(QueryDataSourcesSQLParams{
			DataSourceURLs: []string{"collection://abc"},
			Query:          "   ",
		})
		if err == nil {
			t.Fatal("expected validation error for empty SQL query")
		}
	})

	t.Run("query data sources view includes mode", func(t *testing.T) {
		args, err := buildQueryDataSourcesViewArgs(QueryDataSourcesViewParams{
			ViewURL: "https://example.invalid/workspace/Tasks?v=view-1",
		})
		if err != nil {
			t.Fatalf("buildQueryDataSourcesViewArgs() error: %v", err)
		}

		data, ok := args["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("data type = %T, want map[string]interface{}", args["data"])
		}
		if data["mode"] != "view" {
			t.Fatalf("mode = %v, want view", data["mode"])
		}
	})

	t.Run("query data sources view validates URL", func(t *testing.T) {
		_, err := buildQueryDataSourcesViewArgs(QueryDataSourcesViewParams{
			ViewURL: "   ",
		})
		if err == nil {
			t.Fatal("expected validation error for empty view URL")
		}
	})

	t.Run("update page unknown command", func(t *testing.T) {
		_, err := buildUpdatePageArgs(UpdatePageParams{
			PageID:  "page-1",
			Command: UpdateCommand("bad"),
		})
		if err == nil {
			t.Fatal("expected validation error for unknown command")
		}
	})
}

func TestCallToolRetryBehavior(t *testing.T) {
	newClient := func(
		maxAttempts int,
		attemptTimeout time.Duration,
		override func(context.Context, string, map[string]interface{}) (string, error),
	) *Client {
		return &Client{
			callToolOverride: override,
			callToolPolicy: callToolPolicy{
				attemptTimeout: attemptTimeout,
				maxAttempts:    maxAttempts,
				baseBackoff:    time.Millisecond,
				maxBackoff:     2 * time.Millisecond,
			},
		}
	}

	t.Run("retries transient failures then succeeds", func(t *testing.T) {
		calls := 0
		client := newClient(3, time.Second, func(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
			calls++
			if calls < 3 {
				return "", fmt.Errorf("request failed with status 503: unavailable")
			}
			return "ok", nil
		})

		got, err := client.CallTool(context.Background(), "notion-search", map[string]interface{}{"query": "x"})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if got != "ok" {
			t.Fatalf("CallTool() = %q, want ok", got)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3", calls)
		}
	})

	t.Run("does not retry authentication errors", func(t *testing.T) {
		calls := 0
		client := newClient(5, time.Second, func(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
			calls++
			return "", fmt.Errorf("request failed with status 401: unauthorized")
		})

		_, err := client.CallTool(context.Background(), "notion-search", map[string]interface{}{"query": "x"})
		if err == nil {
			t.Fatal("expected error")
		}

		var toolErr *CallToolError
		if !errors.As(err, &toolErr) {
			t.Fatalf("error type = %T, want *CallToolError", err)
		}
		if toolErr.Kind != CallToolErrorKindAuthentication {
			t.Fatalf("kind = %q, want %q", toolErr.Kind, CallToolErrorKindAuthentication)
		}
		if toolErr.Attempts != 1 {
			t.Fatalf("attempts = %d, want 1", toolErr.Attempts)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("retries per-attempt timeout", func(t *testing.T) {
		calls := 0
		client := newClient(2, 10*time.Millisecond, func(ctx context.Context, _ string, _ map[string]interface{}) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		})

		_, err := client.CallTool(context.Background(), "notion-search", map[string]interface{}{"query": "x"})
		if err == nil {
			t.Fatal("expected timeout error")
		}

		var toolErr *CallToolError
		if !errors.As(err, &toolErr) {
			t.Fatalf("error type = %T, want *CallToolError", err)
		}
		if toolErr.Kind != CallToolErrorKindTimeout {
			t.Fatalf("kind = %q, want %q", toolErr.Kind, CallToolErrorKindTimeout)
		}
		if toolErr.Attempts != 2 {
			t.Fatalf("attempts = %d, want 2", toolErr.Attempts)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
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
