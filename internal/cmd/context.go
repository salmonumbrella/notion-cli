package cmd

import (
	"context"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

type (
	workspaceKey   struct{}
	errorFormatKey struct{}
	configKey      struct{}
)

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

// WithErrorFormat stores the error format in the context.
func WithErrorFormat(ctx context.Context, format string) context.Context {
	return context.WithValue(ctx, errorFormatKey{}, format)
}

// ErrorFormatFromContext retrieves the error format from context.
func ErrorFormatFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(errorFormatKey{}).(string); ok {
		return v
	}
	return ""
}

// WithConfig stores loaded CLI config in context for downstream helpers.
func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}

// ConfigFromContext retrieves CLI config from context.
func ConfigFromContext(ctx context.Context) *config.Config {
	if v, ok := ctx.Value(configKey{}).(*config.Config); ok {
		return v
	}
	return nil
}
