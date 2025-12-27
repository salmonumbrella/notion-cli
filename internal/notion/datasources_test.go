package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDataSource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources" {
			t.Errorf("expected path /data_sources, got %s", r.URL.Path)
		}

		var req CreateDataSourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent == nil {
			t.Error("expected parent to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSource{
			Object:     "data_source",
			ID:         "ds123",
			Properties: req.Properties,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateDataSourceRequest{
		Parent:     map[string]interface{}{"database_id": "db123"},
		Properties: map[string]interface{}{"Name": map[string]interface{}{"title": map[string]interface{}{}}},
	}

	ds, err := client.CreateDataSource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ds.ID != "ds123" {
		t.Errorf("expected ID 'ds123', got %q", ds.ID)
	}
}

func TestCreateDataSource_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CreateDataSource(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "parent is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateDataSource_MissingParent(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateDataSourceRequest{
		Properties: map[string]interface{}{"Name": "test"},
	}

	_, err := client.CreateDataSource(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing parent")
	}

	expected := "parent is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetDataSource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources/ds123" {
			t.Errorf("expected path /data_sources/ds123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSource{
			Object:         "data_source",
			ID:             "ds123",
			CreatedTime:    "2024-01-01T00:00:00.000Z",
			LastEditedTime: "2024-01-01T00:00:00.000Z",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	ds, err := client.GetDataSource(ctx, "ds123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ds.ID != "ds123" {
		t.Errorf("expected ID 'ds123', got %q", ds.ID)
	}
	if ds.Object != "data_source" {
		t.Errorf("expected object 'data_source', got %q", ds.Object)
	}
}

func TestGetDataSource_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetDataSource(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty data source ID")
	}

	expected := "data source ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetDataSource_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "Could not find data source.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.GetDataSource(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent data source")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestUpdateDataSource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources/ds123" {
			t.Errorf("expected path /data_sources/ds123, got %s", r.URL.Path)
		}

		var req UpdateDataSourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSource{
			Object:     "data_source",
			ID:         "ds123",
			Title:      req.Title,
			Properties: req.Properties,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &UpdateDataSourceRequest{
		Title: []RichText{
			{Type: "text", Text: &TextContent{Content: "Updated Title"}},
		},
	}

	ds, err := client.UpdateDataSource(ctx, "ds123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ds.ID != "ds123" {
		t.Errorf("expected ID 'ds123', got %q", ds.ID)
	}
}

func TestUpdateDataSource_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &UpdateDataSourceRequest{
		Title: []RichText{
			{Type: "text", Text: &TextContent{Content: "Test"}},
		},
	}

	_, err := client.UpdateDataSource(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty data source ID")
	}

	expected := "data source ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestQueryDataSource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources/ds123/query" {
			t.Errorf("expected path /data_sources/ds123/query, got %s", r.URL.Path)
		}

		var req QueryDataSourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceQueryResult{
			Object: "list",
			Results: []Page{
				{Object: "page", ID: "page1"},
				{Object: "page", ID: "page2"},
			},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &QueryDataSourceRequest{
		PageSize: 10,
	}

	result, err := client.QueryDataSource(ctx, "ds123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
}

func TestQueryDataSource_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &QueryDataSourceRequest{}

	_, err := client.QueryDataSource(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty data source ID")
	}

	expected := "data source ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestQueryDataSource_WithFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req QueryDataSourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Filter == nil {
			t.Error("expected filter to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceQueryResult{
			Object:  "list",
			Results: []Page{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &QueryDataSourceRequest{
		Filter: map[string]interface{}{
			"property": "Status",
			"select":   map[string]interface{}{"equals": "Done"},
		},
	}

	_, err := client.QueryDataSource(ctx, "ds123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryDataSource_WithSorts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req QueryDataSourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Sorts) == 0 {
			t.Error("expected sorts to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceQueryResult{
			Object:  "list",
			Results: []Page{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &QueryDataSourceRequest{
		Sorts: []map[string]interface{}{
			{"property": "Created", "direction": "descending"},
		},
	}

	_, err := client.QueryDataSource(ctx, "ds123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryDataSource_Pagination(t *testing.T) {
	nextCursor := "cursor456"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceQueryResult{
			Object: "list",
			Results: []Page{
				{Object: "page", ID: "page1"},
			},
			NextCursor: &nextCursor,
			HasMore:    true,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	result, err := client.QueryDataSource(ctx, "ds123", &QueryDataSourceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}
	if result.NextCursor == nil || *result.NextCursor != "cursor456" {
		t.Error("expected NextCursor to be 'cursor456'")
	}
}

func TestListDataSourceTemplates_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources/templates" {
			t.Errorf("expected path /data_sources/templates, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceTemplateList{
			Object: "list",
			Results: []*DataSourceTemplate{
				{ID: "tpl1", Name: "Template 1", Description: "First template"},
				{ID: "tpl2", Name: "Template 2", Description: "Second template"},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	list, err := client.ListDataSourceTemplates(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list.Results) != 2 {
		t.Errorf("expected 2 templates, got %d", len(list.Results))
	}
	if list.Results[0].Name != "Template 1" {
		t.Errorf("expected first template name 'Template 1', got %q", list.Results[0].Name)
	}
}

func TestListDataSourceTemplates_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DataSourceTemplateList{
			Object:  "list",
			Results: []*DataSourceTemplate{},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	list, err := client.ListDataSourceTemplates(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list.Results) != 0 {
		t.Errorf("expected 0 templates, got %d", len(list.Results))
	}
}
