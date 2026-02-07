package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestEnrichPage_WithDatabaseParent(t *testing.T) {
	// Set up mock server
	mux := http.NewServeMux()

	// Mock database endpoint
	mux.HandleFunc("/databases/db-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     "db-123",
			"title": []map[string]interface{}{
				{"plain_text": "My Database"},
			},
		})
	})

	// Mock block children endpoint
	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{map[string]interface{}{"id": "child-1"}, map[string]interface{}{"id": "child-2"}},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID: "page-456",
		Parent: map[string]interface{}{
			"type":        "database_id",
			"database_id": "db-123",
		},
	}

	enriched := enrichPage(ctx, client, page)

	if enriched.ParentTitle != "My Database" {
		t.Errorf("expected parent_title 'My Database', got %q", enriched.ParentTitle)
	}
	if enriched.ChildCount != 2 {
		t.Errorf("expected child_count 2, got %d", enriched.ChildCount)
	}
}

func TestEnrichPage_WithPageParent(t *testing.T) {
	// Set up mock server
	mux := http.NewServeMux()

	// Mock parent page endpoint
	mux.HandleFunc("/pages/parent-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "parent-123",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "title",
					"title": []map[string]interface{}{
						{"plain_text": "Parent Page Title"},
					},
				},
			},
		})
	})

	// Mock block children endpoint (empty)
	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID: "page-456",
		Parent: map[string]interface{}{
			"type":    "page_id",
			"page_id": "parent-123",
		},
	}

	enriched := enrichPage(ctx, client, page)

	if enriched.ParentTitle != "Parent Page Title" {
		t.Errorf("expected parent_title 'Parent Page Title', got %q", enriched.ParentTitle)
	}
	if enriched.ChildCount != 0 {
		t.Errorf("expected child_count 0, got %d", enriched.ChildCount)
	}
}

func TestEnrichPage_NoParent(t *testing.T) {
	// Set up mock server
	mux := http.NewServeMux()

	// Mock block children endpoint
	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{map[string]interface{}{"id": "child-1"}},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID:     "page-456",
		Parent: nil,
	}

	enriched := enrichPage(ctx, client, page)

	if enriched.ParentTitle != "" {
		t.Errorf("expected empty parent_title, got %q", enriched.ParentTitle)
	}
	if enriched.ChildCount != 1 {
		t.Errorf("expected child_count 1, got %d", enriched.ChildCount)
	}
}

func TestExtractPageTitleFromProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		expected   string
	}{
		{
			name:       "nil properties",
			properties: nil,
			expected:   "",
		},
		{
			name:       "empty properties",
			properties: map[string]interface{}{},
			expected:   "",
		},
		{
			name: "title property with plain_text",
			properties: map[string]interface{}{
				"Name": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "My Page Title"},
					},
				},
			},
			expected: "My Page Title",
		},
		{
			name: "title property with text content",
			properties: map[string]interface{}{
				"Title": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{
							"text": map[string]interface{}{
								"content": "Text Content Title",
							},
						},
					},
				},
			},
			expected: "Text Content Title",
		},
		{
			name: "multiple rich text segments",
			properties: map[string]interface{}{
				"Name": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Hello "},
						map[string]interface{}{"plain_text": "World"},
					},
				},
			},
			expected: "Hello World",
		},
		{
			name: "no title type property",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"type": "select",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPageTitleFromProperties(tt.properties)
			if got != tt.expected {
				t.Errorf("extractPageTitleFromProperties() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnrichPage_BreadcrumbPath(t *testing.T) {
	mux := http.NewServeMux()

	// Page -> parent page -> grandparent database
	mux.HandleFunc("/pages/parent-789", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "parent-789",
			"parent": map[string]interface{}{
				"type":        "database_id",
				"database_id": "db-root",
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Parent Page"}},
				},
			},
		})
	})

	mux.HandleFunc("/databases/db-root", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     "db-root",
			"title":  []map[string]interface{}{{"plain_text": "Root DB"}},
		})
	})

	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID: "page-456",
		Parent: map[string]interface{}{
			"type":    "page_id",
			"page_id": "parent-789",
		},
	}

	enriched := enrichPage(ctx, client, page)

	if len(enriched.Path) != 2 {
		t.Fatalf("expected 2 path entries, got %d: %+v", len(enriched.Path), enriched.Path)
	}
	if enriched.Path[0].Type != "database" || enriched.Path[0].Title != "Root DB" {
		t.Errorf("path[0] = %+v, want database 'Root DB'", enriched.Path[0])
	}
	if enriched.Path[1].Type != "page" || enriched.Path[1].Title != "Parent Page" {
		t.Errorf("path[1] = %+v, want page 'Parent Page'", enriched.Path[1])
	}
}

func TestEnrichedPage_JSONSerialization(t *testing.T) {
	page := &notion.Page{
		Object: "page",
		ID:     "page-123",
	}
	enriched := &EnrichedPage{
		Page:        page,
		ParentTitle: "My Database",
		ChildCount:  5,
	}

	data, err := json.Marshal(enriched)
	if err != nil {
		t.Fatalf("failed to marshal EnrichedPage: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal EnrichedPage: %v", err)
	}

	if result["parent_title"] != "My Database" {
		t.Errorf("expected parent_title 'My Database', got %v", result["parent_title"])
	}
	if result["child_count"] != float64(5) {
		t.Errorf("expected child_count 5, got %v", result["child_count"])
	}
	if result["id"] != "page-123" {
		t.Errorf("expected id 'page-123', got %v", result["id"])
	}
}

func TestEnrichPage_BreadcrumbPath_WorkspaceRoot(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID: "page-456",
		Parent: map[string]interface{}{
			"type":      "workspace",
			"workspace": true,
		},
	}

	enriched := enrichPage(ctx, client, page)

	if len(enriched.Path) != 0 {
		t.Errorf("expected empty path for workspace root page, got %d entries", len(enriched.Path))
	}
}

func TestEnrichPage_BreadcrumbPath_SingleDatabaseParent(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/databases/db-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     "db-123",
			"title":  []map[string]interface{}{{"plain_text": "Tasks"}},
		})
	})

	mux.HandleFunc("/blocks/page-456/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  []interface{}{},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := notion.NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	page := &notion.Page{
		ID: "page-456",
		Parent: map[string]interface{}{
			"type":        "database_id",
			"database_id": "db-123",
		},
	}

	enriched := enrichPage(ctx, client, page)

	if len(enriched.Path) != 1 {
		t.Fatalf("expected 1 path entry, got %d", len(enriched.Path))
	}
	if enriched.Path[0].Type != "database" || enriched.Path[0].ID != "db-123" {
		t.Errorf("path[0] = %+v, want database db-123", enriched.Path[0])
	}
}

func TestBuildBreadcrumbPath_NilParent(t *testing.T) {
	client := notion.NewClient("test-token")
	ctx := context.Background()

	path := buildBreadcrumbPath(ctx, client, nil)
	if len(path) != 0 {
		t.Errorf("expected empty path for nil parent, got %d entries", len(path))
	}
}
