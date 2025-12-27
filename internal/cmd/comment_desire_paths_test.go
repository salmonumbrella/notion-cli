package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComment_DefaultAdd_Positional(t *testing.T) {
	const pageID = "12345678123412341234123456789012"

	mux := http.NewServeMux()
	mux.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST /comments, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		parent, _ := body["parent"].(map[string]interface{})
		if got, _ := parent["page_id"].(string); got != pageID {
			t.Fatalf("expected parent.page_id %q, got %q", pageID, got)
		}
		if rt, ok := body["rich_text"].([]interface{}); !ok || len(rt) == 0 {
			t.Fatalf("expected rich_text array in request, got %#v", body["rich_text"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "comment",
			"id":     "c1",
			"parent": map[string]interface{}{
				"type":    "page_id",
				"page_id": pageID,
			},
			"discussion_id":    "d1",
			"created_time":     "2026-01-01T00:00:00Z",
			"last_edited_time": "2026-01-01T00:00:00Z",
			"created_by": map[string]interface{}{
				"object": "user",
				"id":     "u1",
				"type":   "bot",
				"name":   "Test Bot",
			},
			"rich_text": []map[string]interface{}{
				{
					"type":       "text",
					"plain_text": "ok",
					"text": map[string]interface{}{
						"content": "ok",
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := &App{
		Stdout: outWriter(&out),
		Stderr: outWriter(&errBuf),
	}

	root := app.RootCommand()
	root.SetArgs([]string{"comment", pageID, "Hello", "world"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(out.String(), `"object": "comment"`) {
		t.Fatalf("expected comment JSON output, got: %s", out.String())
	}
}

func TestCommentList_ResolvesPageNameViaSearch(t *testing.T) {
	const (
		pageID = "12345678123412341234123456789012"
		query  = "Meeting Notes"
	)

	var searchCalls int
	mux := http.NewServeMux()

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		searchCalls++
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST /search, got %s", r.Method)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode search request: %v", err)
		}
		if got, _ := req["query"].(string); got != query {
			t.Fatalf("expected search query %q, got %q", query, got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     pageID,
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type": "title",
							"title": []map[string]interface{}{
								{"plain_text": query},
							},
						},
					},
				},
			},
			"has_more": false,
		})
	})

	mux.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET /comments, got %s", r.Method)
		}
		if got := r.URL.Query().Get("block_id"); got != pageID {
			t.Fatalf("expected block_id %q, got %q", pageID, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object":        "comment",
					"id":            "c99",
					"discussion_id": "d99",
					"parent": map[string]interface{}{
						"type":    "page_id",
						"page_id": pageID,
					},
					"created_time":     "2026-01-01T00:00:00Z",
					"last_edited_time": "2026-01-01T00:00:00Z",
					"created_by": map[string]interface{}{
						"object": "user",
						"id":     "u1",
						"type":   "bot",
						"name":   "Test Bot",
					},
					"rich_text": []map[string]interface{}{
						{
							"type":       "text",
							"plain_text": "hi",
							"text": map[string]interface{}{
								"content": "hi",
							},
						},
					},
				},
			},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := &App{
		Stdout: outWriter(&out),
		Stderr: outWriter(&errBuf),
	}

	root := app.RootCommand()
	root.SetArgs([]string{"comment", "list", query, "--results-only"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if searchCalls != 1 {
		t.Fatalf("expected 1 search call, got %d", searchCalls)
	}
	if !strings.Contains(out.String(), `"id": "c99"`) {
		t.Fatalf("expected comment list output, got: %s", out.String())
	}
}

func TestCommentAdd_ShortFlags(t *testing.T) {
	const (
		pageID    = "208eaeccf76481aea95bec6d51c70337"
		mentionID = "608a5f14-b513-4fae-b3cc-476d266f227b"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST /comments, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		parent, _ := body["parent"].(map[string]interface{})
		if got, _ := parent["page_id"].(string); got != pageID {
			t.Fatalf("expected parent.page_id %q, got %q", pageID, got)
		}

		rt, ok := body["rich_text"].([]interface{})
		if !ok || len(rt) == 0 {
			t.Fatalf("expected rich_text array in request, got %#v", body["rich_text"])
		}

		foundMention := false
		for _, item := range rt {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if entry["type"] != "mention" {
				continue
			}
			mention, _ := entry["mention"].(map[string]interface{})
			user, _ := mention["user"].(map[string]interface{})
			if got, _ := user["id"].(string); got == mentionID {
				foundMention = true
				break
			}
		}
		if !foundMention {
			t.Fatalf("expected mention with user id %q in rich_text payload", mentionID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "comment",
			"id":     "c-short",
			"parent": map[string]interface{}{
				"type":    "page_id",
				"page_id": pageID,
			},
			"discussion_id":    "d-short",
			"created_time":     "2026-01-01T00:00:00Z",
			"last_edited_time": "2026-01-01T00:00:00Z",
			"created_by": map[string]interface{}{
				"object": "user",
				"id":     "u1",
				"type":   "bot",
				"name":   "Test Bot",
			},
			"rich_text": []map[string]interface{}{
				{
					"type":       "text",
					"plain_text": "ok",
					"text": map[string]interface{}{
						"content": "ok",
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := &App{
		Stdout: outWriter(&out),
		Stderr: outWriter(&errBuf),
	}

	root := app.RootCommand()
	root.SetArgs([]string{"comment", "add", "-p", pageID, "-t", "@Reviewer update sent", "-m", mentionID})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(out.String(), `"id": "c-short"`) {
		t.Fatalf("expected comment JSON output, got: %s", out.String())
	}
}

// outWriter prevents accidental type assertions in isTerminal(*os.File).
func outWriter(buf *bytes.Buffer) *bytes.Buffer { return buf }
