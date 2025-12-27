package auth

import "github.com/99designs/keyring"

// MockKeyring is a test double for KeyringProvider
type MockKeyring struct {
	items map[string]keyring.Item
}

// NewMockKeyringProvider creates a new mock keyring for testing.
// This is exported for use in tests outside the auth package.
func NewMockKeyringProvider() *MockKeyring {
	return &MockKeyring{
		items: make(map[string]keyring.Item),
	}
}

// Get retrieves an item from the mock keyring
func (m *MockKeyring) Get(key string) (keyring.Item, error) {
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

// Set stores an item in the mock keyring
func (m *MockKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

// Remove deletes an item from the mock keyring
func (m *MockKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

// SetToken is a helper for tests to set a token in the mock keyring
func (m *MockKeyring) SetToken(token string) {
	m.items[KeyName] = keyring.Item{
		Key:  KeyName,
		Data: []byte(token),
	}
}

// SetProviderFunc allows tests to inject a mock provider.
// This is exported for use in tests outside the auth package.
func SetProviderFunc(fn func() (KeyringProvider, error)) {
	if fn == nil {
		// Reset to default
		defaultProvider = newOSKeyring
	} else {
		defaultProvider = fn
	}
}
