package notion

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitInfo contains rate limit information from the API response
type RateLimitInfo struct {
	// Remaining requests in current window
	Remaining int
	// Total requests allowed per window
	Limit int
	// Time when the rate limit window resets
	ResetAt time.Time
	// Request ID for debugging
	RequestID string
	// Last updated timestamp
	UpdatedAt time.Time
}

// RateLimitTracker tracks rate limit information from API responses
type RateLimitTracker struct {
	mu   sync.RWMutex
	info *RateLimitInfo
}

// NewRateLimitTracker creates a new rate limit tracker
func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{}
}

// Update updates rate limit info from HTTP response headers
// Notion API headers:
//   - X-RateLimit-Limit: requests per window
//   - X-RateLimit-Remaining: remaining requests
//   - X-RateLimit-Reset: timestamp when window resets
//   - X-Request-Id: unique request identifier
func (t *RateLimitTracker) Update(resp *http.Response) {
	if resp == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	info := &RateLimitInfo{
		RequestID: resp.Header.Get("X-Request-Id"),
		UpdatedAt: time.Now(),
	}

	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		info.Limit, _ = strconv.Atoi(limit)
	}

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		info.Remaining, _ = strconv.Atoi(remaining)
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			info.ResetAt = time.Unix(ts, 0)
		}
	}

	t.info = info
}

// Get returns the current rate limit info (may be nil if no requests made)
func (t *RateLimitTracker) Get() *RateLimitInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.info == nil {
		return nil
	}
	// Return a copy
	info := *t.info
	return &info
}

// IsLow returns true if remaining requests are below threshold (e.g., 10%)
func (t *RateLimitTracker) IsLow() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.info == nil || t.info.Limit == 0 {
		return false
	}
	return float64(t.info.Remaining)/float64(t.info.Limit) < 0.1
}
