package output

import (
	"context"
	"encoding/json"
	"testing"
)

type testList struct {
	Object  string        `json:"object"`
	Results []testElement `json:"results"`
}

type testElement struct {
	ID string `json:"id"`
}

func TestApplyResultsOnly_StructResults(t *testing.T) {
	ctx := WithResultsOnly(context.Background(), true)
	in := testList{
		Object:  "list",
		Results: []testElement{{ID: "a"}, {ID: "b"}},
	}
	got := ApplyResultsOnly(ctx, in)
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `[{"id":"a"},{"id":"b"}]` {
		t.Fatalf("unexpected: %s", string(b))
	}
}

func TestApplyResultsOnly_MapResults(t *testing.T) {
	ctx := WithResultsOnly(context.Background(), true)
	in := map[string]any{
		"object":  "list",
		"results": []any{map[string]any{"id": "x"}},
	}
	got := ApplyResultsOnly(ctx, in)
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `[{"id":"x"}]` {
		t.Fatalf("unexpected: %s", string(b))
	}
}
