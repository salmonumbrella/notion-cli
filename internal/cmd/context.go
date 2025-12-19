package cmd

import "context"

type workspaceKey struct{}

// WithWorkspace stores a workspace name in the context
func WithWorkspace(ctx context.Context, workspace string) context.Context {
	return context.WithValue(ctx, workspaceKey{}, workspace)
}

// WorkspaceFromContext retrieves the workspace name from the context
func WorkspaceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(workspaceKey{}).(string); ok {
		return v
	}
	return ""
}
