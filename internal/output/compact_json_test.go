package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPrinter_PrintJSON_CompactMode(t *testing.T) {
	data := map[string]interface{}{
		"id":    "p1",
		"title": "Page",
	}

	t.Run("default pretty JSON", func(t *testing.T) {
		var buf bytes.Buffer
		printer := NewPrinter(&buf, FormatJSON)

		if err := printer.Print(context.Background(), data); err != nil {
			t.Fatalf("print failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "\n  ") {
			t.Fatalf("expected pretty JSON indentation, got: %s", out)
		}
	})

	t.Run("compact JSON when enabled", func(t *testing.T) {
		var buf bytes.Buffer
		printer := NewPrinter(&buf, FormatJSON)
		ctx := WithCompactJSON(context.Background(), true)

		if err := printer.Print(ctx, data); err != nil {
			t.Fatalf("print failed: %v", err)
		}

		out := buf.String()
		if strings.Contains(out, "\n  ") {
			t.Fatalf("expected compact JSON, got pretty JSON: %s", out)
		}
	})
}

func TestPrinter_PrintJSON_QueryCompactMode(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": "a", "name": "one"},
		},
	}

	var buf bytes.Buffer
	printer := NewPrinter(&buf, FormatJSON)
	ctx := WithQuery(context.Background(), ".items[0]")
	ctx = WithCompactJSON(ctx, true)

	if err := printer.Print(ctx, data); err != nil {
		t.Fatalf("print failed: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\n  ") {
		t.Fatalf("expected compact query output, got pretty JSON: %s", out)
	}
}
