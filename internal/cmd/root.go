package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/config"
	"github.com/salmonumbrella/notion-cli/internal/debug"
	"github.com/salmonumbrella/notion-cli/internal/logging"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat  output.Format
	debugMode     bool
	workspaceName string

	// Agent-friendly flags
	yesFlag   bool
	limitFlag int
	sortBy    string
	descFlag  bool

	// Version information
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "notion",
	Short: "CLI for Notion API",
	Long:  `A command-line interface for interacting with the Notion API`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Configure slog based on debug flag
		logging.Setup(debugMode, os.Stderr)

		// Load config file (skip for config commands to avoid recursion)
		var cfg *config.Config
		if cmd.Name() != "config" && (cmd.Parent() == nil || cmd.Parent().Name() != "config") {
			loadedCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			cfg = loadedCfg
		} else {
			cfg = &config.Config{}
		}

		// Determine workspace: flag > env var > config default
		ws := workspaceName
		if ws == "" {
			ws = os.Getenv("NOTION_WORKSPACE")
		}

		// Get output format from flag or config file
		formatStr, _ := cmd.Flags().GetString("output")
		// If flag was not explicitly set by user, use config file default
		if !cmd.Flags().Changed("output") && cfg.GetOutput() != "" {
			formatStr = cfg.GetOutput()
		}

		// Parse and validate output format
		format, err := output.ParseFormat(formatStr)
		if err != nil {
			return err
		}

		// Store format in both global var (for backwards compatibility)
		// and context (new pattern for dependency injection)
		outputFormat = format

		// Get jq query from flag
		query, _ := cmd.Flags().GetString("query")

		// Inject format, debug mode, query, and workspace into context so subcommands can access them
		ctx := output.WithFormat(cmd.Context(), format)
		ctx = output.WithQuery(ctx, query)
		ctx = debug.WithDebug(ctx, debugMode)
		ctx = WithWorkspace(ctx, ws)

		// Inject agent-friendly flags into context
		ctx = output.WithYes(ctx, yesFlag)
		ctx = output.WithLimit(ctx, limitFlag)
		ctx = output.WithSort(ctx, sortBy, descFlag)

		cmd.SetContext(ctx)

		// Check token age and warn if old (skip for auth and config commands)
		skipCommands := map[string]bool{"auth": true, "config": true}
		if !skipCommands[cmd.Name()] && (cmd.Parent() == nil || !skipCommands[cmd.Parent().Name()]) {
			checkTokenAgeAndWarn()
		}

		return nil
	},
}

func init() {
	// Set version info
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("notion-cli %s (commit: %s, built: %s)\n", version, commit, buildTime))

	// Global flags
	rootCmd.PersistentFlags().String("output", "text", "Output format (text|json|table|yaml)")
	rootCmd.PersistentFlags().String("query", "", "jq expression to filter JSON output")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output (shows HTTP requests/responses)")
	rootCmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace to use (overrides NOTION_WORKSPACE env var)")

	// Agent-friendly flags
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompts (for automation)")
	rootCmd.PersistentFlags().IntVar(&limitFlag, "limit", 0, "Limit number of results (0 = unlimited)")
	rootCmd.PersistentFlags().StringVar(&sortBy, "sort-by", "", "Sort results by field")
	rootCmd.PersistentFlags().BoolVar(&descFlag, "desc", false, "Sort in descending order")

	// Register subcommands
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newWorkspaceCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newBlockCmd())
	rootCmd.AddCommand(newDBCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newCommentCmd())
	rootCmd.AddCommand(newFileCmd())
	rootCmd.AddCommand(newDataSourceCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newAPICmd())
}

// checkTokenAgeAndWarn checks if the token is older than the rotation threshold
// and prints a warning to stderr if it is. This is non-blocking.
func checkTokenAgeAndWarn() {
	// Only check for keyring tokens (not env var tokens)
	if os.Getenv(auth.EnvVarName) != "" {
		return
	}

	// Get token metadata
	metadata, err := auth.GetTokenMetadata()
	if err != nil || metadata == nil {
		return
	}

	// Check if token is old and warn
	if auth.IsTokenExpiringSoon(metadata.CreatedAt) {
		age := auth.TokenAgeDays(metadata.CreatedAt)
		fmt.Fprintf(os.Stderr, "Warning: Your API token is %d days old. Consider rotating it for security.\n", age)
	}
}

// Execute runs the root command with context for graceful shutdown
func Execute(ctx context.Context, args []string) error {
	rootCmd.SetArgs(args)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// GetOutputFormat returns the current output format for use by subcommands
func GetOutputFormat() output.Format {
	return outputFormat
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildTime = b
	rootCmd.Version = v
	rootCmd.SetVersionTemplate(fmt.Sprintf("notion-cli %s (commit: %s, built: %s)\n", v, c, b))
}

// GetDebugMode returns true if debug mode is enabled
func GetDebugMode() bool {
	return debugMode
}

// NewNotionClient creates a new Notion API client with debug mode enabled if the --debug flag was set
func NewNotionClient(token string) *notion.Client {
	client := notion.NewClient(token)
	if debugMode {
		client.WithDebug()
	}
	return client
}

// GetTokenFromContext retrieves the token based on workspace context.
// If a workspace is specified in context, it gets the workspace-specific token.
// Otherwise, falls back to the default token retrieval.
func GetTokenFromContext(ctx context.Context) (string, error) {
	workspace := WorkspaceFromContext(ctx)
	if workspace != "" {
		// Get workspace-specific token
		return auth.GetWorkspaceToken(workspace)
	}
	// Fall back to default token (keyring or env var)
	return auth.GetWorkspaceToken("")
}
