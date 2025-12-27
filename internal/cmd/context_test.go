package cmd

import (
	"context"
	"testing"
)

func TestWorkspaceContext(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
	}{
		{
			name:      "with workspace",
			workspace: "work",
		},
		{
			name:      "with empty workspace",
			workspace: "",
		},
		{
			name:      "with special characters",
			workspace: "my-workspace_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithWorkspace(ctx, tt.workspace)

			got := WorkspaceFromContext(ctx)
			if got != tt.workspace {
				t.Errorf("WorkspaceFromContext() = %q, want %q", got, tt.workspace)
			}
		})
	}
}

func TestWorkspaceFromContext_NoValue(t *testing.T) {
	ctx := context.Background()
	got := WorkspaceFromContext(ctx)
	if got != "" {
		t.Errorf("WorkspaceFromContext() = %q, want empty string", got)
	}
}

func TestWorkspaceFromContext_WrongType(t *testing.T) {
	// This shouldn't happen in practice, but let's be defensive
	ctx := context.WithValue(context.Background(), workspaceKey{}, 123)
	got := WorkspaceFromContext(ctx)
	if got != "" {
		t.Errorf("WorkspaceFromContext() with wrong type = %q, want empty string", got)
	}
}
