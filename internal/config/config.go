package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	// Workspace display name
	Name string `yaml:"name,omitempty"`

	// Token source: "keyring", "env:VAR_NAME", or direct token value
	TokenSource string `yaml:"token_source,omitempty"`

	// Optional custom API URL (for testing/enterprise)
	APIURL string `yaml:"api_url,omitempty"`

	// Override output format for this workspace
	Output string `yaml:"output,omitempty"`
}

// configPathFunc is the function used to get the default config path
// It can be overridden for testing
var configPathFunc = defaultConfigPath

// SetConfigPathFunc sets the config path function for testing.
// Returns the original function so it can be restored.
func SetConfigPathFunc(fn func() (string, error)) func() (string, error) {
	orig := configPathFunc
	configPathFunc = fn
	return orig
}

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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// GetOutput returns the effective output format (config default or empty)
func (c *Config) GetOutput() string {
	return c.Output
}

// GetColor returns the effective color mode (config default or empty)
func (c *Config) GetColor() string {
	return c.Color
}

// GetWorkspace returns the workspace configuration by name
func (c *Config) GetWorkspace(name string) (*WorkspaceConfig, error) {
	if c.Workspaces == nil {
		return nil, fmt.Errorf("workspace %q not found", name)
	}

	ws, ok := c.Workspaces[name]
	if !ok {
		return nil, fmt.Errorf("workspace %q not found", name)
	}

	return &ws, nil
}

// GetDefaultWorkspace returns the default workspace configuration
func (c *Config) GetDefaultWorkspace() (*WorkspaceConfig, error) {
	// If DefaultWorkspace is set, use it
	if c.DefaultWorkspace != "" {
		return c.GetWorkspace(c.DefaultWorkspace)
	}

	// If still no default found and there's exactly one workspace, use it
	if len(c.Workspaces) == 1 {
		for _, ws := range c.Workspaces {
			ws := ws // Create a copy to avoid returning address of loop variable
			return &ws, nil
		}
	}

	return nil, fmt.Errorf("no default workspace configured")
}

// SetDefaultWorkspace sets the default workspace by name
func (c *Config) SetDefaultWorkspace(name string) error {
	// Verify workspace exists
	if _, err := c.GetWorkspace(name); err != nil {
		return err
	}

	// Set the new default
	c.DefaultWorkspace = name

	return nil
}

// AddWorkspace adds a new workspace to the configuration
func (c *Config) AddWorkspace(name string, ws WorkspaceConfig) error {
	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	// Initialize map if needed
	if c.Workspaces == nil {
		c.Workspaces = make(map[string]WorkspaceConfig)
	}

	// Check if workspace already exists
	if _, exists := c.Workspaces[name]; exists {
		return fmt.Errorf("workspace %q already exists", name)
	}

	// Set the name field
	ws.Name = name

	// If this is the first workspace, set it as default
	if len(c.Workspaces) == 0 {
		c.DefaultWorkspace = name
	}

	c.Workspaces[name] = ws
	return nil
}

// RemoveWorkspace removes a workspace from the configuration
func (c *Config) RemoveWorkspace(name string) error {
	if c.Workspaces == nil {
		return fmt.Errorf("workspace %q not found", name)
	}

	if _, exists := c.Workspaces[name]; !exists {
		return fmt.Errorf("workspace %q not found", name)
	}

	// Check if this is the default workspace
	isDefault := c.DefaultWorkspace == name

	delete(c.Workspaces, name)

	// If we removed the default workspace, clear the default
	if isDefault {
		c.DefaultWorkspace = ""

		// If there's exactly one workspace left, make it the default
		if len(c.Workspaces) == 1 {
			for wsName := range c.Workspaces {
				c.DefaultWorkspace = wsName
				break
			}
		}
	}

	return nil
}

// ListWorkspaces returns a list of all workspace names
func (c *Config) ListWorkspaces() []string {
	if c.Workspaces == nil {
		return []string{}
	}

	names := make([]string, 0, len(c.Workspaces))
	for name := range c.Workspaces {
		names = append(names, name)
	}

	// Sort for consistent output
	sort.Strings(names)

	return names
}
