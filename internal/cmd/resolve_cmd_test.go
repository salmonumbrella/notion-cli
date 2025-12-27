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

func TestResolve_IncludesSkillAndSearchCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NOTION_TOKEN", "test-token")

	// Write a minimal skill file with a custom alias.
	skillPath := filepath.Join(home, ".claude", "skills", "notion-cli", "notion-cli.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(`---
name: notion-cli
description: test
---

## Custom Aliases

| Alias | Type | Target ID |
|-------|------|-----------|
| standup | page | page-skill-1 |
`), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object": "page",
					"id":     "page-search-2",
					"url":    "https://example.invalid/page-search-2",
					"properties": map[string]any{
						"Name": map[string]any{
							"type": "title",
							"title": []map[string]any{
								{"plain_text": "standup"},
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
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()
	root.SetArgs([]string{"--results-only", "resolve", "standup"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("resolve failed: %v\nstderr=%s", err, errBuf.String())
	}

	var got []ResolveCandidate
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal output: %v\nout=%s", err, out.String())
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d (%v)", len(got), got)
	}
}
