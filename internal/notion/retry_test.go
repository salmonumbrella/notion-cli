package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoRequest_RetryOn429(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Return 429 for first two attempts
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  429,
				Code:    "rate_limited",
				Message: "Rate limit exceeded.",
			})
			return
		}
		// Success on third attempt
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	start := time.Now()
	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	// Should have waited at least 1s + 2s = 3s (minus jitter tolerance)
	if elapsed < 2500*time.Millisecond {
		t.Errorf("expected at least 2.5s delay from retries, got %v", elapsed)
	}
}

func TestDoRequest_RetryOn500(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 2 {
			// Return 500 for first attempt
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  500,
				Code:    "internal_server_error",
				Message: "Internal server error.",
			})
			return
		}
		// Success on second attempt
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attemptCount != 2 {
		t.Errorf("expected 2 attempts, got %d", attemptCount)
	}
}

func TestDoRequest_RetryOn502(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  502,
				Code:    "bad_gateway",
				Message: "Bad gateway.",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attemptCount != 2 {
		t.Errorf("expected 2 attempts, got %d", attemptCount)
	}
}

func TestDoRequest_NoRetryOn404(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "Page not found.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

	if err == nil {
		t.Fatal("expected error for 404")
	}

	if attemptCount != 1 {
		t.Errorf("expected only 1 attempt (no retry), got %d", attemptCount)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestDoRequest_ExhaustedRetries(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  429,
			Code:    "rate_limited",
			Message: "Rate limit exceeded.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	// Should have made 4 attempts (initial + 3 retries)
	if attemptCount != 4 {
		t.Errorf("expected 4 attempts (initial + 3 retries), got %d", attemptCount)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
}

func TestDoRequest_ContextCancellationDuringRetry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  429,
			Code:    "rate_limited",
			Message: "Rate limit exceeded.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}

	// Should have made at least 1 attempt but not all 4
	if attemptCount == 0 {
		t.Error("expected at least 1 attempt")
	}
	if attemptCount >= 4 {
		t.Errorf("expected fewer than 4 attempts due to cancellation, got %d", attemptCount)
	}
}

func TestDoRequest_RetryAfterHeader(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  429,
				Code:    "rate_limited",
				Message: "Rate limit exceeded.",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	start := time.Now()
	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attemptCount != 2 {
		t.Errorf("expected 2 attempts, got %d", attemptCount)
	}

	// Should have waited at least 2 seconds as specified in Retry-After header
	if elapsed < 2*time.Second {
		t.Errorf("expected at least 2s delay from Retry-After header, got %v", elapsed)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			got := isRetryable(tt.statusCode)
			if got != tt.want {
				t.Errorf("isRetryable(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{
			name:       "empty string",
			retryAfter: "",
			want:       0,
		},
		{
			name:       "seconds as integer",
			retryAfter: "5",
			want:       5 * time.Second,
		},
		{
			name:       "large integer",
			retryAfter: "120",
			want:       120 * time.Second,
		},
		{
			name:       "invalid string",
			retryAfter: "invalid",
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.retryAfter)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.retryAfter, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Use UTC to avoid timezone issues
	futureTime := time.Now().UTC().Add(5 * time.Second)
	retryAfter := futureTime.Format(http.TimeFormat)

	got := parseRetryAfter(retryAfter)

	// Allow 1 second tolerance for timing variations
	if got < 4*time.Second || got > 6*time.Second {
		t.Errorf("parseRetryAfter(%q) = %v, want ~5s", retryAfter, got)
	}
}

func TestCalculateRetryDelay(t *testing.T) {
	client := NewClient("test-token")

	t.Run("exponential backoff without error", func(t *testing.T) {
		// Attempt 1: base delay (1s) * 2^0 = 1s + jitter
		delay1 := client.calculateRetryDelay(1, nil)
		if delay1 < 1*time.Second || delay1 > 1500*time.Millisecond {
			t.Errorf("attempt 1 delay should be ~1s, got %v", delay1)
		}

		// Attempt 2: base delay (1s) * 2^1 = 2s + jitter
		delay2 := client.calculateRetryDelay(2, nil)
		if delay2 < 2*time.Second || delay2 > 2500*time.Millisecond {
			t.Errorf("attempt 2 delay should be ~2s, got %v", delay2)
		}

		// Attempt 3: base delay (1s) * 2^2 = 4s + jitter
		delay3 := client.calculateRetryDelay(3, nil)
		if delay3 < 4*time.Second || delay3 > 5*time.Second {
			t.Errorf("attempt 3 delay should be ~4s, got %v", delay3)
		}
	})

	t.Run("with Retry-After header", func(t *testing.T) {
		apiErr := &APIError{
			StatusCode: 429,
			Response: &ErrorResponse{
				Status:  429,
				Code:    "rate_limited",
				Message: "Rate limited",
			},
			RetryAfter: 10 * time.Second,
		}

		delay := client.calculateRetryDelay(1, apiErr)
		if delay != 10*time.Second {
			t.Errorf("expected 10s from Retry-After, got %v", delay)
		}
	})
}
