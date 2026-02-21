// Package skill provides functionality for parsing and resolving aliases
// from Claude skill files. Skill files are markdown documents that contain
// tables mapping human-friendly aliases to Notion IDs.
package skill

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// SkillFile represents the parsed skill file
type SkillFile struct {
	Databases map[string]DatabaseAlias
	Users     map[string]UserAlias
	Aliases   map[string]CustomAlias
}

// DatabaseAlias represents a database alias entry
type DatabaseAlias struct {
	Alias         string
	Name          string
	ID            string
	TitleProperty string
	DefaultStatus string
}

// UserAlias represents a user alias entry
type UserAlias struct {
	Alias string
	Name  string
	ID    string
}

// CustomAlias represents a custom alias entry
type CustomAlias struct {
	Alias    string
	Type     string
	TargetID string
}

// DefaultPath returns the default skill file path
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "skills", "notion-cli", "notion-cli.md")
}

// Load loads the skill file from the default path
func Load() (*SkillFile, error) {
	f, err := os.Open(DefaultPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &SkillFile{
				Databases: make(map[string]DatabaseAlias),
				Users:     make(map[string]UserAlias),
				Aliases:   make(map[string]CustomAlias),
			}, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse parses a skill file from a reader
func Parse(r io.Reader) (*SkillFile, error) {
	skill := &SkillFile{
		Databases: make(map[string]DatabaseAlias),
		Users:     make(map[string]UserAlias),
		Aliases:   make(map[string]CustomAlias),
	}

	scanner := bufio.NewScanner(r)
	var currentSection string
	var inTable bool
	var headerParsed bool

	for scanner.Scan() {
		line := scanner.Text()

		// Detect section headers
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimPrefix(line, "## ")
			inTable = false
			headerParsed = false
			continue
		}

		// Detect table start
		if strings.HasPrefix(line, "|") && strings.Contains(line, "Alias") {
			inTable = true
			headerParsed = false
			continue
		}

		// Skip separator line
		if inTable && strings.Contains(line, "---") {
			headerParsed = true
			continue
		}

		// Parse table rows
		if inTable && headerParsed && strings.HasPrefix(line, "|") {
			cells := parseTableRow(line)
			if len(cells) < 3 {
				continue
			}

			switch currentSection {
			case "Databases":
				if len(cells) >= 5 {
					alias := cells[0]
					skill.Databases[alias] = DatabaseAlias{
						Alias:         alias,
						Name:          cells[1],
						ID:            cells[2],
						TitleProperty: cells[3],
						DefaultStatus: cells[4],
					}
				}
			case "Users":
				alias := cells[0]
				skill.Users[alias] = UserAlias{
					Alias: alias,
					Name:  cells[1],
					ID:    cells[2],
				}
			case "Custom Aliases":
				alias := cells[0]
				skill.Aliases[alias] = CustomAlias{
					Alias:    alias,
					Type:     cells[1],
					TargetID: cells[2],
				}
			}
		}

		// End of table on empty line
		if inTable && strings.TrimSpace(line) == "" {
			inTable = false
		}
	}

	return skill, scanner.Err()
}

// parseTableRow extracts cells from a markdown table row
func parseTableRow(line string) []string {
	// Remove leading/trailing pipes and split
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// ResolveDatabase resolves an alias or ID to a database ID
func (s *SkillFile) ResolveDatabase(aliasOrID string) (string, bool) {
	// Check if it's an alias
	if db, ok := s.Databases[aliasOrID]; ok {
		return db.ID, true
	}
	// Check if it's already a UUID-like string
	if looksLikeUUID(aliasOrID) {
		return aliasOrID, true
	}
	return "", false
}

// ResolveUser resolves an alias or ID to a user ID
func (s *SkillFile) ResolveUser(aliasOrID string) (string, bool) {
	if user, ok := s.Users[aliasOrID]; ok {
		return user.ID, true
	}
	if looksLikeUUID(aliasOrID) {
		return aliasOrID, true
	}
	return "", false
}

// ResolveAlias resolves a custom alias to its target ID
func (s *SkillFile) ResolveAlias(alias string) (string, string, bool) {
	if a, ok := s.Aliases[alias]; ok {
		return a.TargetID, a.Type, true
	}
	return "", "", false
}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`)

func looksLikeUUID(s string) bool {
	return uuidRegex.MatchString(strings.ToLower(s))
}

const skillTemplate = `---
name: notion-cli
description: "Use when interacting with Notion via the CLI - contains database aliases, user mappings, and shortcuts"
---

# Notion CLI

## Databases

| Alias | Name | ID | Title Property | Default Status |
|-------|------|-----|----------------|----------------|
{{- range .SortedDatabases }}
| {{ .Alias }} | {{ .Name }} | {{ .ID }} | {{ .TitleProperty }} | {{ .DefaultStatus }} |
{{- end }}

## Users

| Alias | Name | ID |
|-------|------|-----|
{{- range .SortedUsers }}
| {{ .Alias }} | {{ .Name }} | {{ .ID }} |
{{- end }}

## Custom Aliases

| Alias | Type | Target ID |
|-------|------|-----------|
{{- range .SortedAliases }}
| {{ .Alias }} | {{ .Type }} | {{ .TargetID }} |
{{- end }}

## Quick Reference

| Operation | Command |
|-----------|---------|
| Create page (fast) | ` + "`ntn create \"My new page\"`" + ` |
| Create page (explicit) | ` + "`ntn p c --ds <database-alias-or-id> --title \"...\" --status \"...\"`" + ` |
| Query database | ` + "`ntn db q <database-alias-or-id> --all --io`" + ` |
| Query a title quickly | ` + "`ntn db q <database-alias-or-id> --i -j --jq '.rs[0].pr.Name.t[0].p'`" + ` |
| Add comment | ` + "`ntn c a <page-id-or-name> \"Looks great\"`" + ` |
`

// Write writes the skill file to a writer
func (s *SkillFile) Write(w io.Writer) error {
	tmpl, err := template.New("skill").Parse(skillTemplate)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	data := struct {
		SortedDatabases []DatabaseAlias
		SortedUsers     []UserAlias
		SortedAliases   []CustomAlias
	}{
		SortedDatabases: s.sortedDatabases(),
		SortedUsers:     s.sortedUsers(),
		SortedAliases:   s.sortedAliases(),
	}

	return tmpl.Execute(w, data)
}

// Save saves the skill file to the default path
func (s *SkillFile) Save() error {
	path := DefaultPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create skill file: %w", err)
	}
	defer func() { _ = f.Close() }()

	return s.Write(f)
}

func (s *SkillFile) sortedDatabases() []DatabaseAlias {
	dbs := make([]DatabaseAlias, 0, len(s.Databases))
	for _, db := range s.Databases {
		dbs = append(dbs, db)
	}
	sort.Slice(dbs, func(i, j int) bool {
		return dbs[i].Alias < dbs[j].Alias
	})
	return dbs
}

func (s *SkillFile) sortedUsers() []UserAlias {
	users := make([]UserAlias, 0, len(s.Users))
	for _, u := range s.Users {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Alias < users[j].Alias
	})
	return users
}

func (s *SkillFile) sortedAliases() []CustomAlias {
	aliases := make([]CustomAlias, 0, len(s.Aliases))
	for _, a := range s.Aliases {
		aliases = append(aliases, a)
	}
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Alias < aliases[j].Alias
	})
	return aliases
}
