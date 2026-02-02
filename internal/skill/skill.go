// Package skill provides functionality for parsing and resolving aliases
// from Claude skill files. Skill files are markdown documents that contain
// tables mapping human-friendly aliases to Notion IDs.
package skill

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
