package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommentList_LightOutput(t *testing.T) {
	const pageID = "12345678-1234-1234-1234-123456789012"

	mux := http.NewServeMux()
	mux.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET /comments, got %s", r.Method)
		}
		if got := r.URL.Query().Get("block_id"); got == "" {
			t.Fatal("expected block_id query parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object":        "comment",
					"id":            "c-light",
					"discussion_id": "d-light",
					"parent": map[string]interface{}{
						"type":    "page_id",
						"page_id": pageID,
					},
					"created_time": "2026-02-16T00:00:00Z",
					"created_by": map[string]interface{}{
						"object": "user",
						"id":     "u-light",
						"type":   "person",
						"name":   "Alice",
					},
					"rich_text": []map[string]interface{}{
						{
							"type":       "text",
							"plain_text": "Looks good",
							"text": map[string]interface{}{
								"content": "Looks good",
							},
						},
					},
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
	root.SetArgs([]string{"comment", "list", pageID, "--li", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output JSON: %v\nraw: %s", err, out.String())
	}

	results, ok := payload["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected one result, got %#v", payload["results"])
	}
	first, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %#v", results[0])
	}

	if got := first["id"]; got != "c-light" {
		t.Fatalf("expected id c-light, got %#v", got)
	}
	if got := first["discussion_id"]; got != "d-light" {
		t.Fatalf("expected discussion_id d-light, got %#v", got)
	}
	if got := first["created_by"]; got != "Alice" {
		t.Fatalf("expected created_by Alice, got %#v", got)
	}
	if got := first["text"]; got != "Looks good" {
		t.Fatalf("expected text 'Looks good', got %#v", got)
	}
	if _, exists := first["rich_text"]; exists {
		t.Fatalf("did not expect rich_text in light output: %#v", first)
	}
}

func TestFileList_LightOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/file_uploads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET /file_uploads, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object":       "file_upload",
					"id":           "f-light",
					"created_time": "2026-02-16T00:00:00Z",
					"expiry_time":  "2026-02-17T00:00:00Z",
					"file_name":    "invoice.pdf",
					"size":         1234,
					"status":       "uploaded",
					"upload_url":   "https://example.com/upload",
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
	root.SetArgs([]string{"file", "list", "--li", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output JSON: %v\nraw: %s", err, out.String())
	}

	results, ok := payload["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected one result, got %#v", payload["results"])
	}
	first, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %#v", results[0])
	}

	if got := first["id"]; got != "f-light" {
		t.Fatalf("expected id f-light, got %#v", got)
	}
	if got := first["file_name"]; got != "invoice.pdf" {
		t.Fatalf("expected file_name invoice.pdf, got %#v", got)
	}
	if got := first["status"]; got != "uploaded" {
		t.Fatalf("expected status uploaded, got %#v", got)
	}
	if _, exists := first["upload_url"]; exists {
		t.Fatalf("did not expect upload_url in light output: %#v", first)
	}
}
