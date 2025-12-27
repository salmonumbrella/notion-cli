package output

import (
	"bytes"
	"context"
	"strings"
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

func TestPrinter_WithQuery_UnexpectedEOFHint(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), `.props | map({key`)
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, map[string]string{"key": "value"})
	if err == nil {
		t.Fatal("expected error for incomplete jq query")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid --query:") {
		t.Fatalf("expected invalid --query prefix, got: %s", msg)
	}
	if !strings.Contains(msg, "query looks incomplete") {
		t.Fatalf("expected incomplete-query hint, got: %s", msg)
	}
	if !strings.Contains(msg, "--query-file") {
		t.Fatalf("expected --query-file guidance, got: %s", msg)
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

func TestNormalizeQuery_RemovesEscapedBangOutsideStrings(t *testing.T) {
	query := `.results[] | select(.status \!= "Done")`
	got, changed := NormalizeQuery(query)
	if !changed {
		t.Fatalf("expected change for escaped bang")
	}
	if got != `.results[] | select(.status != "Done")` {
		t.Errorf("normalized query = %q, want %q", got, `.results[] | select(.status != "Done")`)
	}
}

func TestNormalizeQuery_LeavesEscapedBangInsideStrings(t *testing.T) {
	query := `test("\\!=")`
	got, changed := NormalizeQuery(query)
	if changed {
		t.Fatalf("unexpected change for escaped bang inside string")
	}
	if got != query {
		t.Errorf("normalized query = %q, want %q", got, query)
	}
}

func TestNormalizeQuery_NoChange(t *testing.T) {
	query := `.results[] | select(.status != "Done")`
	got, changed := NormalizeQuery(query)
	if changed {
		t.Fatalf("unexpected change for clean query")
	}
	if got != query {
		t.Errorf("normalized query = %q, want %q", got, query)
	}
}

func TestNormalizeQuery_ExpandsPathAliases(t *testing.T) {
	query := `.props["Invoice Alert"].rt[0].pt`
	got, changed := NormalizeQuery(query)
	// The bool is reserved for shell "\!" normalization warnings.
	if changed {
		t.Fatalf("unexpected escape-normalization change for alias-only query")
	}
	want := `.properties["Invoice Alert"].rich_text[0].plain_text`
	if got != want {
		t.Errorf("normalized query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_DoesNotRewriteStringsOrMixedCase(t *testing.T) {
	query := `.Status | .rt | "pt" | .properties["rt"]`
	got, _ := NormalizeQuery(query)
	want := `.Status | .rich_text | "pt" | .properties["rt"]`
	if got != want {
		t.Errorf("normalized query = %q, want %q", got, want)
	}
}

func TestPrinter_WithQuery_PathAliases(t *testing.T) {
	data := map[string]interface{}{
		"properties": map[string]interface{}{
			"Invoice Alert": map[string]interface{}{
				"rich_text": []interface{}{
					map[string]interface{}{"plain_text": "ready"},
				},
			},
		},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), `.props["Invoice Alert"].rt[0].pt`)
	printer := NewPrinter(&buf, FormatJSON)

	if err := printer.Print(ctx, data); err != nil {
		t.Fatalf("print with alias query failed: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != `"ready"` {
		t.Errorf("expected \"ready\", got %s", got)
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

func TestPrinter_WithQuery_RuntimeError_NoPanicFormatting_JSON(t *testing.T) {
	type page struct {
		ID string `json:"id"`
	}

	data := map[string]interface{}{
		"results": []page{{ID: "1"}},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".results.foo")
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, data)
	if err == nil {
		t.Fatal("expected runtime query error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "query error:") {
		t.Fatalf("expected query error prefix, got: %s", msg)
	}
	if strings.Contains(msg, "PANIC=Error method") {
		t.Fatalf("query error leaked panic formatting: %s", msg)
	}
	if !strings.Contains(msg, "invalid type:") {
		t.Fatalf("expected invalid type message, got: %s", msg)
	}
}

func TestPrinter_WithQuery_RuntimeError_NoPanicFormatting_NDJSON(t *testing.T) {
	type page struct {
		ID string `json:"id"`
	}

	data := map[string]interface{}{
		"results": []page{{ID: "1"}},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".results.foo")
	printer := NewPrinter(&buf, FormatNDJSON)

	err := printer.Print(ctx, data)
	if err == nil {
		t.Fatal("expected runtime query error")
	}

	msg := err.Error()
	if strings.Contains(msg, "PANIC=Error method") {
		t.Fatalf("query error leaked panic formatting: %s", msg)
	}
}

// TestPrinter_WithQuery_TypedStruct_JSON verifies that --query works on typed Go
// structs (like *notion.Page), not just map[string]interface{}. This was the root
// cause of the panic: gojq received a struct it couldn't traverse.
func TestPrinter_WithQuery_TypedStruct_JSON(t *testing.T) {
	type page struct {
		Object string                 `json:"object"`
		ID     string                 `json:"id"`
		Props  map[string]interface{} `json:"properties"`
	}

	data := &page{
		Object: "page",
		ID:     "abc-123",
		Props: map[string]interface{}{
			"Name": map[string]interface{}{"type": "title"},
		},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".id")
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print with --query on typed struct failed: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != `"abc-123"` {
		t.Errorf("expected \"abc-123\", got: %s", got)
	}
}

// TestPrinter_WithQuery_TypedStruct_NDJSON verifies the same fix for NDJSON output.
func TestPrinter_WithQuery_TypedStruct_NDJSON(t *testing.T) {
	type page struct {
		Object string `json:"object"`
		ID     string `json:"id"`
	}

	data := &page{Object: "page", ID: "abc-123"}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".id")
	printer := NewPrinter(&buf, FormatNDJSON)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print with --query on typed struct failed: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != `"abc-123"` {
		t.Errorf("expected \"abc-123\", got: %s", got)
	}
}

// TestPrinter_WithQuery_TypedStruct_NestedAccess verifies deep property access
// on typed structs — the exact scenario from the bug report:
// notion page get ID -o json --query '.properties["Name"]'
func TestPrinter_WithQuery_TypedStruct_NestedAccess(t *testing.T) {
	type page struct {
		Object     string                 `json:"object"`
		ID         string                 `json:"id"`
		Properties map[string]interface{} `json:"properties"`
	}

	data := &page{
		Object: "page",
		ID:     "abc-123",
		Properties: map[string]interface{}{
			"Name": map[string]interface{}{
				"type": "title",
				"title": []interface{}{
					map[string]interface{}{"plain_text": "Hello"},
				},
			},
		},
	}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), `.properties["Name"].type`)
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print with --query on nested struct property failed: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != `"title"` {
		t.Errorf("expected \"title\", got: %s", got)
	}
}

// TestPrinter_WithQuery_TextFormat verifies that --query works with text output.
func TestPrinter_WithQuery_TextFormat(t *testing.T) {
	type page struct {
		Object string `json:"object"`
		ID     string `json:"id"`
	}

	data := &page{Object: "page", ID: "abc-123"}

	var buf bytes.Buffer
	ctx := WithQuery(context.Background(), ".id")
	printer := NewPrinter(&buf, FormatText)

	err := printer.Print(ctx, data)
	if err != nil {
		t.Fatalf("print with --query on text format failed: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "abc-123" {
		t.Errorf("expected abc-123, got: %s", got)
	}
}

func TestNormalizeQuery_Idempotent(t *testing.T) {
	input := `.props["Invoice Alert"].rt[0].pt`
	first, _ := NormalizeQuery(input)
	second, _ := NormalizeQuery(first)
	if first != second {
		t.Fatalf("NormalizeQuery is not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestNormalizeQuery_PipeSeparated(t *testing.T) {
	query := `.props.Status | .rt | map(.pt)`
	got, _ := NormalizeQuery(query)
	want := `.properties.Status | .rich_text | map(.plain_text)`
	if got != want {
		t.Errorf("pipe-separated query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_RecursiveDescent(t *testing.T) {
	query := `..rt`
	got, _ := NormalizeQuery(query)
	want := `..rich_text`
	if got != want {
		t.Errorf("recursive descent query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_OptionalOperator(t *testing.T) {
	query := `.rt?`
	got, _ := NormalizeQuery(query)
	want := `.rich_text?`
	if got != want {
		t.Errorf("optional operator query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_MultipleAliases(t *testing.T) {
	query := `.rs[0].props.Name.ti[0].pt`
	got, _ := NormalizeQuery(query)
	want := `.results[0].properties.Name.title[0].plain_text`
	if got != want {
		t.Errorf("multiple aliases query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_ShortestAliases(t *testing.T) {
	query := `.rs[0].pr.Name.t[0].p`
	got, _ := NormalizeQuery(query)
	want := `.results[0].properties.Name.title[0].plain_text`
	if got != want {
		t.Errorf("shortest aliases query = %q, want %q", got, want)
	}
}

func TestNormalizeQuery_EmptyAndWhitespace(t *testing.T) {
	got, changed := NormalizeQuery("")
	if changed || got != "" {
		t.Fatalf("expected no-op for empty query, got %q changed=%v", got, changed)
	}
	got, changed = NormalizeQuery("   ")
	if changed || got != "   " {
		t.Fatalf("expected no-op for whitespace query, got %q changed=%v", got, changed)
	}
}

func TestNormalizeQuery_CommentPreserved(t *testing.T) {
	query := ".props.Name # rt is alias\n.rt"
	got, _ := NormalizeQuery(query)
	want := ".properties.Name # rt is alias\n.rich_text"
	if got != want {
		t.Errorf("comment handling: got %q, want %q", got, want)
	}
}

func TestNormalizeQuery_NoDotPrefix(t *testing.T) {
	// Bare identifiers without a leading dot should not be rewritten
	got, _ := NormalizeQuery("rt")
	if got != "rt" {
		t.Fatalf("bare token without dot should not be rewritten, got %q", got)
	}
}
