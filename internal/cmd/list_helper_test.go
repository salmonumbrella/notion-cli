// internal/cmd/list_helper_test.go
package cmd

import (
	"context"
	"testing"
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
		Fetch: func(ctx context.Context, page, pageSize int) (ListResult[testItem], error) {
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
