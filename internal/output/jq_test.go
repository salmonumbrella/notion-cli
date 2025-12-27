package output

import (
	"bytes"
	"context"
	"testing"
)

func TestPrinter_WithQuery_FilterArray(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": "1", "name": "Alice"},
			map[string]interface{}{"id": "2", "name": "Bob"},
		},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".items[].name")
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print failed: %v", err)
	}

	output := buf.String()
	// Each result on its own line
	if output != "\"Alice\"\n\"Bob\"\n" {
		t.Errorf("expected filtered output, got: %q", output)
	}
}

func TestPrinter_WithQuery_InvalidQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".invalid[")
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error for invalid jq query")
	}
}

func TestPrinter_WithQuery_NoQuery(t *testing.T) {
	data := map[string]string{"key": "value"}

	var buf bytes.Buffer
	ctx := context.Background() // No query
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print failed: %v", err)
	}

	// Should output full JSON
	if !bytes.Contains(buf.Bytes(), []byte(`"key"`)) {
		t.Errorf("expected full JSON output, got: %s", buf.String())
	}
}

func TestQueryFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	query := QueryFromContext(ctx)
	if query != "" {
		t.Errorf("expected empty query, got: %q", query)
	}
}

func TestWithQuery_RoundTrip(t *testing.T) {
	ctx := WithQuery(context.Background(), ".foo.bar")
	query := QueryFromContext(ctx)
	if query != ".foo.bar" {
		t.Errorf("expected .foo.bar, got: %q", query)
	}
}
