package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/pages/page123" {
			t.Errorf("expected path /pages/page123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object:     "page",
			ID:         "page123",
			Properties: map[string]interface{}{"title": "Test Page"},
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page, err := client.GetPage(ctx, "page123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ID != "page123" {
		t.Errorf("expected ID 'page123', got %q", page.ID)
	}
	if page.Object != "page" {
		t.Errorf("expected object 'page', got %q", page.Object)
	}
}

func TestGetPage_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetPage(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty page ID")
	}

	expected := "page ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetPage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "Could not find page with ID.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.GetPage(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent page")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestCreatePage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/pages" {
			t.Errorf("expected path /pages, got %s", r.URL.Path)
		}

		var req CreatePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent == nil {
			t.Error("expected parent to be set")
		}
		if req.Properties == nil {
			t.Error("expected properties to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object:     "page",
			ID:         "newpage123",
			Properties: req.Properties,
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreatePageRequest{
		Parent:     map[string]interface{}{"page_id": "parent123"},
		Properties: map[string]interface{}{"title": "New Page"},
	}

	page, err := client.CreatePage(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ID != "newpage123" {
		t.Errorf("expected ID 'newpage123', got %q", page.ID)
	}
}

func TestCreatePage_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CreatePage(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "create page request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreatePage_MissingParent(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreatePageRequest{
		Properties: map[string]interface{}{"title": "New Page"},
	}

	_, err := client.CreatePage(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing parent")
	}

	expected := "parent is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreatePage_MissingProperties(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreatePageRequest{
		Parent: map[string]interface{}{"page_id": "parent123"},
	}

	_, err := client.CreatePage(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing properties")
	}

	expected := "properties are required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdatePage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/pages/page123" {
			t.Errorf("expected path /pages/page123, got %s", r.URL.Path)
		}

		var req UpdatePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object:     "page",
			ID:         "page123",
			Properties: req.Properties,
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &UpdatePageRequest{
		Properties: map[string]interface{}{"title": "Updated Page"},
	}

	page, err := client.UpdatePage(ctx, "page123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ID != "page123" {
		t.Errorf("expected ID 'page123', got %q", page.ID)
	}
}

func TestUpdatePage_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &UpdatePageRequest{
		Properties: map[string]interface{}{"title": "Updated Page"},
	}

	_, err := client.UpdatePage(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty page ID")
	}

	expected := "page ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdatePage_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.UpdatePage(ctx, "page123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "update page request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdatePage_Archive(t *testing.T) {
	archived := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req UpdatePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Archived == nil || *req.Archived != true {
			t.Error("expected archived to be true")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object:   "page",
			ID:       "page123",
			Archived: *req.Archived,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &UpdatePageRequest{
		Archived: &archived,
	}

	page, err := client.UpdatePage(ctx, "page123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !page.Archived {
		t.Error("expected page to be archived")
	}
}

func TestGetPageProperty_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/pages/page123/properties/prop123" {
			t.Errorf("expected path /pages/page123/properties/prop123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "property_item",
			"id":     "prop123",
			"type":   "title",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	result, err := client.GetPageProperty(ctx, "page123", "prop123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["id"] != "prop123" {
		t.Errorf("expected property ID 'prop123', got %v", result["id"])
	}
}

func TestGetPageProperty_EmptyPageID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetPageProperty(ctx, "", "prop123")
	if err == nil {
		t.Fatal("expected error for empty page ID")
	}

	expected := "page ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetPageProperty_EmptyPropertyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetPageProperty(ctx, "page123", "")
	if err == nil {
		t.Fatal("expected error for empty property ID")
	}

	expected := "property ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestMovePage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/pages/page123" {
			t.Errorf("expected path /pages/page123, got %s", r.URL.Path)
		}

		var req MovePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent == nil {
			t.Error("expected parent to be set")
		}
		if req.Parent["page_id"] != "newparent123" {
			t.Errorf("expected parent page_id 'newparent123', got %v", req.Parent["page_id"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object:   "page",
			ID:       "page123",
			Parent:   req.Parent,
			Archived: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &MovePageRequest{
		Parent: map[string]interface{}{"page_id": "newparent123"},
	}

	page, err := client.MovePage(ctx, "page123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ID != "page123" {
		t.Errorf("expected ID 'page123', got %q", page.ID)
	}
}

func TestMovePage_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &MovePageRequest{
		Parent: map[string]interface{}{"page_id": "newparent123"},
	}

	_, err := client.MovePage(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty page ID")
	}

	expected := "page ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestMovePage_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.MovePage(ctx, "page123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "move page request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestMovePage_MissingParent(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &MovePageRequest{}

	_, err := client.MovePage(ctx, "page123", req)
	if err == nil {
		t.Fatal("expected error for missing parent")
	}

	expected := "parent is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestMovePage_ToDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req MovePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent["database_id"] != "db123" {
			t.Errorf("expected parent database_id 'db123', got %v", req.Parent["database_id"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object: "page",
			ID:     "page123",
			Parent: req.Parent,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &MovePageRequest{
		Parent: map[string]interface{}{"database_id": "db123"},
	}

	page, err := client.MovePage(ctx, "page123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ID != "page123" {
		t.Errorf("expected ID 'page123', got %q", page.ID)
	}
}

func TestMovePage_WithAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req MovePageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.After != "block456" {
			t.Errorf("expected after 'block456', got %q", req.After)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Page{
			Object: "page",
			ID:     "page123",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &MovePageRequest{
		Parent: map[string]interface{}{"page_id": "newparent123"},
		After:  "block456",
	}

	_, err := client.MovePage(ctx, "page123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
