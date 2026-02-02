package cmd

import (
	"context"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func TestResolveID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Aliases: map[string]skill.CustomAlias{
			"standup": {TargetID: "page-789", Type: "page"},
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"me", "user-456"},
		{"standup", "page-789"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveID_NilSkillFile(t *testing.T) {
	got := resolveID(nil, "test-input")
	if got != "test-input" {
		t.Errorf("resolveID(nil, %q) = %q, want %q", "test-input", got, "test-input")
	}
}

func TestResolveUserID(t *testing.T) {
	sf := &skill.SkillFile{
		Users: map[string]skill.UserAlias{
			"me": {ID: "user-456"},
		},
		Databases: map[string]skill.DatabaseAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"me", "user-456"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveUserID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveUserID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveDatabaseID(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{
			"issues": {ID: "db-123"},
		},
		Users:   map[string]skill.UserAlias{},
		Aliases: map[string]skill.CustomAlias{},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"issues", "db-123"},
		{"abc12345-1234-1234-1234-123456789012", "abc12345-1234-1234-1234-123456789012"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveDatabaseID(sf, tt.input)
			if got != tt.expected {
				t.Errorf("resolveDatabaseID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWithSkillFile(t *testing.T) {
	sf := &skill.SkillFile{
		Databases: map[string]skill.DatabaseAlias{},
		Users:     map[string]skill.UserAlias{},
		Aliases:   map[string]skill.CustomAlias{},
	}

	ctx := context.Background()
	ctx = WithSkillFile(ctx, sf)

	got := SkillFileFromContext(ctx)
	if got != sf {
		t.Errorf("SkillFileFromContext() did not return the expected skill file")
	}
}

func TestSkillFileFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := SkillFileFromContext(ctx)
	if got != nil {
		t.Errorf("SkillFileFromContext() = %v, want nil", got)
	}
}
