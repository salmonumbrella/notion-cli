package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageGet_IncludeChildren(t *testing.T) {
	const (
		rawID        = "12345678123412341234123456789012"
		normalizedID = "12345678-1234-1234-1234-123456789012"
	)

	var childrenCalls int
	mux := http.NewServeMux()

	mux.HandleFunc("/pages/"+rawID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               normalizedID,
			"created_time":     "2026-01-01T00:00:00.000Z",
			"last_edited_time": "2026-01-01T00:00:00.000Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Demo"},
					},
				},
			},
		})
	})

	mux.HandleFunc("/blocks/"+rawID+"/children", func(w http.ResponseWriter, r *http.Request) {
		childrenCalls++
		w.Header().Set("Content-Type", "application/json")

		if childrenCalls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []interface{}{
					map[string]interface{}{
						"object":           "block",
						"id":               "block-1",
						"type":             "paragraph",
						"created_time":     "2026-01-01T00:00:00.000Z",
						"last_edited_time": "2026-01-01T00:00:00.000Z",
						"has_children":     false,
						"paragraph": map[string]interface{}{
							"rich_text": []interface{}{
								map[string]interface{}{
									"plain_text": "one",
								},
							},
						},
					},
				},
				"has_more":    true,
				"next_cursor": "cursor-2",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object":           "block",
					"id":               "block-2",
					"type":             "image",
					"created_time":     "2026-01-01T00:00:00.000Z",
					"last_edited_time": "2026-01-01T00:00:00.000Z",
					"has_children":     false,
					"image": map[string]interface{}{
						"type": "external",
						"external": map[string]interface{}{
							"url": "https://example.com/demo.png",
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
	root.SetArgs([]string{"page", "get", rawID, "--include-children", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v\nraw: %s", err, out.String())
	}

	if gotID, _ := payload["id"].(string); gotID != normalizedID {
		t.Fatalf("expected page id %q, got %q", normalizedID, gotID)
	}

	children, ok := payload["children"].([]interface{})
	if !ok {
		t.Fatalf("expected children array, got %T", payload["children"])
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	if childrenCalls != 2 {
		t.Fatalf("expected paginated children fetch (2 calls), got %d", childrenCalls)
	}
}

func TestPageGet_IncludeChildrenDepthRecursive(t *testing.T) {
	const (
		rawID        = "abcdefabcdefabcdefabcdefabcdefab"
		normalizedID = "abcdefab-cdef-abcd-efab-cdefabcdefab"
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/pages/"+rawID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               normalizedID,
			"created_time":     "2026-01-01T00:00:00.000Z",
			"last_edited_time": "2026-01-01T00:00:00.000Z",
			"archived":         false,
			"properties":       map[string]interface{}{},
		})
	})

	mux.HandleFunc("/blocks/"+rawID+"/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object":           "block",
					"id":               "child-1",
					"type":             "toggle",
					"created_time":     "2026-01-01T00:00:00.000Z",
					"last_edited_time": "2026-01-01T00:00:00.000Z",
					"has_children":     true,
					"toggle": map[string]interface{}{
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "toggle"},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	mux.HandleFunc("/blocks/child-1/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []interface{}{
				map[string]interface{}{
					"object":           "block",
					"id":               "grandchild-1",
					"type":             "paragraph",
					"created_time":     "2026-01-01T00:00:00.000Z",
					"last_edited_time": "2026-01-01T00:00:00.000Z",
					"has_children":     false,
					"paragraph": map[string]interface{}{
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "nested"},
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
	root.SetArgs([]string{"page", "get", rawID, "--include-children", "--children-depth", "2", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v\nraw: %s", err, out.String())
	}

	children, ok := payload["children"].([]interface{})
	if !ok || len(children) != 1 {
		t.Fatalf("expected 1 top-level child, got %v", payload["children"])
	}

	child, ok := children[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected child object, got %T", children[0])
	}
	grandchildren, ok := child["children"].([]interface{})
	if !ok || len(grandchildren) != 1 {
		t.Fatalf("expected 1 nested child, got %v", child["children"])
	}
}
