// Package batch provides support for reading JSON arrays and NDJSON files
// for bulk operations like batch page creation.
package batch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

const (
	// MaxInputSize is the maximum file size for batch input (10MB).
	MaxInputSize = 10 * 1024 * 1024
	// MaxItemCount is the maximum number of items in a batch.
	MaxItemCount = 10000
)

// Result represents the outcome of a batch operation on a single item.
type Result struct {
	Index   int                    `json:"index"`
	Success bool                   `json:"success"`
	ID      string                 `json:"id,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Input   map[string]interface{} `json:"input,omitempty"`
}

// ReadItems reads items from a JSON array or NDJSON file.
func ReadItems(path string) ([]map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}

	if info.Size() > MaxInputSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", MaxInputSize)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Try to parse as JSON array first
	var items []map[string]interface{}
	dec := json.NewDecoder(f)
	if err := dec.Decode(&items); err == nil {
		if len(items) > MaxItemCount {
			return nil, fmt.Errorf("file exceeds maximum item count of %d", MaxItemCount)
		}
		return items, nil
	}

	// Fallback to NDJSON (one JSON object per line)
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("cannot seek file: %w", err)
	}
	items = nil
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("invalid JSON on line %d: %w", len(items)+1, err)
		}

		items = append(items, item)
		if len(items) > MaxItemCount {
			return nil, fmt.Errorf("file exceeds maximum item count of %d", MaxItemCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return items, nil
}

// WriteResults writes batch results as JSON.
func WriteResults(path string, results []Result) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
