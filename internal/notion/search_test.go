package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("expected path /search, got %s", r.URL.Path)
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object: "list",
			Results: []map[string]interface{}{
				{
					"object": "page",
					"id":     "page123",
				},
			},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query: "test query",
	}

	result, err := client.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestSearch_NilRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Verify empty request is sent
		if req.Query != "" {
			t.Errorf("expected empty query, got %q", req.Query)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object:  "list",
			Results: []map[string]interface{}{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// nil request should be handled gracefully
	result, err := client.Search(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
}

func TestSearch_WithFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Filter == nil {
			t.Error("expected filter to be set")
		}

		filterValue, ok := req.Filter["value"]
		if !ok || filterValue != "page" {
			t.Errorf("expected filter value 'page', got %v", filterValue)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object:  "list",
			Results: []map[string]interface{}{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query: "test",
		Filter: map[string]interface{}{
			"property": "object",
			"value":    "page",
		},
	}

	_, err := client.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_WithSort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Sort == nil {
			t.Error("expected sort to be set")
		}

		direction, ok := req.Sort["direction"]
		if !ok || direction != "descending" {
			t.Errorf("expected sort direction 'descending', got %v", direction)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object:  "list",
			Results: []map[string]interface{}{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query: "test",
		Sort: map[string]interface{}{
			"direction": "descending",
			"timestamp": "last_edited_time",
		},
	}

	_, err := client.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.StartCursor != "cursor123" {
			t.Errorf("expected start_cursor 'cursor123', got %q", req.StartCursor)
		}
		if req.PageSize != 50 {
			t.Errorf("expected page_size 50, got %d", req.PageSize)
		}

		w.WriteHeader(http.StatusOK)
		nextCursor := "nextcursor456"
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object:     "list",
			Results:    []map[string]interface{}{},
			HasMore:    true,
			NextCursor: &nextCursor,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query:       "test",
		StartCursor: "cursor123",
		PageSize:    50,
	}

	result, err := client.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasMore {
		t.Error("expected has_more to be true")
	}
	if result.NextCursor == nil || *result.NextCursor != "nextcursor456" {
		t.Errorf("expected next_cursor 'nextcursor456', got %v", result.NextCursor)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Empty query is valid - returns all pages
		if req.Query != "" {
			t.Errorf("expected empty query, got %q", req.Query)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResult{
			Object:  "list",
			Results: []map[string]interface{}{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query: "",
	}

	result, err := client.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
}

func TestSearch_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  401,
			Code:    "unauthorized",
			Message: "Invalid API token",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &SearchRequest{
		Query: "test",
	}

	_, err := client.Search(ctx, req)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}
