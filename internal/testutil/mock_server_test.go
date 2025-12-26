// internal/testutil/mock_server_test.go
package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMockServer_HandleJSON(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	response := map[string]string{"id": "page-123", "object": "page"}
	ms.HandleJSON("GET", "/v1/pages/123", http.StatusOK, response)

	resp, err := http.Get(ms.URL() + "/v1/pages/123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if result["id"] != "page-123" {
		t.Errorf("expected id page-123, got %s", result["id"])
	}
}

func TestMockServer_HandleError(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.HandleError("GET", "/v1/notfound", http.StatusNotFound, "object_not_found", "Page not found")

	resp, err := http.Get(ms.URL() + "/v1/notfound")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "object_not_found") {
		t.Errorf("expected error code in body: %s", body)
	}
}

func TestMockServer_HandleRateLimit(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.HandleRateLimit("GET", "/v1/ratelimited", 5)

	resp, err := http.Get(ms.URL() + "/v1/ratelimited")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestMockServer_Reset(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.HandleJSON("GET", "/v1/test", http.StatusOK, map[string]string{"ok": "true"})
	ms.Reset()

	resp, err := http.Get(ms.URL() + "/v1/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after reset, got %d", resp.StatusCode)
	}
}
