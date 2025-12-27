//go:build integration
// +build integration

package auth

import (
	"os"
	"testing"
)

// Integration tests for actual keyring operations.
// Run with: go test -tags=integration ./internal/auth/...

func TestKeyring_StoreAndRetrieve(t *testing.T) {
	// Clear any existing env var to ensure we test keyring
	os.Unsetenv("NOTION_TOKEN")

	testToken := "secret_integration_test_token_xyz123"

	// Clean up any existing token
	_ = DeleteToken()

	// Store token
	err := StoreToken(testToken)
	if err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Retrieve token
	retrievedToken, err := GetToken()
	if err != nil {
		t.Fatalf("failed to retrieve token: %v", err)
	}

	if retrievedToken != testToken {
		t.Errorf("expected token %q, got %q", testToken, retrievedToken)
	}

	// Verify HasToken returns true
	if !HasToken() {
		t.Error("HasToken should return true after storing token")
	}

	// Clean up
	err = DeleteToken()
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify token is gone
	_, err = GetToken()
	if err == nil {
		t.Error("expected error after deleting token, got nil")
	}

	if HasToken() {
		t.Error("HasToken should return false after deleting token")
	}
}

func TestKeyring_UpdateToken(t *testing.T) {
	os.Unsetenv("NOTION_TOKEN")

	firstToken := "secret_first_token"
	secondToken := "secret_second_token"

	// Clean up
	_ = DeleteToken()

	// Store first token
	err := StoreToken(firstToken)
	if err != nil {
		t.Fatalf("failed to store first token: %v", err)
	}

	// Verify first token
	token, err := GetToken()
	if err != nil {
		t.Fatalf("failed to get first token: %v", err)
	}
	if token != firstToken {
		t.Errorf("expected first token %q, got %q", firstToken, token)
	}

	// Update to second token
	err = StoreToken(secondToken)
	if err != nil {
		t.Fatalf("failed to store second token: %v", err)
	}

	// Verify second token
	token, err = GetToken()
	if err != nil {
		t.Fatalf("failed to get second token: %v", err)
	}
	if token != secondToken {
		t.Errorf("expected second token %q, got %q", secondToken, token)
	}

	// Clean up
	_ = DeleteToken()
}

func TestKeyring_EnvironmentVariableTakesPrecedence(t *testing.T) {
	keyringToken := "secret_keyring_token"
	envToken := "secret_env_token"

	// Clean up and store in keyring
	_ = DeleteToken()
	err := StoreToken(keyringToken)
	if err != nil {
		t.Fatalf("failed to store keyring token: %v", err)
	}

	// Without env var, should get keyring token
	token, err := GetToken()
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if token != keyringToken {
		t.Errorf("expected keyring token %q, got %q", keyringToken, token)
	}

	// Set env var — env takes precedence over keyring (standard CLI convention)
	os.Setenv("NOTION_TOKEN", envToken)
	defer os.Unsetenv("NOTION_TOKEN")

	token, err = GetToken()
	if err != nil {
		t.Fatalf("failed to get token with env var: %v", err)
	}
	if token != envToken {
		t.Errorf("env var should take precedence: expected %q, got %q", envToken, token)
	}

	// Clean up keyring
	_ = DeleteToken()

	// With no keyring token, env var still works
	token, err = GetToken()
	if err != nil {
		t.Fatalf("failed to get env token: %v", err)
	}
	if token != envToken {
		t.Errorf("expected env token %q, got %q", envToken, token)
	}
}
