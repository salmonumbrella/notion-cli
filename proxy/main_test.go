package main

import (
	"net/url"
	"os"
	"testing"
	"time"
)

func TestIsLocalhostURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{"localhost", "http://localhost:8080/callback", true},
		{"localhost no port", "http://localhost/callback", true},
		{"127.0.0.1", "http://127.0.0.1:8080/callback", true},
		{"127.0.0.1 no port", "http://127.0.0.1/callback", true},
		{"ipv6 localhost", "http://[::1]:8080/callback", true},
		{"external domain", "http://evil.com/callback", false},
		{"external with localhost in path", "http://evil.com/localhost/callback", false},
		{"localhost subdomain attack", "http://localhost.evil.com/callback", false},
		{"empty host", "http:///callback", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.uri)
			if err != nil {
				t.Fatalf("failed to parse URI: %v", err)
			}
			result := isLocalhostURI(u)
			if result != tt.expected {
				t.Errorf("isLocalhostURI(%q) = %v, want %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestSignAndVerifyState(t *testing.T) {
	// Set a known signing key for testing
	_ = os.Setenv("CLIENT_SECRET", "test-secret-key")
	defer func() { _ = os.Unsetenv("CLIENT_SECRET") }()

	clientState := "abc123"
	redirectURI := "http://localhost:12345/callback"

	// Sign a state
	signed := signState(clientState, redirectURI)
	if signed == "" {
		t.Fatal("signState returned empty string")
	}

	// Verify it works
	verified, err := verifyState(signed)
	if err != nil {
		t.Fatalf("verifyState failed: %v", err)
	}

	if verified.ClientState != clientState {
		t.Errorf("ClientState = %q, want %q", verified.ClientState, clientState)
	}
	if verified.RedirectURI != redirectURI {
		t.Errorf("RedirectURI = %q, want %q", verified.RedirectURI, redirectURI)
	}
}

func TestVerifyState_InvalidSignature(t *testing.T) {
	_ = os.Setenv("CLIENT_SECRET", "test-secret-key")
	defer func() { _ = os.Unsetenv("CLIENT_SECRET") }()

	// Sign with one key
	signed := signState("state1", "http://localhost:8080/callback")

	// Change key and try to verify
	_ = os.Setenv("CLIENT_SECRET", "different-key")
	_, err := verifyState(signed)
	if err == nil {
		t.Error("verifyState should fail with different signing key")
	}
}

func TestVerifyState_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{"empty", ""},
		{"not base64", "!!!invalid!!!"},
		{"valid base64 but not json", "aGVsbG8gd29ybGQ="},    // "hello world"
		{"json but wrong structure", "eyJmb28iOiJiYXIifQ=="}, // {"foo":"bar"}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := verifyState(tt.state)
			if err == nil {
				t.Error("verifyState should fail for invalid input")
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    3,
		window:   time.Minute,
	}

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.allow(ip) {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   100 * time.Millisecond,
	}

	ip := "192.168.1.1"

	// Use up the limit
	limiter.allow(ip)
	limiter.allow(ip)

	// Should be blocked
	if limiter.allow(ip) {
		t.Error("should be blocked at limit")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.allow(ip) {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    10,
		window:   50 * time.Millisecond,
	}

	// Add some old requests
	oldTime := time.Now().Add(-time.Hour)
	limiter.requests["old-ip"] = []time.Time{oldTime}
	limiter.requests["empty-ip"] = []time.Time{}

	// Add a recent request
	limiter.requests["recent-ip"] = []time.Time{time.Now()}

	// Run cleanup
	limiter.cleanup()

	// Old and empty IPs should be removed
	if _, exists := limiter.requests["old-ip"]; exists {
		t.Error("old-ip should be cleaned up")
	}
	if _, exists := limiter.requests["empty-ip"]; exists {
		t.Error("empty-ip should be cleaned up")
	}

	// Recent IP should remain
	if _, exists := limiter.requests["recent-ip"]; !exists {
		t.Error("recent-ip should not be cleaned up")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	limiter := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    1,
		window:   time.Minute,
	}

	// Each IP gets its own limit
	if !limiter.allow("ip1") {
		t.Error("ip1 first request should be allowed")
	}
	if !limiter.allow("ip2") {
		t.Error("ip2 first request should be allowed")
	}

	// Both should now be at limit
	if limiter.allow("ip1") {
		t.Error("ip1 should be at limit")
	}
	if limiter.allow("ip2") {
		t.Error("ip2 should be at limit")
	}
}
