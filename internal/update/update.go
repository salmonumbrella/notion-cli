// Package update provides non-blocking update checking for the CLI.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// HTTPDoer abstracts an HTTP client for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Checker encapsulates update check configuration.
type Checker struct {
	httpClient    HTTPDoer
	cacheDir      func() (string, error)
	cachePath     string
	cacheFile     string
	checkInterval time.Duration
	now           func() time.Time
	readFile      func(string) ([]byte, error)
	writeFile     func(string, []byte, os.FileMode) error
	mkdirAll      func(string, os.FileMode) error
	repo          string
	logger        *slog.Logger
}

// Option configures a Checker.
type Option func(*Checker)

// NewChecker creates a Checker with defaults and applies options.
func NewChecker(opts ...Option) *Checker {
	c := &Checker{
		httpClient:    http.DefaultClient,
		cacheDir:      os.UserCacheDir,
		cacheFile:     CacheFile,
		checkInterval: CheckInterval,
		now:           time.Now,
		readFile:      os.ReadFile,
		writeFile:     os.WriteFile,
		mkdirAll:      os.MkdirAll,
		repo:          GitHubRepo,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client HTTPDoer) Option {
	return func(c *Checker) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithCacheDirFunc overrides the cache dir resolver.
func WithCacheDirFunc(fn func() (string, error)) Option {
	return func(c *Checker) {
		if fn != nil {
			c.cacheDir = fn
		}
	}
}

// WithCachePath overrides the full cache path.
func WithCachePath(path string) Option {
	return func(c *Checker) {
		c.cachePath = path
	}
}

// WithNow overrides the clock.
func WithNow(fn func() time.Time) Option {
	return func(c *Checker) {
		if fn != nil {
			c.now = fn
		}
	}
}

// WithCheckInterval overrides the check interval.
func WithCheckInterval(interval time.Duration) Option {
	return func(c *Checker) {
		if interval > 0 {
			c.checkInterval = interval
		}
	}
}

// WithRepo overrides the GitHub repo slug.
func WithRepo(repo string) Option {
	return func(c *Checker) {
		if strings.TrimSpace(repo) != "" {
			c.repo = repo
		}
	}
}

// WithLogger overrides the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Checker) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// UpdateError wraps update-check failures with context.
type UpdateError struct {
	Op  string
	Err error
}

func (e *UpdateError) Error() string {
	return fmt.Sprintf("update check %s: %v", e.Op, e.Err)
}

func (e *UpdateError) Unwrap() error {
	return e.Err
}

// Check checks for updates and returns a message if a newer version exists.
func (c *Checker) Check(ctx context.Context, currentVersion string) (string, error) {
	cachePath, err := c.resolveCachePath()
	if err != nil {
		return "", &UpdateError{Op: "cache path", Err: err}
	}

	cached, err := c.loadCache(cachePath)
	if err != nil {
		return "", &UpdateError{Op: "load cache", Err: err}
	}

	if !c.shouldCheck(cached.LatestVersion, cached.LastCheck) {
		if isNewer(currentVersion, cached.LatestVersion) {
			return updateMessage(cached.LatestVersion, currentVersion, c.repo), nil
		}
		return "", nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	latest, err := c.fetchLatestRelease(ctx)
	if err != nil {
		return "", &UpdateError{Op: "fetch latest release", Err: err}
	}

	saveErr := c.saveCache(cachePath, cache{
		LastCheck:     c.now(),
		LatestVersion: latest,
	})
	if saveErr != nil {
		return updateMessage(latest, currentVersion, c.repo), &UpdateError{Op: "save cache", Err: saveErr}
	}

	if isNewer(currentVersion, latest) {
		return updateMessage(latest, currentVersion, c.repo), nil
	}

	return "", nil
}

// Check checks for updates and returns a message if a newer version exists.
// This is non-blocking and logs failures at debug level.
func Check(ctx context.Context, currentVersion string) string {
	checker := NewChecker()
	msg, err := checker.Check(ctx, currentVersion)
	if err != nil {
		checker.logger.Debug("update check failed", "error", err)
	}
	return msg
}

// CheckWithOptions runs an update check with custom options.
func CheckWithOptions(ctx context.Context, currentVersion string, opts ...Option) (string, error) {
	checker := NewChecker(opts...)
	return checker.Check(ctx, currentVersion)
}

func (c *Checker) resolveCachePath() (string, error) {
	if strings.TrimSpace(c.cachePath) != "" {
		return c.cachePath, nil
	}
	dir, err := c.cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "notion-cli", c.cacheFile), nil
}

func (c *Checker) loadCache(path string) (cache, error) {
	data, err := c.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache{}, nil
		}
		return cache{}, err
	}
	var parsed cache
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cache{}, err
	}
	return parsed, nil
}

func (c *Checker) saveCache(path string, value cache) error {
	if err := c.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.writeFile(path, data, 0o644)
}

func (c *Checker) shouldCheck(cachedVersion string, lastCheck time.Time) bool {
	if cachedVersion == "" || lastCheck.IsZero() {
		return true
	}
	return c.now().Sub(lastCheck) > c.checkInterval
}

func (c *Checker) fetchLatestRelease(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
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

func updateMessage(latest, current, repo string) string {
	return fmt.Sprintf("A new version is available: %s (current: %s)\nRun: go install github.com/%s/cmd/notion@latest", latest, current, repo)
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
