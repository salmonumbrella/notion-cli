package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const (
	// ServiceName is the keyring service name for notion-cli
	ServiceName = "notion-cli"
	// KeyName is the key used to store the token in the keyring (legacy)
	KeyName = "notion-token"
	// TokenMetadataKey is the key used to store token metadata in the keyring
	TokenMetadataKey = "notion-token-metadata"
	// UserInfoKey is the key used to store user metadata in the keyring
	UserInfoKey = "notion-user-info"
	// AuthSourceKey is the key used to store the authentication source
	AuthSourceKey = "notion-auth-source"
	// EnvVarName is the environment variable fallback for the token
	EnvVarName = "NOTION_TOKEN"
	// CredentialsDirEnvVarName controls the credential storage root directory.
	// notion-cli keyring files are stored under: <dir>/notion-cli/keyring
	CredentialsDirEnvVarName = "NOTION_CREDENTIALS_DIR"
	// SharedCredentialsDirEnvVarName is a shared OpenClaw-compatible
	// credential root used when NOTION_CREDENTIALS_DIR is unset.
	SharedCredentialsDirEnvVarName = "OPENCLAW_CREDENTIALS_DIR"
	// KeyringPasswordEnvVarName sets the file keyring passphrase for non-interactive setups.
	KeyringPasswordEnvVarName = "NOTION_KEYRING_PASSWORD"
	// SharedKeyringPasswordEnvVarName is a shared OpenClaw-compatible keyring password env.
	SharedKeyringPasswordEnvVarName = "CW_KEYRING_PASSWORD"
	// OpenClawKeyringPasswordEnvVarName is an OpenClaw-specific fallback keyring password env.
	OpenClawKeyringPasswordEnvVarName = "OPENCLAW_KEYRING_PASSWORD"
	// DBUSSessionAddressEnvVarName is used to detect Linux headless mode.
	DBUSSessionAddressEnvVarName = "DBUS_SESSION_BUS_ADDRESS"
	// TokenRotationThresholdDays is the number of days before warning about token age
	TokenRotationThresholdDays = 90
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

// TokenMetadata contains metadata about the stored token
type TokenMetadata struct {
	Token     string    `json:"token"`
	Source    string    `json:"source"` // "oauth" or "internal"
	CreatedAt time.Time `json:"created_at"`
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

func keyringFileDir() string {
	if dir := strings.TrimSpace(os.Getenv(CredentialsDirEnvVarName)); dir != "" {
		return filepath.Join(dir, ServiceName, "keyring")
	}
	if dir := strings.TrimSpace(os.Getenv(SharedCredentialsDirEnvVarName)); dir != "" {
		return filepath.Join(dir, ServiceName, "keyring")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}

	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return string(os.PathSeparator) + filepath.Join(ServiceName, "keyring")
	}
	return filepath.Join(configDir, ServiceName, "keyring")
}

func keyringFilePassword() string {
	if password := strings.TrimSpace(os.Getenv(KeyringPasswordEnvVarName)); password != "" {
		return password
	}
	if password := strings.TrimSpace(os.Getenv(SharedKeyringPasswordEnvVarName)); password != "" {
		return password
	}
	if password := strings.TrimSpace(os.Getenv(OpenClawKeyringPasswordEnvVarName)); password != "" {
		return password
	}
	return ServiceName
}

func shouldForceFileBackend(goos string, dbusAddr string) bool {
	return goos == "linux" && strings.TrimSpace(dbusAddr) == ""
}

// newOSKeyring creates a new OS keyring provider
func newOSKeyring() (KeyringProvider, error) {
	fileDir := keyringFileDir()
	cfg := keyring.Config{
		ServiceName: ServiceName,
		// macOS Keychain settings
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
		// File-based fallback (for environments without GUI keyring)
		FileDir:          fileDir,
		FilePasswordFunc: func(_ string) (string, error) { return keyringFilePassword(), nil },
	}

	if shouldForceFileBackend(runtime.GOOS, os.Getenv(DBUSSessionAddressEnvVarName)) {
		cfg.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
	}

	ring, err := keyring.Open(cfg)
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
// Can be overridden for testing using SetProviderFunc (in keyring_test.go)
var defaultProvider func() (KeyringProvider, error) = newOSKeyring

// StoreToken stores the Notion API token in the system keyring.
// Returns an error if the token is empty or if keyring storage fails.
// This function preserves the CreatedAt timestamp if the token hasn't changed.
func StoreToken(token string) error {
	return StoreTokenWithSource(token, "")
}

// StoreTokenWithSource stores the Notion API token with source metadata.
// If source is empty, it's stored as "unknown".
// Preserves CreatedAt if the token hasn't changed.
func StoreTokenWithSource(token string, source string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	provider, err := defaultProvider()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	// Check if we have existing metadata
	var createdAt time.Time
	existingMeta, err := GetTokenMetadata()
	if err == nil && existingMeta != nil && existingMeta.Token == token {
		// Token hasn't changed, preserve CreatedAt
		createdAt = existingMeta.CreatedAt
	} else {
		// New token or first save, use current time
		createdAt = time.Now()
	}

	// Default source if not specified
	if source == "" {
		source = "unknown"
	}

	// Store metadata
	metadata := TokenMetadata{
		Token:     token,
		Source:    source,
		CreatedAt: createdAt,
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal token metadata: %w", err)
	}

	err = provider.Set(keyring.Item{
		Key:   TokenMetadataKey,
		Label: "Notion CLI Token Metadata",
		Data:  data,
	})
	if err != nil {
		return fmt.Errorf("failed to store token metadata in keyring: %w", err)
	}

	// Also store in legacy location for backwards compatibility
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
	// Check environment variable first — this avoids blocking keychain
	// prompts in CI, tests, and headless environments.
	if token := os.Getenv(EnvVarName); token != "" {
		return token, nil
	}

	// Fall back to keyring
	provider, err := defaultProvider()
	if err == nil {
		item, err := provider.Get(KeyName)
		if err == nil && len(item.Data) > 0 {
			return string(item.Data), nil
		}
	}

	return "", fmt.Errorf("no Notion token found in %s environment variable or keyring", EnvVarName)
}

// HasToken checks if a token is available (either in keyring or env var).
// Returns true if a token exists, false otherwise.
func HasToken() bool {
	_, err := GetToken()
	return err == nil
}

// GetTokenMetadata retrieves token metadata from the keyring.
// Returns nil if no metadata is stored or if metadata is invalid.
// This only returns metadata for tokens stored in the keyring, not env vars.
func GetTokenMetadata() (*TokenMetadata, error) {
	provider, err := defaultProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	item, err := provider.Get(TokenMetadataKey)
	if err != nil {
		return nil, err
	}

	var metadata TokenMetadata
	if err := json.Unmarshal(item.Data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token metadata: %w", err)
	}

	return &metadata, nil
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

	// Also remove token metadata, user info and auth source
	_ = provider.Remove(TokenMetadataKey)
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

// TokenAgeDays calculates the age of a token in days from its creation time.
// Returns 0 if createdAt is zero (token age unknown).
func TokenAgeDays(createdAt time.Time) int {
	if createdAt.IsZero() {
		return 0
	}
	return int(time.Since(createdAt).Hours() / 24)
}

// IsTokenExpiringSoon checks if a token is older than the rotation threshold.
// Returns false if createdAt is zero (token age unknown).
func IsTokenExpiringSoon(createdAt time.Time) bool {
	if createdAt.IsZero() {
		return false
	}
	return TokenAgeDays(createdAt) > TokenRotationThresholdDays
}

// FormatTokenAge formats the token creation time and age in a human-readable way.
// Returns empty string if createdAt is zero (token age unknown).
func FormatTokenAge(createdAt time.Time) string {
	if createdAt.IsZero() {
		return ""
	}
	age := TokenAgeDays(createdAt)
	dateStr := createdAt.Format("2006-01-02")
	switch age {
	case 0:
		return fmt.Sprintf("created today (%s)", dateStr)
	case 1:
		return fmt.Sprintf("1 day ago (created %s)", dateStr)
	default:
		return fmt.Sprintf("%d days ago (created %s)", age, dateStr)
	}
}
