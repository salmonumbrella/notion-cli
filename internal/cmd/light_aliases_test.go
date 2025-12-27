package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func TestLightAliases_OnListCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "search", cmd: newSearchCmd()},
		{name: "db list", cmd: newDBListCmd()},
		{name: "datasource list", cmd: newDataSourceListCmd()},
		{name: "page list", cmd: newPageListCmd()},
		{name: "comment list", cmd: newCommentListCmd()},
		{name: "file list", cmd: newFileListCmd()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lightFlag := tt.cmd.Flags().Lookup("light")
			if lightFlag == nil {
				t.Fatal("expected --light flag")
			}

			liFlag := tt.cmd.Flags().Lookup("li")
			if liFlag == nil {
				t.Fatal("expected --li alias flag")
			}
			if !liFlag.Hidden {
				t.Fatal("--li alias should be hidden")
			}

			if err := tt.cmd.Flags().Set("li", "true"); err != nil {
				t.Fatalf("set --li: %v", err)
			}
			light, err := tt.cmd.Flags().GetBool("light")
			if err != nil {
				t.Fatalf("get --light: %v", err)
			}
			if !light {
				t.Fatal("--li should set --light")
			}
		})
	}
}

func TestPageList_LightForwarding(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST /search, got %s", r.Method)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		filter, _ := req["filter"].(map[string]interface{})
		if got, _ := filter["value"].(string); got != "page" {
			t.Fatalf("expected page filter, got %#v", req["filter"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "p-light",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Page Light"}},
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
	root.SetArgs([]string{"page", "list", "--li", "--output", "json"})
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
	if got := first["id"]; got != "p-light" {
		t.Fatalf("expected id p-light, got %#v", got)
	}
	if got := first["title"]; got != "Page Light" {
		t.Fatalf("expected title Page Light, got %#v", got)
	}
	if _, exists := first["properties"]; exists {
		t.Fatalf("did not expect properties in light output: %#v", first)
	}
}
