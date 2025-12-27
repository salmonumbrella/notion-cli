package output

import (
	"time"
)

// injectMeta adds a _meta object to list-envelope responses.
// Only applies to map[string]interface{} with object="list" and a results array.
// Non-list data is returned unchanged.
func injectMeta(data interface{}) interface{} {
	m, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	obj, _ := m["object"].(string)
	if obj != "list" {
		return data
	}

	results, _ := m["results"].([]interface{})

	meta := map[string]interface{}{
		"fetched_count": len(results),
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	m["_meta"] = meta
	return m
}
