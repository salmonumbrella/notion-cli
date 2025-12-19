package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/debug"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat output.Format
	debugMode    bool

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
		// Parse and validate output format from flag
		formatStr, _ := cmd.Flags().GetString("output")
		format, err := output.ParseFormat(formatStr)
		if err != nil {
			return err
		}

		// Store format in both global var (for backwards compatibility)
		// and context (new pattern for dependency injection)
		outputFormat = format

		// Inject format and debug mode into context so subcommands can access them
		ctx := output.WithFormat(cmd.Context(), format)
		ctx = debug.WithDebug(ctx, debugMode)
		cmd.SetContext(ctx)

		// Check token age and warn if old (skip for auth commands)
		if cmd.Name() != "auth" && !cmd.HasParent() || (cmd.Parent() != nil && cmd.Parent().Name() != "auth") {
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
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output (shows HTTP requests/responses)")

	// Register subcommands
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newBlockCmd())
	rootCmd.AddCommand(newDBCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newCommentCmd())
	rootCmd.AddCommand(newFileCmd())
	rootCmd.AddCommand(newDataSourceCmd())
	rootCmd.AddCommand(newCompletionCmd())
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
