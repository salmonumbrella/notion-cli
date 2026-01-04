package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func resolveJSONInput(raw string, file string) (string, error) {
	if raw != "" && file != "" {
		return "", fmt.Errorf("use only one of inline JSON or --file")
	}

	if file != "" {
		return readInputSource(file)
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "-" {
		return readInputSource("-")
	}
	if strings.HasPrefix(trimmed, "@") {
		path := trimmed[1:]
		return readInputSource(path)
	}

	return raw, nil
}

// readJSONInput resolves a single JSON value with @file and - (stdin) support.
func readJSONInput(value string) (string, error) {
	return resolveJSONInput(value, "")
}

func readInputSource(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("input file path is required")
	}
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}
