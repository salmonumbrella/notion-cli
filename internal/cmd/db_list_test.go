package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDBList_WrapsEnvelope verifies the --all path returns a standard list
// envelope with object, results, has_more, and next_cursor fields.
func TestDBList_WrapsEnvelope(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		// First call returns one result with has_more=true; second call returns
		// one result with has_more=false. fetchAllPages merges both pages.
		if callCount == 1 {
			next := "cursor-2"
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "data_source",
						"id":     "ds-1",
						"title":  []interface{}{map[string]interface{}{"plain_text": "DB One"}},
					},
				},
				"has_more":    true,
				"next_cursor": next,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "data_source",
					"id":     "ds-2",
					"title":  []interface{}{map[string]interface{}{"plain_text": "DB Two"}},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out bytes.Buffer
	app := &App{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	root.SetArgs([]string{"db", "list", "--all", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON object envelope, got parse error: %v\nraw: %s", err, out.String())
	}

	if obj, ok := payload["object"].(string); !ok || obj != "list" {
		t.Fatalf("expected object=list, got %v", payload["object"])
	}

	results, ok := payload["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T: %v", payload["results"], payload["results"])
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// After fetching all pages, has_more should be false.
	if hm, ok := payload["has_more"].(bool); !ok || hm {
		t.Fatalf("expected has_more=false, got %v", payload["has_more"])
	}
}

// TestDBList_SinglePage_WrapsEnvelope verifies the single-page path (no --all)
// also returns a standard list envelope with has_more and next_cursor.
func TestDBList_SinglePage_WrapsEnvelope(t *testing.T) {
	nextCursorVal := "cursor-next"
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "data_source",
					"id":     "ds-1",
					"title":  []interface{}{map[string]interface{}{"plain_text": "Projects"}},
				},
			},
			"has_more":    true,
			"next_cursor": nextCursorVal,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out bytes.Buffer
	app := &App{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	root.SetArgs([]string{"db", "list", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON object envelope, got parse error: %v\nraw: %s", err, out.String())
	}

	if obj, ok := payload["object"].(string); !ok || obj != "list" {
		t.Fatalf("expected object=list, got %v", payload["object"])
	}

	results, ok := payload["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", payload["results"])
	}

	if hm, ok := payload["has_more"].(bool); !ok || !hm {
		t.Fatalf("expected has_more=true, got %v", payload["has_more"])
	}

	if nc, ok := payload["next_cursor"].(string); !ok || nc != nextCursorVal {
		t.Fatalf("expected next_cursor=%q, got %v", nextCursorVal, payload["next_cursor"])
	}
}
