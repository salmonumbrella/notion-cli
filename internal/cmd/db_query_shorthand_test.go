package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDBQuery_ShorthandFilters_BuildsServerSideFilter(t *testing.T) {
	const (
		dbID = "12345678123412341234123456789012"
		dsID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		meID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NOTION_TOKEN", "test-token")

	// Skill file so --assignee me works.
	skillPath := filepath.Join(home, ".claude", "skills", "notion-cli", "notion-cli.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(`---
name: notion-cli
description: test
---

## Users

| Alias | Name | ID |
|-------|------|----|
| me | Me | `+meID+` |
`), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	var gotFilter map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "database",
			"id":     dbID,
			"data_sources": []map[string]any{
				{"id": dsID, "name": "Primary"},
			},
		})
	})
	mux.HandleFunc("/data_sources/"+dsID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "data_source",
			"id":     dsID,
			"properties": map[string]any{
				"Status":   map[string]any{"type": "status"},
				"Assignee": map[string]any{"type": "people"},
				"Priority": map[string]any{"type": "select"},
			},
		})
	})
	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode query: %v", err)
		}
		if f, ok := body["filter"].(map[string]any); ok {
			gotFilter = f
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"results":  []any{},
			"has_more": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"db", "query", dbID,
		"--status", "Done",
		"--assignee", "me",
		"--priority", "High",
		"--results-only",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("db query failed: %v\nstderr=%s", err, errBuf.String())
	}

	and, ok := gotFilter["and"].([]any)
	if !ok || len(and) != 3 {
		t.Fatalf("expected filter.and with 3 items, got %#v", gotFilter)
	}
}
