package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestLightSchemaContract_CommandCoverage(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	root := app.RootCommand()

	got := collectCommandsWithLocalFlag(root, "light")
	want := lightSchemaCommands()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("light schema registry mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestLightSchemaContract_RegistryIncludesGuarantees(t *testing.T) {
	for _, command := range lightSchemaCommands() {
		schema, ok := lookupLightSchema(command)
		if !ok {
			t.Fatalf("missing schema for %q", command)
		}
		if len(schema.ItemFields) == 0 {
			t.Fatalf("schema %q must define item fields", command)
		}
		if len(schema.Guarantees) == 0 {
			t.Fatalf("schema %q must define guarantees", command)
		}
	}
}

func TestLightSchemaContract_FieldShapeAndGuarantees(t *testing.T) {
	tests := []struct {
		command string
		build   func(*testing.T) map[string]interface{}
		verify  func(*testing.T, map[string]interface{})
	}{
		{
			command: "list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightSearchResults([]map[string]interface{}{
					{
						"object": "page",
						"id":     "list-schema-1",
						"url":    "https://example.invalid/pages/list-schema-1",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{
								"type":  "title",
								"title": []interface{}{map[string]interface{}{"plain_text": "Top List"}},
							},
						},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "page" {
					t.Fatalf("top-level list light object mismatch, got %#v", payload["object"])
				}
			},
		},
		{
			command: "search",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightSearchResults([]map[string]interface{}{
					{
						"object": "data_source",
						"id":     "ds-schema-1",
						"url":    "https://example.invalid/ds/schema-1",
						"title":  []interface{}{map[string]interface{}{"plain_text": "Schema DS"}},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "ds" {
					t.Fatalf("search light object should normalize to ds, got %#v", payload["object"])
				}
				if payload["title"] != "Schema DS" {
					t.Fatalf("search light title mismatch, got %#v", payload["title"])
				}
			},
		},
		{
			command: "page list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightSearchResults([]map[string]interface{}{
					{
						"object": "page",
						"id":     "page-list-schema-1",
						"url":    "https://example.invalid/pages/list-schema-1",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{
								"type":  "title",
								"title": []interface{}{map[string]interface{}{"plain_text": "List Page"}},
							},
						},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "page" {
					t.Fatalf("page list light object mismatch, got %#v", payload["object"])
				}
			},
		},
		{
			command: "db list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightSearchResults([]map[string]interface{}{
					{
						"object": "data_source",
						"id":     "db-list-schema-1",
						"url":    "https://example.invalid/ds/db-list-schema-1",
						"title":  []interface{}{map[string]interface{}{"plain_text": "Database List"}},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "ds" {
					t.Fatalf("db list light object should normalize to ds, got %#v", payload["object"])
				}
			},
		},
		{
			command: "datasource list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightSearchResults([]map[string]interface{}{
					{
						"object": "data_source",
						"id":     "ds-list-schema-1",
						"url":    "https://example.invalid/ds/list-schema-1",
						"title":  []interface{}{map[string]interface{}{"plain_text": "Data Source List"}},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "ds" {
					t.Fatalf("datasource list light object should normalize to ds, got %#v", payload["object"])
				}
			},
		},
		{
			command: "page get",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightPage(&notion.Page{
					ID:  "page-get-schema-1",
					URL: "https://example.invalid/pages/get-schema-1",
					Properties: map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []interface{}{map[string]interface{}{"plain_text": "Get Page"}},
						},
					},
				})
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["object"] != "page" {
					t.Fatalf("page get light should default object=page, got %#v", payload["object"])
				}
				if _, exists := payload["properties"]; exists {
					t.Fatalf("page get light should omit properties, got %#v", payload)
				}
			},
		},
		{
			command: "user list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightUsers([]*notion.User{
					{
						ID:   "user-schema-1",
						Name: "Reviewer",
						Type: "person",
						Person: &notion.Person{
							Email: "reviewer@example.invalid",
						},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["email"] != "reviewer@example.invalid" {
					t.Fatalf("user list light should include email when present, got %#v", payload["email"])
				}
			},
		},
		{
			command: "comment list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightComments([]*notion.Comment{
					{
						ID:           "comment-schema-1",
						DiscussionID: "discussion-schema-1",
						Parent: notion.Parent{
							Type:   "page_id",
							PageID: "parent-page-schema-1",
						},
						CreatedBy: notion.User{
							ID:   "comment-user-schema-1",
							Name: "Reviewer",
						},
						CreatedTime: "2026-01-01T00:00:00.000Z",
						RichText: []notion.RichText{
							{PlainText: "Part A "},
							{Text: &notion.TextContent{Content: "Part B"}},
						},
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if payload["text"] != "Part A Part B" {
					t.Fatalf("comment list light should flatten rich text, got %#v", payload["text"])
				}
			},
		},
		{
			command: "file list",
			build: func(t *testing.T) map[string]interface{} {
				item := toLightFileUploads([]*notion.FileUpload{
					{
						ID:          "file-schema-1",
						FileName:    "artifact.txt",
						Status:      "uploaded",
						Size:        256,
						CreatedTime: "2026-01-01T00:00:00.000Z",
						ExpiryTime:  "2026-01-02T00:00:00.000Z",
						UploadURL:   "https://example.invalid/uploads/file-schema-1",
					},
				})[0]
				return toJSONMap(t, item)
			},
			verify: func(t *testing.T, payload map[string]interface{}) {
				if _, exists := payload["upload_url"]; exists {
					t.Fatalf("file list light should omit upload_url, got %#v", payload)
				}
			},
		},
	}

	caseCommands := make([]string, 0, len(tests))
	for _, tt := range tests {
		caseCommands = append(caseCommands, tt.command)
	}
	sort.Strings(caseCommands)
	if !reflect.DeepEqual(caseCommands, lightSchemaCommands()) {
		t.Fatalf("field-shape test cases out of sync with registry\n got: %v\nwant: %v", caseCommands, lightSchemaCommands())
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			schema, ok := lookupLightSchema(tt.command)
			if !ok {
				t.Fatalf("missing schema for %q", tt.command)
			}

			payload := tt.build(t)
			requireExactKeys(t, payload, schema.ItemFields)
			tt.verify(t, payload)
		})
	}
}

func collectCommandsWithLocalFlag(root *cobra.Command, flagName string) []string {
	commands := make(map[string]struct{})

	var walk func(*cobra.Command)
	walk = func(current *cobra.Command) {
		if current == nil {
			return
		}

		if current.Flags().Lookup(flagName) != nil {
			path := strings.TrimPrefix(current.CommandPath(), root.Name()+" ")
			commands[path] = struct{}{}
		}

		for _, child := range current.Commands() {
			walk(child)
		}
	}

	walk(root)

	out := make([]string, 0, len(commands))
	for command := range commands {
		out = append(out, command)
	}
	sort.Strings(out)
	return out
}

func toJSONMap(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	return payload
}

func requireExactKeys(t *testing.T, payload map[string]interface{}, expected []string) {
	t.Helper()

	gotKeys := make([]string, 0, len(payload))
	for key := range payload {
		gotKeys = append(gotKeys, key)
	}
	sort.Strings(gotKeys)

	wantKeys := append([]string(nil), expected...)
	sort.Strings(wantKeys)

	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("unexpected keys\n got: %v\nwant: %v\npayload: %#v", gotKeys, wantKeys, payload)
	}
}
