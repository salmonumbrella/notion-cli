// Package testutil provides testing utilities for the Notion CLI.
package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockServer provides a test HTTP server for API mocking.
type MockServer struct {
	server   *httptest.Server
	handlers map[string]http.HandlerFunc
	mu       sync.RWMutex
}

// NewMockServer creates a new mock server.
func NewMockServer() *MockServer {
	ms := &MockServer{
		handlers: make(map[string]http.HandlerFunc),
	}

	ms.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path

		ms.mu.RLock()
		handler, ok := ms.handlers[key]
		ms.mu.RUnlock()

		if ok {
			handler(w, r)
			return
		}

		http.NotFound(w, r)
	}))

	return ms
}

// URL returns the server's base URL.
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close shuts down the server.
func (ms *MockServer) Close() {
	ms.server.Close()
}

// Handle registers a custom handler for a method+path.
func (ms *MockServer) Handle(method, path string, handler http.HandlerFunc) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.handlers[method+" "+path] = handler
}

// HandleJSON registers a handler that returns JSON with the given status.
func (ms *MockServer) HandleJSON(method, path string, status int, response interface{}) {
	ms.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	})
}

// HandleError registers a handler that returns a Notion API error.
func (ms *MockServer) HandleError(method, path string, status int, code, message string) {
	ms.HandleJSON(method, path, status, map[string]interface{}{
		"object":  "error",
		"status":  status,
		"code":    code,
		"message": message,
	})
}

// HandleRateLimit registers a 429 response with Retry-After header.
func (ms *MockServer) HandleRateLimit(method, path string, retryAfter int) {
	ms.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  429,
			"code":    "rate_limited",
			"message": "Rate limited",
		})
	})
}

// Reset clears all registered handlers.
func (ms *MockServer) Reset() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.handlers = make(map[string]http.HandlerFunc)
}
