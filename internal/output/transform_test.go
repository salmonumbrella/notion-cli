package output

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty is ok", input: "", wantErr: false},
		{name: "simple fields", input: "id,name", wantErr: false},
		{name: "alias and index", input: "first=items[0],name", wantErr: false},
		{name: "dot notation index", input: "first=items.0,name", wantErr: false},
		{name: "nested dot notation index", input: "val=props.Name.title.0.plain_text", wantErr: false},
		{name: "quoted key", input: "status=props['My Status']", wantErr: false},
		{name: "invalid path", input: "name=", wantErr: true},
		{name: "invalid bracket", input: "name[", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFields(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

func TestApplyOutputTransforms_ProjectFields(t *testing.T) {
	data := []map[string]interface{}{
		{
			"id":   "1",
			"name": "Alpha",
			"arr":  []interface{}{"x", "y"},
		},
	}

	ctx := WithFields(context.Background(), "id,name,first=arr[0]")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	want := []interface{}{
		map[string]interface{}{
			"id":    "1",
			"name":  "Alpha",
			"first": "x",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projected fields mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestApplyOutputTransforms_ProjectFields_DotNotationIndex(t *testing.T) {
	data := []map[string]interface{}{
		{
			"id": "1",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Hello"},
					},
				},
			},
		},
	}

	ctx := WithFields(context.Background(), "id,name=properties.Name.title.0.plain_text")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	want := []interface{}{
		map[string]interface{}{
			"id":   "1",
			"name": "Hello",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dot notation index mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestApplyOutputTransforms_ProjectFields_PathAliases(t *testing.T) {
	data := []map[string]interface{}{
		{
			"id": "1",
			"properties": map[string]interface{}{
				"Invoice Alert": map[string]interface{}{
					"rich_text": []interface{}{
						map[string]interface{}{"plain_text": "Ready"},
					},
				},
			},
		},
	}

	ctx := WithFields(context.Background(), "id,msg=props['Invoice Alert'].rt.0.pt")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	want := []interface{}{
		map[string]interface{}{
			"id":  "1",
			"msg": "Ready",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alias projection mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestApplyOutputTransforms_ProjectFields_ShortestPathAliases(t *testing.T) {
	data := []map[string]interface{}{
		{
			"id": "1",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Ready"},
					},
				},
			},
		},
	}

	ctx := WithFields(context.Background(), "id,name=pr.Name.t.0.p")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	want := []interface{}{
		map[string]interface{}{
			"id":   "1",
			"name": "Ready",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shortest alias projection mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestApplyOutputTransforms_BracketAndDotNotationEquivalent(t *testing.T) {
	data := map[string]interface{}{
		"arr": []interface{}{"a", "b", "c"},
	}

	bracketCtx := WithFields(context.Background(), "val=arr[1]")
	bracketGot, err := applyOutputTransforms(bracketCtx, data, FormatJSON)
	if err != nil {
		t.Fatalf("bracket notation error: %v", err)
	}

	dotCtx := WithFields(context.Background(), "val=arr.1")
	dotGot, err := applyOutputTransforms(dotCtx, data, FormatJSON)
	if err != nil {
		t.Fatalf("dot notation error: %v", err)
	}

	if !reflect.DeepEqual(bracketGot, dotGot) {
		t.Fatalf("bracket and dot notation should produce identical results\nbracket: %#v\ndot: %#v", bracketGot, dotGot)
	}
}

func TestApplyOutputTransforms_JSONPath(t *testing.T) {
	data := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{"id": "abc"},
		},
	}

	ctx := WithJSONPath(context.Background(), ".results[0].id")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	if got != "abc" {
		t.Fatalf("expected jsonpath result %q, got %#v", "abc", got)
	}
}

func TestApplyOutputTransforms_JSONPath_PathAliases(t *testing.T) {
	data := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"Invoice Alert": map[string]interface{}{
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "Done"},
						},
					},
				},
			},
		},
	}

	ctx := WithJSONPath(context.Background(), "$.rs[0].props[\"Invoice Alert\"].rt[0].pt")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	if got != "Done" {
		t.Fatalf("expected jsonpath alias result %q, got %#v", "Done", got)
	}
}

func TestApplyOutputTransforms_JSONPath_ShortestPathAliases(t *testing.T) {
	data := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"title": []interface{}{
							map[string]interface{}{"plain_text": "Done"},
						},
					},
				},
			},
		},
	}

	ctx := WithJSONPath(context.Background(), "$.rs[0].pr.Name.t[0].p")
	got, err := applyOutputTransforms(ctx, data, FormatJSON)
	if err != nil {
		t.Fatalf("applyOutputTransforms returned error: %v", err)
	}

	if got != "Done" {
		t.Fatalf("expected jsonpath shortest alias result %q, got %#v", "Done", got)
	}
}

func TestPrinter_FailEmpty(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFailEmpty(context.Background(), true)
	printer := NewPrinter(&buf, FormatJSON)

	err := printer.Print(ctx, []interface{}{})
	if err == nil {
		t.Fatalf("expected error for empty result with --fail-empty")
	}
	if !clierrors.IsUserError(err) {
		t.Fatalf("expected user error, got %T", err)
	}
}
