// internal/batch/batch_test.go
package batch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadItems_JSONArray(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "items.json")

	content := `[{"id": "1"}, {"id": "2"}, {"id": "3"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ReadItems(path)
	if err != nil {
		t.Fatalf("ReadItems failed: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestReadItems_NDJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "items.ndjson")

	content := `{"id": "1"}
{"id": "2"}
{"id": "3"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ReadItems(path)
	if err != nil {
		t.Fatalf("ReadItems failed: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestReadItems_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.json")

	// Create file larger than MaxInputSize
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxInputSize+1; i++ {
		if _, err := f.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = ReadItems(path)
	if err == nil {
		t.Error("expected error for file exceeding MaxInputSize")
	}
}

func TestReadItems_TooManyItems(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "many.ndjson")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxItemCount+1; i++ {
		if _, err := f.WriteString(`{"id":"x"}` + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = ReadItems(path)
	if err == nil {
		t.Error("expected error for file exceeding MaxItemCount")
	}
}

func TestReadItems_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ReadItems(path)
	if err != nil {
		t.Fatalf("ReadItems failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestReadItems_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("{not valid json}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadItems(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteResults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "results.json")

	results := []Result{
		{Index: 0, Success: true, ID: "abc123"},
		{Index: 1, Success: false, Error: "failed to create"},
	}

	if err := WriteResults(path, results); err != nil {
		t.Fatalf("WriteResults failed: %v", err)
	}

	// Verify file was created and contains valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read output file: %v", err)
	}

	var loaded []Result
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 results, got %d", len(loaded))
	}

	if loaded[0].ID != "abc123" {
		t.Errorf("expected first result ID 'abc123', got '%s'", loaded[0].ID)
	}

	if loaded[1].Error != "failed to create" {
		t.Errorf("expected second result error 'failed to create', got '%s'", loaded[1].Error)
	}
}

func TestReadItems_NonexistentFile(t *testing.T) {
	_, err := ReadItems("/nonexistent/path/file.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadItems_NDJSONWithEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "items.ndjson")

	content := `{"id": "1"}

{"id": "2"}

{"id": "3"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ReadItems(path)
	if err != nil {
		t.Fatalf("ReadItems failed: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 items (skipping empty lines), got %d", len(items))
	}
}
