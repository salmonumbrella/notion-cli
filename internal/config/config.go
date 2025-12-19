package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the CLI configuration
type Config struct {
	// Default output format (text, json, table, yaml)
	Output string `yaml:"output,omitempty"`

	// Default color mode (auto, always, never)
	Color string `yaml:"color,omitempty"`

	// Default workspace name (for multi-workspace support)
	DefaultWorkspace string `yaml:"default_workspace,omitempty"`

	// Workspaces configuration
	Workspaces map[string]WorkspaceConfig `yaml:"workspaces,omitempty"`
}

// WorkspaceConfig represents workspace-specific configuration
type WorkspaceConfig struct {
	// Token source: "keyring" or "env:VAR_NAME"
	TokenSource string `yaml:"token_source,omitempty"`

	// Override output format for this workspace
	Output string `yaml:"output,omitempty"`
}

// configPathFunc is the function used to get the default config path
// It can be overridden for testing
var configPathFunc = defaultConfigPath

// defaultConfigPath returns ~/.config/notion-cli/config.yaml
func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "notion-cli", "config.yaml"), nil
}

// DefaultConfigPath returns ~/.config/notion-cli/config.yaml
func DefaultConfigPath() (string, error) {
	return configPathFunc()
}

// Load loads config from the default path, returns empty config if not found
func Load() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return &Config{}, nil
	}
	return LoadFromPath(path)
}

// LoadFromPath loads config from a specific path
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil // Return empty config if file doesn't exist
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}
	return &cfg, nil
}

// Save saves config to the default path
func (c *Config) Save() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return c.SaveToPath(path)
}

// SaveToPath saves config to a specific path
func (c *Config) SaveToPath(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetOutput returns the effective output format (config default or empty)
func (c *Config) GetOutput() string {
	return c.Output
}

// GetColor returns the effective color mode (config default or empty)
func (c *Config) GetColor() string {
	return c.Color
}
