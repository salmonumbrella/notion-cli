package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestHelpText_LightExamplesMapToSupportedCommands(t *testing.T) {
	lines := strings.Split(rootHelpText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "ntn ") || !strings.Contains(trimmed, "--li") {
			continue
		}

		command, err := helpExampleCommandPath(trimmed)
		if err != nil {
			t.Fatalf("failed to parse help line %q: %v", trimmed, err)
		}
		if _, ok := lookupLightSchema(command); !ok {
			t.Fatalf("help line advertises --li for unsupported command %q: %s", command, trimmed)
		}
	}
}

func helpExampleCommandPath(line string) (string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "ntn" {
		return "", fmt.Errorf("not a command line")
	}

	cmd := canonicalTopLevel(fields[1])
	if cmd == "" {
		return "", fmt.Errorf("unknown top-level command %q", fields[1])
	}

	if len(fields) > 2 && !strings.HasPrefix(fields[2], "-") {
		if sub := canonicalSubcommand(fields[2]); sub != "" {
			return cmd + " " + sub, nil
		}
	}

	return cmd, nil
}

func canonicalTopLevel(token string) string {
	switch token {
	case "list", "ls":
		return "list"
	case "search", "s", "find", "q":
		return "search"
	case "page", "p", "pages":
		return "page"
	case "db", "database", "databases":
		return "db"
	case "datasource", "ds":
		return "datasource"
	case "comment", "comments", "c":
		return "comment"
	case "file", "files", "f":
		return "file"
	case "user", "users", "u":
		return "user"
	default:
		return ""
	}
}

func canonicalSubcommand(token string) string {
	switch token {
	case "list", "ls":
		return "list"
	case "get", "g":
		return "get"
	default:
		return ""
	}
}
