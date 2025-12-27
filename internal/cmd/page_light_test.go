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

func TestPageGetCmd_LightAlias(t *testing.T) {
	cmd := newPageGetCmd()

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

func TestPageGet_LightOutput(t *testing.T) {
	const (
		rawID = "11111111111111111111111111111111"
		url   = "https://example.invalid/pages/light-page-1"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/pages/"+rawID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET /pages/<id>, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               rawID,
			"url":              url,
			"created_time":     "2025-06-04T06:07:00.000Z",
			"last_edited_time": "2025-06-04T06:07:00.000Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Sample Light Page"},
					},
				},
				"Status": map[string]interface{}{
					"type": "status",
					"status": map[string]interface{}{
						"name": "Pending",
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

	var out bytes.Buffer
	app := &App{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	root.SetArgs([]string{"page", "get", rawID, "--li", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output JSON: %v\nraw: %s", err, out.String())
	}

	if got := payload["id"]; got != rawID {
		t.Fatalf("expected id %q, got %#v", rawID, got)
	}
	if got := payload["object"]; got != "page" {
		t.Fatalf("expected object page, got %#v", got)
	}
	if got := payload["title"]; got != "Sample Light Page" {
		t.Fatalf("expected title Sample Light Page, got %#v", got)
	}
	if got := payload["url"]; got != url {
		t.Fatalf("expected url %q, got %#v", url, got)
	}
	if _, exists := payload["properties"]; exists {
		t.Fatalf("did not expect properties in light output: %#v", payload)
	}
}

func TestPageGet_LightOutput_DefaultsToCompactJSON(t *testing.T) {
	const (
		rawID = "22222222222222222222222222222222"
		url   = "https://example.invalid/pages/light-page-2"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/pages/"+rawID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":           "page",
			"id":               rawID,
			"url":              url,
			"created_time":     "2025-06-04T06:07:00.000Z",
			"last_edited_time": "2025-06-04T06:07:00.000Z",
			"archived":         false,
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Sample Compact Page"},
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

	var out bytes.Buffer
	app := &App{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}

	root := app.RootCommand()
	root.SetArgs([]string{"page", "get", rawID, "--li"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	raw := out.String()
	if strings.Contains(raw, "\n  ") {
		t.Fatalf("expected compact JSON output, got pretty JSON:\n%s", raw)
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		t.Fatalf("expected JSON object output, got: %s", raw)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output JSON: %v\nraw: %s", err, raw)
	}
	if got := payload["id"]; got != rawID {
		t.Fatalf("expected id %q, got %#v", rawID, got)
	}
	if got := payload["title"]; got != "Sample Compact Page" {
		t.Fatalf("expected title Sample Compact Page, got %#v", got)
	}
}

func TestPageGet_LightIncompatibleFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOTION_TOKEN", "test-token")

	cases := []struct {
		name     string
		args     []string
		wantText string
	}{
		{
			name:     "editable",
			args:     []string{"page", "get", "33333333333333333333333333333333", "--li", "--editable"},
			wantText: "--light cannot be combined with --editable",
		},
		{
			name:     "enrich",
			args:     []string{"page", "get", "33333333333333333333333333333333", "--li", "--enrich"},
			wantText: "--light cannot be combined with --enrich",
		},
		{
			name:     "include-children",
			args:     []string{"page", "get", "33333333333333333333333333333333", "--li", "--include-children"},
			wantText: "--light cannot be combined with --include-children",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
			}

			root := app.RootCommand()
			root.SetArgs(tc.args)
			err := root.ExecuteContext(context.Background())
			if err == nil {
				t.Fatalf("expected command error containing %q", tc.wantText)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("expected error containing %q, got %q", tc.wantText, err.Error())
			}
		})
	}
}
