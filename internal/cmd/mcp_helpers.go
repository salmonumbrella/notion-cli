package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
)

func parseMCPJSONObject(input, flagName string) (map[string]interface{}, error) {
	if input == "" {
		return nil, nil
	}
	out, _, err := readAndDecodeJSON[map[string]interface{}](input, fmt.Sprintf("invalid --%s JSON", flagName))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseMCPJSONArray(input, flagName string) ([]interface{}, error) {
	if input == "" {
		return nil, nil
	}
	out, _, err := readAndDecodeJSON[[]interface{}](input, fmt.Sprintf("invalid --%s JSON array", flagName))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseMCPJSONObjectFromInlineOrFile(inline, filePath, inlineFlag, fileFlag string) (map[string]interface{}, error) {
	inlineSet := strings.TrimSpace(inline) != ""
	fileSet := strings.TrimSpace(filePath) != ""
	if inlineSet && fileSet {
		return nil, fmt.Errorf("use only one of --%s or --%s", inlineFlag, fileFlag)
	}
	if !inlineSet && !fileSet {
		return nil, nil
	}

	resolved, err := cmdutil.ResolveJSONInput(inline, filePath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resolved) == "" {
		// Preserve previous meeting-notes behavior: empty/whitespace input means no filter.
		return nil, nil
	}

	var out map[string]interface{}
	if err := cmdutil.UnmarshalJSONInput(resolved, &out); err != nil {
		return nil, fmt.Errorf("invalid --%s JSON: %w", inlineFlag, err)
	}
	return out, nil
}

func readTextFileForFlag(path, label string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", label, err)
	}
	return string(data), nil
}

func ensureMutuallyExclusiveFlags(flagA string, setA bool, flagB string, setB bool) error {
	if setA && setB {
		return fmt.Errorf("%s and %s are mutually exclusive", flagA, flagB)
	}
	return nil
}
