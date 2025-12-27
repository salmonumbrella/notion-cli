package testhygiene

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	emailPattern                  = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	allowedFilesWithNotionDomains = map[string]struct{}{
		"internal/cmdutil/id_normalize_test.go": {},
		"internal/notion/url_fuzz_test.go":      {},
		"internal/notion/url_bench_test.go":     {},
	}
	disallowedFixturePhrases = []string{
		"ace canada express",
		"invoice 33922",
		"airwallex",
	}
)

func TestFixtureHygiene_NoIdentifyingContent(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var findings []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			base := filepath.Base(path)
			switch base {
			case ".git", ".idea", ".vscode", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}

		if !shouldScanFixtureFile(rel) {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		lower := strings.ToLower(content)

		if strings.Contains(lower, "notion.so") || strings.Contains(lower, "notion.site") {
			if _, ok := allowedFilesWithNotionDomains[rel]; !ok {
				findings = append(findings, fmt.Sprintf("%s: contains notion.so/notion.site; use example.invalid fixture URLs", rel))
			}
		}

		for _, phrase := range disallowedFixturePhrases {
			if strings.Contains(lower, phrase) {
				findings = append(findings, fmt.Sprintf("%s: contains disallowed identifying phrase %q", rel, phrase))
			}
		}

		for _, email := range emailPattern.FindAllString(content, -1) {
			domain := emailDomain(email)
			if !isAllowedFixtureEmailDomain(domain) {
				findings = append(findings, fmt.Sprintf("%s: contains non-synthetic email %q", rel, email))
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("fixture hygiene scan failed: %v", err)
	}

	if len(findings) > 0 {
		t.Fatalf("fixture hygiene violations:\n%s", strings.Join(findings, "\n"))
	}
}

func shouldScanFixtureFile(rel string) bool {
	if strings.HasPrefix(rel, "internal/testhygiene/") {
		return false
	}
	if strings.HasSuffix(rel, "_test.go") {
		return true
	}
	if strings.Contains(rel, "/testdata/") {
		return true
	}
	return false
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %q", dir)
		}
		dir = parent
	}
}

func emailDomain(email string) string {
	parts := strings.SplitN(strings.ToLower(email), "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func isAllowedFixtureEmailDomain(domain string) bool {
	switch domain {
	case "example.com", "example.org", "example.net", "example.test", "example.invalid", "localhost", "test.local":
		return true
	default:
		return strings.HasSuffix(domain, ".example.invalid")
	}
}
