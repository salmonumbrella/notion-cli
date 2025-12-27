package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToLightSearchResults(t *testing.T) {
	results := []map[string]interface{}{
		{
			"object": "page",
			"id":     "page-1",
			"url":    "https://example.invalid/page-1",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{map[string]interface{}{"plain_text": "Roadmap"}},
				},
			},
		},
		{
			"object": "data_source",
			"id":     "ds-1",
			"title":  []interface{}{map[string]interface{}{"plain_text": "Projects"}},
		},
		{
			"object": "page",
			"id":     "",
		},
	}

	light := toLightSearchResults(results)
	if len(light) != 2 {
		t.Fatalf("expected 2 light results, got %d", len(light))
	}

	if light[0].ID != "page-1" || light[0].Object != "page" || light[0].Title != "Roadmap" {
		t.Fatalf("unexpected first result: %#v", light[0])
	}
	if light[1].ID != "ds-1" || light[1].Object != "ds" || light[1].Title != "Projects" {
		t.Fatalf("unexpected second result: %#v", light[1])
	}
}

func TestSearchCmd_LightAliasFlags(t *testing.T) {
	cmd := newSearchCmd()

	lightFlag := cmd.Flags().Lookup("light")
	if lightFlag == nil {
		t.Fatal("expected --light flag")
	}

	liFlag := cmd.Flags().Lookup("li")
	if liFlag == nil {
		t.Fatal("expected --li alias flag")
	}
	if !liFlag.Hidden {
		t.Fatal("--li alias should be hidden")
	}

	if err := cmd.Flags().Set("li", "true"); err != nil {
		t.Fatalf("set --li: %v", err)
	}
	light, err := cmd.Flags().GetBool("light")
	if err != nil {
		t.Fatalf("get --light: %v", err)
	}
	if !light {
		t.Fatal("--li should set --light")
	}
}

func TestSearch_LightOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST /search, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "p1",
					"url":    "https://example.invalid/p1",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Search Result"}},
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
	root.SetArgs([]string{"search", "result", "--li", "--output", "json"})
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
	if got := first["id"]; got != "p1" {
		t.Fatalf("expected id p1, got %#v", got)
	}
	if got := first["object"]; got != "page" {
		t.Fatalf("expected object page, got %#v", got)
	}
	if got := first["title"]; got != "Search Result" {
		t.Fatalf("expected title Search Result, got %#v", got)
	}
	if _, exists := first["properties"]; exists {
		t.Fatalf("did not expect properties in light output: %#v", first)
	}
}
