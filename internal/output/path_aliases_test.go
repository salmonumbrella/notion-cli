package output

import (
	"context"
	"reflect"
	"testing"
)

func TestNormalizeSortPath(t *testing.T) {
	got, changed := NormalizeSortPath("ct")
	if !changed {
		t.Fatal("expected ct to be normalized")
	}
	if got != "created_time" {
		t.Fatalf("NormalizeSortPath(ct) = %q, want %q", got, "created_time")
	}

	got, changed = NormalizeSortPath("created_time")
	if changed {
		t.Fatal("did not expect canonical sort path to change")
	}
	if got != "created_time" {
		t.Fatalf("NormalizeSortPath(created_time) = %q, want %q", got, "created_time")
	}
}

func TestApplyAgentOptions_SortAlias(t *testing.T) {
	data := []map[string]interface{}{
		{
			"id":           "older",
			"created_time": "2026-01-01T00:00:00Z",
		},
		{
			"id":           "newer",
			"created_time": "2026-01-02T00:00:00Z",
		},
	}

	ctx := WithSort(context.Background(), "ct", true)
	got := ApplyAgentOptions(ctx, data)

	typed, ok := got.([]map[string]interface{})
	if !ok {
		t.Fatalf("ApplyAgentOptions returned %T, want []map[string]interface{}", got)
	}

	want := []map[string]interface{}{
		{
			"id":           "newer",
			"created_time": "2026-01-02T00:00:00Z",
		},
		{
			"id":           "older",
			"created_time": "2026-01-01T00:00:00Z",
		},
	}

	if !reflect.DeepEqual(typed, want) {
		t.Fatalf("sorted data mismatch\nwant: %#v\ngot: %#v", want, typed)
	}
}

func TestNormalizeSortPath_Empty(t *testing.T) {
	got, changed := NormalizeSortPath("")
	if changed || got != "" {
		t.Fatalf("expected no-op for empty sort path, got %q changed=%v", got, changed)
	}
}

func TestNormalizeSortPath_DottedPath(t *testing.T) {
	got, changed := NormalizeSortPath("props.Name.ct")
	if !changed {
		t.Fatal("expected dotted sort path to be normalized")
	}
	if got != "properties.Name.created_time" {
		t.Fatalf("NormalizeSortPath(props.Name.ct) = %q, want %q", got, "properties.Name.created_time")
	}
}

func TestNormalizeSortPath_MixedCase(t *testing.T) {
	got, changed := NormalizeSortPath("Status")
	if changed {
		t.Fatal("mixed-case sort path should not change")
	}
	if got != "Status" {
		t.Fatalf("NormalizeSortPath(Status) = %q, want %q", got, "Status")
	}
}
