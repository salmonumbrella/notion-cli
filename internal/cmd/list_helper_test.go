// internal/cmd/list_helper_test.go
package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestListConfig_BuildCommand(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "Test"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %q", cmd.Use)
	}
	if cmd.Short != "List test items" {
		t.Errorf("expected Short 'List test items', got %q", cmd.Short)
	}
}

func TestNewListCommand_RunE_JSONOutput(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "First"}, {ID: "2", Name: "Second"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set JSON format in context
	ctx := output.WithFormat(context.Background(), output.FormatJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify JSON output contains expected data
	if !strings.Contains(got, `"id": "1"`) {
		t.Errorf("expected JSON to contain id '1', got: %s", got)
	}
	if !strings.Contains(got, `"name": "First"`) {
		t.Errorf("expected JSON to contain name 'First', got: %s", got)
	}
	if !strings.Contains(got, `"id": "2"`) {
		t.Errorf("expected JSON to contain id '2', got: %s", got)
	}
}

func TestNewListCommand_RunE_TableOutput(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "First"}, {ID: "2", Name: "Second"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set table format in context
	ctx := output.WithFormat(context.Background(), output.FormatTable)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify table output contains headers and data
	if !strings.Contains(got, "ID") {
		t.Errorf("expected table to contain header 'ID', got: %s", got)
	}
	if !strings.Contains(got, "NAME") {
		t.Errorf("expected table to contain header 'NAME', got: %s", got)
	}
	if !strings.Contains(got, "First") {
		t.Errorf("expected table to contain 'First', got: %s", got)
	}
	if !strings.Contains(got, "Second") {
		t.Errorf("expected table to contain 'Second', got: %s", got)
	}
}

func TestNewListCommand_RunE_YAMLOutput(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "First"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set YAML format in context
	ctx := output.WithFormat(context.Background(), output.FormatYAML)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify YAML output contains expected data
	if !strings.Contains(got, "id: \"1\"") {
		t.Errorf("expected YAML to contain 'id: \"1\"', got: %s", got)
	}
	if !strings.Contains(got, "name: First") {
		t.Errorf("expected YAML to contain 'name: First', got: %s", got)
	}
}

func TestNewListCommand_RunE_EmptyResult(t *testing.T) {
	config := ListConfig[testItem]{
		Use:          "list",
		Short:        "List test items",
		Headers:      []string{"ID", "NAME"},
		EmptyMessage: "No test items found",
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stderr (empty message goes to stderr)
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := output.WithFormat(context.Background(), output.FormatTable)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "No test items found") {
		t.Errorf("expected stderr to contain empty message, got: %s", got)
	}
}

func TestNewListCommand_RunE_EmptyResultDefaultMessage(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		// EmptyMessage not set - should use default
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := output.WithFormat(context.Background(), output.FormatTable)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "No items found") {
		t.Errorf("expected stderr to contain default empty message, got: %s", got)
	}
}

func TestNewListCommand_RunE_FetchError(t *testing.T) {
	expectedErr := errors.New("fetch failed: connection timeout")

	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{}, expectedErr
		},
	}

	cmd := NewListCommand(config)

	ctx := output.WithFormat(context.Background(), output.FormatTable)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestNewListCommand_RunE_TextOutput(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "First"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set text format in context
	ctx := output.WithFormat(context.Background(), output.FormatText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Text format for slices outputs one item per line
	// The output should contain the struct representation
	if got == "" {
		t.Error("expected non-empty text output")
	}
}

func TestNewListCommand_RunE_HasMoreIndicator(t *testing.T) {
	config := ListConfig[testItem]{
		Use:     "list",
		Short:   "List test items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{item.ID, item.Name}
		},
		Fetch: func(ctx context.Context, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: "1", Name: "First"}},
				HasMore: true, // More results available
			}, nil
		},
	}

	cmd := NewListCommand(config)

	// Capture both stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	ctx := output.WithFormat(context.Background(), output.FormatTable)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufErr bytes.Buffer
	_, _ = bufErr.ReadFrom(rErr)
	_ = rOut.Close()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stderrContent := bufErr.String()
	if !strings.Contains(stderrContent, "more results available") {
		t.Errorf("expected stderr to contain 'more results available', got: %s", stderrContent)
	}
}

func TestListResult_Empty(t *testing.T) {
	result := ListResult[testItem]{
		Items:   []testItem{},
		HasMore: false,
	}

	if len(result.Items) != 0 {
		t.Error("expected empty items")
	}
	if result.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestListResult_WithItems(t *testing.T) {
	result := ListResult[testItem]{
		Items:   []testItem{{ID: "1", Name: "Test"}, {ID: "2", Name: "Test2"}},
		HasMore: true,
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}
}
