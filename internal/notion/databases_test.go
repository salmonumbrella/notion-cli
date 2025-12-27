package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDatabase_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/databases/db123" {
			t.Errorf("expected path /databases/db123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Database{
			Object:     "database",
			ID:         "db123",
			Title:      []map[string]interface{}{{"type": "text", "text": map[string]interface{}{"content": "Test Database"}}},
			Properties: map[string]map[string]interface{}{},
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	database, err := client.GetDatabase(ctx, "db123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if database.ID != "db123" {
		t.Errorf("expected ID 'db123', got %q", database.ID)
	}
	if database.Object != "database" {
		t.Errorf("expected object 'database', got %q", database.Object)
	}
}

func TestGetDatabase_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetDatabase(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty database ID")
	}

	expected := "database ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestQueryDatabase_Success(t *testing.T) {
	// API 2025-09-03+ requires: GET database first to get data_source_id, then POST to /data_sources/{id}/query
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request: GET database to retrieve data_source_id
			if r.Method != http.MethodGet {
				t.Errorf("expected GET method for database fetch, got %s", r.Method)
			}
			if r.URL.Path != "/databases/db123" {
				t.Errorf("expected path /databases/db123, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Database{
				Object:      "database",
				ID:          "db123",
				DataSources: []DataSourceRef{{ID: "ds456"}},
			})
			return
		}
		// Second request: POST query to data source
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method for query, got %s", r.Method)
		}
		if r.URL.Path != "/data_sources/ds456/query" {
			t.Errorf("expected path /data_sources/ds456/query, got %s", r.URL.Path)
		}

		var req DatabaseQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DatabaseQueryResult{
			Object:  "list",
			Results: []Page{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &DatabaseQueryRequest{
		PageSize: 10,
	}

	result, err := client.QueryDatabase(ctx, "db123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests (GET database + POST query), got %d", requestCount)
	}
}

func TestQueryDatabase_MultipleDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method for database fetch, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Database{
			Object: "database",
			ID:     "db123",
			DataSources: []DataSourceRef{
				{ID: "ds1", Name: "Primary"},
				{ID: "ds2", Name: "Secondary"},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.QueryDatabase(ctx, "db123", &DatabaseQueryRequest{})
	if err == nil {
		t.Fatal("expected error for multiple data sources")
	}

	expected := "database has multiple data sources; specify a data source ID"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestQueryDatabase_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.QueryDatabase(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty database ID")
	}

	expected := "database ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestQueryDatabase_NilRequest(t *testing.T) {
	// API 2025-09-03+ requires: GET database first, then POST query
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request: GET database
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Database{
				Object:      "database",
				ID:          "db123",
				DataSources: []DataSourceRef{{ID: "ds456"}},
			})
			return
		}
		// Second request: POST query - nil request should be handled gracefully
		var req DatabaseQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DatabaseQueryResult{
			Object:  "list",
			Results: []Page{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// nil request should be handled gracefully
	result, err := client.QueryDatabase(ctx, "db123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("expected object 'list', got %q", result.Object)
	}
}

func TestCreateDatabase_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/databases" {
			t.Errorf("expected path /databases, got %s", r.URL.Path)
		}

		var req CreateDatabaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent == nil {
			t.Error("expected parent to be set")
		}
		if req.InitialDataSource == nil {
			t.Error("expected initial_data_source to be set")
		} else if len(req.InitialDataSource.Properties) == 0 {
			t.Error("expected initial_data_source.properties to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Database{
			Object:     "database",
			ID:         "newdb123",
			Title:      req.Title,
			Properties: req.InitialDataSource.Properties,
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateDatabaseRequest{
		Parent: map[string]interface{}{"page_id": "parent123"},
		InitialDataSource: &InitialDataSource{
			Properties: map[string]map[string]interface{}{
				"Name": {"title": map[string]interface{}{}},
			},
		},
	}

	database, err := client.CreateDatabase(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if database.ID != "newdb123" {
		t.Errorf("expected ID 'newdb123', got %q", database.ID)
	}
}

func TestCreateDatabase_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CreateDatabase(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "create database request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateDatabase_MissingParent(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateDatabaseRequest{
		InitialDataSource: &InitialDataSource{
			Properties: map[string]map[string]interface{}{
				"Name": {"title": map[string]interface{}{}},
			},
		},
	}

	_, err := client.CreateDatabase(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing parent")
	}

	expected := "parent is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateDatabase_MissingInitialDataSource(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateDatabaseRequest{
		Parent: map[string]interface{}{"page_id": "parent123"},
	}

	_, err := client.CreateDatabase(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing initial_data_source")
	}

	expected := "initial_data_source is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateDatabase_MissingProperties(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateDatabaseRequest{
		Parent:            map[string]interface{}{"page_id": "parent123"},
		InitialDataSource: &InitialDataSource{},
	}

	_, err := client.CreateDatabase(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing properties")
	}

	expected := "initial_data_source.properties are required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdateDatabase_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}
		if r.URL.Path != "/databases/db123" {
			t.Errorf("expected path /databases/db123, got %s", r.URL.Path)
		}

		var req UpdateDatabaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Database{
			Object:     "database",
			ID:         "db123",
			Title:      req.Title,
			Properties: map[string]map[string]interface{}{},
			Archived:   false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &UpdateDatabaseRequest{
		Title: []map[string]interface{}{{"type": "text", "text": map[string]interface{}{"content": "Updated Title"}}},
	}

	database, err := client.UpdateDatabase(ctx, "db123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if database.ID != "db123" {
		t.Errorf("expected ID 'db123', got %q", database.ID)
	}
}

func TestUpdateDatabase_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &UpdateDatabaseRequest{
		Title: []map[string]interface{}{},
	}

	_, err := client.UpdateDatabase(ctx, "", req)
	if err == nil {
		t.Fatal("expected error for empty database ID")
	}

	expected := "database ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUpdateDatabase_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.UpdateDatabase(ctx, "db123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "update database request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestBuildQueryURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		opts     *BuildDatabaseQueryOptions
		expected string
	}{
		{
			name:     "nil options",
			baseURL:  "https://api.notion.com/v1/databases/db123/query",
			opts:     nil,
			expected: "https://api.notion.com/v1/databases/db123/query",
		},
		{
			name:    "with page size",
			baseURL: "https://api.notion.com/v1/databases/db123/query",
			opts: &BuildDatabaseQueryOptions{
				PageSize: 50,
			},
			expected: "https://api.notion.com/v1/databases/db123/query?page_size=50",
		},
		{
			name:    "with start cursor",
			baseURL: "https://api.notion.com/v1/databases/db123/query",
			opts: &BuildDatabaseQueryOptions{
				StartCursor: "cursor123",
			},
			expected: "https://api.notion.com/v1/databases/db123/query?start_cursor=cursor123",
		},
		{
			name:    "with both",
			baseURL: "https://api.notion.com/v1/databases/db123/query",
			opts: &BuildDatabaseQueryOptions{
				StartCursor: "cursor123",
				PageSize:    50,
			},
			expected: "https://api.notion.com/v1/databases/db123/query?page_size=50&start_cursor=cursor123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildQueryURL(tt.baseURL, tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected URL %q, got %q", tt.expected, result)
			}
		})
	}
}
