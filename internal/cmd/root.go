package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/config"
	"github.com/salmonumbrella/notion-cli/internal/debug"
	"github.com/salmonumbrella/notion-cli/internal/iocontext"
	"github.com/salmonumbrella/notion-cli/internal/logging"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/skill"
	"github.com/salmonumbrella/notion-cli/internal/ui"
)

func newRootCmd(app *App) *cobra.Command {
	// Global flags
	var (
		debugMode     bool
		workspaceName string
		queryFile     string
		errorFormat   string
		quietFlag     bool

		// Agent-friendly flags
		yesFlag   bool
		limitFlag int
		sortBy    string
		descFlag  bool
	)

	rootCmd := &cobra.Command{
		Use:   "notion",
		Short: "CLI for Notion API",
		Long:  `A command-line interface for interacting with the Notion API`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure Cobra doesn't emit its own error/usage text; we handle error output centrally.
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			// Configure slog based on debug flag
			logging.Setup(debugMode, app.Stderr)

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
			// Priority: --format (alias) > --output > config file > default
			formatStr, _ := cmd.Flags().GetString("output")
			if cmd.Flags().Changed("format") {
				// --format alias takes precedence if explicitly used
				formatStr, _ = cmd.Flags().GetString("format")
			} else if !cmd.Flags().Changed("output") && cfg.GetOutput() != "" {
				// Fall back to config file default
				formatStr = cfg.GetOutput()
			} else if !cmd.Flags().Changed("output") {
				// Default to JSON when stdout is not a TTY (agent-friendly)
				if !isTerminal(app.Stdout) {
					formatStr = string(output.FormatJSON)
				}
			}

			// Parse and validate output format
			format, err := output.ParseFormat(formatStr)
			if err != nil {
				return err
			}

			// In non-interactive JSON/YAML output, suppress non-essential warnings by default.
			if !cmd.Flags().Changed("quiet") && !isTerminal(app.Stdout) {
				switch format {
				case output.FormatJSON, output.FormatNDJSON, output.FormatYAML:
					quietFlag = true
				}
			}

			// Get jq query from flags
			query, _ := cmd.Flags().GetString("query")
			queryFileFlag, _ := cmd.Flags().GetString("query-file")
			if query != "" && queryFileFlag != "" {
				return fmt.Errorf("use only one of --query or --query-file")
			}
			if queryFileFlag != "" {
				loaded, err := cmdutil.ReadInputSource(queryFileFlag)
				if err != nil {
					return err
				}
				query = loaded
			}

			// Inject format, debug mode, query, and workspace into context so subcommands can access them
			ctx := cmd.Context()
			ctx = iocontext.WithIO(ctx, app.Stdout, app.Stderr)
			ctx = output.WithFormat(ctx, format)
			ctx = output.WithQuery(ctx, query)
			ctx = debug.WithDebug(ctx, debugMode)
			ctx = WithWorkspace(ctx, ws)

			// Inject agent-friendly flags into context
			ctx = output.WithYes(ctx, yesFlag)
			ctx = output.WithLimit(ctx, limitFlag)
			ctx = output.WithSort(ctx, sortBy, descFlag)
			ctx = output.WithQuiet(ctx, quietFlag)
			ctx = WithErrorFormat(ctx, errorFormat)
			ctx = ui.WithUI(ctx, ui.New(parseColorMode(cfg.GetColor())))

			// Load skill file for alias resolution (non-fatal if missing)
			sf, _ := skill.Load()
			ctx = WithSkillFile(ctx, sf)

			cmd.SetContext(ctx)

			// Check token age and warn if old (skip for auth and config commands)
			skipCommands := map[string]bool{"auth": true, "config": true}
			if !skipCommands[cmd.Name()] && (cmd.Parent() == nil || !skipCommands[cmd.Parent().Name()]) {
				checkTokenAgeAndWarn(ctx, quietFlag)
			}

			if err := validateErrorFormat(errorFormat); err != nil {
				return err
			}

			// Suppress Cobra's default usage output when emitting structured errors.
			// We handle error printing ourselves to keep machine-readable output clean.
			if effectiveErrorFormat(ctx) != "text" {
				cmd.SilenceUsage = true
			}

			return nil
		},
	}

	// Set version info
	rootCmd.Version = app.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("notion-cli %s (commit: %s, built: %s)\n", app.Version, app.Commit, app.BuildTime))

	// Global flags
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text|json|ndjson|table|yaml)")
	// Alias --format to --output for agent discoverability
	rootCmd.PersistentFlags().String("format", "text", "Alias for --output")
	_ = rootCmd.PersistentFlags().MarkHidden("format")
	rootCmd.PersistentFlags().String("query", "", "jq expression to filter JSON output")
	rootCmd.PersistentFlags().StringVar(&queryFile, "query-file", "", "Read jq expression from file (use - for stdin)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output (shows HTTP requests/responses)")
	rootCmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace to use (overrides NOTION_WORKSPACE env var)")
	rootCmd.PersistentFlags().StringVar(&errorFormat, "error-format", "auto", "Error output format (auto|text|json)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Suppress non-essential output")

	// Agent-friendly flags
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompts (for automation)")
	rootCmd.PersistentFlags().BoolVar(&yesFlag, "no-input", false, "Alias for --yes (non-interactive)")
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
	rootCmd.AddCommand(newFetchCmd())
	rootCmd.AddCommand(newWebhookCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newAPICmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newSkillCmd())

	return rootCmd
}

// checkTokenAgeAndWarn checks if the token is older than the rotation threshold
// and prints a warning to stderr if it is. This is non-blocking.
func checkTokenAgeAndWarn(ctx context.Context, quiet bool) {
	if quiet {
		return
	}
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
		_, _ = fmt.Fprintf(stderrFromContext(ctx), "Warning: Your API token is %d days old. Consider rotating it for security.\n", age)
	}
}

// NewNotionClient creates a new Notion API client with debug mode enabled if the --debug flag was set.
func NewNotionClient(ctx context.Context, token string) *notion.Client {
	client := notion.NewClient(token)
	if debug.IsDebug(ctx) {
		client.WithDebugOutput(stderrFromContext(ctx))
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

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func parseColorMode(value string) ui.ColorMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always":
		return ui.ColorAlways
	case "never":
		return ui.ColorNever
	default:
		return ui.ColorAuto
	}
}
