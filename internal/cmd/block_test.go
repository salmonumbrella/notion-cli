package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func TestIsArchivedBlockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "archived block error",
			err:      errors.New("notion API error 400 (validation_error): Can't edit block that is archived. You must unarchive the block before editing."),
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.New("notion API error 404: block not found"),
			expected: false,
		},
		{
			name:     "partial match - archived only",
			err:      errors.New("block is archived"),
			expected: false,
		},
		{
			name:     "partial match - edit only",
			err:      errors.New("cannot edit block"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isArchivedBlockError(tt.err)
			if result != tt.expected {
				t.Errorf("isArchivedBlockError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestPtrBool(t *testing.T) {
	truePtr := ptrBool(true)
	if truePtr == nil || *truePtr != true {
		t.Errorf("ptrBool(true) = %v, want pointer to true", truePtr)
	}

	falsePtr := ptrBool(false)
	if falsePtr == nil || *falsePtr != false {
		t.Errorf("ptrBool(false) = %v, want pointer to false", falsePtr)
	}
}

type stubBlockChildrenWriter struct {
	calls     []*notion.AppendBlockChildrenRequest
	responses []*notion.BlockList
}

func (s *stubBlockChildrenWriter) AppendBlockChildren(
	_ context.Context,
	_ string,
	req *notion.AppendBlockChildrenRequest,
) (*notion.BlockList, error) {
	clone := &notion.AppendBlockChildrenRequest{
		After: req.After,
	}
	clone.Children = append(clone.Children, req.Children...)
	s.calls = append(s.calls, clone)

	idx := len(s.calls) - 1
	if idx < len(s.responses) {
		return s.responses[idx], nil
	}

	return nil, fmt.Errorf("no stub response for call %d", idx)
}

func makeChildren(n int) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, map[string]interface{}{
			"object": "block",
			"type":   "paragraph",
		})
	}
	return out
}

func TestAppendBlockChildrenBatched_SplitsAndChainsAfter(t *testing.T) {
	writer := &stubBlockChildrenWriter{
		responses: []*notion.BlockList{
			{
				Object:  "list",
				Results: []notion.Block{{ID: "batch-1-last"}},
			},
			{
				Object:  "list",
				Results: []notion.Block{{ID: "batch-2-last"}},
			},
		},
	}

	children := makeChildren(162)
	got, err := appendBlockChildrenBatched(context.Background(), writer, "page-1", children, "anchor-1")
	if err != nil {
		t.Fatalf("appendBlockChildrenBatched() error = %v", err)
	}

	if got == nil {
		t.Fatal("appendBlockChildrenBatched() returned nil result")
	}

	if len(writer.calls) != 2 {
		t.Fatalf("AppendBlockChildren called %d times, want 2", len(writer.calls))
	}
	if len(writer.calls[0].Children) != 100 {
		t.Fatalf("first batch size = %d, want 100", len(writer.calls[0].Children))
	}
	if len(writer.calls[1].Children) != 62 {
		t.Fatalf("second batch size = %d, want 62", len(writer.calls[1].Children))
	}
	if writer.calls[0].After != "anchor-1" {
		t.Fatalf("first batch after = %q, want %q", writer.calls[0].After, "anchor-1")
	}
	if writer.calls[1].After != "batch-1-last" {
		t.Fatalf("second batch after = %q, want %q", writer.calls[1].After, "batch-1-last")
	}
}

func TestAppendBlockChildrenBatched_MissingBatchIDForChaining(t *testing.T) {
	writer := &stubBlockChildrenWriter{
		responses: []*notion.BlockList{
			{
				Object:  "list",
				Results: []notion.Block{},
			},
		},
	}

	_, err := appendBlockChildrenBatched(context.Background(), writer, "page-1", makeChildren(101), "anchor-1")
	if err == nil {
		t.Fatal("appendBlockChildrenBatched() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing last block ID for chaining") {
		t.Fatalf("error = %q, want missing chaining ID message", err.Error())
	}
}

func TestAppendBlockChildrenBatched_SingleBatch(t *testing.T) {
	writer := &stubBlockChildrenWriter{
		responses: []*notion.BlockList{
			{
				Object:  "list",
				Results: []notion.Block{{ID: "only-batch-last"}},
			},
		},
	}

	children := makeChildren(50)
	got, err := appendBlockChildrenBatched(context.Background(), writer, "page-1", children, "anchor-1")
	if err != nil {
		t.Fatalf("appendBlockChildrenBatched() error = %v", err)
	}

	if got == nil {
		t.Fatal("appendBlockChildrenBatched() returned nil result")
	}

	if len(writer.calls) != 1 {
		t.Fatalf("AppendBlockChildren called %d times, want 1", len(writer.calls))
	}
	if len(writer.calls[0].Children) != 50 {
		t.Fatalf("batch size = %d, want 50", len(writer.calls[0].Children))
	}
	if writer.calls[0].After != "anchor-1" {
		t.Fatalf("batch after = %q, want %q", writer.calls[0].After, "anchor-1")
	}
	if len(got.Results) != 1 {
		t.Fatalf("combined results = %d, want 1", len(got.Results))
	}
}
