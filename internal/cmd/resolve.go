// Package cmd provides CLI commands for the notion-cli tool
package cmd

import (
	"context"

	"github.com/salmonumbrella/notion-cli/internal/skill"
)

// skillFileKey is the context key for the loaded skill file
type skillFileKey struct{}

// WithSkillFile adds the skill file to context
func WithSkillFile(ctx context.Context, sf *skill.SkillFile) context.Context {
	return context.WithValue(ctx, skillFileKey{}, sf)
}

// SkillFileFromContext retrieves the skill file from context
func SkillFileFromContext(ctx context.Context) *skill.SkillFile {
	if sf, ok := ctx.Value(skillFileKey{}).(*skill.SkillFile); ok {
		return sf
	}
	return nil
}

// resolveID resolves an alias to its target ID, or returns the input unchanged
func resolveID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}

	// Try database alias
	if id, ok := sf.ResolveDatabase(input); ok {
		return id
	}

	// Try user alias
	if id, ok := sf.ResolveUser(input); ok {
		return id
	}

	// Try custom alias
	if id, _, ok := sf.ResolveAlias(input); ok {
		return id
	}

	// Return unchanged
	return input
}

// resolveUserID specifically resolves user aliases
func resolveUserID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}
	if id, ok := sf.ResolveUser(input); ok {
		return id
	}
	return input
}

// resolveDatabaseID specifically resolves database aliases
func resolveDatabaseID(sf *skill.SkillFile, input string) string {
	if sf == nil {
		return input
	}
	if id, ok := sf.ResolveDatabase(input); ok {
		return id
	}
	return input
}
