package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	token := "test-token"
	client := NewClient(token)

	if client.token != token {
		t.Errorf("expected token %q, got %q", token, client.token)
	}

	if client.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", defaultBaseURL, client.baseURL)
	}

	if client.version != apiVersion {
		t.Errorf("expected version %q, got %q", apiVersion, client.version)
	}

	if client.httpClient == nil {
		t.Error("expected httpClient to be set")
	}

	if client.httpClient.Timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, client.httpClient.Timeout)
	}
}

func TestWithHTTPClient(t *testing.T) {
	client := NewClient("token")
	customHTTP := &http.Client{Timeout: 5 * time.Second}

	result := client.WithHTTPClient(customHTTP)

	if result != client {
		t.Error("expected WithHTTPClient to return same client for chaining")
	}

	if client.httpClient != customHTTP {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestWithBaseURL(t *testing.T) {
	client := NewClient("token")
	customURL := "https://custom.api.com"

	result := client.WithBaseURL(customURL)

	if result != client {
		t.Error("expected WithBaseURL to return same client for chaining")
	}

	if client.baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, client.baseURL)
	}
}

func TestDoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", auth)
		}

		if version := r.Header.Get("Notion-Version"); version != apiVersion {
			t.Errorf("expected Notion-Version header %q, got %q", apiVersion, version)
		}

		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDoRequest_NoAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}

		if version := r.Header.Get("Notion-Version"); version != apiVersion {
			t.Errorf("expected Notion-Version header %q, got %q", apiVersion, version)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL).WithAuthHeaderDisabled()
	ctx := context.Background()

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDoRequest_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body["key"] != "value" {
			t.Errorf("expected body key=value, got %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	requestBody := map[string]string{"key": "value"}
	resp, err := client.doRequest(ctx, http.MethodPost, "/test", requestBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestDoRequest_ErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   ErrorResponse
		expectError    bool
		expectedStatus int
		expectedCode   string
	}{
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			responseBody: ErrorResponse{
				Object:  "error",
				Status:  401,
				Code:    "unauthorized",
				Message: "API token is invalid.",
			},
			expectError:    true,
			expectedStatus: 401,
			expectedCode:   "unauthorized",
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			responseBody: ErrorResponse{
				Object:  "error",
				Status:  404,
				Code:    "object_not_found",
				Message: "Could not find page with ID.",
			},
			expectError:    true,
			expectedStatus: 404,
			expectedCode:   "object_not_found",
		},
		{
			name:       "429 Rate Limited",
			statusCode: http.StatusTooManyRequests,
			responseBody: ErrorResponse{
				Object:  "error",
				Status:  429,
				Code:    "rate_limited",
				Message: "Rate limit exceeded.",
			},
			expectError:    true,
			expectedStatus: 429,
			expectedCode:   "rate_limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient("test-token").WithBaseURL(server.URL)
			ctx := context.Background()

			_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

			if !tt.expectError {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected error to wrap *APIError, got %T", err)
			}

			if apiErr.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, apiErr.StatusCode)
			}

			if apiErr.Response == nil {
				t.Fatal("expected error response to be set")
			}

			if apiErr.Response.Code != tt.expectedCode {
				t.Errorf("expected code %q, got %q", tt.expectedCode, apiErr.Response.Code)
			}

			// Verify error message format
			expectedMsg := tt.responseBody.Error()
			if apiErr.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, apiErr.Error())
			}
		})
	}
}

func TestDoRequest_MalformedErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, apiErr.StatusCode)
	}

	if apiErr.Response != nil {
		t.Error("expected Response to be nil for malformed error")
	}
}

func TestDoGet(t *testing.T) {
	type testResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(testResponse{
			ID:   "123",
			Name: "Test",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	var result testResponse
	err := client.doGet(ctx, "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "123" {
		t.Errorf("expected ID '123', got %q", result.ID)
	}

	if result.Name != "Test" {
		t.Errorf("expected Name 'Test', got %q", result.Name)
	}
}

func TestDoGet_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		// Verify query parameters
		if r.URL.Query().Get("page_size") != "50" {
			t.Errorf("expected page_size=50, got %s", r.URL.Query().Get("page_size"))
		}
		if r.URL.Query().Get("start_cursor") != "abc123" {
			t.Errorf("expected start_cursor=abc123, got %s", r.URL.Query().Get("start_cursor"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	query := url.Values{}
	query.Set("page_size", "50")
	query.Set("start_cursor", "abc123")

	var result map[string]string
	err := client.doGet(ctx, "/test", query, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", result["status"])
	}
}

func TestDoPost(t *testing.T) {
	type testRequest struct {
		Title string `json:"title"`
	}

	type testResponse struct {
		ID string `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var req testRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Title != "Test Title" {
			t.Errorf("expected title 'Test Title', got %q", req.Title)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(testResponse{ID: "456"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	request := testRequest{Title: "Test Title"}
	var result testResponse

	err := client.doPost(ctx, "/test", request, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "456" {
		t.Errorf("expected ID '456', got %q", result.ID)
	}
}

func TestDoPatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	var result map[string]string
	err := client.doPatch(ctx, "/test", map[string]string{"key": "value"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", result["status"])
	}
}

func TestDoDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"archived": true})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	var result map[string]bool
	err := client.doDelete(ctx, "/test", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result["archived"] {
		t.Error("expected archived to be true")
	}
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)

	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context cancellation error, got %v", err)
	}
}

func TestDoRequest_InvalidJSON(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	// Try to marshal an invalid type (channels can't be marshaled)
	invalidBody := make(chan int)

	_, err := client.doRequest(ctx, http.MethodPost, "/test", invalidBody)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to marshal request body") {
		t.Errorf("expected marshal error, got %v", err)
	}
}

func TestErrorResponse_Error(t *testing.T) {
	err := ErrorResponse{
		Object:  "error",
		Status:  401,
		Code:    "unauthorized",
		Message: "Invalid token",
	}

	expected := "notion API error 401 (unauthorized): Invalid token"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestAPIError_Error(t *testing.T) {
	t.Run("with response", func(t *testing.T) {
		apiErr := &APIError{
			StatusCode: 404,
			Response: &ErrorResponse{
				Object:  "error",
				Status:  404,
				Code:    "object_not_found",
				Message: "Page not found",
			},
		}

		expected := "notion API error 404 (object_not_found): Page not found"
		if apiErr.Error() != expected {
			t.Errorf("expected error message %q, got %q", expected, apiErr.Error())
		}
	})

	t.Run("without response", func(t *testing.T) {
		apiErr := &APIError{
			StatusCode: 500,
		}

		expected := "notion API error 500"
		if apiErr.Error() != expected {
			t.Errorf("expected error message %q, got %q", expected, apiErr.Error())
		}
	})
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  503,
			Code:    "service_unavailable",
			Message: "Service temporarily unavailable",
		})
	}))
	defer server.Close()

	// Create client with circuit breaker enabled; disable retries to avoid slow backoff
	client := NewClient("test-token").
		WithBaseURL(server.URL).
		WithMaxRetries(0).
		EnableCircuitBreaker()

	ctx := context.Background()

	// Make 5 requests to hit the threshold (no retries, so 1 attempt per call)
	for i := 0; i < defaultCircuitBreakerThreshold; i++ {
		_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
		if err == nil {
			t.Fatal("expected error from failing service")
		}

		// Should not be circuit open error yet
		if errors.Is(err, ErrCircuitOpen) {
			t.Fatalf("circuit opened too early at failure %d", i+1)
		}
	}

	// Next request should fail immediately with circuit open error
	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	// Verify the circuit breaker prevented the request (1 attempt per call, no retries)
	if attempts != defaultCircuitBreakerThreshold {
		t.Errorf("expected %d attempts before circuit opened, got %d", defaultCircuitBreakerThreshold, attempts)
	}
}

func TestCircuitBreaker_ResetOnSuccess(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= 3 {
			// Fail first 3 requests
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  503,
				Code:    "service_unavailable",
				Message: "Service temporarily unavailable",
			})
		} else {
			// Succeed on 4th request
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()

	client := NewClient("test-token").
		WithBaseURL(server.URL).
		WithMaxRetries(0).
		EnableCircuitBreaker()

	ctx := context.Background()

	// Make 3 failing requests
	for i := 0; i < 3; i++ {
		_, _ = client.doRequest(ctx, http.MethodGet, "/test", nil)
	}

	// Make successful request - should reset failure counter
	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	_ = resp.Body.Close()

	// Now make more failing requests - should need 5 more to open circuit
	failCount = 0 // Reset server counter
	for i := 0; i < defaultCircuitBreakerThreshold-1; i++ {
		_, _ = client.doRequest(ctx, http.MethodGet, "/test", nil)
	}

	// Circuit should not be open yet
	_, err = client.doRequest(ctx, http.MethodGet, "/test", nil)
	if errors.Is(err, ErrCircuitOpen) {
		t.Error("circuit opened prematurely after success reset")
	}
}

func TestCircuitBreaker_AutoRecovery(t *testing.T) {
	doRequestCallCount := 0
	mu := &sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		currentCall := doRequestCallCount
		mu.Unlock()

		// First 5 doRequest calls should fail (to trigger circuit opening)
		// Note: each doRequest makes multiple HTTP attempts due to retries
		// After circuit opens and recovers, we succeed
		if currentCall < defaultCircuitBreakerThreshold {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Object:  "error",
				Status:  503,
				Code:    "service_unavailable",
				Message: "Service temporarily unavailable",
			})
		} else {
			// Succeed after circuit recovery
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()

	// Create client with short recovery timeout for testing; disable retries to avoid slow backoff
	client := NewClient("test-token").
		WithBaseURL(server.URL).
		WithMaxRetries(0).
		WithCircuitBreaker(defaultCircuitBreakerThreshold, 100*time.Millisecond)

	ctx := context.Background()

	// Open the circuit - make 5 failing requests
	for i := 0; i < defaultCircuitBreakerThreshold; i++ {
		mu.Lock()
		doRequestCallCount = i
		mu.Unlock()
		_, _ = client.doRequest(ctx, http.MethodGet, "/test", nil)
	}

	// Verify circuit is open
	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Update counter for recovery attempt
	mu.Lock()
	doRequestCallCount = defaultCircuitBreakerThreshold
	mu.Unlock()

	// Circuit should be half-open and allow a request through
	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("expected success after recovery timeout, got error: %v", err)
	}
	_ = resp.Body.Close()

	// Circuit should now be fully closed
	if client.circuitBreaker.isOpen() {
		t.Error("circuit should be closed after successful recovery")
	}
}

func TestCircuitBreaker_DisabledByDefault(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  503,
			Code:    "service_unavailable",
			Message: "Service temporarily unavailable",
		})
	}))
	defer server.Close()

	// Create client WITHOUT enabling circuit breaker; disable retries to avoid slow backoff
	client := NewClient("test-token").WithBaseURL(server.URL).WithMaxRetries(0)

	ctx := context.Background()

	// Make many requests - circuit breaker should not open
	for i := 0; i < defaultCircuitBreakerThreshold+5; i++ {
		_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
		if errors.Is(err, ErrCircuitOpen) {
			t.Fatal("circuit breaker should be disabled by default")
		}
	}
}

func TestCircuitBreaker_Only5xxErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Return 4xx errors which should not trigger circuit breaker
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "Not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").
		WithBaseURL(server.URL).
		EnableCircuitBreaker()

	ctx := context.Background()

	// Make many 4xx requests - should not open circuit
	for i := 0; i < defaultCircuitBreakerThreshold+5; i++ {
		_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
		if errors.Is(err, ErrCircuitOpen) {
			t.Fatal("circuit should not open for 4xx errors")
		}
	}
}
