package cmd

import (
	"net/http"
	"testing"
)

func TestHasAuthorizationHeader(t *testing.T) {
	t.Run("nil headers", func(t *testing.T) {
		if hasAuthorizationHeader(nil) {
			t.Fatal("expected false for nil headers")
		}
	})

	t.Run("missing", func(t *testing.T) {
		if hasAuthorizationHeader(http.Header{}) {
			t.Fatal("expected false for empty headers")
		}
	})

	t.Run("present", func(t *testing.T) {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer test")
		if !hasAuthorizationHeader(headers) {
			t.Fatal("expected true when Authorization header is present")
		}
	})

	t.Run("present lowercase", func(t *testing.T) {
		headers := http.Header{}
		headers.Add("authorization", "Bearer test")
		if !hasAuthorizationHeader(headers) {
			t.Fatal("expected true when authorization header is present")
		}
	})

	t.Run("other headers only", func(t *testing.T) {
		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		if hasAuthorizationHeader(headers) {
			t.Fatal("expected false when Authorization header is missing")
		}
	})
}
