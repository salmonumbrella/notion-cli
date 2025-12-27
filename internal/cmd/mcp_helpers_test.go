package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMCPJSONObject_WhitespaceIsError(t *testing.T) {
	_, err := parseMCPJSONObject("   ", "properties")
	if err == nil {
		t.Fatal("expected whitespace JSON to return an error")
	}
	if !strings.Contains(err.Error(), "invalid --properties JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMCPJSONArray_WhitespaceIsError(t *testing.T) {
	_, err := parseMCPJSONArray("   ", "params")
	if err == nil {
		t.Fatal("expected whitespace JSON array to return an error")
	}
	if !strings.Contains(err.Error(), "invalid --params JSON array") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMCPJSONObjectFromInlineOrFile_EmptyFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "filter.json")
	if err := os.WriteFile(path, []byte("   \n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := parseMCPJSONObjectFromInlineOrFile("", path, "filter", "filter-file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil filter for empty/whitespace file, got: %#v", got)
	}
}

func TestParseMCPJSONObjectFromInlineOrFile_MutuallyExclusive(t *testing.T) {
	_, err := parseMCPJSONObjectFromInlineOrFile("{}", "filter.json", "filter", "filter-file")
	if err == nil {
		t.Fatal("expected error when both inline and file are provided")
	}
	if !strings.Contains(err.Error(), "use only one of --filter or --filter-file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
