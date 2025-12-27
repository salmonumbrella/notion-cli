package cmd

import (
	"reflect"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestPlainTextFromRichTextArray(t *testing.T) {
	in := []interface{}{
		map[string]interface{}{"plain_text": "Hello"},
		map[string]interface{}{"plain_text": " "},
		map[string]interface{}{"plain_text": "world"},
	}
	if got := plainTextFromRichTextArray(in); got != "Hello world" {
		t.Fatalf("expected %q, got %q", "Hello world", got)
	}
}

func TestSimplifyPropertyValue(t *testing.T) {
	tests := []struct {
		name     string
		propType string
		value    interface{}
		want     interface{}
	}{
		{
			name:     "title",
			propType: "title",
			value: []interface{}{
				map[string]interface{}{"plain_text": "A"},
				map[string]interface{}{"plain_text": "B"},
			},
			want: "AB",
		},
		{
			name:     "select",
			propType: "select",
			value:    map[string]interface{}{"name": "Done"},
			want:     "Done",
		},
		{
			name:     "multi_select",
			propType: "multi_select",
			value: []interface{}{
				map[string]interface{}{"name": "A"},
				map[string]interface{}{"name": "B"},
			},
			want: []string{"A", "B"},
		},
		{
			name:     "relation",
			propType: "relation",
			value: []interface{}{
				map[string]interface{}{"id": "1"},
				map[string]interface{}{"id": "2"},
			},
			want: []string{"1", "2"},
		},
		{
			name:     "formula_number",
			propType: "formula",
			value:    map[string]interface{}{"type": "number", "number": float64(42)},
			want:     float64(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simplifyPropertyValue(tt.propType, tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestSimplifyBlocks(t *testing.T) {
	blocks := []notion.Block{
		{
			ID:          "b1",
			Type:        "paragraph",
			HasChildren: false,
			Content: map[string]interface{}{
				"rich_text": []interface{}{
					map[string]interface{}{"plain_text": "Hi"},
				},
			},
		},
		{
			ID:          "b2",
			Type:        "to_do",
			HasChildren: false,
			Content: map[string]interface{}{
				"rich_text": []interface{}{
					map[string]interface{}{"plain_text": "Task"},
				},
				"checked": true,
			},
		},
	}

	out := simplifyBlocks(blocks)
	if len(out) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(out))
	}
	if out[0]["id"] != "b1" || out[0]["text"] != "Hi" {
		t.Fatalf("unexpected simplified block: %#v", out[0])
	}
	if out[1]["id"] != "b2" || out[1]["text"] != "Task" || out[1]["checked"] != true {
		t.Fatalf("unexpected simplified block: %#v", out[1])
	}
}
