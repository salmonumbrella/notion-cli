package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFM   map[string]string
		wantBody string
	}{
		{
			name:  "full frontmatter",
			input: "---\nnotion-id: abc-123\ntitle: My Page\nlast-synced: 2026-01-01T00:00:00Z\n---\n\n# Hello\n",
			wantFM: map[string]string{
				"notion-id":   "abc-123",
				"title":       "My Page",
				"last-synced": "2026-01-01T00:00:00Z",
			},
			wantBody: "\n# Hello\n",
		},
		{
			name:     "no frontmatter",
			input:    "# Just a heading\n\nSome text.\n",
			wantFM:   map[string]string{},
			wantBody: "# Just a heading\n\nSome text.\n",
		},
		{
			name:     "empty input",
			input:    "",
			wantFM:   map[string]string{},
			wantBody: "",
		},
		{
			name:     "unclosed frontmatter",
			input:    "---\nnotion-id: abc-123\ntitle: My Page\n# No closing delimiter\n",
			wantFM:   map[string]string{},
			wantBody: "---\nnotion-id: abc-123\ntitle: My Page\n# No closing delimiter\n",
		},
		{
			name:  "frontmatter with empty lines",
			input: "---\nnotion-id: abc-123\n\ntitle: My Page\n---\nBody\n",
			wantFM: map[string]string{
				"notion-id": "abc-123",
				"title":     "My Page",
			},
			wantBody: "Body\n",
		},
		{
			name:  "frontmatter with colon in value",
			input: "---\ntitle: Meeting: Planning Session\n---\nContent\n",
			wantFM: map[string]string{
				"title": "Meeting: Planning Session",
			},
			wantBody: "Content\n",
		},
		{
			name:     "frontmatter only no body",
			input:    "---\nnotion-id: abc-123\n---\n",
			wantFM:   map[string]string{"notion-id": "abc-123"},
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body := parseFrontmatter(tt.input)

			if len(fm) != len(tt.wantFM) {
				t.Errorf("frontmatter length: got %d, want %d", len(fm), len(tt.wantFM))
			}
			for key, want := range tt.wantFM {
				if got := fm[key]; got != want {
					t.Errorf("fm[%q] = %q, want %q", key, got, want)
				}
			}

			if body != tt.wantBody {
				t.Errorf("body:\ngot:  %q\nwant: %q", body, tt.wantBody)
			}
		})
	}
}

func TestBuildFrontmatterString(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]string
		want string
	}{
		{
			name: "empty map",
			fm:   map[string]string{},
			want: "",
		},
		{
			name: "standard keys in order",
			fm: map[string]string{
				"notion-id":   "abc-123",
				"title":       "My Page",
				"last-synced": "2026-01-01T00:00:00Z",
			},
			want: "---\nnotion-id: abc-123\ntitle: My Page\nlast-synced: 2026-01-01T00:00:00Z\n---\n",
		},
		{
			name: "only notion-id",
			fm:   map[string]string{"notion-id": "abc-123"},
			want: "---\nnotion-id: abc-123\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFrontmatterString(tt.fm)
			if got != tt.want {
				t.Errorf("buildFrontmatterString:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestWriteFrontmatterToFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")

	fm := map[string]string{
		"notion-id":   "12345678-1234-1234-1234-123456789012",
		"title":       "Test Page",
		"last-synced": "2026-02-12T10:30:00Z",
	}
	body := "\n# Test Page\n\nSome content here.\n"

	err := writeFrontmatterToFile(filePath, fm, body)
	if err != nil {
		t.Fatalf("writeFrontmatterToFile: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		t.Error("file should start with ---")
	}
	if !strings.Contains(content, "notion-id: 12345678-1234-1234-1234-123456789012") {
		t.Error("file should contain notion-id")
	}
	if !strings.Contains(content, "title: Test Page") {
		t.Error("file should contain title")
	}
	if !strings.Contains(content, "# Test Page") {
		t.Error("file should contain body heading")
	}
	if !strings.HasSuffix(content, "\n") {
		t.Error("file should end with newline")
	}
}

func TestExtractTitleFromBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "h1 heading",
			body: "# Meeting Notes\n\nSome text.",
			want: "Meeting Notes",
		},
		{
			name: "h2 heading only",
			body: "## Not a title\n\nSome text.",
			want: "",
		},
		{
			name: "no heading",
			body: "Just a paragraph.",
			want: "",
		},
		{
			name: "h1 after other content",
			body: "Some text.\n\n# Title Here\n\nMore text.",
			want: "Title Here",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitleFromBody(tt.body)
			if got != tt.want {
				t.Errorf("extractTitleFromBody: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPageSyncFrontmatterRoundTrip(t *testing.T) {
	original := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: My Notes\nlast-synced: 2026-02-12T10:30:00Z\n---\n\n# My Notes\n\nThis is the content.\n\n- Item 1\n- Item 2\n"

	fm, body := parseFrontmatter(original)

	if fm["notion-id"] != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("notion-id mismatch: %q", fm["notion-id"])
	}
	if fm["title"] != "My Notes" {
		t.Errorf("title mismatch: %q", fm["title"])
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "roundtrip.md")
	if err := writeFrontmatterToFile(filePath, fm, body); err != nil {
		t.Fatalf("writeFrontmatterToFile: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	fm2, body2 := parseFrontmatter(string(data))

	if fm2["notion-id"] != fm["notion-id"] {
		t.Errorf("notion-id round-trip: got %q, want %q", fm2["notion-id"], fm["notion-id"])
	}
	if fm2["title"] != fm["title"] {
		t.Errorf("title round-trip: got %q, want %q", fm2["title"], fm["title"])
	}
	if fm2["last-synced"] != fm["last-synced"] {
		t.Errorf("last-synced round-trip: got %q, want %q", fm2["last-synced"], fm["last-synced"])
	}

	if body2 != body {
		t.Errorf("body round-trip:\ngot:  %q\nwant: %q", body2, body)
	}
}

func TestPageSyncPush_ExistingPage(t *testing.T) {
	var blockChildrenCalls int
	var deleteBlockCalls []string
	var appendCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pages/") && !strings.Contains(r.URL.Path, "/blocks/"):
			// Title update — accept silently
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			blockChildrenCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{"object": "block", "id": "block-1", "type": "paragraph", "has_children": false},
					{"object": "block", "id": "block-2", "type": "paragraph", "has_children": false},
				},
				"has_more": false,
			})

		case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/blocks/"):
			blockID := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			deleteBlockCalls = append(deleteBlockCalls, blockID)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "block",
				"id":       blockID,
				"type":     "paragraph",
				"archived": true,
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			appendCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: Test\nlast-synced: 2026-01-01T00:00:00Z\n---\n\n# Updated Content\n\nNew paragraph.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPush(ctx, client, &stderr, mdFile, "", "", false, true)
	if err != nil {
		t.Fatalf("runSyncPush: %v", err)
	}

	if blockChildrenCalls != 1 {
		t.Errorf("expected 1 block children call, got %d", blockChildrenCalls)
	}
	if len(deleteBlockCalls) != 2 {
		t.Errorf("expected 2 delete calls, got %d", len(deleteBlockCalls))
	}
	if appendCalls != 1 {
		t.Errorf("expected 1 append call, got %d", appendCalls)
	}

	data, _ := os.ReadFile(mdFile)
	fm, _ := parseFrontmatter(string(data))
	if fm["last-synced"] == "2026-01-01T00:00:00Z" {
		t.Error("last-synced should have been updated")
	}
	if fm["notion-id"] != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("notion-id should be preserved, got %q", fm["notion-id"])
	}
}

func TestPageSyncPush_NewPage(t *testing.T) {
	var createdPageID string
	var appendCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/databases/"):
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "error",
				"status":  404,
				"code":    "object_not_found",
				"message": "Could not find database",
			})

		case r.Method == "POST" && r.URL.Path == "/v1/pages":
			createdPageID = "new-page-id-1234"
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     createdPageID,
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []map[string]interface{}{
							{"plain_text": "My Document"},
						},
					},
				},
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			appendCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "new.md")
	content := "# My Document\n\nSome content.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPush(ctx, client, &stderr, mdFile, "parent-page-id-1234567890ab", "", false, false)
	if err != nil {
		t.Fatalf("runSyncPush: %v", err)
	}

	data, _ := os.ReadFile(mdFile)
	fm, body := parseFrontmatter(string(data))

	if fm["notion-id"] != "new-page-id-1234" {
		t.Errorf("expected notion-id to be written back, got %q", fm["notion-id"])
	}
	if fm["title"] != "My Document" {
		t.Errorf("expected title 'My Document', got %q", fm["title"])
	}
	if fm["last-synced"] == "" {
		t.Error("expected last-synced to be set")
	}
	if !strings.Contains(body, "# My Document") {
		t.Error("body should still contain the heading")
	}
}

func TestPageSyncPush_NoIDNoParent(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "orphan.md")
	content := "# No Parent\n\nJust content.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPush(ctx, nil, &stderr, mdFile, "", "", false, false)
	if err == nil {
		t.Fatal("expected error when no notion-id and no --parent")
	}
	if !strings.Contains(err.Error(), "no notion-id in frontmatter") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPageSyncPull_ToFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []map[string]interface{}{
							{"plain_text": "Pulled Page"},
						},
					},
				},
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "block-1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Hello world"},
							},
						},
						"has_children": false,
					},
				},
				"has_more": false,
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "pulled.md")

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPull(ctx, client, &stderr, "12345678-1234-1234-1234-123456789012", outFile, false)
	if err != nil {
		t.Fatalf("runSyncPull: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	fm, body := parseFrontmatter(content)
	if fm["notion-id"] != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("notion-id: got %q", fm["notion-id"])
	}
	if fm["title"] != "Pulled Page" {
		t.Errorf("title: got %q", fm["title"])
	}
	if fm["last-synced"] == "" {
		t.Error("last-synced should be set")
	}

	if !strings.Contains(body, "Hello world") {
		t.Errorf("body should contain block text, got: %q", body)
	}
}

func TestPageSyncPull_ToStdout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type": "title",
						"title": []map[string]interface{}{
							{"plain_text": "Stdout Page"},
						},
					},
				},
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "list",
				"results":  []map[string]interface{}{},
				"has_more": false,
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPull(ctx, client, &stderr, "12345678-1234-1234-1234-123456789012", "", false)
	if err != nil {
		t.Fatalf("runSyncPull to stdout: %v", err)
	}
}

func TestPageSyncPush_DryRun(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "dryrun.md")
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: Test\n---\n\n# Dry Run\n\nContent.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr strings.Builder
	ctx := t.Context()
	err := runSyncPush(ctx, nil, &stderr, mdFile, "", "", true, false)
	if err != nil {
		t.Fatalf("runSyncPush dry-run: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("dry-run output should contain [DRY-RUN]")
	}
	if !strings.Contains(output, "replace all blocks") {
		t.Error("dry-run output should describe the action")
	}
	if !strings.Contains(output, "No changes made") {
		t.Error("dry-run output should say no changes made")
	}

	data, _ := os.ReadFile(mdFile)
	if string(data) != content {
		t.Error("file should not be modified during dry-run")
	}
}

func TestPageSyncResolveParentForSync(t *testing.T) {
	t.Run("explicit page", func(t *testing.T) {
		parent, err := resolveParentForSync(t.Context(), nil, "parent-id", "page")
		if err != nil {
			t.Fatal(err)
		}
		if parent["page_id"] != "parent-id" {
			t.Errorf("expected page_id, got %v", parent)
		}
	})

	t.Run("explicit database", func(t *testing.T) {
		parent, err := resolveParentForSync(t.Context(), nil, "parent-id", "database")
		if err != nil {
			t.Fatal(err)
		}
		if parent["database_id"] != "parent-id" {
			t.Errorf("expected database_id, got %v", parent)
		}
	})

	t.Run("auto detect database", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "GET" && strings.Contains(r.URL.Path, "/databases/") {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"object": "database",
					"id":     "db-id",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		client := newTestSyncClient(t, srv)
		parent, err := resolveParentForSync(t.Context(), client, "db-id", "")
		if err != nil {
			t.Fatal(err)
		}
		if parent["database_id"] != "db-id" {
			t.Errorf("expected database_id from auto-detect, got %v", parent)
		}
	})

	t.Run("auto detect fallback to page", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "GET" && strings.Contains(r.URL.Path, "/databases/") {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"object":  "error",
					"status":  404,
					"code":    "object_not_found",
					"message": "not found",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		client := newTestSyncClient(t, srv)
		parent, err := resolveParentForSync(t.Context(), client, "page-id", "")
		if err != nil {
			t.Fatal(err)
		}
		if parent["page_id"] != "page-id" {
			t.Errorf("expected page_id fallback, got %v", parent)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := resolveParentForSync(t.Context(), nil, "parent-id", "invalid")
		if err == nil {
			t.Fatal("expected error for invalid parent type")
		}
	})
}

func TestAppendBlocksInBatches(t *testing.T) {
	var batchSizes []int
	var decodeErr error

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children") {
			var req notion.AppendBlockChildrenRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				decodeErr = err
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			batchSizes = append(batchSizes, len(req.Children))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newTestSyncClient(t, srv)

	blocks := make([]map[string]interface{}, 205)
	for i := range blocks {
		blocks[i] = map[string]interface{}{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": fmt.Sprintf("line %d", i),
						},
					},
				},
			},
		}
	}

	if err := appendBlocksInBatches(t.Context(), client, "page-id", blocks); err != nil {
		t.Fatalf("appendBlocksInBatches: %v", err)
	}
	if decodeErr != nil {
		t.Fatalf("server decode error: %v", decodeErr)
	}

	if len(batchSizes) != 3 {
		t.Fatalf("expected 3 batched requests, got %d", len(batchSizes))
	}
	if batchSizes[0] != 100 || batchSizes[1] != 100 || batchSizes[2] != 5 {
		t.Fatalf("unexpected batch sizes: %v", batchSizes)
	}
}

func TestPageSyncPush_ConflictDetected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			// Page was edited AFTER the last sync
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "12345678-1234-1234-1234-123456789012",
				"last_edited_time": "2026-02-13T12:00:00.000Z",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Test"}},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "conflict.md")
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: Test\nlast-synced: 2026-02-13T10:00:00Z\n---\n\n# Test\n\nContent.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	err := runSyncPush(t.Context(), client, &stderr, mdFile, "", "", false, false)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "modified on Notion since last sync") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPageSyncPush_ConflictForced(t *testing.T) {
	var appendCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "12345678-1234-1234-1234-123456789012",
				"last_edited_time": "2026-02-13T12:00:00.000Z",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Test"}},
					},
				},
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pages/") && !strings.Contains(r.URL.Path, "/blocks/"):
			// Title update — accept silently
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "list",
				"results":  []map[string]interface{}{},
				"has_more": false,
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			appendCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "forced.md")
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: Test\nlast-synced: 2026-02-13T10:00:00Z\n---\n\n# Test\n\nContent.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	err := runSyncPush(t.Context(), client, &stderr, mdFile, "", "", false, true)
	if err != nil {
		t.Fatalf("runSyncPush with --force: %v", err)
	}
	if appendCalls != 1 {
		t.Errorf("expected 1 append call, got %d", appendCalls)
	}
}

func TestPageSyncPush_UpdatesTitle(t *testing.T) {
	var updatedTitle string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "12345678-1234-1234-1234-123456789012",
				"last_edited_time": "2026-02-13T09:00:00.000Z",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Old Title"}},
					},
				},
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pages/") && !strings.Contains(r.URL.Path, "/blocks/"):
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if props, ok := body["properties"].(map[string]interface{}); ok {
				if titleProp, ok := props["title"].(map[string]interface{}); ok {
					if titleArr, ok := titleProp["title"].([]interface{}); ok && len(titleArr) > 0 {
						if first, ok := titleArr[0].(map[string]interface{}); ok {
							if textObj, ok := first["text"].(map[string]interface{}); ok {
								updatedTitle, _ = textObj["content"].(string)
							}
						}
					}
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "list",
				"results":  []map[string]interface{}{},
				"has_more": false,
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "title-update.md")
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\ntitle: New Title\nlast-synced: 2026-02-13T10:00:00Z\n---\n\n# New Title\n\nContent.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	err := runSyncPush(t.Context(), client, &stderr, mdFile, "", "", false, true)
	if err != nil {
		t.Fatalf("runSyncPush: %v", err)
	}

	if updatedTitle != "New Title" {
		t.Errorf("expected title update to 'New Title', got %q", updatedTitle)
	}
}

func TestPageSyncPush_UpdatesTitleFromH1(t *testing.T) {
	var updatedTitle string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":           "page",
				"id":               "12345678-1234-1234-1234-123456789012",
				"last_edited_time": "2026-02-13T09:00:00.000Z",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Old Title"}},
					},
				},
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pages/") && !strings.Contains(r.URL.Path, "/blocks/"):
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if props, ok := body["properties"].(map[string]interface{}); ok {
				if titleProp, ok := props["title"].(map[string]interface{}); ok {
					if titleArr, ok := titleProp["title"].([]interface{}); ok && len(titleArr) > 0 {
						if first, ok := titleArr[0].(map[string]interface{}); ok {
							if textObj, ok := first["text"].(map[string]interface{}); ok {
								updatedTitle, _ = textObj["content"].(string)
							}
						}
					}
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     "12345678-1234-1234-1234-123456789012",
			})

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":   "list",
				"results":  []map[string]interface{}{},
				"has_more": false,
			})

		case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/blocks/") && strings.HasSuffix(r.URL.Path, "/children"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "list",
				"results": []map[string]interface{}{},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "title-from-h1.md")
	// No title in frontmatter, should extract from first H1
	content := "---\nnotion-id: 12345678-1234-1234-1234-123456789012\nlast-synced: 2026-02-13T10:00:00Z\n---\n\n# Heading Title\n\nContent.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	client := newTestSyncClient(t, srv)

	var stderr strings.Builder
	err := runSyncPush(t.Context(), client, &stderr, mdFile, "", "", false, true)
	if err != nil {
		t.Fatalf("runSyncPush: %v", err)
	}

	if updatedTitle != "Heading Title" {
		t.Errorf("expected title update to 'Heading Title', got %q", updatedTitle)
	}
}

// newTestSyncClient creates a notion.Client pointed at a test server.
func newTestSyncClient(t *testing.T, srv *httptest.Server) *notion.Client {
	t.Helper()
	client := notion.NewClient("test-token")
	client.WithBaseURL(fmt.Sprintf("%s/v1", srv.URL))
	return client
}
