package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/salmonumbrella/notion-cli/internal/config"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
		Long:  "Manage multiple workspace configurations for different Notion accounts or integrations.",
	}

	cmd.AddCommand(newWorkspaceListCmd())
	cmd.AddCommand(newWorkspaceAddCmd())
	cmd.AddCommand(newWorkspaceRemoveCmd())
	cmd.AddCommand(newWorkspaceUseCmd())
	cmd.AddCommand(newWorkspaceShowCmd())

	return cmd
}

func newWorkspaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured workspaces",
		Long:  "List all configured workspaces with their token sources and default status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if len(cfg.Workspaces) == 0 {
				fmt.Println("No workspaces configured.")
				fmt.Println("\nTo add a workspace, use:")
				fmt.Println("  notion workspace add <name> --token-source <source>")
				return nil
			}

			// Create tabwriter for aligned columns
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tTOKEN SOURCE\tDEFAULT")

			// Print each workspace
			names := cfg.ListWorkspaces()
			for _, name := range names {
				ws := cfg.Workspaces[name]
				source := formatTokenSource(ws.TokenSource)
				defaultMarker := ""
				// Show marker if explicitly the default or if it's the only workspace (implicit default)
				isDefault := cfg.DefaultWorkspace == name || (cfg.DefaultWorkspace == "" && len(cfg.Workspaces) == 1)
				if isDefault {
					defaultMarker = "*"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, source, defaultMarker)
			}

			_ = w.Flush()
			return nil
		},
	}
}

func newWorkspaceAddCmd() *cobra.Command {
	var tokenSource string
	var apiURL string
	var setDefault bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new workspace",
		Long: `Add a new workspace configuration.

Token sources:
  keyring          - Store token in system keyring
  env:VAR_NAME     - Read token from environment variable
  <direct-token>   - Use token directly (not recommended for production)

Examples:
  notion workspace add personal --token-source keyring
  notion workspace add work --token-source env:NOTION_WORK_TOKEN
  notion workspace add test --token-source secret_abc123 --default`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate token source is provided
			if tokenSource == "" {
				return fmt.Errorf("--token-source is required")
			}

			// Validate token source format
			if err := validateTokenSource(tokenSource); err != nil {
				return err
			}

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create workspace config
			ws := config.WorkspaceConfig{
				TokenSource: tokenSource,
				APIURL:      apiURL,
			}

			// Add workspace
			if err := cfg.AddWorkspace(name, ws); err != nil {
				return err
			}

			// Set as default if requested
			if setDefault {
				cfg.DefaultWorkspace = name
			}

			// Save config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			path, _ := config.DefaultConfigPath()
			fmt.Printf("Added workspace %q to %s\n", name, path)
			if cfg.DefaultWorkspace == name {
				fmt.Println("Set as default workspace")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tokenSource, "token-source", "", "Token source (keyring, env:VAR_NAME, or direct token)")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "Custom API URL (optional)")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set as default workspace")

	return cmd
}

func newWorkspaceRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a workspace",
		Long:  "Remove a workspace configuration from the config file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Check if this is the default workspace
			isDefault := cfg.DefaultWorkspace == name

			// Confirm if removing default workspace (unless -y flag is set)
			ctx := cmd.Context()
			if isDefault && !output.YesFromContext(ctx) {
				fmt.Printf("Warning: %q is the default workspace.\n", name)
				if !confirmAction("Are you sure you want to remove it?") {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Remove workspace
			if err := cfg.RemoveWorkspace(name); err != nil {
				return err
			}

			// Save config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			path, _ := config.DefaultConfigPath()
			fmt.Printf("Removed workspace %q from %s\n", name, path)

			// If a new default was auto-selected, notify user
			if isDefault && cfg.DefaultWorkspace != "" {
				fmt.Printf("Default workspace is now %q\n", cfg.DefaultWorkspace)
			}

			return nil
		},
	}
}

func newWorkspaceUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set default workspace",
		Long:  "Set the default workspace to use when --workspace flag is not specified.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Set default workspace
			if err := cfg.SetDefaultWorkspace(name); err != nil {
				return err
			}

			// Save config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			path, _ := config.DefaultConfigPath()
			fmt.Printf("Set default workspace to %q in %s\n", name, path)

			return nil
		},
	}
}

func newWorkspaceShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show workspace details",
		Long:  "Show detailed configuration for a workspace. If no name is provided, shows the default workspace.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Determine which workspace to show
			var name string
			var ws *config.WorkspaceConfig

			if len(args) > 0 {
				name = args[0]
				ws, err = cfg.GetWorkspace(name)
				if err != nil {
					return err
				}
			} else {
				// Show default workspace
				ws, err = cfg.GetDefaultWorkspace()
				if err != nil {
					return fmt.Errorf("no default workspace configured: %w", err)
				}
				// Get the name from either the explicit default or find the single workspace
				if cfg.DefaultWorkspace != "" {
					name = cfg.DefaultWorkspace
				} else if len(cfg.Workspaces) == 1 {
					// Single workspace case - find the name
					for wsName := range cfg.Workspaces {
						name = wsName
						break
					}
				}
			}

			// Display workspace details
			fmt.Printf("Workspace: %s\n", name)
			// Show "yes" if it's explicitly the default or if it's the only workspace (implicit default)
			isDefault := cfg.DefaultWorkspace == name || (cfg.DefaultWorkspace == "" && len(cfg.Workspaces) == 1)
			if isDefault {
				fmt.Println("Default: yes")
			} else {
				fmt.Println("Default: no")
			}
			fmt.Printf("Token Source: %s\n", formatTokenSource(ws.TokenSource))
			if ws.APIURL != "" {
				fmt.Printf("API URL: %s\n", ws.APIURL)
			}
			if ws.Output != "" {
				fmt.Printf("Output Format: %s\n", ws.Output)
			}

			return nil
		},
	}
}

// formatTokenSource formats a token source for display, redacting sensitive information
func formatTokenSource(source string) string {
	if source == "" {
		return "(not set)"
	}

	// Check if it's an env var reference
	if strings.HasPrefix(source, "env:") {
		return source
	}

	// Check if it's "keyring"
	if source == "keyring" {
		return "keyring"
	}

	// Otherwise, it's a direct token - redact it
	return "(direct)"
}

// validateTokenSource validates the token source format
func validateTokenSource(source string) error {
	if source == "" {
		return fmt.Errorf("token source cannot be empty")
	}

	// Valid formats:
	// - "keyring"
	// - "env:VAR_NAME"
	// - Direct token (any non-empty string)

	if source == "keyring" {
		return nil
	}

	if strings.HasPrefix(source, "env:") {
		varName := strings.TrimPrefix(source, "env:")
		if varName == "" {
			return fmt.Errorf("environment variable name cannot be empty in token source")
		}
		return nil
	}

	// Direct token - accept any non-empty string
	return nil
}

// confirmAction prompts the user for confirmation
func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
