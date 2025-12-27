package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootLightMode_DefaultsToCompactJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "p1",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Result"}},
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
	app := &App{Stdout: &out, Stderr: &bytes.Buffer{}}
	root := app.RootCommand()
	root.SetArgs([]string{"search", "result", "--li"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	raw := out.String()
	if strings.Contains(raw, "\n  ") {
		t.Fatalf("expected compact JSON in light mode, got pretty JSON:\n%s", raw)
	}
}

func TestRootLightMode_CompactJSONCanBeDisabled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "p1",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Result"}},
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
	app := &App{Stdout: &out, Stderr: &bytes.Buffer{}}
	root := app.RootCommand()
	root.SetArgs([]string{"search", "result", "--li", "--cj=false", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "\n  ") {
		t.Fatalf("expected pretty JSON when compact mode is disabled, got:\n%s", raw)
	}
}

func TestRootLightMode_OutAliasTextIsRespected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "p1",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Result"}},
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
	app := &App{Stdout: &out, Stderr: &bytes.Buffer{}}
	root := app.RootCommand()
	root.SetArgs([]string{"search", "result", "--li", "--out", "text"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	raw := strings.TrimSpace(out.String())
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		t.Fatalf("expected text output with explicit --out text, got JSON:\n%s", raw)
	}
	if !strings.Contains(raw, "ID") || !strings.Contains(raw, "OBJECT") {
		t.Fatalf("expected tabular text output with explicit --out text, got:\n%s", raw)
	}
}

func TestCommandFlagChanged_DetectsParentPersistentAlias(t *testing.T) {
	var sawOutChanged bool
	var sawOutputChanged bool

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("output", "text", "output format")
	flagAlias(root.PersistentFlags(), "output", "out")

	child := &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
			sawOutChanged = commandFlagChanged(cmd, "out")
			sawOutputChanged = commandFlagChanged(cmd, "output")
			return nil
		},
	}
	root.AddCommand(child)

	root.SetArgs([]string{"child", "--out", "text"})
	if err := root.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !sawOutChanged {
		t.Fatal("expected commandFlagChanged(cmd, \"out\") to be true")
	}
	if sawOutputChanged {
		t.Fatal("expected canonical output flag to remain unchanged when alias is used")
	}
}
