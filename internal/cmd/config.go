package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/salmonumbrella/notion-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Manage CLI configuration",
		Long:    `Manage notion-cli configuration file at ~/.config/notion-cli/config.yaml`,
	}
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigPathCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		Long:  `Display the current configuration from ~/.config/notion-cli/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := stdoutFromContext(cmd.Context())
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Marshal to YAML for display
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to format config: %w", err)
			}

			// If config is empty, show a helpful message
			if len(data) == 0 || string(data) == "{}\n" {
				path, _ := config.DefaultConfigPath()
				_, _ = fmt.Fprintf(out, "No configuration file found at %s\n", path)
				_, _ = fmt.Fprintln(out, "\nTo create a config file, use:")
				_, _ = fmt.Fprintln(out, "  notion config set output json")
				return nil
			}

			_, _ = fmt.Fprint(out, string(data))
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value in ~/.config/notion-cli/config.yaml

Supported keys:
  output            - Default output format (text, json, table, yaml)
  color             - Default color mode (auto, always, never)
  default_workspace - Default workspace name

Examples:
  notion config set output json
  notion config set color always
  notion config set default_workspace personal`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := stdoutFromContext(cmd.Context())
			key := args[0]
			value := args[1]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Set the value based on key
			switch key {
			case "output":
				// Validate output format
				validFormats := []string{"text", "json", "table", "yaml"}
				if !contains(validFormats, value) {
					return fmt.Errorf("invalid output format %q, must be one of: %s", value, strings.Join(validFormats, ", "))
				}
				cfg.Output = value
			case "color":
				// Validate color mode
				validModes := []string{"auto", "always", "never"}
				if !contains(validModes, value) {
					return fmt.Errorf("invalid color mode %q, must be one of: %s", value, strings.Join(validModes, ", "))
				}
				cfg.Color = value
			case "default_workspace":
				cfg.DefaultWorkspace = value
			default:
				return fmt.Errorf("unknown config key %q\n\nSupported keys: output, color, default_workspace", key)
			}

			// Save the config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			path, _ := config.DefaultConfigPath()
			_, _ = fmt.Fprintf(out, "Set %s = %s in %s\n", key, value, path)
			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long:  `Display the path to the configuration file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := stdoutFromContext(cmd.Context())
			path, err := config.DefaultConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine config path: %w", err)
			}

			_, _ = fmt.Fprintln(out, path)

			// Show if file exists
			if _, err := os.Stat(path); err == nil {
				_, _ = fmt.Fprintln(out, "(file exists)")
			} else if os.IsNotExist(err) {
				_, _ = fmt.Fprintln(out, "(file does not exist)")
			}

			return nil
		},
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
