package debug

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithDebug(t *testing.T) {
	ctx := context.Background()

	// Test setting debug to true
	ctx = WithDebug(ctx, true)
	if !IsDebug(ctx) {
		t.Error("Expected IsDebug to return true")
	}

	// Test setting debug to false
	ctx = WithDebug(ctx, false)
	if IsDebug(ctx) {
		t.Error("Expected IsDebug to return false")
	}
}

func TestIsDebug_NoValue(t *testing.T) {
	ctx := context.Background()
	if IsDebug(ctx) {
		t.Error("Expected IsDebug to return false for context without debug value")
	}
}

func TestDebugTransport_Request(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret_token_12345678")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify debug output
	output := buf.String()

	// Check for request logging
	if !strings.Contains(output, "--> GET") {
		t.Error("Expected request method and URL in output")
	}

	// Check for token redaction
	if strings.Contains(output, "secret_token_12345678") {
		t.Error("Token should be redacted")
	}
	if !strings.Contains(output, "...5678") {
		t.Error("Expected last 4 characters of token to be shown")
	}

	// Check for response logging
	if !strings.Contains(output, "<-- 200") {
		t.Error("Expected response status in output")
	}

	// Check for response body
	if !strings.Contains(output, `"message": "success"`) {
		t.Error("Expected response body in output")
	}
}

func TestDebugTransport_RequestBody(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request with body
	requestBody := `{"name": "Test User", "email": "test@example.com"}`
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify debug output
	output := buf.String()

	// Check for request body
	if !strings.Contains(output, requestBody) {
		t.Error("Expected request body in output")
	}
}

func TestDebugTransport_LongBody(t *testing.T) {
	// Create a test server that returns a large response
	largeBody := strings.Repeat("x", 2000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify debug output is truncated
	output := buf.String()
	if !strings.Contains(output, "[truncated]") {
		t.Error("Expected large response body to be truncated")
	}
}

func TestDebugTransport_Error(t *testing.T) {
	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request to invalid URL
	req, err := http.NewRequest("GET", "http://invalid.localhost.test:99999", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Fatal("Expected request to fail")
	}

	// Verify debug output shows error
	output := buf.String()
	if !strings.Contains(output, "<-- ERROR:") {
		t.Error("Expected error to be logged in output")
	}
}

func TestNewDebugTransport_Defaults(t *testing.T) {
	// Test with nil transport and nil output
	dt := NewDebugTransport(nil, nil)
	if dt.Transport != http.DefaultTransport {
		t.Error("Expected default transport when nil is passed")
	}
	// We can't directly test if Output is os.Stderr, but we can verify it's not nil
	if dt.Output == nil {
		t.Error("Expected output to be set to os.Stderr when nil is passed")
	}
}

func TestDebugTransport_RateLimitHeaders(t *testing.T) {
	// Create a test server that returns rate limit headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "75")
		w.Header().Set("X-RateLimit-Reset", "1766149200") // Some future timestamp
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify debug output contains rate limit info
	output := buf.String()

	// Check for rate limit display
	if !strings.Contains(output, "Rate-Limit: 75/100 remaining") {
		t.Errorf("Expected rate limit info in output, got: %s", output)
	}
}

func TestDebugTransport_NoRateLimitHeaders(t *testing.T) {
	// Create a test server without rate limit headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create a buffer to capture debug output
	var buf bytes.Buffer

	// Create client with debug transport
	transport := NewDebugTransport(nil, &buf)
	client := &http.Client{Transport: transport}

	// Make request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify debug output does NOT contain rate limit info
	output := buf.String()

	// Should not show rate limit line when headers are absent
	if strings.Contains(output, "Rate-Limit:") {
		t.Errorf("Should not show rate limit info when headers are absent")
	}
}
