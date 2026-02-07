package output

import (
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
