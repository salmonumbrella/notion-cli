//go:build integration
// +build integration

package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Integration tests for EnhanceStatusError.
// Run with: go test -tags=integration ./internal/notion/...

func TestEnhanceStatusError_Integration(t *testing.T) {
	// This test requires a mock server or is skipped when run with -tags=integration
	// against real API. For CI, we use the mock server below.
	t.Skip("Integration test - run manually with real API")

	// Example of how the full flow would work:
	// 1. Create a client with a real API token
	// 2. Make an update with invalid status
	// 3. Verify enhanced error contains valid options

	// client := NewClient(os.Getenv("NOTION_TOKEN"))
	// pageID := "your-test-page-id"
	// ctx := context.Background()
	//
	// req := &UpdatePageRequest{
	//     Properties: map[string]interface{}{
	//         "Status": map[string]interface{}{
	//             "status": map[string]interface{}{
	//                 "name": "InvalidStatusValue",
	//             },
	//         },
	//     },
	// }
	//
	// _, err := client.UpdatePage(ctx, pageID, req)
	// if err != nil {
	//     enhanced := EnhanceStatusError(ctx, client, pageID, err)
	//     var statusErr *EnhancedStatusError
	//     if errors.As(enhanced, &statusErr) {
	//         t.Logf("Enhanced error: %s", statusErr.Error())
	//         // Verify the error contains valid options
	//     }
	// }
}

func TestEnhanceStatusError_WithMockServer(t *testing.T) {
	// This test demonstrates the full flow using a mock server
	// that simulates the Notion API responses.

	// Set up mock server with multiple endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/pages/page123" && r.Method == http.MethodGet:
			// Return page with database parent
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "page123",
				"parent": map[string]interface{}{
					"database_id": "db123",
				},
			})

		case r.URL.Path == "/databases/db123" && r.Method == http.MethodGet:
			// Return database with data source
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "database",
				"id":     "db123",
				"data_sources": []map[string]interface{}{
					{"id": "ds123"},
				},
			})

		case r.URL.Path == "/data_sources/ds123" && r.Method == http.MethodGet:
			// Return data source with status property
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "data_source",
				"id":     "ds123",
				"properties": map[string]interface{}{
					"Status": map[string]interface{}{
						"type": "status",
						"status": map[string]interface{}{
							"options": []map[string]interface{}{
								{"name": "Not Started", "color": "gray"},
								{"name": "In Progress", "color": "blue"},
								{"name": "Done", "color": "green"},
							},
						},
					},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "error",
				"status":  404,
				"code":    "object_not_found",
				"message": "Not found",
			})
		}
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// Create a mock status validation error
	originalErr := &APIError{
		Response: &ErrorResponse{
			Status:  400,
			Code:    "validation_error",
			Message: `Invalid status option. Status option "InvalidStatus" does not exist".`,
		},
	}

	enhanced := EnhanceStatusError(ctx, client, "page123", originalErr)

	var statusErr *EnhancedStatusError
	if !errors.As(enhanced, &statusErr) {
		t.Fatalf("expected EnhancedStatusError, got %T: %v", enhanced, enhanced)
	}

	if statusErr.InvalidValue != "InvalidStatus" {
		t.Errorf("expected invalid value 'InvalidStatus', got %q", statusErr.InvalidValue)
	}

	if len(statusErr.StatusProperties) != 1 {
		t.Errorf("expected 1 status property, got %d", len(statusErr.StatusProperties))
	}

	errMsg := statusErr.Error()
	if !strings.Contains(errMsg, "Not Started") {
		t.Error("error message should contain 'Not Started' option")
	}
	if !strings.Contains(errMsg, "In Progress") {
		t.Error("error message should contain 'In Progress' option")
	}
	if !strings.Contains(errMsg, "Done") {
		t.Error("error message should contain 'Done' option")
	}

	// Verify unwrapping preserves original error
	if !errors.Is(enhanced, originalErr) {
		t.Error("enhanced error should unwrap to original error")
	}
}

func TestEnhanceStatusError_FetchFailure(t *testing.T) {
	// When fetching schema fails, should return original error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return error (use 400 to avoid triggering client retry logic)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  400,
			"code":    "invalid_request",
			"message": "Bad request",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	originalErr := &APIError{
		Response: &ErrorResponse{
			Status:  400,
			Code:    "validation_error",
			Message: `Invalid status option. Status option "Test" does not exist".`,
		},
	}

	result := EnhanceStatusError(ctx, client, "page123", originalErr)

	// Should return original error since fetch failed
	if result != originalErr {
		t.Errorf("expected original error when fetch fails, got %v", result)
	}
}

func TestEnhanceStatusError_NoStatusProperties(t *testing.T) {
	// When database has no status properties, should return original error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/pages/page123":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "page123",
				"parent": map[string]interface{}{
					"database_id": "db123",
				},
			})
		case "/databases/db123":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "database",
				"id":     "db123",
				"data_sources": []map[string]interface{}{
					{"id": "ds123"},
				},
			})
		case "/data_sources/ds123":
			// Data source with no status properties
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "data_source",
				"id":     "ds123",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type": "title",
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	originalErr := &APIError{
		Response: &ErrorResponse{
			Status:  400,
			Code:    "validation_error",
			Message: `Invalid status option. Status option "Test" does not exist".`,
		},
	}

	result := EnhanceStatusError(ctx, client, "page123", originalErr)

	// Should return original error since no status properties found
	if result != originalErr {
		t.Errorf("expected original error when no status properties, got %v", result)
	}
}

func TestEnhanceStatusError_PageWithDataSourceParent(t *testing.T) {
	// Test when page parent is directly a data_source_id (newer API)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/pages/page123":
			// Page with data_source_id parent (newer API format)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "page123",
				"parent": map[string]interface{}{
					"data_source_id": "ds123",
				},
			})
		case "/data_sources/ds123":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "data_source",
				"id":     "ds123",
				"properties": map[string]interface{}{
					"Status": map[string]interface{}{
						"type": "status",
						"status": map[string]interface{}{
							"options": []map[string]interface{}{
								{"name": "Open", "color": "blue"},
								{"name": "Closed", "color": "red"},
							},
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	originalErr := &APIError{
		Response: &ErrorResponse{
			Status:  400,
			Code:    "validation_error",
			Message: `Invalid status option. Status option "Invalid" does not exist".`,
		},
	}

	enhanced := EnhanceStatusError(ctx, client, "page123", originalErr)

	var statusErr *EnhancedStatusError
	if !errors.As(enhanced, &statusErr) {
		t.Fatalf("expected EnhancedStatusError, got %T: %v", enhanced, enhanced)
	}

	errMsg := statusErr.Error()
	if !strings.Contains(errMsg, "Open") {
		t.Error("error message should contain 'Open' option")
	}
	if !strings.Contains(errMsg, "Closed") {
		t.Error("error message should contain 'Closed' option")
	}
}
