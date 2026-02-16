package cmd

import (
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestNormalizeDataSourceSearchResults(t *testing.T) {
	tests := []struct {
		name                 string
		input                []map[string]interface{}
		wantTitleIsArray     bool
		wantName             string
		wantTitlePlainText   string
		wantExistingNameKept bool
	}{
		{
			name: "nil title becomes empty array with empty fallbacks",
			input: []map[string]interface{}{
				{"id": "ds1", "title": nil},
			},
			wantTitleIsArray:   true,
			wantName:           "",
			wantTitlePlainText: "",
		},
		{
			name: "missing title becomes empty array with empty fallbacks",
			input: []map[string]interface{}{
				{"id": "ds1"},
			},
			wantTitleIsArray:   true,
			wantName:           "",
			wantTitlePlainText: "",
		},
		{
			name: "plain_text title populates normalized fields",
			input: []map[string]interface{}{
				{
					"id": "ds1",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Sample Shipments"},
					},
				},
			},
			wantTitleIsArray:   true,
			wantName:           "Sample Shipments",
			wantTitlePlainText: "Sample Shipments",
		},
		{
			name: "existing name is preserved",
			input: []map[string]interface{}{
				{
					"id":   "ds1",
					"name": "Custom Name",
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Actual Title"},
					},
				},
			},
			wantTitleIsArray:     true,
			wantName:             "Custom Name",
			wantTitlePlainText:   "Actual Title",
			wantExistingNameKept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDataSourceSearchResults(tt.input)
			if len(got) != 1 {
				t.Fatalf("expected 1 result, got %d", len(got))
			}

			item := got[0]
			if _, ok := item["title"].([]interface{}); tt.wantTitleIsArray && !ok {
				t.Fatalf("title should be []interface{}, got %T", item["title"])
			}

			name, _ := item["name"].(string)
			if name != tt.wantName {
				t.Fatalf("name = %q, want %q", name, tt.wantName)
			}

			titlePlainText, _ := item["title_plain_text"].(string)
			if titlePlainText != tt.wantTitlePlainText {
				t.Fatalf("title_plain_text = %q, want %q", titlePlainText, tt.wantTitlePlainText)
			}

			if tt.wantExistingNameKept && name != "Custom Name" {
				t.Fatalf("existing name should be preserved, got %q", name)
			}
		})
	}
}

func TestExtractSelectNames(t *testing.T) {
	tests := []struct {
		name string
		prop map[string]interface{}
		want []string
	}{
		{
			name: "nil prop returns nil",
			prop: nil,
			want: nil,
		},
		{
			name: "empty map returns nil",
			prop: map[string]interface{}{},
			want: nil,
		},
		{
			name: "select with name returns single-element slice",
			prop: map[string]interface{}{
				"select": map[string]interface{}{
					"name": "Active",
				},
			},
			want: []string{"Active"},
		},
		{
			name: "select without name returns empty",
			prop: map[string]interface{}{
				"select": map[string]interface{}{
					"id": "some-id",
				},
			},
			want: nil,
		},
		{
			name: "select with empty name returns empty",
			prop: map[string]interface{}{
				"select": map[string]interface{}{
					"name": "",
				},
			},
			want: nil,
		},
		{
			name: "select is null returns nil",
			prop: map[string]interface{}{
				"select": nil,
			},
			want: nil,
		},
		{
			name: "status with name returns single-element slice",
			prop: map[string]interface{}{
				"status": map[string]interface{}{
					"name": "In Progress",
				},
			},
			want: []string{"In Progress"},
		},
		{
			name: "status without name returns empty",
			prop: map[string]interface{}{
				"status": map[string]interface{}{
					"id": "some-id",
				},
			},
			want: nil,
		},
		{
			name: "multi_select with multiple names returns all",
			prop: map[string]interface{}{
				"multi_select": []interface{}{
					map[string]interface{}{"name": "Tag1"},
					map[string]interface{}{"name": "Tag2"},
					map[string]interface{}{"name": "Tag3"},
				},
			},
			want: []string{"Tag1", "Tag2", "Tag3"},
		},
		{
			name: "multi_select with empty array returns empty",
			prop: map[string]interface{}{
				"multi_select": []interface{}{},
			},
			want: []string{},
		},
		{
			name: "multi_select with some empty names skips them",
			prop: map[string]interface{}{
				"multi_select": []interface{}{
					map[string]interface{}{"name": "Valid"},
					map[string]interface{}{"name": ""},
					map[string]interface{}{"id": "no-name"},
				},
			},
			want: []string{"Valid"},
		},
		{
			name: "property with neither select nor multi_select returns nil",
			prop: map[string]interface{}{
				"type":  "text",
				"title": "Some title",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSelectNames(tt.prop)

			if tt.want == nil {
				if got != nil {
					t.Errorf("extractSelectNames() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("extractSelectNames() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("extractSelectNames()[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestFilterResultsBySelect(t *testing.T) {
	// Helper to create test pages with select properties
	makePage := func(id string, propName string, propValue map[string]interface{}) notion.Page {
		return notion.Page{
			ID: id,
			Properties: map[string]interface{}{
				propName: propValue,
			},
		}
	}

	tests := []struct {
		name      string
		results   []notion.Page
		propName  string
		equals    string
		notEquals string
		match     string
		wantIDs   []string
		wantErr   bool
	}{
		{
			name: "exact match finds matching pages",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
				makePage("page2", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Inactive"},
				}),
				makePage("page3", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
			},
			propName: "Status",
			equals:   "Active",
			match:    "",
			wantIDs:  []string{"page1", "page3"},
			wantErr:  false,
		},
		{
			name: "regex match finds matching pages",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "In Progress"},
				}),
				makePage("page2", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Done"},
				}),
				makePage("page3", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "In Review"},
				}),
			},
			propName: "Status",
			equals:   "",
			match:    "^In ",
			wantIDs:  []string{"page1", "page3"},
			wantErr:  false,
		},
		{
			name: "not equals excludes matching pages",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"status": map[string]interface{}{"name": "已完成"},
				}),
				makePage("page2", "Status", map[string]interface{}{
					"status": map[string]interface{}{"name": "未發送"},
				}),
			},
			propName:  "Status",
			notEquals: "已完成",
			wantIDs:   []string{"page2"},
			wantErr:   false,
		},
		{
			name: "case insensitive regex match",
			results: []notion.Page{
				makePage("page1", "Priority", map[string]interface{}{
					"select": map[string]interface{}{"name": "HIGH"},
				}),
				makePage("page2", "Priority", map[string]interface{}{
					"select": map[string]interface{}{"name": "low"},
				}),
				makePage("page3", "Priority", map[string]interface{}{
					"select": map[string]interface{}{"name": "High"},
				}),
			},
			propName: "Priority",
			equals:   "",
			match:    "(?i)high",
			wantIDs:  []string{"page1", "page3"},
			wantErr:  false,
		},
		{
			name: "no matches returns empty slice",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
				makePage("page2", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Done"},
				}),
			},
			propName: "Status",
			equals:   "Archived",
			match:    "",
			wantIDs:  []string{},
			wantErr:  false,
		},
		{
			name: "missing property is skipped",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
				makePage("page2", "OtherProp", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
			},
			propName: "Status",
			equals:   "Active",
			match:    "",
			wantIDs:  []string{"page1"},
			wantErr:  false,
		},
		{
			name: "multi_select works with exact match",
			results: []notion.Page{
				makePage("page1", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "Frontend"},
						map[string]interface{}{"name": "React"},
					},
				}),
				makePage("page2", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "Backend"},
						map[string]interface{}{"name": "Go"},
					},
				}),
				makePage("page3", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "Frontend"},
						map[string]interface{}{"name": "Vue"},
					},
				}),
			},
			propName: "Tags",
			equals:   "Frontend",
			match:    "",
			wantIDs:  []string{"page1", "page3"},
			wantErr:  false,
		},
		{
			name: "multi_select works with regex match",
			results: []notion.Page{
				makePage("page1", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "bug"},
						map[string]interface{}{"name": "urgent"},
					},
				}),
				makePage("page2", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "feature"},
					},
				}),
				makePage("page3", "Tags", map[string]interface{}{
					"multi_select": []interface{}{
						map[string]interface{}{"name": "bugfix"},
					},
				}),
			},
			propName: "Tags",
			equals:   "",
			match:    "bug",
			wantIDs:  []string{"page1", "page3"},
			wantErr:  false,
		},
		{
			name: "invalid regex returns error",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
			},
			propName: "Status",
			equals:   "",
			match:    "[invalid",
			wantIDs:  nil,
			wantErr:  true,
		},
		{
			name:     "empty results returns empty slice",
			results:  []notion.Page{},
			propName: "Status",
			equals:   "Active",
			match:    "",
			wantIDs:  []string{},
			wantErr:  false,
		},
		{
			name: "property is not a map is skipped",
			results: []notion.Page{
				{
					ID: "page1",
					Properties: map[string]interface{}{
						"Status": "not a map",
					},
				},
				makePage("page2", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
			},
			propName: "Status",
			equals:   "Active",
			match:    "",
			wantIDs:  []string{"page2"},
			wantErr:  false,
		},
		{
			name: "select with empty names is skipped",
			results: []notion.Page{
				makePage("page1", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": ""},
				}),
				makePage("page2", "Status", map[string]interface{}{
					"select": map[string]interface{}{"name": "Active"},
				}),
			},
			propName: "Status",
			equals:   "Active",
			match:    "",
			wantIDs:  []string{"page2"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterResultsBySelect(tt.results, tt.propName, tt.equals, tt.notEquals, tt.match)

			if tt.wantErr {
				if err == nil {
					t.Errorf("filterResultsBySelect() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("filterResultsBySelect() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.wantIDs) {
				t.Errorf("filterResultsBySelect() returned %d pages, want %d", len(got), len(tt.wantIDs))
				return
			}

			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("filterResultsBySelect()[%d].ID = %q, want %q", i, got[i].ID, wantID)
				}
			}
		})
	}
}
