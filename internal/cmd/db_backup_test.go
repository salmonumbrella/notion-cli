package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugifyDBTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple title",
			input: "My Projects",
			want:  "my-projects",
		},
		{
			name:  "already lowercase",
			input: "tasks",
			want:  "tasks",
		},
		{
			name:  "special characters stripped",
			input: "My (Cool) Project!",
			want:  "my-cool-project",
		},
		{
			name:  "underscores become hyphens",
			input: "my_project_db",
			want:  "my-project-db",
		},
		{
			name:  "multiple spaces collapsed",
			input: "my   project",
			want:  "my-project",
		},
		{
			name:  "leading and trailing spaces trimmed",
			input: "  my project  ",
			want:  "my-project",
		},
		{
			name:  "numbers preserved",
			input: "Project 2024",
			want:  "project-2024",
		},
		{
			name:  "empty string returns untitled",
			input: "",
			want:  "untitled",
		},
		{
			name:  "only special characters returns untitled",
			input: "!!!@@@###",
			want:  "untitled",
		},
		{
			name:  "long title truncated to 50 chars",
			input: "this is a very long database title that should be truncated to fifty characters maximum",
			want:  "this-is-a-very-long-database-title-that-should-be",
		},
		{
			name:  "truncation does not end on hyphen",
			input: "this is a very long database title that should be-truncated here",
			want:  "this-is-a-very-long-database-title-that-should-be",
		},
		{
			name:  "mixed case and special chars",
			input: "Team Alpha -- Sprint Board (Q1/2024)",
			want:  "team-alpha-sprint-board-q12024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugifyDBTitle(tt.input)
			if got != tt.want {
				t.Errorf("slugifyDBTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadBackupMeta(t *testing.T) {
	t.Run("valid meta file", func(t *testing.T) {
		dir := t.TempDir()
		metaPath := filepath.Join(dir, ".backup-meta.json")
		content := `{"last_backup":"2026-02-12T10:00:00Z","page_count":42,"version":1}`
		if err := os.WriteFile(metaPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write meta file: %v", err)
		}

		meta, err := readBackupMeta(metaPath)
		if err != nil {
			t.Fatalf("readBackupMeta() error = %v", err)
		}
		if meta.LastBackup != "2026-02-12T10:00:00Z" {
			t.Errorf("LastBackup = %q, want %q", meta.LastBackup, "2026-02-12T10:00:00Z")
		}
		if meta.PageCount != 42 {
			t.Errorf("PageCount = %d, want %d", meta.PageCount, 42)
		}
		if meta.Version != 1 {
			t.Errorf("Version = %d, want %d", meta.Version, 1)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := readBackupMeta("/nonexistent/path/.backup-meta.json")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		metaPath := filepath.Join(dir, ".backup-meta.json")
		if err := os.WriteFile(metaPath, []byte("not json"), 0o644); err != nil {
			t.Fatalf("failed to write meta file: %v", err)
		}

		_, err := readBackupMeta(metaPath)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}

func TestToInterfaceSlice(t *testing.T) {
	input := []map[string]interface{}{
		{"plain_text": "Hello"},
		{"plain_text": "World"},
	}
	result := toInterfaceSlice(input)
	items, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestDBBackup_FullBackup(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"
	pageID := "page-1111-2222-3333-4444-555555555555"

	mux := http.NewServeMux()

	// GET /databases/{id} - return database schema
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Test Database"},
			},
			"properties": map[string]any{
				"Name": map[string]any{"type": "title", "title": map[string]any{}},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	// POST /data_sources/{id}/query - return pages
	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object": "page",
					"id":     pageID,
					"properties": map[string]any{
						"Name": map[string]any{
							"type":  "title",
							"title": []map[string]any{{"plain_text": "Test Page"}},
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
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	// Verify directory structure
	backupDir := filepath.Join(outputDir, "test-database")

	// schema.json exists and is valid JSON
	schemaData, err := os.ReadFile(filepath.Join(backupDir, "schema.json"))
	if err != nil {
		t.Fatalf("schema.json not found: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("schema.json is not valid JSON: %v", err)
	}
	if schema["id"] != dbID {
		t.Errorf("schema.id = %v, want %v", schema["id"], dbID)
	}

	// .backup-meta.json exists and has correct format
	metaData, err := os.ReadFile(filepath.Join(backupDir, ".backup-meta.json"))
	if err != nil {
		t.Fatalf(".backup-meta.json not found: %v", err)
	}
	var meta backupMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf(".backup-meta.json is not valid JSON: %v", err)
	}
	if meta.PageCount != 1 {
		t.Errorf("meta.PageCount = %d, want 1", meta.PageCount)
	}
	if meta.Version != 1 {
		t.Errorf("meta.Version = %d, want 1", meta.Version)
	}
	// Verify timestamp is recent (within last minute)
	ts, err := time.Parse(time.RFC3339, meta.LastBackup)
	if err != nil {
		t.Fatalf("failed to parse LastBackup timestamp: %v", err)
	}
	if time.Since(ts) > time.Minute {
		t.Errorf("LastBackup timestamp is too old: %s", meta.LastBackup)
	}

	// Page JSON exists
	pageData, err := os.ReadFile(filepath.Join(backupDir, "pages", pageID+".json"))
	if err != nil {
		t.Fatalf("page JSON not found: %v", err)
	}
	var page map[string]interface{}
	if err := json.Unmarshal(pageData, &page); err != nil {
		t.Fatalf("page JSON is not valid: %v", err)
	}
	if page["id"] != pageID {
		t.Errorf("page.id = %v, want %v", page["id"], pageID)
	}

	// Verify summary was printed to stderr
	if !strings.Contains(errBuf.String(), "Backed up 1 pages") {
		t.Errorf("expected summary in stderr, got: %s", errBuf.String())
	}
}

func TestDBBackup_Incremental(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"

	var receivedFilter map[string]interface{}

	mux := http.NewServeMux()

	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Incremental DB"},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		// Capture the filter from the request body
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if f, ok := body["filter"].(map[string]interface{}); ok {
			receivedFilter = f
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":      "list",
			"results":     []map[string]any{},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	// Pre-create a .backup-meta.json to simulate a previous backup
	backupDir := filepath.Join(outputDir, "incremental-db")
	if err := os.MkdirAll(filepath.Join(backupDir, "pages"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	prevTimestamp := "2026-02-10T12:00:00Z"
	metaContent, _ := json.Marshal(backupMeta{
		LastBackup: prevTimestamp,
		PageCount:  10,
		Version:    1,
	})
	if err := os.WriteFile(filepath.Join(backupDir, ".backup-meta.json"), metaContent, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir, "--incremental"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	// Verify that the filter included the last_edited_time constraint
	if receivedFilter == nil {
		t.Fatal("expected filter to be sent in query, got nil")
	}
	if receivedFilter["timestamp"] != "last_edited_time" {
		t.Errorf("filter.timestamp = %v, want last_edited_time", receivedFilter["timestamp"])
	}
	letFilter, ok := receivedFilter["last_edited_time"].(map[string]interface{})
	if !ok {
		t.Fatalf("filter.last_edited_time is not a map: %v", receivedFilter["last_edited_time"])
	}
	if letFilter["after"] != prevTimestamp {
		t.Errorf("filter.last_edited_time.after = %v, want %v", letFilter["after"], prevTimestamp)
	}
}

func TestDBBackup_WithContent(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"
	pageID := "page-content-1111-2222-333344445555"

	var blocksFetched bool

	mux := http.NewServeMux()

	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Content DB"},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object": "page",
					"id":     pageID,
					"properties": map[string]any{
						"Name": map[string]any{
							"type":  "title",
							"title": []map[string]any{{"plain_text": "Content Page"}},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	mux.HandleFunc("/blocks/"+pageID+"/children", func(w http.ResponseWriter, r *http.Request) {
		blocksFetched = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object":       "block",
					"id":           "block-1",
					"type":         "paragraph",
					"has_children": false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "Hello world"},
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
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir, "--content"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	if !blocksFetched {
		t.Fatal("expected block children to be fetched with --content flag")
	}

	// Verify blocks file was written
	blocksPath := filepath.Join(outputDir, "content-db", "pages", pageID+".blocks.json")
	blocksData, err := os.ReadFile(blocksPath)
	if err != nil {
		t.Fatalf("blocks file not found: %v", err)
	}

	var blocks []interface{}
	if err := json.Unmarshal(blocksData, &blocks); err != nil {
		t.Fatalf("blocks file is not valid JSON: %v", err)
	}
	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
}

func TestDBBackup_MarkdownFormat(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"
	pageID := "page-md-1111-2222-3333-444455556666"

	mux := http.NewServeMux()

	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Markdown DB"},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object": "page",
					"id":     pageID,
					"properties": map[string]any{
						"Name": map[string]any{
							"type":  "title",
							"title": []map[string]any{{"plain_text": "My Note"}},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	mux.HandleFunc("/blocks/"+pageID+"/children", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object":       "block",
					"id":           "block-md-1",
					"type":         "heading_2",
					"has_children": false,
					"heading_2": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "Section One"},
						},
					},
				},
				{
					"object":       "block",
					"id":           "block-md-2",
					"type":         "paragraph",
					"has_children": false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "Some content here."},
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
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir, "--export-format", "markdown"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	// Verify markdown file was written
	mdPath := filepath.Join(outputDir, "markdown-db", "pages", pageID+".md")
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("markdown file not found: %v", err)
	}

	mdContent := string(mdData)
	if !strings.Contains(mdContent, "# My Note") {
		t.Errorf("expected page title in markdown, got: %s", mdContent)
	}
	if !strings.Contains(mdContent, "## Section One") {
		t.Errorf("expected heading in markdown, got: %s", mdContent)
	}
	if !strings.Contains(mdContent, "Some content here.") {
		t.Errorf("expected paragraph in markdown, got: %s", mdContent)
	}

	// Page JSON should also be written
	pageJSONPath := filepath.Join(outputDir, "markdown-db", "pages", pageID+".json")
	if _, err := os.Stat(pageJSONPath); os.IsNotExist(err) {
		t.Error("page JSON file should also be written alongside markdown")
	}
}

func TestDBBackup_NoPages(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"

	mux := http.NewServeMux()

	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Empty DB"},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":      "list",
			"results":     []map[string]any{},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	// Should still create the directory and meta file
	backupDir := filepath.Join(outputDir, "empty-db")
	if _, err := os.Stat(filepath.Join(backupDir, "schema.json")); os.IsNotExist(err) {
		t.Error("schema.json should exist even for empty database")
	}

	metaData, err := os.ReadFile(filepath.Join(backupDir, ".backup-meta.json"))
	if err != nil {
		t.Fatalf(".backup-meta.json not found: %v", err)
	}
	var meta backupMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("invalid meta JSON: %v", err)
	}
	if meta.PageCount != 0 {
		t.Errorf("meta.PageCount = %d, want 0", meta.PageCount)
	}

	if !strings.Contains(errBuf.String(), "Backed up 0 pages") {
		t.Errorf("expected zero-page summary, got: %s", errBuf.String())
	}
}

func TestDBBackup_IncrementalWithoutPreviousMeta(t *testing.T) {
	// When --incremental is used but no previous .backup-meta.json exists,
	// it should fall through to a full backup (no filter applied).
	t.Setenv("NOTION_TOKEN", "test-token")

	dbID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dsID := "ds-11111111-2222-3333-4444-555555555555"

	var receivedFilter map[string]interface{}

	mux := http.NewServeMux()

	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"title": []map[string]any{
				{"plain_text": "Fresh DB"},
			},
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Default"},
			},
		})
	})

	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if f, ok := body["filter"].(map[string]interface{}); ok {
			receivedFilter = f
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":      "list",
			"results":     []map[string]any{},
			"has_more":    false,
			"next_cursor": nil,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	outputDir := t.TempDir()

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"db", "backup", dbID, "--output-dir", outputDir, "--incremental"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("backup failed: %v\nstderr=%s", err, errBuf.String())
	}

	// No filter should have been applied (full backup fallback)
	if receivedFilter != nil {
		t.Errorf("expected no filter for first incremental backup, got: %v", receivedFilter)
	}
}
