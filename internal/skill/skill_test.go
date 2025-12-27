package skill

import (
	"strings"
	"testing"
)

func TestParseSkillFile(t *testing.T) {
	content := `# Notion CLI

## Databases

| Alias | Name | ID | Title Property | Default Status |
|-------|------|-----|----------------|----------------|
| issues | Issue Tracker | 1d4eaecc-f764-8195-a931-000bd5878943 | Title | Todo |
| projects | Projects | 1d5eaecc-f76480b2acaee35061bff442 | Project | Not Started |

## Users

| Alias | Name | ID |
|-------|------|-----|
| me | Alice Smith | abc123 |
| bob | Bob Jones | def456 |

## Custom Aliases

| Alias | Type | Target ID |
|-------|------|-----------|
| standup | page | 324acbe8da524a7a9c7e58cd2ab2b522 |
`
	skill, err := Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check databases
	if len(skill.Databases) != 2 {
		t.Errorf("expected 2 databases, got %d", len(skill.Databases))
	}
	if skill.Databases["issues"].ID != "1d4eaecc-f764-8195-a931-000bd5878943" {
		t.Errorf("issues ID mismatch")
	}
	if skill.Databases["issues"].TitleProperty != "Title" {
		t.Errorf("issues TitleProperty mismatch")
	}

	// Check users
	if len(skill.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(skill.Users))
	}
	if skill.Users["me"].ID != "abc123" {
		t.Errorf("me user ID mismatch")
	}

	// Check custom aliases
	if len(skill.Aliases) != 1 {
		t.Errorf("expected 1 alias, got %d", len(skill.Aliases))
	}
	if skill.Aliases["standup"].TargetID != "324acbe8da524a7a9c7e58cd2ab2b522" {
		t.Errorf("standup alias ID mismatch")
	}
}

func TestResolveDatabaseAlias(t *testing.T) {
	tests := []struct {
		name      string
		aliasOrID string
		wantID    string
		wantFound bool
	}{
		{
			name:      "resolve existing alias",
			aliasOrID: "issues",
			wantID:    "1d4eaecc-f764-8195-a931-000bd5878943",
			wantFound: true,
		},
		{
			name:      "pass through UUID with dashes",
			aliasOrID: "12345678-1234-1234-1234-123456789abc",
			wantID:    "12345678-1234-1234-1234-123456789abc",
			wantFound: true,
		},
		{
			name:      "pass through UUID without dashes",
			aliasOrID: "12345678123412341234123456789abc",
			wantID:    "12345678123412341234123456789abc",
			wantFound: true,
		},
		{
			name:      "non-existent alias",
			aliasOrID: "nonexistent",
			wantID:    "",
			wantFound: false,
		},
	}

	skill := &SkillFile{
		Databases: map[string]DatabaseAlias{
			"issues": {
				Alias: "issues",
				Name:  "Issue Tracker",
				ID:    "1d4eaecc-f764-8195-a931-000bd5878943",
			},
		},
		Users:   make(map[string]UserAlias),
		Aliases: make(map[string]CustomAlias),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotFound := skill.ResolveDatabase(tt.aliasOrID)
			if gotFound != tt.wantFound {
				t.Errorf("ResolveDatabase() found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveDatabase() id = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResolveUserAlias(t *testing.T) {
	tests := []struct {
		name      string
		aliasOrID string
		wantID    string
		wantFound bool
	}{
		{
			name:      "resolve existing alias",
			aliasOrID: "me",
			wantID:    "abc123",
			wantFound: true,
		},
		{
			name:      "pass through UUID",
			aliasOrID: "12345678-1234-1234-1234-123456789abc",
			wantID:    "12345678-1234-1234-1234-123456789abc",
			wantFound: true,
		},
		{
			name:      "non-existent alias",
			aliasOrID: "unknown",
			wantID:    "",
			wantFound: false,
		},
	}

	skill := &SkillFile{
		Databases: make(map[string]DatabaseAlias),
		Users: map[string]UserAlias{
			"me": {
				Alias: "me",
				Name:  "Alice Smith",
				ID:    "abc123",
			},
		},
		Aliases: make(map[string]CustomAlias),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotFound := skill.ResolveUser(tt.aliasOrID)
			if gotFound != tt.wantFound {
				t.Errorf("ResolveUser() found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveUser() id = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResolveCustomAlias(t *testing.T) {
	tests := []struct {
		name      string
		alias     string
		wantID    string
		wantType  string
		wantFound bool
	}{
		{
			name:      "resolve existing alias",
			alias:     "standup",
			wantID:    "324acbe8da524a7a9c7e58cd2ab2b522",
			wantType:  "page",
			wantFound: true,
		},
		{
			name:      "non-existent alias",
			alias:     "unknown",
			wantID:    "",
			wantType:  "",
			wantFound: false,
		},
	}

	skill := &SkillFile{
		Databases: make(map[string]DatabaseAlias),
		Users:     make(map[string]UserAlias),
		Aliases: map[string]CustomAlias{
			"standup": {
				Alias:    "standup",
				Type:     "page",
				TargetID: "324acbe8da524a7a9c7e58cd2ab2b522",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotType, gotFound := skill.ResolveAlias(tt.alias)
			if gotFound != tt.wantFound {
				t.Errorf("ResolveAlias() found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveAlias() id = %v, want %v", gotID, tt.wantID)
			}
			if gotType != tt.wantType {
				t.Errorf("ResolveAlias() type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"12345678-1234-1234-1234-123456789abc", true},
		{"12345678123412341234123456789abc", true},
		{"12345678-1234-1234-1234-123456789ABC", true}, // uppercase
		{"1d4eaecc-f764-8195-a931-000bd5878943", true},
		{"1d5eaeccf76480b2acaee35061bff442", true},
		{"issues", false},
		{"me", false},
		{"short", false},
		{"", false},
		{"12345678-1234-1234-1234-12345678", false}, // too short
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeUUID(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseEmptyFile(t *testing.T) {
	content := ""
	skill, err := Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(skill.Databases) != 0 {
		t.Errorf("expected 0 databases, got %d", len(skill.Databases))
	}
	if len(skill.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(skill.Users))
	}
	if len(skill.Aliases) != 0 {
		t.Errorf("expected 0 aliases, got %d", len(skill.Aliases))
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "basic row",
			input: "| a | b | c |",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "row with spaces",
			input: "|  spaced  |  values  |",
			want:  []string{"spaced", "values"},
		},
		{
			name:  "row with empty cells",
			input: "| a |  | c |",
			want:  []string{"a", "", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTableRow(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseTableRow() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseTableRow()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestWriteSkillFile(t *testing.T) {
	skill := &SkillFile{
		Databases: map[string]DatabaseAlias{
			"issues": {
				Alias:         "issues",
				Name:          "Issue Tracker",
				ID:            "abc123",
				TitleProperty: "Title",
				DefaultStatus: "Todo",
			},
		},
		Users: map[string]UserAlias{
			"me": {Alias: "me", Name: "Test User", ID: "user123"},
		},
		Aliases: map[string]CustomAlias{
			"standup": {Alias: "standup", Type: "page", TargetID: "page123"},
		},
	}

	var buf strings.Builder
	err := skill.Write(&buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()

	// Verify structure
	if !strings.Contains(output, "## Databases") {
		t.Error("missing Databases section")
	}
	if !strings.Contains(output, "| issues |") {
		t.Error("missing issues row")
	}
	if !strings.Contains(output, "## Users") {
		t.Error("missing Users section")
	}
	if !strings.Contains(output, "## Custom Aliases") {
		t.Error("missing Custom Aliases section")
	}

	// Verify it can be round-tripped
	parsed, err := Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Round-trip parse failed: %v", err)
	}
	if parsed.Databases["issues"].ID != "abc123" {
		t.Error("round-trip failed for database ID")
	}
}
