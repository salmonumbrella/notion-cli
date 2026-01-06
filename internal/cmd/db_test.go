package cmd

import (
	"regexp"
	"testing"
)

func TestExtractTitlePlainText(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "nil value returns empty string",
			value: nil,
			want:  "",
		},
		{
			name:  "non-slice value returns empty string",
			value: "not a slice",
			want:  "",
		},
		{
			name:  "map value returns empty string",
			value: map[string]interface{}{"key": "value"},
			want:  "",
		},
		{
			name:  "empty array returns empty string",
			value: []interface{}{},
			want:  "",
		},
		{
			name: "plain_text path returns text",
			value: []interface{}{
				map[string]interface{}{
					"plain_text": "Database Title",
				},
			},
			want: "Database Title",
		},
		{
			name: "text.content fallback works when plain_text missing",
			value: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"content": "Fallback Title",
					},
				},
			},
			want: "Fallback Title",
		},
		{
			name: "plain_text takes priority over text.content",
			value: []interface{}{
				map[string]interface{}{
					"plain_text": "Primary",
					"text": map[string]interface{}{
						"content": "Secondary",
					},
				},
			},
			want: "Primary",
		},
		{
			name: "multiple parts are joined",
			value: []interface{}{
				map[string]interface{}{
					"plain_text": "Part One",
				},
				map[string]interface{}{
					"plain_text": " Part Two",
				},
				map[string]interface{}{
					"text": map[string]interface{}{
						"content": " Part Three",
					},
				},
			},
			want: "Part One Part Two Part Three",
		},
		{
			name: "empty plain_text falls back to text.content",
			value: []interface{}{
				map[string]interface{}{
					"plain_text": "",
					"text": map[string]interface{}{
						"content": "Fallback Used",
					},
				},
			},
			want: "Fallback Used",
		},
		{
			name: "skips entries that are not maps",
			value: []interface{}{
				"not a map",
				map[string]interface{}{
					"plain_text": "Valid Entry",
				},
				42,
			},
			want: "Valid Entry",
		},
		{
			name: "skips entries with empty text.content",
			value: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"content": "",
					},
				},
				map[string]interface{}{
					"plain_text": "Valid",
				},
			},
			want: "Valid",
		},
		{
			name: "handles text without content key",
			value: []interface{}{
				map[string]interface{}{
					"text": map[string]interface{}{
						"link": "https://example.com",
					},
				},
				map[string]interface{}{
					"plain_text": "Title",
				},
			},
			want: "Title",
		},
		{
			name: "text is not a map returns empty for that entry",
			value: []interface{}{
				map[string]interface{}{
					"text": "not a map",
				},
				map[string]interface{}{
					"plain_text": "Valid",
				},
			},
			want: "Valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitlePlainText(tt.value)
			if got != tt.want {
				t.Errorf("extractTitlePlainText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterDataSourcesByTitle(t *testing.T) {
	// Helper to create test data source results
	makeDataSource := func(id string, title []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"id":    id,
			"title": title,
		}
	}

	// Helper to create a title with plain_text
	makeTitlePlainText := func(text string) []interface{} {
		return []interface{}{
			map[string]interface{}{
				"plain_text": text,
			},
		}
	}

	// Helper to create a title with text.content fallback
	makeTitleTextContent := func(text string) []interface{} {
		return []interface{}{
			map[string]interface{}{
				"text": map[string]interface{}{
					"content": text,
				},
			},
		}
	}

	tests := []struct {
		name    string
		results []map[string]interface{}
		pattern string
		wantIDs []string
	}{
		{
			name: "regex matches returns matching results",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Vendors Database")),
				makeDataSource("ds2", makeTitlePlainText("Products List")),
				makeDataSource("ds3", makeTitlePlainText("Vendor Contacts")),
			},
			pattern: "Vendor",
			wantIDs: []string{"ds1", "ds3"},
		},
		{
			name: "no matches returns empty slice",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Vendors Database")),
				makeDataSource("ds2", makeTitlePlainText("Products List")),
			},
			pattern: "Tasks",
			wantIDs: []string{},
		},
		{
			name: "empty title is skipped",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("")),
				makeDataSource("ds2", makeTitlePlainText("Valid Title")),
				makeDataSource("ds3", []interface{}{}),
			},
			pattern: ".",
			wantIDs: []string{"ds2"},
		},
		{
			name: "regex with start anchor works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Project Tasks")),
				makeDataSource("ds2", makeTitlePlainText("My Project")),
				makeDataSource("ds3", makeTitlePlainText("Project Notes")),
			},
			pattern: "^Project",
			wantIDs: []string{"ds1", "ds3"},
		},
		{
			name: "regex with end anchor works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Task List")),
				makeDataSource("ds2", makeTitlePlainText("Shopping List")),
				makeDataSource("ds3", makeTitlePlainText("Listed Items")),
			},
			pattern: "List$",
			wantIDs: []string{"ds1", "ds2"},
		},
		{
			name: "regex with both anchors works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Tasks")),
				makeDataSource("ds2", makeTitlePlainText("My Tasks")),
				makeDataSource("ds3", makeTitlePlainText("Tasks List")),
			},
			pattern: "^Tasks$",
			wantIDs: []string{"ds1"},
		},
		{
			name: "case insensitive regex with (?i) works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("VENDORS")),
				makeDataSource("ds2", makeTitlePlainText("vendors")),
				makeDataSource("ds3", makeTitlePlainText("Vendors")),
				makeDataSource("ds4", makeTitlePlainText("Products")),
			},
			pattern: "(?i)vendors",
			wantIDs: []string{"ds1", "ds2", "ds3"},
		},
		{
			name: "case sensitive by default",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("VENDORS")),
				makeDataSource("ds2", makeTitlePlainText("vendors")),
				makeDataSource("ds3", makeTitlePlainText("Vendors")),
			},
			pattern: "Vendors",
			wantIDs: []string{"ds3"},
		},
		{
			name:    "empty results returns empty slice",
			results: []map[string]interface{}{},
			pattern: "anything",
			wantIDs: []string{},
		},
		{
			name: "nil title field is skipped",
			results: []map[string]interface{}{
				{"id": "ds1", "title": nil},
				makeDataSource("ds2", makeTitlePlainText("Valid")),
			},
			pattern: ".",
			wantIDs: []string{"ds2"},
		},
		{
			name: "missing title field is skipped",
			results: []map[string]interface{}{
				{"id": "ds1", "object": "data_source"},
				makeDataSource("ds2", makeTitlePlainText("Has Title")),
			},
			pattern: ".",
			wantIDs: []string{"ds2"},
		},
		{
			name: "text.content fallback is used for matching",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitleTextContent("Fallback Title")),
				makeDataSource("ds2", makeTitlePlainText("Plain Text Title")),
			},
			pattern: "Title",
			wantIDs: []string{"ds1", "ds2"},
		},
		{
			name: "complex regex pattern works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Task-2024-001")),
				makeDataSource("ds2", makeTitlePlainText("Task-2024-002")),
				makeDataSource("ds3", makeTitlePlainText("Note-2024-001")),
				makeDataSource("ds4", makeTitlePlainText("Task-2023-001")),
			},
			pattern: `Task-2024-\d{3}`,
			wantIDs: []string{"ds1", "ds2"},
		},
		{
			name: "regex with alternation works",
			results: []map[string]interface{}{
				makeDataSource("ds1", makeTitlePlainText("Active Projects")),
				makeDataSource("ds2", makeTitlePlainText("Archived Items")),
				makeDataSource("ds3", makeTitlePlainText("Active Tasks")),
				makeDataSource("ds4", makeTitlePlainText("Pending Work")),
			},
			pattern: "Active|Archived",
			wantIDs: []string{"ds1", "ds2", "ds3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile(tt.pattern)
			got := filterDataSourcesByTitle(tt.results, re)

			if len(got) != len(tt.wantIDs) {
				t.Errorf("filterDataSourcesByTitle() returned %d results, want %d", len(got), len(tt.wantIDs))
				return
			}

			for i, wantID := range tt.wantIDs {
				gotID, ok := got[i]["id"].(string)
				if !ok {
					t.Errorf("filterDataSourcesByTitle()[%d] has no id field", i)
					continue
				}
				if gotID != wantID {
					t.Errorf("filterDataSourcesByTitle()[%d].id = %q, want %q", i, gotID, wantID)
				}
			}
		})
	}
}
