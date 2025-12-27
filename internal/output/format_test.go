package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{
			name:  "text lowercase",
			input: "text",
			want:  FormatText,
		},
		{
			name:  "text uppercase",
			input: "TEXT",
			want:  FormatText,
		},
		{
			name:  "text with whitespace",
			input: "  text  ",
			want:  FormatText,
		},
		{
			name:  "empty defaults to text",
			input: "",
			want:  FormatText,
		},
		{
			name:  "json lowercase",
			input: "json",
			want:  FormatJSON,
		},
		{
			name:  "json uppercase",
			input: "JSON",
			want:  FormatJSON,
		},
		{
			name:  "ndjson lowercase",
			input: "ndjson",
			want:  FormatNDJSON,
		},
		{
			name:  "ndjson uppercase",
			input: "NDJSON",
			want:  FormatNDJSON,
		},
		{
			name:  "jsonl lowercase",
			input: "jsonl",
			want:  FormatNDJSON,
		},
		{
			name:  "jsonl uppercase",
			input: "JSONL",
			want:  FormatNDJSON,
		},
		{
			name:  "table lowercase",
			input: "table",
			want:  FormatTable,
		},
		{
			name:  "table uppercase",
			input: "TABLE",
			want:  FormatTable,
		},
		{
			name:  "yaml lowercase",
			input: "yaml",
			want:  FormatYAML,
		},
		{
			name:  "yaml uppercase",
			input: "YAML",
			want:  FormatYAML,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid format xml",
			input:   "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrinter_PrintJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("simple struct", func(t *testing.T) {
		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatJSON)

		data := Person{Name: "Alice", Age: 30}
		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		// Verify it's valid JSON
		var result Person
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if result.Name != "Alice" || result.Age != 30 {
			t.Errorf("got %+v, want {Name:Alice Age:30}", result)
		}
	})

	t.Run("slice of structs", func(t *testing.T) {
		type Item struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatJSON)

		data := []Item{
			{ID: "1", Title: "First"},
			{ID: "2", Title: "Second"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		var result []Item
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("got %d items, want 2", len(result))
		}
	})

	t.Run("map", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatJSON)

		data := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if result["key1"] != "value1" {
			t.Errorf("got %v, want value1", result["key1"])
		}
	})

	t.Run("nil data", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatJSON)

		if err := p.Print(ctx, nil); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		if buf.Len() == 0 {
			// nil is acceptable output for nil data
			return
		}
	})
}

func TestPrinter_PrintNDJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("slice outputs lines", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatNDJSON)

		data := []map[string]interface{}{
			{"id": 1},
			{"id": 2},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		var first map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
			t.Fatalf("invalid JSON on line 1: %v", err)
		}
	})
}

func TestPrinter_PrintYAML(t *testing.T) {
	ctx := context.Background()

	t.Run("simple struct", func(t *testing.T) {
		type Person struct {
			Name string `yaml:"name"`
			Age  int    `yaml:"age"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatYAML)

		data := Person{Name: "Alice", Age: 30}
		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		// Verify it's valid YAML
		var result Person
		if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid YAML: %v", err)
		}

		if result.Name != "Alice" || result.Age != 30 {
			t.Errorf("got %+v, want {Name:Alice Age:30}", result)
		}
	})

	t.Run("slice of structs", func(t *testing.T) {
		type Item struct {
			ID    string `yaml:"id"`
			Title string `yaml:"title"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatYAML)

		data := []Item{
			{ID: "1", Title: "First"},
			{ID: "2", Title: "Second"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		var result []Item
		if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid YAML: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("got %d items, want 2", len(result))
		}
	})

	t.Run("map", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatYAML)

		data := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		var result map[string]interface{}
		if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid YAML: %v", err)
		}

		if result["key1"] != "value1" {
			t.Errorf("got %v, want value1", result["key1"])
		}
	})

	t.Run("nil data", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatYAML)

		if err := p.Print(ctx, nil); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		if buf.Len() == 0 {
			// nil is acceptable output for nil data
			return
		}
	})
}

func TestPrinter_PrintText(t *testing.T) {
	ctx := context.Background()

	t.Run("struct with json tags", func(t *testing.T) {
		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
			City string `json:"city,omitempty"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		data := Person{Name: "Alice", Age: 30, City: "NYC"}
		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "name: Alice") {
			t.Errorf("output missing 'name: Alice': %s", output)
		}
		if !strings.Contains(output, "age: 30") {
			t.Errorf("output missing 'age: 30': %s", output)
		}
		if !strings.Contains(output, "city: NYC") {
			t.Errorf("output missing 'city: NYC': %s", output)
		}
	})

	t.Run("map", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		data := map[string]string{
			"id":    "123",
			"title": "Test",
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "id:") && !strings.Contains(output, "title:") {
			t.Errorf("output missing expected keys: %s", output)
		}
	})

	t.Run("slice of strings", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		data := []string{"item1", "item2", "item3"}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 3 {
			t.Errorf("got %d lines, want 3: %s", len(lines), output)
		}
	})

	t.Run("primitive value", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		if err := p.Print(ctx, "simple string"); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "simple string" {
			t.Errorf("got %q, want 'simple string'", output)
		}
	})

	t.Run("nil data", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		if err := p.Print(ctx, nil); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output for nil, got: %s", buf.String())
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type Item struct {
			ID string `json:"id"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		data := &Item{ID: "123"}
		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "id: 123") {
			t.Errorf("output missing 'id: 123': %s", output)
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		type Item struct {
			ID string
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatText)

		var data *Item
		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output for nil pointer, got: %s", buf.String())
		}
	})
}

func TestPrinter_PrintTable(t *testing.T) {
	ctx := context.Background()

	t.Run("slice of structs", func(t *testing.T) {
		type Item struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Count int    `json:"count"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []Item{
			{ID: "1", Title: "First", Count: 10},
			{ID: "2", Title: "Second", Count: 20},
			{ID: "3", Title: "Third", Count: 30},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 3 data rows
		if len(lines) != 4 {
			t.Errorf("got %d lines, want 4: %s", len(lines), output)
		}

		// Check header
		header := strings.ToUpper(lines[0])
		if !strings.Contains(header, "ID") || !strings.Contains(header, "TITLE") || !strings.Contains(header, "COUNT") {
			t.Errorf("header missing expected columns: %s", lines[0])
		}

		// Check that data appears in output
		if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
			t.Errorf("output missing expected data: %s", output)
		}
	})

	t.Run("slice of maps", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []map[string]interface{}{
			{"id": "1", "name": "Alice"},
			{"id": "2", "name": "Bob"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 2 data rows
		if len(lines) < 3 {
			t.Errorf("got %d lines, want at least 3: %s", len(lines), output)
		}

		// Check that data appears
		if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
			t.Errorf("output missing expected data: %s", output)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []map[string]string{}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output for empty slice, got: %s", buf.String())
		}
	})

	t.Run("non-slice data returns error", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := map[string]string{"key": "value"}

		err := p.Print(ctx, data)
		if err == nil {
			t.Error("expected error for non-slice data, got nil")
		}
		if !strings.Contains(err.Error(), "slice or array") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("slice of primitives returns error", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []string{"a", "b", "c"}

		err := p.Print(ctx, data)
		if err == nil {
			t.Error("expected error for slice of primitives, got nil")
		}
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		type Item struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []*Item{
			{ID: "1", Name: "First"},
			{ID: "2", Name: "Second"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
			t.Errorf("output missing expected data: %s", output)
		}
	})

	t.Run("maps with missing keys show dash", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		data := []map[string]interface{}{
			{"id": "1", "name": "Alice", "age": 30},
			{"id": "2", "name": "Bob"}, // missing age
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Second row should have a dash for missing age
		if len(lines) >= 3 {
			// The exact position depends on column order, but there should be a dash
			if !strings.Contains(output, "-") {
				t.Errorf("expected dash for missing value: %s", output)
			}
		}
	})

	t.Run("slice with nil pointer maps skips nil entries", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		// Create slice with nil pointer to map
		data := []*map[string]interface{}{
			{"id": "1", "name": "Alice"},
			nil, // nil pointer should be skipped, not cause infinite loop
			{"id": "3", "name": "Charlie"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 2 data rows (nil skipped)
		if len(lines) != 3 {
			t.Errorf("got %d lines, want 3 (header + 2 rows, nil skipped): %s", len(lines), output)
		}

		// Check that non-nil data appears
		if !strings.Contains(output, "Alice") || !strings.Contains(output, "Charlie") {
			t.Errorf("output missing expected data: %s", output)
		}
	})

	t.Run("slice with nil pointer structs skips nil entries", func(t *testing.T) {
		type Item struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf, FormatTable)

		// Create slice with nil pointer to struct
		data := []*Item{
			{ID: "1", Name: "First"},
			nil, // nil pointer should be skipped, not cause infinite loop
			{ID: "3", Name: "Third"},
		}

		if err := p.Print(ctx, data); err != nil {
			t.Fatalf("Print() error = %v", err)
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have header + 2 data rows (nil skipped)
		if len(lines) != 3 {
			t.Errorf("got %d lines, want 3 (header + 2 rows, nil skipped): %s", len(lines), output)
		}

		// Check that non-nil data appears
		if !strings.Contains(output, "First") || !strings.Contains(output, "Third") {
			t.Errorf("output missing expected data: %s", output)
		}
	})
}

func TestPrinter_UnsupportedFormat(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	// Create printer with invalid format (bypassing ParseFormat)
	p := &Printer{
		w:      &buf,
		format: Format("invalid"),
	}

	err := p.Print(ctx, "test")
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewPrinter(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, FormatJSON)

	if p == nil {
		t.Fatal("NewPrinter returned nil")
	}
	if p.w != &buf {
		t.Error("writer not set correctly")
	}
	if p.format != FormatJSON {
		t.Errorf("format = %v, want %v", p.format, FormatJSON)
	}
}
