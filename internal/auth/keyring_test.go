package auth

import (
	"fmt"
	"os"
	"testing"

	"github.com/99designs/keyring"
)

// mockKeyring is a test double for KeyringProvider that simulates no keyring access
type mockKeyring struct {
	items map[string]keyring.Item
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{
		items: make(map[string]keyring.Item),
	}
}

func (m *mockKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *mockKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

func (m *mockKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

// setupMockKeyring configures tests to use a mock keyring with no stored token
func setupMockKeyring() func() {
	mock := newMockKeyring()
	originalProvider := defaultProvider

	// Set provider to return an empty mock (simulating no keyring access)
	setProviderFunc(func() (KeyringProvider, error) {
		return mock, nil
	})

	// Return cleanup function
	return func() {
		defaultProvider = originalProvider
	}
}

// setupNoKeyring configures tests to simulate environments where keyring is unavailable
func setupNoKeyring() func() {
	originalProvider := defaultProvider

	// Set provider to always return error (simulating CI/container environment)
	setProviderFunc(func() (KeyringProvider, error) {
		return nil, fmt.Errorf("keyring not available")
	})

	// Return cleanup function
	return func() {
		defaultProvider = originalProvider
	}
}

func TestGetToken_FromEnvironment(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	// Set environment variable
	expectedToken := "secret_test_token_12345"
	os.Setenv("NOTION_TOKEN", expectedToken)
	defer os.Unsetenv("NOTION_TOKEN")

	token, err := GetToken()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, token)
	}
}

func TestGetToken_NoTokenAvailable(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	// Ensure env var is not set
	os.Unsetenv("NOTION_TOKEN")

	_, err := GetToken()
	if err == nil {
		t.Fatal("expected error when no token available, got nil")
	}
}

func TestHasToken_WithEnvironment(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	os.Setenv("NOTION_TOKEN", "test_token")
	defer os.Unsetenv("NOTION_TOKEN")

	if !HasToken() {
		t.Error("expected HasToken to return true with env var set")
	}
}

func TestHasToken_WithoutToken(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	os.Unsetenv("NOTION_TOKEN")

	if HasToken() {
		t.Error("expected HasToken to return false with no token available")
	}
}

func TestStoreToken_EmptyToken(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	err := StoreToken("")
	if err == nil {
		t.Error("expected error when storing empty token, got nil")
	}
}

func TestDeleteToken_NoError(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	// Should not error even if token doesn't exist
	err := DeleteToken()
	if err != nil {
		t.Errorf("expected no error from DeleteToken, got: %v", err)
	}
}

// Test that mock keyring works correctly
func TestMockKeyring_StoreAndRetrieve(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	// Store a token
	testToken := "test_token_abc123"
	err := StoreToken(testToken)
	if err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Retrieve the token (should come from mock keyring, not env var)
	os.Unsetenv("NOTION_TOKEN")
	token, err := GetToken()
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if token != testToken {
		t.Errorf("expected token %q, got %q", testToken, token)
	}

	// Delete the token
	err = DeleteToken()
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify it's gone
	_, err = GetToken()
	if err == nil {
		t.Error("expected error after deleting token, got nil")
	}
}
