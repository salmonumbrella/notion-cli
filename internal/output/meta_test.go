package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestInjectMeta(t *testing.T) {
	data := map[string]interface{}{
		"object": "list",
		"results": []interface{}{
			map[string]interface{}{"id": "a"},
			map[string]interface{}{"id": "b"},
		},
		"has_more":    true,
		"next_cursor": "cursor-abc",
	}

	result := injectMeta(data)

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	meta, ok := m["_meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected _meta map, got %T", m["_meta"])
	}

	if meta["fetched_count"] != 2 {
		t.Errorf("expected fetched_count 2, got %v", meta["fetched_count"])
	}

	if _, ok := meta["timestamp"].(string); !ok {
		t.Errorf("expected timestamp string, got %T", meta["timestamp"])
	}
}

func TestInjectMeta_NonList(t *testing.T) {
	data := map[string]interface{}{
		"object": "page",
		"id":     "page-123",
	}

	result := injectMeta(data)

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if _, exists := m["_meta"]; exists {
		t.Error("should not inject _meta into non-list responses")
	}
}

func TestInjectMeta_EmptyResults(t *testing.T) {
	data := map[string]interface{}{
		"object":   "list",
		"results":  []interface{}{},
		"has_more": false,
	}

	result := injectMeta(data)
	m := result.(map[string]interface{})
	meta := m["_meta"].(map[string]interface{})

	if meta["fetched_count"] != 0 {
		t.Errorf("expected fetched_count 0, got %v", meta["fetched_count"])
	}
}

func TestInjectMeta_NotAMap(t *testing.T) {
	data := []interface{}{"a", "b"}
	result := injectMeta(data)

	// Should return data unchanged
	if _, ok := result.([]interface{}); !ok {
		t.Errorf("expected slice returned unchanged, got %T", result)
	}
}

func TestPrintInjectsMeta(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	ctx = WithFormat(ctx, FormatJSON)

	printer := NewPrinter(&buf, FormatJSON)

	data := map[string]interface{}{
		"object":      "list",
		"results":     []interface{}{map[string]interface{}{"id": "a"}},
		"has_more":    false,
		"next_cursor": nil,
	}

	if err := printer.Print(ctx, data); err != nil {
		t.Fatalf("Print failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if _, ok := result["_meta"]; !ok {
		t.Error("expected _meta in JSON output")
	}
}

func TestPrintSkipsMetaForTable(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	ctx = WithFormat(ctx, FormatTable)

	printer := NewPrinter(&buf, FormatTable)

	// Use a typed slice that the table formatter can handle directly.
	// The key assertion is that _meta never appears in table output.
	data := []map[string]interface{}{
		{"id": "a", "name": "Alice"},
	}

	if err := printer.Print(ctx, data); err != nil {
		t.Fatalf("Print failed: %v", err)
	}

	if strings.Contains(buf.String(), "_meta") {
		t.Error("table output should not contain _meta")
	}
}
