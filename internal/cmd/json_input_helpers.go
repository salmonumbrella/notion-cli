package cmd

import (
	"fmt"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
)

func readAndDecodeJSON[T any](input, parseErrPrefix string) (T, string, error) {
	var out T
	resolved, err := cmdutil.ReadJSONInput(input)
	if err != nil {
		return out, "", err
	}
	if err := cmdutil.UnmarshalJSONInput(resolved, &out); err != nil {
		return out, "", fmt.Errorf("%s: %w", parseErrPrefix, err)
	}
	return out, resolved, nil
}

func resolveAndDecodeJSON[T any](inline, file, parseErrPrefix string) (T, string, error) {
	var out T
	resolved, err := cmdutil.ResolveJSONInput(inline, file)
	if err != nil {
		return out, "", err
	}
	if err := cmdutil.UnmarshalJSONInput(resolved, &out); err != nil {
		return out, "", fmt.Errorf("%s: %w", parseErrPrefix, err)
	}
	return out, resolved, nil
}

func hasJSONInput(inline, file string) bool {
	return strings.TrimSpace(inline) != "" || strings.TrimSpace(file) != ""
}
