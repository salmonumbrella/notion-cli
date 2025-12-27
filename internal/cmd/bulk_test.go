package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		clause    string
		wantProp  string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple key=value",
			clause:    "Status=Done",
			wantProp:  "Status",
			wantValue: "Done",
		},
		{
			name:      "value with spaces",
			clause:    "Status=In Progress",
			wantProp:  "Status",
			wantValue: "In Progress",
		},
		{
			name:      "key with spaces",
			clause:    " Priority = High ",
			wantProp:  "Priority",
			wantValue: "High",
		},
		{
			name:      "value with equals sign",
			clause:    "Formula=x=1",
			wantProp:  "Formula",
			wantValue: "x=1",
		},
		{
			name:    "no equals sign",
			clause:  "StatusDone",
			wantErr: true,
		},
		{
			name:    "empty property name",
			clause:  "=Done",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, value, err := parseWhereClause(tt.clause)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if prop != tt.wantProp {
				t.Errorf("prop = %q, want %q", prop, tt.wantProp)
			}
			if value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestBuildFilterForType(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		propType string
		value    string
		wantJSON string
		wantErr  bool
	}{
		{
			name:     "status property",
			propName: "Status",
			propType: "status",
			value:    "Done",
			wantJSON: `{"property":"Status","status":{"equals":"Done"}}`,
		},
		{
			name:     "select property",
			propName: "Priority",
			propType: "select",
			value:    "High",
			wantJSON: `{"property":"Priority","select":{"equals":"High"}}`,
		},
		{
			name:     "checkbox true",
			propName: "Complete",
			propType: "checkbox",
			value:    "true",
			wantJSON: `{"checkbox":{"equals":true},"property":"Complete"}`,
		},
		{
			name:     "checkbox false",
			propName: "Complete",
			propType: "checkbox",
			value:    "false",
			wantJSON: `{"checkbox":{"equals":false},"property":"Complete"}`,
		},
		{
			name:     "checkbox case insensitive",
			propName: "Complete",
			propType: "checkbox",
			value:    "TRUE",
			wantJSON: `{"checkbox":{"equals":true},"property":"Complete"}`,
		},
		{
			name:     "title property",
			propName: "Name",
			propType: "title",
			value:    "Test",
			wantJSON: `{"property":"Name","title":{"contains":"Test"}}`,
		},
		{
			name:     "rich_text property",
			propName: "Notes",
			propType: "rich_text",
			value:    "important",
			wantJSON: `{"property":"Notes","rich_text":{"contains":"important"}}`,
		},
		{
			name:     "number property",
			propName: "Score",
			propType: "number",
			value:    "42.5",
			wantJSON: `{"number":{"equals":42.5},"property":"Score"}`,
		},
		{
			name:     "number property integer",
			propName: "Count",
			propType: "number",
			value:    "7",
			wantJSON: `{"number":{"equals":7},"property":"Count"}`,
		},
		{
			name:     "invalid number",
			propName: "Score",
			propType: "number",
			value:    "abc",
			wantErr:  true,
		},
		{
			name:     "unsupported type",
			propName: "Files",
			propType: "files",
			value:    "test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFilterForType(tt.propName, tt.propType, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != tt.wantJSON {
				t.Errorf("got  %s\nwant %s", gotJSON, tt.wantJSON)
			}
		})
	}
}

func TestParseWhereClauses_SingleFilter(t *testing.T) {
	schema := map[string]interface{}{
		"Status": map[string]interface{}{"type": "status"},
	}

	filter, err := parseWhereClauses([]string{"Status=Done"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotJSON, _ := json.Marshal(filter)
	wantJSON := `{"property":"Status","status":{"equals":"Done"}}`
	if string(gotJSON) != wantJSON {
		t.Errorf("got  %s\nwant %s", gotJSON, wantJSON)
	}
}

func TestParseWhereClauses_MultipleFilters_AND(t *testing.T) {
	schema := map[string]interface{}{
		"Status":   map[string]interface{}{"type": "status"},
		"Priority": map[string]interface{}{"type": "select"},
	}

	filter, err := parseWhereClauses([]string{"Status=Done", "Priority=Low"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	and, ok := filter["and"].([]interface{})
	if !ok {
		t.Fatalf("expected filter.and array, got %#v", filter)
	}
	if len(and) != 2 {
		t.Fatalf("expected 2 items in and array, got %d", len(and))
	}

	// Verify each sub-filter
	first, ok := and[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for first filter, got %T", and[0])
	}
	if first["property"] != "Status" {
		t.Errorf("first filter property = %q, want Status", first["property"])
	}

	second, ok := and[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for second filter, got %T", and[1])
	}
	if second["property"] != "Priority" {
		t.Errorf("second filter property = %q, want Priority", second["property"])
	}
}

func TestParseWhereClauses_FuzzyPropertyMatch(t *testing.T) {
	schema := map[string]interface{}{
		"Due Date": map[string]interface{}{"type": "rich_text"},
	}

	// "due_date" should fuzzy-match "Due Date"
	filter, err := parseWhereClauses([]string{"due_date=tomorrow"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The canonical name should be "Due Date"
	if filter["property"] != "Due Date" {
		t.Errorf("property = %q, want %q", filter["property"], "Due Date")
	}
}

func TestParseWhereClauses_UnknownProperty(t *testing.T) {
	schema := map[string]interface{}{
		"Status": map[string]interface{}{"type": "status"},
	}

	_, err := parseWhereClauses([]string{"Unknown=Done"}, schema)
	if err == nil {
		t.Fatal("expected error for unknown property, got nil")
	}
}

func TestBuildPropertyPayloadForType(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		propType string
		value    string
		wantJSON string
		wantErr  bool
	}{
		{
			name:     "status property",
			propName: "Status",
			propType: "status",
			value:    "Done",
			wantJSON: `{"status":{"name":"Done"}}`,
		},
		{
			name:     "select property",
			propName: "Priority",
			propType: "select",
			value:    "High",
			wantJSON: `{"select":{"name":"High"}}`,
		},
		{
			name:     "checkbox true",
			propName: "Complete",
			propType: "checkbox",
			value:    "true",
			wantJSON: `{"checkbox":true}`,
		},
		{
			name:     "checkbox false",
			propName: "Complete",
			propType: "checkbox",
			value:    "false",
			wantJSON: `{"checkbox":false}`,
		},
		{
			name:     "title property",
			propName: "Name",
			propType: "title",
			value:    "Test Title",
			wantJSON: `{"title":[{"text":{"content":"Test Title"}}]}`,
		},
		{
			name:     "rich_text property",
			propName: "Notes",
			propType: "rich_text",
			value:    "Some notes",
			wantJSON: `{"rich_text":[{"text":{"content":"Some notes"}}]}`,
		},
		{
			name:     "number property",
			propName: "Score",
			propType: "number",
			value:    "99.5",
			wantJSON: `{"number":99.5}`,
		},
		{
			name:     "people property",
			propName: "Assignee",
			propType: "people",
			value:    "user-id-123",
			wantJSON: `{"people":[{"id":"user-id-123"}]}`,
		},
		{
			name:     "unsupported type",
			propName: "Files",
			propType: "files",
			value:    "test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPropertyPayloadForType(tt.propName, tt.propType, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != tt.wantJSON {
				t.Errorf("got  %s\nwant %s", gotJSON, tt.wantJSON)
			}
		})
	}
}

func TestParseSetClauses(t *testing.T) {
	schema := map[string]interface{}{
		"Status":   map[string]interface{}{"type": "status"},
		"Priority": map[string]interface{}{"type": "select"},
		"DRI":      map[string]interface{}{"type": "people"},
	}

	props, err := parseSetClauses([]string{
		"Status=In Progress",
		"Priority=High",
		"DRI=user-id-123",
	}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(props) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(props))
	}

	// Check Status
	statusProp, ok := props["Status"].(map[string]interface{})
	if !ok {
		t.Fatal("Status property not a map")
	}
	statusInner, ok := statusProp["status"].(map[string]interface{})
	if !ok {
		t.Fatal("Status.status not a map")
	}
	if statusInner["name"] != "In Progress" {
		t.Errorf("Status.status.name = %q, want %q", statusInner["name"], "In Progress")
	}
}

func TestParseSetClauses_InvalidFormat(t *testing.T) {
	schema := map[string]interface{}{
		"Status": map[string]interface{}{"type": "status"},
	}

	_, err := parseSetClauses([]string{"StatusDone"}, schema)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestResolveSchemaProperty(t *testing.T) {
	schema := map[string]interface{}{
		"Status":   map[string]interface{}{"type": "status"},
		"Due Date": map[string]interface{}{"type": "date"},
	}

	tests := []struct {
		name          string
		input         string
		wantCanonical string
		wantType      string
		wantErr       bool
	}{
		{
			name:          "exact match",
			input:         "Status",
			wantCanonical: "Status",
			wantType:      "status",
		},
		{
			name:          "fuzzy match with underscores",
			input:         "due_date",
			wantCanonical: "Due Date",
			wantType:      "date",
		},
		{
			name:          "fuzzy match case insensitive",
			input:         "STATUS",
			wantCanonical: "Status",
			wantType:      "status",
		},
		{
			name:    "unknown property",
			input:   "Nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, typ, err := resolveSchemaProperty(schema, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if canonical != tt.wantCanonical {
				t.Errorf("canonical = %q, want %q", canonical, tt.wantCanonical)
			}
			if typ != tt.wantType {
				t.Errorf("type = %q, want %q", typ, tt.wantType)
			}
		})
	}
}

// bulkTestRoot creates a root command with the bulk subcommand registered for testing.
func bulkTestRoot(out, errBuf *bytes.Buffer) *App {
	return &App{Stdout: out, Stderr: errBuf}
}

// bulkTestCommand creates a root cobra.Command with newBulkCmd registered.
func bulkTestCommand(app *App) *cobra.Command {
	root := app.RootCommand()
	root.AddCommand(newBulkCmd())
	return root
}

func TestBulkUpdate_ErrorWhenNoWhere(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "update", "12345678123412341234123456789012",
		"--set", "Status=Done",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --where is not provided")
	}
	if !strings.Contains(err.Error(), "--where is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBulkUpdate_ErrorWhenNoSet(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "update", "12345678123412341234123456789012",
		"--where", "Status=Done",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --set is not provided")
	}
	if !strings.Contains(err.Error(), "--set is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBulkArchive_ErrorWhenNoWhere(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "archive", "12345678123412341234123456789012",
		"--yes",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --where is not provided")
	}
	if !strings.Contains(err.Error(), "--where is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBulkUpdate_DryRun_NoUpdateCalls(t *testing.T) {
	const (
		dbID   = "12345678123412341234123456789012"
		dsID   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		pageID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	t.Setenv("NOTION_TOKEN", "test-token")

	var mu sync.Mutex
	updateCalled := false

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
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "data_source",
				"id":     dsID,
				"properties": map[string]any{
					"Status": map[string]any{"type": "status"},
				},
			})
			return
		}
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
							"type": "title",
							"title": []map[string]any{
								{"plain_text": "Test Page"},
							},
						},
					},
				},
			},
			"has_more": false,
		})
	})
	mux.HandleFunc("/pages/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			mu.Lock()
			updateCalled = true
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "page",
			"id":     pageID,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "update", dbID,
		"--where", "Status=Done",
		"--set", "Status=Archived",
		"--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, errBuf.String())
	}

	mu.Lock()
	called := updateCalled
	mu.Unlock()

	if called {
		t.Fatal("update API was called during --dry-run")
	}

	// Check stderr has dry-run output
	if !strings.Contains(errBuf.String(), "[DRY-RUN]") {
		t.Errorf("expected dry-run message in stderr, got: %s", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Test Page") {
		t.Errorf("expected page title in dry-run output, got: %s", errBuf.String())
	}
}

func TestBulkUpdate_Integration_QueryAndUpdate(t *testing.T) {
	const (
		dbID    = "12345678123412341234123456789012"
		dsID    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		pageID1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		pageID2 = "cccccccccccccccccccccccccccccccc"
	)

	t.Setenv("NOTION_TOKEN", "test-token")

	var mu sync.Mutex
	var queryFilter map[string]any
	updatedPages := make(map[string]map[string]any)

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
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "data_source",
				"id":     dsID,
				"properties": map[string]any{
					"Status":   map[string]any{"type": "status"},
					"Priority": map[string]any{"type": "select"},
				},
			})
			return
		}
	})
	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode query: %v", err)
		}
		mu.Lock()
		if f, ok := body["filter"].(map[string]any); ok {
			queryFilter = f
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{
					"object": "page",
					"id":     pageID1,
					"properties": map[string]any{
						"Name":   map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "Page 1"}}},
						"Status": map[string]any{"type": "status", "status": map[string]any{"name": "Done"}},
					},
				},
				{
					"object": "page",
					"id":     pageID2,
					"properties": map[string]any{
						"Name":   map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "Page 2"}}},
						"Status": map[string]any{"type": "status", "status": map[string]any{"name": "Done"}},
					},
				},
			},
			"has_more": false,
		})
	})
	mux.HandleFunc("/pages/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode update: %v", err)
			}
			// Extract page ID from URL path
			pageID := r.URL.Path[len("/pages/"):]
			mu.Lock()
			updatedPages[pageID] = body
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "page",
			"id":     "updated",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "update", dbID,
		"--where", "Status=Done",
		"--set", "Status=Archived",
		"--set", "Priority=Low",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, errBuf.String())
	}

	// Verify filter sent to query
	mu.Lock()
	defer mu.Unlock()
	if queryFilter == nil {
		t.Fatal("expected query filter to be set")
	}
	if queryFilter["property"] != "Status" {
		t.Errorf("filter property = %q, want Status", queryFilter["property"])
	}

	// Verify both pages were updated
	if len(updatedPages) != 2 {
		t.Errorf("expected 2 pages updated, got %d", len(updatedPages))
	}

	// Verify the properties sent in the update
	for pageID, body := range updatedPages {
		props, ok := body["properties"].(map[string]any)
		if !ok {
			t.Errorf("page %s: expected properties map, got %T", pageID, body["properties"])
			continue
		}
		statusProp, ok := props["Status"].(map[string]any)
		if !ok {
			t.Errorf("page %s: expected Status property map", pageID)
			continue
		}
		statusInner, ok := statusProp["status"].(map[string]any)
		if !ok {
			t.Errorf("page %s: expected Status.status map", pageID)
			continue
		}
		if statusInner["name"] != "Archived" {
			t.Errorf("page %s: Status.status.name = %q, want Archived", pageID, statusInner["name"])
		}
	}

	// Verify summary in stderr
	if !strings.Contains(errBuf.String(), "Updated 2 page(s)") {
		t.Errorf("expected summary in stderr, got: %s", errBuf.String())
	}
}

func TestBulkArchive_Integration(t *testing.T) {
	const (
		dbID   = "12345678123412341234123456789012"
		dsID   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		pageID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	t.Setenv("NOTION_TOKEN", "test-token")

	var mu sync.Mutex
	var updateBody map[string]any

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
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "data_source",
				"id":     dsID,
				"properties": map[string]any{
					"Status": map[string]any{"type": "status"},
				},
			})
			return
		}
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
						"Name": map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "To Archive"}}},
					},
				},
			},
			"has_more": false,
		})
	})
	mux.HandleFunc("/pages/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode update: %v", err)
			}
			mu.Lock()
			updateBody = body
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":   "page",
			"id":       pageID,
			"archived": true,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "archive", dbID,
		"--where", "Status=Cancelled",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, errBuf.String())
	}

	// Verify archived was set to true
	mu.Lock()
	defer mu.Unlock()
	if updateBody == nil {
		t.Fatal("expected update call")
	}
	archived, ok := updateBody["archived"].(bool)
	if !ok || !archived {
		t.Errorf("expected archived=true, got %v", updateBody["archived"])
	}

	// Verify summary
	if !strings.Contains(errBuf.String(), "Archived 1 page(s)") {
		t.Errorf("expected archive summary in stderr, got: %s", errBuf.String())
	}
}

func TestBulkUpdate_Limit(t *testing.T) {
	const (
		dbID    = "12345678123412341234123456789012"
		dsID    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		pageID1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		pageID2 = "cccccccccccccccccccccccccccccccc"
		pageID3 = "dddddddddddddddddddddddddddddddd"
	)

	t.Setenv("NOTION_TOKEN", "test-token")

	var mu sync.Mutex
	updateCount := 0

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
				"Status": map[string]any{"type": "status"},
			},
		})
	})
	mux.HandleFunc("/data_sources/"+dsID+"/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"results": []map[string]any{
				{"object": "page", "id": pageID1, "properties": map[string]any{"Name": map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "P1"}}}}},
				{"object": "page", "id": pageID2, "properties": map[string]any{"Name": map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "P2"}}}}},
				{"object": "page", "id": pageID3, "properties": map[string]any{"Name": map[string]any{"type": "title", "title": []map[string]any{{"plain_text": "P3"}}}}},
			},
			"has_more": false,
		})
	})
	mux.HandleFunc("/pages/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			mu.Lock()
			updateCount++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "page", "id": "x"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)

	var out, errBuf bytes.Buffer
	app := bulkTestRoot(&out, &errBuf)
	root := bulkTestCommand(app)

	root.SetArgs([]string{
		"bulk", "update", dbID,
		"--where", "Status=Done",
		"--set", "Status=Archived",
		"--limit", "2",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v\nstderr=%s", err, errBuf.String())
	}

	mu.Lock()
	count := updateCount
	mu.Unlock()

	if count != 2 {
		t.Errorf("expected 2 updates (limited), got %d", count)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    interface{ Milliseconds() int64 }
		want string
	}{
		// We'll test via the function directly
	}
	_ = tests

	// Sub-second
	if got := formatDuration(500 * 1e6); got != "500ms" {
		t.Errorf("formatDuration(500ms) = %q, want %q", got, "500ms")
	}
	// Seconds
	if got := formatDuration(1300 * 1e6); got != "1.3s" {
		t.Errorf("formatDuration(1.3s) = %q, want %q", got, "1.3s")
	}
}
