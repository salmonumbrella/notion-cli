package cmd

import (
	"context"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

func TestAgentFlags_YesFromContext(t *testing.T) {
	ctx := output.WithYes(context.Background(), true)
	if !output.YesFromContext(ctx) {
		t.Error("expected Yes to be true")
	}

	ctx2 := output.WithYes(context.Background(), false)
	if output.YesFromContext(ctx2) {
		t.Error("expected Yes to be false")
	}
}

func TestAgentFlags_YesFromContext_Default(t *testing.T) {
	ctx := context.Background()
	if output.YesFromContext(ctx) {
		t.Error("expected default Yes to be false")
	}
}

func TestAgentFlags_LimitFromContext(t *testing.T) {
	ctx := output.WithLimit(context.Background(), 10)
	if output.LimitFromContext(ctx) != 10 {
		t.Errorf("expected limit 10, got %d", output.LimitFromContext(ctx))
	}
}

func TestAgentFlags_LimitFromContext_Default(t *testing.T) {
	ctx := context.Background()
	if output.LimitFromContext(ctx) != 0 {
		t.Errorf("expected default limit 0, got %d", output.LimitFromContext(ctx))
	}
}

func TestAgentFlags_SortFromContext(t *testing.T) {
	ctx := output.WithSort(context.Background(), "created_time", true)
	field, desc := output.SortFromContext(ctx)
	if field != "created_time" {
		t.Errorf("expected sort field created_time, got %s", field)
	}
	if !desc {
		t.Error("expected descending to be true")
	}
}

func TestAgentFlags_SortFromContext_Default(t *testing.T) {
	ctx := context.Background()
	field, desc := output.SortFromContext(ctx)
	if field != "" {
		t.Errorf("expected empty sort field, got %s", field)
	}
	if desc {
		t.Error("expected default descending to be false")
	}
}

func TestAgentFlags_QuietFromContext(t *testing.T) {
	ctx := output.WithQuiet(context.Background(), true)
	if !output.QuietFromContext(ctx) {
		t.Error("expected Quiet to be true")
	}

	ctx2 := output.WithQuiet(context.Background(), false)
	if output.QuietFromContext(ctx2) {
		t.Error("expected Quiet to be false")
	}
}

func TestAgentFlags_QuietFromContext_Default(t *testing.T) {
	ctx := context.Background()
	if output.QuietFromContext(ctx) {
		t.Error("expected default Quiet to be false")
	}
}
