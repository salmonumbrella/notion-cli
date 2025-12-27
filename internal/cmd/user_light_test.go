package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserListCmd_LightAlias(t *testing.T) {
	cmd := newUserListCmd()

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

func TestUserList_LightOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET /users, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object":     "user",
					"id":         "u1",
					"type":       "person",
					"name":       "Alice",
					"avatar_url": "https://example.com/avatar.png",
					"person": map[string]interface{}{
						"email": "alice@example.com",
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
	root.SetArgs([]string{"user", "list", "--light", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output JSON: %v\nraw: %s", err, out.String())
	}

	resultsAny, ok := payload["results"]
	if !ok {
		t.Fatalf("expected results in output: %v", payload)
	}
	results, ok := resultsAny.([]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected one result, got %#v", resultsAny)
	}

	first, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %#v", results[0])
	}
	if got := first["id"]; got != "u1" {
		t.Fatalf("expected id u1, got %#v", got)
	}
	if got := first["name"]; got != "Alice" {
		t.Fatalf("expected name Alice, got %#v", got)
	}
	if got := first["email"]; got != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %#v", got)
	}
	if got := first["type"]; got != "person" {
		t.Fatalf("expected type person, got %#v", got)
	}

	if _, exists := first["avatar_url"]; exists {
		t.Fatalf("did not expect avatar_url in light output: %#v", first)
	}
	if _, exists := first["person"]; exists {
		t.Fatalf("did not expect person object in light output: %#v", first)
	}
}
