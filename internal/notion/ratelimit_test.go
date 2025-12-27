package notion

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRateLimitTracker_Update(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantInfo *RateLimitInfo
	}{
		{
			name: "all headers present",
			headers: map[string]string{
				"X-RateLimit-Limit":     "1000",
				"X-RateLimit-Remaining": "750",
				"X-RateLimit-Reset":     "1640000000",
				"X-Request-Id":          "req-123",
			},
			wantInfo: &RateLimitInfo{
				Limit:     1000,
				Remaining: 750,
				ResetAt:   time.Unix(1640000000, 0),
				RequestID: "req-123",
			},
		},
		{
			name: "partial headers",
			headers: map[string]string{
				"X-RateLimit-Limit":     "1000",
				"X-RateLimit-Remaining": "500",
			},
			wantInfo: &RateLimitInfo{
				Limit:     1000,
				Remaining: 500,
			},
		},
		{
			name: "no headers",
			headers: map[string]string{
				"X-Request-Id": "req-456",
			},
			wantInfo: &RateLimitInfo{
				RequestID: "req-456",
			},
		},
		{
			name: "invalid values",
			headers: map[string]string{
				"X-RateLimit-Limit":     "invalid",
				"X-RateLimit-Remaining": "also-invalid",
				"X-RateLimit-Reset":     "not-a-number",
				"X-Request-Id":          "req-789",
			},
			wantInfo: &RateLimitInfo{
				RequestID: "req-789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewRateLimitTracker()

			// Create mock response with headers
			resp := &http.Response{
				Header: make(http.Header),
			}
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			tracker.Update(resp)
			got := tracker.Get()

			if got == nil {
				t.Fatal("Get() returned nil")
			}

			if got.Limit != tt.wantInfo.Limit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantInfo.Limit)
			}
			if got.Remaining != tt.wantInfo.Remaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tt.wantInfo.Remaining)
			}
			if !got.ResetAt.Equal(tt.wantInfo.ResetAt) {
				t.Errorf("ResetAt = %v, want %v", got.ResetAt, tt.wantInfo.ResetAt)
			}
			if got.RequestID != tt.wantInfo.RequestID {
				t.Errorf("RequestID = %s, want %s", got.RequestID, tt.wantInfo.RequestID)
			}
			if got.UpdatedAt.IsZero() {
				t.Error("UpdatedAt should be set")
			}
		})
	}
}

func TestRateLimitTracker_UpdateNilResponse(t *testing.T) {
	tracker := NewRateLimitTracker()
	tracker.Update(nil) // Should not panic

	got := tracker.Get()
	if got != nil {
		t.Error("Get() should return nil when Update() called with nil response")
	}
}

func TestRateLimitTracker_Get(t *testing.T) {
	tracker := NewRateLimitTracker()

	// Before any updates
	if got := tracker.Get(); got != nil {
		t.Error("Get() should return nil before any updates")
	}

	// After update
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("X-RateLimit-Limit", "1000")
	resp.Header.Set("X-RateLimit-Remaining", "750")

	tracker.Update(resp)

	info1 := tracker.Get()
	info2 := tracker.Get()

	// Should return copies, not the same pointer
	if info1 == info2 {
		t.Error("Get() should return a copy, not the same pointer")
	}

	// But values should be equal
	if info1.Limit != info2.Limit || info1.Remaining != info2.Remaining {
		t.Error("Get() copies should have equal values")
	}
}

func TestRateLimitTracker_IsLow(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		remaining int
		wantLow   bool
	}{
		{
			name:      "well above threshold (75%)",
			limit:     1000,
			remaining: 750,
			wantLow:   false,
		},
		{
			name:      "at threshold (10%)",
			limit:     1000,
			remaining: 100,
			wantLow:   false,
		},
		{
			name:      "just below threshold (9.9%)",
			limit:     1000,
			remaining: 99,
			wantLow:   true,
		},
		{
			name:      "very low (1%)",
			limit:     1000,
			remaining: 10,
			wantLow:   true,
		},
		{
			name:      "empty",
			limit:     1000,
			remaining: 0,
			wantLow:   true,
		},
		{
			name:      "no limit set",
			limit:     0,
			remaining: 0,
			wantLow:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewRateLimitTracker()

			if tt.limit > 0 {
				resp := &http.Response{
					Header: make(http.Header),
				}
				resp.Header.Set("X-RateLimit-Limit", string(rune(tt.limit+'0')))
				resp.Header.Set("X-RateLimit-Remaining", string(rune(tt.remaining+'0')))

				// Use proper string conversion
				resp.Header.Set("X-RateLimit-Limit", formatInt(tt.limit))
				resp.Header.Set("X-RateLimit-Remaining", formatInt(tt.remaining))

				tracker.Update(resp)
			}

			got := tracker.IsLow()
			if got != tt.wantLow {
				t.Errorf("IsLow() = %v, want %v", got, tt.wantLow)
			}
		})
	}
}

func TestRateLimitTracker_IsLowBeforeUpdate(t *testing.T) {
	tracker := NewRateLimitTracker()
	if tracker.IsLow() {
		t.Error("IsLow() should return false before any updates")
	}
}

func TestRateLimitTracker_ThreadSafety(t *testing.T) {
	tracker := NewRateLimitTracker()

	var wg sync.WaitGroup
	iterations := 100

	// Spawn multiple goroutines that update and read concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp := &http.Response{
					Header: make(http.Header),
				}
				resp.Header.Set("X-RateLimit-Limit", "1000")
				resp.Header.Set("X-RateLimit-Remaining", formatInt(1000-j))
				resp.Header.Set("X-Request-Id", formatInt(id*iterations+j))

				tracker.Update(resp)
			}
		}(i)
	}

	// Spawn readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tracker.Get()
				tracker.IsLow()
			}
		}()
	}

	wg.Wait()

	// Verify we can still read after concurrent access
	info := tracker.Get()
	if info == nil {
		t.Error("tracker should have info after concurrent updates")
	}
}

func TestRateLimitTracker_MultipleUpdates(t *testing.T) {
	tracker := NewRateLimitTracker()

	// First update
	resp1 := &http.Response{
		Header: make(http.Header),
	}
	resp1.Header.Set("X-RateLimit-Limit", "1000")
	resp1.Header.Set("X-RateLimit-Remaining", "750")
	resp1.Header.Set("X-Request-Id", "req-1")

	tracker.Update(resp1)
	info1 := tracker.Get()

	if info1.Remaining != 750 {
		t.Errorf("First update: Remaining = %d, want 750", info1.Remaining)
	}

	// Second update (simulate another request)
	time.Sleep(10 * time.Millisecond)
	resp2 := &http.Response{
		Header: make(http.Header),
	}
	resp2.Header.Set("X-RateLimit-Limit", "1000")
	resp2.Header.Set("X-RateLimit-Remaining", "749")
	resp2.Header.Set("X-Request-Id", "req-2")

	tracker.Update(resp2)
	info2 := tracker.Get()

	if info2.Remaining != 749 {
		t.Errorf("Second update: Remaining = %d, want 749", info2.Remaining)
	}
	if info2.RequestID != "req-2" {
		t.Errorf("Second update: RequestID = %s, want req-2", info2.RequestID)
	}
	if !info2.UpdatedAt.After(info1.UpdatedAt) {
		t.Error("Second update should have later UpdatedAt timestamp")
	}
}

// formatInt converts an int to a string for header values
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
