package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// NormalizeNotionID normalizes a Notion ID or URL into a raw ID string.
func NormalizeNotionID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("id is required")
	}

	if looksLikeURL(trimmed) {
		id, err := notion.ExtractIDFromNotionURL(trimmed)
		if err != nil {
			return "", err
		}
		return id, nil
	}

	return trimmed, nil
}

// ResolveJSONInput resolves JSON input from inline, @file, or stdin.
func ResolveJSONInput(raw string, file string) (string, error) {
	if raw != "" && file != "" {
		return "", fmt.Errorf("use only one of inline JSON or --file")
	}

	if file != "" {
		return ReadInputSource(file)
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "-" {
		return ReadInputSource("-")
	}
	if strings.HasPrefix(trimmed, "@") {
		path := trimmed[1:]
		return ReadInputSource(path)
	}

	return raw, nil
}

// ReadJSONInput resolves a single JSON value with @file and - (stdin) support.
func ReadJSONInput(value string) (string, error) {
	return ResolveJSONInput(value, "")
}

// NormalizeJSONInput unwraps double-serialized JSON strings when possible.
// If the input is a JSON string containing JSON, it returns the inner JSON.
//
// This handles cases where JSON has been inadvertently quoted, such as:
//   - Shell escaping issues: "{\"key\": \"value\"}" -> {"key": "value"}
//   - Copy-paste from string literals: "[1, 2, 3]" -> [1, 2, 3]
//
// The function only unwraps one level - triple-serialized JSON will
// only have one layer removed.
//
// If the input is not a double-serialized JSON string, it is returned unchanged.
func NormalizeJSONInput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}

	var inner string
	if err := json.Unmarshal([]byte(trimmed), &inner); err != nil {
		return raw
	}

	innerTrimmed := strings.TrimSpace(inner)
	if innerTrimmed == "" {
		return raw
	}
	if json.Valid([]byte(innerTrimmed)) {
		return innerTrimmed
	}

	return raw
}

// UnmarshalJSONInput unmarshals JSON input, supporting double-serialized JSON strings.
func UnmarshalJSONInput(raw string, target interface{}) error {
	normalized := NormalizeJSONInput(raw)
	return json.Unmarshal([]byte(normalized), target)
}

// ReadInputSource reads input from a file path or stdin when path is "-".
func ReadInputSource(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("input file path is required")
	}
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// looksLikeURL determines whether the given value appears to be a URL rather than
// a raw Notion ID. It uses several heuristics:
//
//  1. Explicit URL schemes: Matches "http://" or "https://" prefixes
//  2. Notion domains: Matches strings containing "notion.so" or "notion.site"
//  3. Path separators: Matches strings containing "/" to catch URL fragments
//     (e.g., "notion.so/Page-Title-abc123" pasted without the scheme)
//
// The "/" check is intentionally broad to catch partial URLs that users might copy
// from their browser. This is unlikely to cause false positives since Notion IDs
// are UUIDs (32 hex characters with optional dashes), which never contain slashes.
func looksLikeURL(value string) bool {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	if strings.Contains(lower, "notion.so") || strings.Contains(lower, "notion.site") {
		return true
	}
	return strings.Contains(value, "/")
}
