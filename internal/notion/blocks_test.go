package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBlock_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/blocks/block123" {
			t.Errorf("expected path /blocks/block123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":       "block",
			"id":           "block123",
			"type":         "paragraph",
			"has_children": false,
			"archived":     false,
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	block, err := client.GetBlock(ctx, "block123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.ID != "block123" {
		t.Errorf("expected ID 'block123', got %q", block.ID)
	}
	if block.Type != "paragraph" {
		t.Errorf("expected type 'paragraph', got %q", block.Type)
	}
}

func TestGetBlock_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetBlock(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetBlockChildren_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/blocks/block123/children" {
			t.Errorf("expected path /blocks/block123/children, got %s", r.URL.Path)
		}

		// Check query parameters if provided
		if r.URL.Query().Get("page_size") != "" {
			if r.URL.Query().Get("page_size") != "50" {
				t.Errorf("expected page_size=50, got %s", r.URL.Query().Get("page_size"))
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object": "block",
					"id":     "child1",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []interface{}{},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &BlockChildrenOptions{
		PageSize: 50,
	}

	blockList, err := client.GetBlockChildren(ctx, "block123", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blockList.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(blockList.Results))
	}
	if blockList.Results[0].ID != "child1" {
		t.Errorf("expected child ID 'child1', got %q", blockList.Results[0].ID)
	}
}

func TestGetBlockChildren_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetBlockChildren(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetBlockChildren_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start_cursor") != "cursor123" {
			t.Errorf("expected start_cursor=cursor123, got %s", r.URL.Query().Get("start_cursor"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":      "list",
			"results":     []interface{}{},
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &BlockChildrenOptions{
		StartCursor: "cursor123",
	}

	_, err := client.GetBlockChildren(ctx, "block123", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendBlockChildren_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/blocks/block123/children" {
			t.Errorf("expected path /blocks/block123/children, got %s", r.URL.Path)
		}

		var req AppendBlockChildrenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Children) == 0 {
			t.Error("expected children to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":      "list",
			"results":     req.Children,
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &AppendBlockChildrenRequest{
		Children: []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []interface{}{},
				},
			},
		},
	}

	blockList, err := client.AppendBlockChildren(ctx, "block123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blockList.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(blockList.Results))
	}
}

func TestAppendBlockChildren_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &AppendBlockChildrenRequest{
		Children: []map[string]interface{}{},
	}

	_, err := client.AppendBlockChildren(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestAppendBlockChildren_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.AppendBlockChildren(ctx, "block123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "append block children request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestAppendBlockChildren_EmptyChildren(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &AppendBlockChildrenRequest{
		Children: []map[string]interface{}{},
	}

	_, err := client.AppendBlockChildren(ctx, "block123", req)
	if err == nil {
		t.Fatal("expected error for empty children")
	}

	expected := "children are required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdateBlock_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/blocks/block123" {
			t.Errorf("expected path /blocks/block123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "block",
			"id":       "block123",
			"type":     "paragraph",
			"archived": false,
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &UpdateBlockRequest{
		Content: map[string]interface{}{
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		},
	}

	block, err := client.UpdateBlock(ctx, "block123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.ID != "block123" {
		t.Errorf("expected ID 'block123', got %q", block.ID)
	}
}

func TestUpdateBlock_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &UpdateBlockRequest{
		Content: map[string]interface{}{},
	}

	_, err := client.UpdateBlock(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdateBlock_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.UpdateBlock(ctx, "block123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "update block request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestDeleteBlock_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/blocks/block123" {
			t.Errorf("expected path /blocks/block123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "block",
			"id":       "block123",
			"type":     "paragraph",
			"archived": true,
			"paragraph": map[string]interface{}{
				"rich_text": []interface{}{},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	block, err := client.DeleteBlock(ctx, "block123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if block.ID != "block123" {
		t.Errorf("expected ID 'block123', got %q", block.ID)
	}
	if !block.Archived {
		t.Error("expected block to be archived")
	}
}

func TestDeleteBlock_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.DeleteBlock(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}
