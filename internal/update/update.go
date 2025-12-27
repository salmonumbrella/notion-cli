// Package update provides non-blocking update checking for the CLI.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// CheckInterval is the minimum time between update checks.
	CheckInterval = 24 * time.Hour
	// GitHubRepo is the repository to check for releases.
	GitHubRepo = "salmonumbrella/notion-cli"
	// CacheFile stores the last check time and version.
	CacheFile = "update-check.json"
)

type cache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

// Check checks for updates and returns a message if a newer version exists.
// This is non-blocking and fails silently on errors.
func Check(ctx context.Context, currentVersion string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	cachePath := filepath.Join(cacheDir, "notion-cli", CacheFile)
	cached := loadCache(cachePath)

	if !shouldCheck(cached.LatestVersion, cached.LastCheck) {
		// Use cached result
		if isNewer(currentVersion, cached.LatestVersion) {
			return fmt.Sprintf("A new version is available: %s (current: %s)\nRun: go install github.com/%s/cmd/notion@latest", cached.LatestVersion, currentVersion, GitHubRepo)
		}
		return ""
	}

	// Fetch latest release (with timeout)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	latest, err := fetchLatestRelease(ctx)
	if err != nil {
		return ""
	}

	// Update cache
	saveCache(cachePath, cache{
		LastCheck:     time.Now(),
		LatestVersion: latest,
	})

	if isNewer(currentVersion, latest) {
		return fmt.Sprintf("A new version is available: %s (current: %s)\nRun: go install github.com/%s/cmd/notion@latest", latest, currentVersion, GitHubRepo)
	}

	return ""
}

func loadCache(path string) cache {
	data, err := os.ReadFile(path)
	if err != nil {
		return cache{}
	}
	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return cache{}
	}
	return c
}

func saveCache(path string, c cache) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644) // Ignore error - best effort caching
}

func shouldCheck(cachedVersion string, lastCheck time.Time) bool {
	if cachedVersion == "" || lastCheck.IsZero() {
		return true
	}
	return time.Since(lastCheck) > CheckInterval
}

func fetchLatestRelease(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return parseVersion(release.TagName), nil
}

func parseVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

func isNewer(current, latest string) bool {
	// Skip check for dev versions
	if current == "dev" || current == "unknown" || current == "" {
		return false
	}

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := 0; i < len(currentParts) && i < len(latestParts); i++ {
		c, _ := strconv.Atoi(currentParts[i])
		l, _ := strconv.Atoi(latestParts[i])
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}
