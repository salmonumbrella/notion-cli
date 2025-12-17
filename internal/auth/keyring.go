package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

const (
	// ServiceName is the keyring service name for notion-cli
	ServiceName = "notion-cli"
	// KeyName is the key used to store the token in the keyring
	KeyName = "notion-token"
	// UserInfoKey is the key used to store user metadata in the keyring
	UserInfoKey = "notion-user-info"
	// AuthSourceKey is the key used to store the authentication source
	AuthSourceKey = "notion-auth-source"
	// EnvVarName is the environment variable fallback for the token
	EnvVarName = "NOTION_TOKEN"
)

// AuthSource represents how the user authenticated
type AuthSource string

const (
	// SourceOAuth indicates OAuth authentication
	SourceOAuth AuthSource = "oauth"
	// SourceInternal indicates internal integration token
	SourceInternal AuthSource = "internal"
	// SourceUnknown indicates unknown authentication source
	SourceUnknown AuthSource = "unknown"
)

// UserInfo contains metadata about the authenticated user/bot
type UserInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Email     string `json:"email,omitempty"`
}

// KeyringProvider defines an interface for keyring operations
type KeyringProvider interface {
	Get(key string) (keyring.Item, error)
	Set(item keyring.Item) error
	Remove(key string) error
}

// osKeyring wraps the actual OS keyring implementation
type osKeyring struct {
	ring keyring.Keyring
}

// newOSKeyring creates a new OS keyring provider
func newOSKeyring() (KeyringProvider, error) {
	// Get config directory for file-based fallback
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	fileDir := configDir + "/notion-cli/keyring"

	ring, err := keyring.Open(keyring.Config{
		ServiceName: ServiceName,
		// macOS Keychain settings
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
		// File-based fallback (for environments without GUI keyring)
		FileDir:          fileDir,
		FilePasswordFunc: func(_ string) (string, error) { return ServiceName, nil },
	})
	if err != nil {
		return nil, err
	}
	return &osKeyring{ring: ring}, nil
}

func (k *osKeyring) Get(key string) (keyring.Item, error) {
	return k.ring.Get(key)
}

func (k *osKeyring) Set(item keyring.Item) error {
	return k.ring.Set(item)
}

func (k *osKeyring) Remove(key string) error {
	return k.ring.Remove(key)
}

// defaultProvider is the keyring provider used by the package
// Can be overridden for testing
var defaultProvider func() (KeyringProvider, error) = newOSKeyring

// setProviderFunc allows tests to inject a mock provider
func setProviderFunc(fn func() (KeyringProvider, error)) {
	defaultProvider = fn
}

// StoreToken stores the Notion API token in the system keyring.
// Returns an error if the token is empty or if keyring storage fails.
func StoreToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	provider, err := defaultProvider()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	err = provider.Set(keyring.Item{
		Key:   KeyName,
		Label: "Notion CLI Token",
		Data:  []byte(token),
	})
	if err != nil {
		return fmt.Errorf("failed to store token in keyring: %w", err)
	}

	return nil
}

// GetToken retrieves the Notion API token from the keyring or environment variable.
// Priority: keyring first, then NOTION_TOKEN env var.
// Returns an error if no token is found in either location.
func GetToken() (string, error) {
	// Try keyring first
	provider, err := defaultProvider()
	if err == nil {
		item, err := provider.Get(KeyName)
		if err == nil && len(item.Data) > 0 {
			return string(item.Data), nil
		}
	}

	// Fallback to environment variable
	token := os.Getenv(EnvVarName)
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no Notion token found in keyring or %s environment variable", EnvVarName)
}

// HasToken checks if a token is available (either in keyring or env var).
// Returns true if a token exists, false otherwise.
func HasToken() bool {
	_, err := GetToken()
	return err == nil
}

// DeleteToken removes the Notion API token from the keyring.
// Does not return an error if the token doesn't exist.
func DeleteToken() error {
	provider, err := defaultProvider()
	if err != nil {
		// If we can't open the keyring, there's nothing to delete
		return nil
	}

	err = provider.Remove(KeyName)
	if err != nil && err != keyring.ErrKeyNotFound {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}

	// Also remove user info and auth source
	_ = provider.Remove(UserInfoKey)
	_ = provider.Remove(AuthSourceKey)

	return nil
}

// StoreUserInfo stores user metadata in the keyring.
func StoreUserInfo(info *UserInfo) error {
	if info == nil {
		return fmt.Errorf("user info cannot be nil")
	}

	provider, err := defaultProvider()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	err = provider.Set(keyring.Item{
		Key:   UserInfoKey,
		Label: "Notion CLI User Info",
		Data:  data,
	})
	if err != nil {
		return fmt.Errorf("failed to store user info in keyring: %w", err)
	}

	return nil
}

// GetUserInfo retrieves user metadata from the keyring.
// Returns nil if no user info is stored.
func GetUserInfo() (*UserInfo, error) {
	provider, err := defaultProvider()
	if err != nil {
		return nil, nil // No keyring access, no user info
	}

	item, err := provider.Get(UserInfoKey)
	if err != nil {
		return nil, nil // No user info stored
	}

	var info UserInfo
	if err := json.Unmarshal(item.Data, &info); err != nil {
		return nil, nil // Invalid data, treat as no user info
	}

	return &info, nil
}

// StoreAuthSource stores the authentication source (OAuth or Internal) in the keyring.
func StoreAuthSource(source AuthSource) error {
	provider, err := defaultProvider()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	err = provider.Set(keyring.Item{
		Key:   AuthSourceKey,
		Label: "Notion CLI Auth Source",
		Data:  []byte(source),
	})
	if err != nil {
		return fmt.Errorf("failed to store auth source in keyring: %w", err)
	}

	return nil
}

// GetAuthSource retrieves the authentication source from the keyring.
// Returns SourceUnknown if no source is stored.
func GetAuthSource() AuthSource {
	provider, err := defaultProvider()
	if err != nil {
		return SourceUnknown
	}

	item, err := provider.Get(AuthSourceKey)
	if err != nil || len(item.Data) == 0 {
		return SourceUnknown
	}

	source := AuthSource(item.Data)
	switch source {
	case SourceOAuth, SourceInternal:
		return source
	default:
		return SourceUnknown
	}
}
