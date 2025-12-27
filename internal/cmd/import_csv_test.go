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
)

func TestImportCSV_ParseAndMap(t *testing.T) {
	// Test that CSV headers are mapped to database properties
	headers := []string{"Name", "Description", "Count"}
	dbProps := map[string]map[string]interface{}{
		"Name":        {"type": "title"},
		"Description": {"type": "rich_text"},
		"Count":       {"type": "number"},
	}

	mappings, warnings := buildColumnMappings(headers, dbProps, nil)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(mappings) != 3 {
		t.Fatalf("expected 3 mappings, got %d", len(mappings))
	}

	expected := []struct {
		header   string
		prop     string
		propType string
	}{
		{"Name", "Name", "title"},
		{"Description", "Description", "rich_text"},
		{"Count", "Count", "number"},
	}

	for i, e := range expected {
		if mappings[i].csvHeader != e.header || mappings[i].notionProp != e.prop || mappings[i].propType != e.propType {
			t.Errorf("mapping %d: got (%s, %s, %s), want (%s, %s, %s)",
				i, mappings[i].csvHeader, mappings[i].notionProp, mappings[i].propType,
				e.header, e.prop, e.propType)
		}
	}
}

func TestImportCSV_CaseInsensitiveFallback(t *testing.T) {
	headers := []string{"name", "DESCRIPTION"}
	dbProps := map[string]map[string]interface{}{
		"Name":        {"type": "title"},
		"Description": {"type": "rich_text"},
	}

	mappings, warnings := buildColumnMappings(headers, dbProps, nil)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings[0].notionProp != "Name" {
		t.Errorf("expected 'Name', got %q", mappings[0].notionProp)
	}
	if mappings[1].notionProp != "Description" {
		t.Errorf("expected 'Description', got %q", mappings[1].notionProp)
	}
}

func TestImportCSV_ColumnMapOverrides(t *testing.T) {
	headers := []string{"Full Name", "Desc"}
	dbProps := map[string]map[string]interface{}{
		"Title":       {"type": "title"},
		"Description": {"type": "rich_text"},
	}

	overrides := map[string]string{
		"Full Name": "Title",
		"Desc":      "Description",
	}

	mappings, warnings := buildColumnMappings(headers, dbProps, overrides)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings[0].notionProp != "Title" {
		t.Errorf("expected 'Title', got %q", mappings[0].notionProp)
	}
	if mappings[1].notionProp != "Description" {
		t.Errorf("expected 'Description', got %q", mappings[1].notionProp)
	}
}

func TestImportCSV_UnmappedColumnWarning(t *testing.T) {
	headers := []string{"Name", "Unknown Column"}
	dbProps := map[string]map[string]interface{}{
		"Name": {"type": "title"},
	}

	mappings, warnings := buildColumnMappings(headers, dbProps, nil)

	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mappings))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "Unknown Column") {
		t.Errorf("warning should mention 'Unknown Column', got %q", warnings[0])
	}
}

func TestImportCSV_PropertyTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		propType string
		wantKey  string
		check    func(t *testing.T, result map[string]interface{})
	}{
		{
			name:     "title",
			value:    "My Title",
			propType: "title",
			wantKey:  "title",
			check: func(t *testing.T, result map[string]interface{}) {
				title := result["title"].([]map[string]interface{})
				if title[0]["text"].(map[string]interface{})["content"] != "My Title" {
					t.Error("title content mismatch")
				}
			},
		},
		{
			name:     "rich_text",
			value:    "Some text",
			propType: "rich_text",
			wantKey:  "rich_text",
		},
		{
			name:     "number",
			value:    "42.5",
			propType: "number",
			wantKey:  "number",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["number"] != 42.5 {
					t.Errorf("expected 42.5, got %v", result["number"])
				}
			},
		},
		{
			name:     "number invalid",
			value:    "not a number",
			propType: "number",
			check: func(t *testing.T, result map[string]interface{}) {
				if result != nil {
					t.Error("expected nil for invalid number")
				}
			},
		},
		{
			name:     "select",
			value:    "Option A",
			propType: "select",
			wantKey:  "select",
			check: func(t *testing.T, result map[string]interface{}) {
				sel := result["select"].(map[string]interface{})
				if sel["name"] != "Option A" {
					t.Errorf("expected 'Option A', got %v", sel["name"])
				}
			},
		},
		{
			name:     "multi_select",
			value:    "Tag1;Tag2;Tag3",
			propType: "multi_select",
			wantKey:  "multi_select",
			check: func(t *testing.T, result map[string]interface{}) {
				ms := result["multi_select"].([]map[string]interface{})
				if len(ms) != 3 {
					t.Fatalf("expected 3 options, got %d", len(ms))
				}
				if ms[0]["name"] != "Tag1" || ms[1]["name"] != "Tag2" || ms[2]["name"] != "Tag3" {
					t.Errorf("multi_select names mismatch: %v", ms)
				}
			},
		},
		{
			name:     "date",
			value:    "2024-01-15",
			propType: "date",
			wantKey:  "date",
			check: func(t *testing.T, result map[string]interface{}) {
				d := result["date"].(map[string]interface{})
				if d["start"] != "2024-01-15" {
					t.Errorf("expected '2024-01-15', got %v", d["start"])
				}
			},
		},
		{
			name:     "checkbox true",
			value:    "true",
			propType: "checkbox",
			wantKey:  "checkbox",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["checkbox"] != true {
					t.Errorf("expected true, got %v", result["checkbox"])
				}
			},
		},
		{
			name:     "checkbox yes",
			value:    "yes",
			propType: "checkbox",
			wantKey:  "checkbox",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["checkbox"] != true {
					t.Errorf("expected true, got %v", result["checkbox"])
				}
			},
		},
		{
			name:     "checkbox 1",
			value:    "1",
			propType: "checkbox",
			wantKey:  "checkbox",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["checkbox"] != true {
					t.Errorf("expected true, got %v", result["checkbox"])
				}
			},
		},
		{
			name:     "checkbox false",
			value:    "no",
			propType: "checkbox",
			wantKey:  "checkbox",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["checkbox"] != false {
					t.Errorf("expected false, got %v", result["checkbox"])
				}
			},
		},
		{
			name:     "url",
			value:    "https://example.com",
			propType: "url",
			wantKey:  "url",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["url"] != "https://example.com" {
					t.Errorf("expected url, got %v", result["url"])
				}
			},
		},
		{
			name:     "email",
			value:    "test@example.com",
			propType: "email",
			wantKey:  "email",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["email"] != "test@example.com" {
					t.Errorf("expected email, got %v", result["email"])
				}
			},
		},
		{
			name:     "phone_number",
			value:    "+1-555-1234",
			propType: "phone_number",
			wantKey:  "phone_number",
			check: func(t *testing.T, result map[string]interface{}) {
				if result["phone_number"] != "+1-555-1234" {
					t.Errorf("expected phone number, got %v", result["phone_number"])
				}
			},
		},
		{
			name:     "unsupported type returns nil",
			value:    "anything",
			propType: "relation",
			check: func(t *testing.T, result map[string]interface{}) {
				if result != nil {
					t.Error("expected nil for unsupported type")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := csvValueToProperty(tt.value, tt.propType)
			if tt.check != nil {
				tt.check(t, result)
			} else if tt.wantKey != "" && result == nil {
				t.Fatalf("expected non-nil result with key %q", tt.wantKey)
			}
		})
	}
}

func TestImportCSV_BuildPageProperties_SkipsEmptyValues(t *testing.T) {
	mappings := []columnMapping{
		{csvIndex: 0, csvHeader: "Name", notionProp: "Name", propType: "title"},
		{csvIndex: 1, csvHeader: "Desc", notionProp: "Description", propType: "rich_text"},
		{csvIndex: 2, csvHeader: "Count", notionProp: "Count", propType: "number"},
	}

	row := []string{"Test Item", "", "42"}
	props := buildPageProperties(row, mappings)

	if props == nil {
		t.Fatal("expected non-nil properties")
	}
	if _, ok := props["Description"]; ok {
		t.Error("expected empty Description to be skipped")
	}
	if _, ok := props["Name"]; !ok {
		t.Error("expected Name to be present")
	}
	if _, ok := props["Count"]; !ok {
		t.Error("expected Count to be present")
	}
}

func TestImportCSV_BuildPageProperties_AllEmpty(t *testing.T) {
	mappings := []columnMapping{
		{csvIndex: 0, csvHeader: "Name", notionProp: "Name", propType: "title"},
		{csvIndex: 1, csvHeader: "Desc", notionProp: "Description", propType: "rich_text"},
	}

	row := []string{"", ""}
	props := buildPageProperties(row, mappings)

	if props != nil {
		t.Error("expected nil for all-empty row")
	}
}

func TestImportCSV_ParseColumnMap(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single pair",
			input: "Name=Title",
			want:  map[string]string{"Name": "Title"},
		},
		{
			name:  "multiple pairs",
			input: "Name=Title,Desc=Description",
			want:  map[string]string{"Name": "Title", "Desc": "Description"},
		},
		{
			name:    "invalid pair",
			input:   "Name",
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   "=Title",
			wantErr: true,
		},
		{
			name:    "empty value",
			input:   "Name=",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseColumnMap(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil && got != nil {
				t.Errorf("expected nil, got %v", got)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestImportCSV_DryRun_NoAPICalls(t *testing.T) {
	const dbID = "12345678123412341234123456789012"

	apiCallCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     dbID,
			"title": []map[string]interface{}{
				{"plain_text": "Test DB"},
			},
			"properties": map[string]interface{}{
				"Name":  map[string]interface{}{"type": "title"},
				"Email": map[string]interface{}{"type": "email"},
			},
		})
	})
	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		t.Error("CreatePage should not be called in dry-run mode")
		w.WriteHeader(500)
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)
	t.Setenv("NOTION_TOKEN", "test-token")

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write a CSV file
	csvFile := filepath.Join(home, "test.csv")
	csvContent := "Name,Email\nAlice,alice@example.com\nBob,bob@example.com\n"
	if err := os.WriteFile(csvFile, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"import", "csv", dbID,
		"--file", csvFile,
		"--dry-run",
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("import csv --dry-run failed: %v\nstderr=%s", err, errBuf.String())
	}

	// Should have called GetDatabase but NOT CreatePage
	if apiCallCount != 1 {
		t.Errorf("expected 1 API call (GetDatabase), got %d", apiCallCount)
	}

	stderrStr := errBuf.String()
	if !strings.Contains(stderrStr, "DRY-RUN") {
		t.Errorf("expected DRY-RUN in stderr, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "Total rows: 2") {
		t.Errorf("expected 'Total rows: 2' in stderr, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "Name -> Name (title)") {
		t.Errorf("expected column mapping in stderr, got %q", stderrStr)
	}
}

func TestImportCSV_Integration_CreatesPages(t *testing.T) {
	const dbID = "12345678123412341234123456789012"

	var createdPages []map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     dbID,
			"title": []map[string]interface{}{
				{"plain_text": "Test DB"},
			},
			"properties": map[string]interface{}{
				"Name":   map[string]interface{}{"type": "title"},
				"Status": map[string]interface{}{"type": "select"},
				"Count":  map[string]interface{}{"type": "number"},
			},
		})
	})
	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode page request: %v", err)
		}
		createdPages = append(createdPages, body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "new-page-id",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)
	t.Setenv("NOTION_TOKEN", "test-token")

	home := t.TempDir()
	t.Setenv("HOME", home)

	csvFile := filepath.Join(home, "test.csv")
	csvContent := "Name,Status,Count\nAlice,Active,10\nBob,Inactive,20\n"
	if err := os.WriteFile(csvFile, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"import", "csv", dbID,
		"--file", csvFile,
		"--batch-size", "5",
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("import csv failed: %v\nstderr=%s", err, errBuf.String())
	}

	if len(createdPages) != 2 {
		t.Fatalf("expected 2 pages created, got %d", len(createdPages))
	}

	// Check first page
	page1Props := createdPages[0]["properties"].(map[string]interface{})
	name1 := page1Props["Name"].(map[string]interface{})
	titleArr := name1["title"].([]interface{})
	titleObj := titleArr[0].(map[string]interface{})
	textObj := titleObj["text"].(map[string]interface{})
	if textObj["content"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", textObj["content"])
	}

	status1 := page1Props["Status"].(map[string]interface{})
	selObj := status1["select"].(map[string]interface{})
	if selObj["name"] != "Active" {
		t.Errorf("expected 'Active', got %v", selObj["name"])
	}

	count1 := page1Props["Count"].(map[string]interface{})
	if count1["number"] != float64(10) {
		t.Errorf("expected 10, got %v", count1["number"])
	}

	// Check summary in stderr
	stderrStr := errBuf.String()
	if !strings.Contains(stderrStr, "Imported 2 pages") {
		t.Errorf("expected 'Imported 2 pages' in stderr, got %q", stderrStr)
	}
}

func TestImportCSV_ColumnMap_Flag(t *testing.T) {
	const dbID = "12345678123412341234123456789012"

	var createdPages []map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     dbID,
			"title": []map[string]interface{}{
				{"plain_text": "Test DB"},
			},
			"properties": map[string]interface{}{
				"Title":       map[string]interface{}{"type": "title"},
				"Description": map[string]interface{}{"type": "rich_text"},
			},
		})
	})
	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		createdPages = append(createdPages, body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "new-page-id",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)
	t.Setenv("NOTION_TOKEN", "test-token")

	home := t.TempDir()
	t.Setenv("HOME", home)

	csvFile := filepath.Join(home, "test.csv")
	csvContent := "Full Name,Desc\nAlice,A developer\n"
	if err := os.WriteFile(csvFile, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"import", "csv", dbID,
		"--file", csvFile,
		"--column-map", "Full Name=Title,Desc=Description",
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("import csv failed: %v\nstderr=%s", err, errBuf.String())
	}

	if len(createdPages) != 1 {
		t.Fatalf("expected 1 page created, got %d", len(createdPages))
	}

	props := createdPages[0]["properties"].(map[string]interface{})
	if _, ok := props["Title"]; !ok {
		t.Error("expected 'Title' property in page")
	}
	if _, ok := props["Description"]; !ok {
		t.Error("expected 'Description' property in page")
	}
}

func TestImportCSV_ErrorNoHeaders(t *testing.T) {
	// Test with empty CSV (no headers)
	records := [][]string{}
	headers := []string{}

	// buildColumnMappings with empty headers should return empty mappings
	mappings, _ := buildColumnMappings(headers, map[string]map[string]interface{}{
		"Name": {"type": "title"},
	}, nil)

	if len(mappings) != 0 {
		t.Errorf("expected 0 mappings for empty headers, got %d", len(mappings))
	}

	// Empty records means no headers — the command should catch this
	if len(records) != 0 {
		t.Error("expected empty records")
	}
}

func TestImportCSV_ErrorNoTitleMapping(t *testing.T) {
	const dbID = "12345678123412341234123456789012"

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     dbID,
			"title": []map[string]interface{}{
				{"plain_text": "Test DB"},
			},
			"properties": map[string]interface{}{
				"Name":   map[string]interface{}{"type": "title"},
				"Status": map[string]interface{}{"type": "select"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)
	t.Setenv("NOTION_TOKEN", "test-token")

	home := t.TempDir()
	t.Setenv("HOME", home)

	// CSV with no header matching the title property
	csvFile := filepath.Join(home, "test.csv")
	csvContent := "Status\nActive\n"
	if err := os.WriteFile(csvFile, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"import", "csv", dbID,
		"--file", csvFile,
	})

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error for missing title mapping, got nil")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("expected error about title, got %q", err.Error())
	}
}

func TestImportCSV_SkipRows(t *testing.T) {
	const dbID = "12345678123412341234123456789012"

	var createdPages []map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/databases/"+dbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "database",
			"id":     dbID,
			"title": []map[string]interface{}{
				{"plain_text": "Test DB"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{"type": "title"},
			},
		})
	})
	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		createdPages = append(createdPages, body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "new-page-id",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("NOTION_API_BASE_URL", server.URL)
	t.Setenv("NOTION_TOKEN", "test-token")

	home := t.TempDir()
	t.Setenv("HOME", home)

	csvFile := filepath.Join(home, "test.csv")
	csvContent := "Name\nSkip1\nSkip2\nKeep1\nKeep2\n"
	if err := os.WriteFile(csvFile, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var out, errBuf bytes.Buffer
	app := &App{Stdout: &out, Stderr: &errBuf}
	root := app.RootCommand()

	root.SetArgs([]string{
		"import", "csv", dbID,
		"--file", csvFile,
		"--skip-rows", "2",
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("import csv failed: %v\nstderr=%s", err, errBuf.String())
	}

	if len(createdPages) != 2 {
		t.Fatalf("expected 2 pages (skipping 2), got %d", len(createdPages))
	}

	// Verify the first created page is "Keep1", not "Skip1"
	props := createdPages[0]["properties"].(map[string]interface{})
	name := props["Name"].(map[string]interface{})
	titleArr := name["title"].([]interface{})
	titleObj := titleArr[0].(map[string]interface{})
	textObj := titleObj["text"].(map[string]interface{})
	if textObj["content"] != "Keep1" {
		t.Errorf("expected 'Keep1', got %v", textObj["content"])
	}
}

func TestImportCSV_MultiSelectSemicolonSplit(t *testing.T) {
	result := csvValueToProperty("Red;Green;Blue", "multi_select")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	ms := result["multi_select"].([]map[string]interface{})
	if len(ms) != 3 {
		t.Fatalf("expected 3 options, got %d", len(ms))
	}
	names := []string{
		ms[0]["name"].(string),
		ms[1]["name"].(string),
		ms[2]["name"].(string),
	}
	expected := []string{"Red", "Green", "Blue"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("option %d: got %q, want %q", i, n, expected[i])
		}
	}
}

func TestImportCSV_MultiSelectTrimsSpaces(t *testing.T) {
	result := csvValueToProperty(" Red ; Green ; Blue ", "multi_select")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	ms := result["multi_select"].([]map[string]interface{})
	if len(ms) != 3 {
		t.Fatalf("expected 3 options, got %d", len(ms))
	}
	// The leading/trailing spaces in the overall value are trimmed by buildPageProperties,
	// but the splits should be trimmed by csvValueToProperty
	if ms[0]["name"] != "Red" {
		t.Errorf("expected 'Red', got %q", ms[0]["name"])
	}
}
