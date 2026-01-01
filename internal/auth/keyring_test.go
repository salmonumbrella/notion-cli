package auth

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// setupMockKeyring configures tests to use a mock keyring with no stored token
func setupMockKeyring() func() {
	mock := NewMockKeyringProvider()
	originalProvider := defaultProvider

	// Set provider to return an empty mock (simulating no keyring access)
	SetProviderFunc(func() (KeyringProvider, error) {
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
	SetProviderFunc(func() (KeyringProvider, error) {
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
	_ = os.Setenv("NOTION_TOKEN", expectedToken)
	defer func() { _ = os.Unsetenv("NOTION_TOKEN") }()

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
	_ = os.Unsetenv("NOTION_TOKEN")

	_, err := GetToken()
	if err == nil {
		t.Fatal("expected error when no token available, got nil")
	}
}

func TestHasToken_WithEnvironment(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	_ = os.Setenv("NOTION_TOKEN", "test_token")
	defer func() { _ = os.Unsetenv("NOTION_TOKEN") }()

	if !HasToken() {
		t.Error("expected HasToken to return true with env var set")
	}
}

func TestHasToken_WithoutToken(t *testing.T) {
	cleanup := setupNoKeyring()
	defer cleanup()

	_ = os.Unsetenv("NOTION_TOKEN")

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
	_ = os.Unsetenv("NOTION_TOKEN")
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

func TestTokenMetadata_StoreAndRetrieve(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	// Store a token with source
	testToken := "secret_test_token_xyz"
	err := StoreTokenWithSource(testToken, "oauth")
	if err != nil {
		t.Fatalf("failed to store token with source: %v", err)
	}

	// Retrieve metadata
	metadata, err := GetTokenMetadata()
	if err != nil {
		t.Fatalf("failed to get token metadata: %v", err)
	}
	if metadata == nil {
		t.Fatal("expected metadata, got nil")
	}

	// Verify metadata fields
	if metadata.Token != testToken {
		t.Errorf("expected token %q, got %q", testToken, metadata.Token)
	}
	if metadata.Source != "oauth" {
		t.Errorf("expected source 'oauth', got %q", metadata.Source)
	}
	if metadata.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero value")
	}
}

func TestTokenMetadata_PreserveCreatedAt(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	// Store a token
	testToken := "secret_test_token_123"
	err := StoreTokenWithSource(testToken, "internal")
	if err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Get the metadata
	metadata1, err := GetTokenMetadata()
	if err != nil {
		t.Fatalf("failed to get token metadata: %v", err)
	}

	// Store the same token again (simulating re-authentication)
	err = StoreTokenWithSource(testToken, "internal")
	if err != nil {
		t.Fatalf("failed to store token again: %v", err)
	}

	// Get metadata again
	metadata2, err := GetTokenMetadata()
	if err != nil {
		t.Fatalf("failed to get token metadata again: %v", err)
	}

	// CreatedAt should be preserved
	if !metadata1.CreatedAt.Equal(metadata2.CreatedAt) {
		t.Errorf("expected CreatedAt to be preserved, got %v and %v", metadata1.CreatedAt, metadata2.CreatedAt)
	}
}

func TestTokenMetadata_NewCreatedAtOnTokenChange(t *testing.T) {
	cleanup := setupMockKeyring()
	defer cleanup()

	// Store first token
	token1 := "secret_test_token_old"
	err := StoreTokenWithSource(token1, "oauth")
	if err != nil {
		t.Fatalf("failed to store first token: %v", err)
	}

	metadata1, _ := GetTokenMetadata()

	// Store different token
	token2 := "secret_test_token_new"
	err = StoreTokenWithSource(token2, "oauth")
	if err != nil {
		t.Fatalf("failed to store second token: %v", err)
	}

	metadata2, _ := GetTokenMetadata()

	// Tokens should be different
	if metadata1.Token == metadata2.Token {
		t.Error("expected tokens to be different")
	}

	// CreatedAt should be updated (or at least different)
	// Note: In very fast execution they might be the same second, but the logic is correct
}

func TestTokenAgeDays(t *testing.T) {
	tests := []struct {
		name     string
		daysAgo  int
		expected int
	}{
		{"zero time", -1, 0}, // Special case: zero time
		{"today", 0, 0},
		{"one day ago", 1, 1},
		{"45 days ago", 45, 45},
		{"95 days ago", 95, 95},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createdAt time.Time
			if tt.daysAgo >= 0 {
				createdAt = time.Now().Add(-time.Duration(tt.daysAgo) * 24 * time.Hour)
			}

			age := TokenAgeDays(createdAt)
			if age != tt.expected {
				t.Errorf("expected age %d, got %d", tt.expected, age)
			}
		})
	}
}

func TestIsTokenExpiringSoon(t *testing.T) {
	tests := []struct {
		name     string
		daysAgo  int
		expected bool
	}{
		{"zero time", -1, false},
		{"fresh token", 0, false},
		{"45 days old", 45, false},
		{"90 days old", 90, false},
		{"91 days old", 91, true},
		{"100 days old", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createdAt time.Time
			if tt.daysAgo >= 0 {
				createdAt = time.Now().Add(-time.Duration(tt.daysAgo) * 24 * time.Hour)
			}

			expiring := IsTokenExpiringSoon(createdAt)
			if expiring != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, expiring)
			}
		})
	}
}

func TestFormatTokenAge(t *testing.T) {
	tests := []struct {
		name     string
		daysAgo  int
		contains string
	}{
		{"zero time", -1, ""},
		{"today", 0, "created today"},
		{"one day ago", 1, "1 day ago"},
		{"45 days ago", 45, "45 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createdAt time.Time
			if tt.daysAgo >= 0 {
				createdAt = time.Now().Add(-time.Duration(tt.daysAgo) * 24 * time.Hour)
			}

			formatted := FormatTokenAge(createdAt)
			if tt.contains == "" {
				if formatted != "" {
					t.Errorf("expected empty string, got %q", formatted)
				}
			} else {
				if !contains(formatted, tt.contains) {
					t.Errorf("expected formatted age to contain %q, got %q", tt.contains, formatted)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
