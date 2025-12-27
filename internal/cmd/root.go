package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/config"
	"github.com/salmonumbrella/notion-cli/internal/debug"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/logging"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
	"github.com/salmonumbrella/notion-cli/internal/ui"
)

//go:embed help.txt
var rootHelpText string

func newRootCmd(app *App) *cobra.Command {
	// Global flags
	var (
		debugMode     bool
		workspaceName string
		queryFlag     string
		jqFlag        string
		fieldsFlag    string
		pickFlag      string
		jsonPathFlag  string
		queryFile     string
		errorFormat   string
		quietFlag     bool
		failEmptyFlag bool
		compactJSON   bool
		latestFlag    bool
		recentFlag    int

		// Agent-friendly flags
		yesFlag         bool
		limitFlag       int
		sortBy          string
		descFlag        bool
		resultsOnlyFlag bool
	)

	rootCmd := &cobra.Command{
		Use:   "ntn",
		Short: "CLI for Notion API",
		Long:  `A command-line interface for interacting with the Notion API`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure Cobra doesn't emit its own error/usage text; we handle error output centrally.
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			// Configure slog based on debug flag
			logging.Setup(debugMode, app.Stderr)

			// Load OpenClaw environment defaults from ~/.openclaw/.env when present.
			if err := loadOpenClawEnvIfPresent(); err != nil {
				slog.Debug("Skipping OpenClaw env auto-load", "error", err)
			}

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

			opts, err := parseGlobalOptions(cmd, cfg, app.Stdout, globalFlagInput{
				workspaceName:   workspaceName,
				queryFlag:       queryFlag,
				jqFlag:          jqFlag,
				fieldsFlag:      fieldsFlag,
				pickFlag:        pickFlag,
				jsonPathFlag:    jsonPathFlag,
				quietFlag:       quietFlag,
				failEmptyFlag:   failEmptyFlag,
				compactJSON:     compactJSON,
				latestFlag:      latestFlag,
				recentFlag:      recentFlag,
				yesFlag:         yesFlag,
				limitFlag:       limitFlag,
				sortBy:          sortBy,
				descFlag:        descFlag,
				resultsOnlyFlag: resultsOnlyFlag,
				errorFormat:     errorFormat,
			})
			if err != nil {
				return err
			}
			if err := validateGlobalOptions(&opts); err != nil {
				return err
			}

			// Inject parsed global options into context so subcommands can access them.
			ctx := buildRootContext(cmd.Context(), app, cfg, debugMode, opts)
			if opts.queryNormalized && !opts.quiet {
				ui.FromContext(ctx).Warning("Normalized --query by removing \\! (shell escape); use ! without backslash.")
			}

			// Load skill file for alias resolution (non-fatal if missing)
			sf, _ := skill.Load()
			ctx = WithSkillFile(ctx, sf)

			// Initialize search cache for the duration of this command
			ctx = WithSearchCache(ctx, NewSearchCache())

			cmd.SetContext(ctx)

			// Check token age and warn if old (skip for auth and config commands)
			skipCommands := map[string]bool{"auth": true, "config": true}
			if !skipCommands[cmd.Name()] && (cmd.Parent() == nil || !skipCommands[cmd.Parent().Name()]) {
				checkTokenAgeAndWarn(ctx, opts.quiet)
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
	rootCmd.SetVersionTemplate(fmt.Sprintf("ntn %s (commit: %s, built: %s)\n", app.Version, app.Commit, app.BuildTime))

	// Global flags
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text|json|ndjson|jsonl|table|yaml")
	// Alias --format to --output for agent discoverability
	rootCmd.PersistentFlags().String("format", "text", "Alias for --output")
	_ = rootCmd.PersistentFlags().MarkHidden("format")
	// Shorthand: --json is equivalent to -o json
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Shorthand for --output json")
	_ = rootCmd.PersistentFlags().MarkHidden("json")
	rootCmd.PersistentFlags().StringVarP(&queryFlag, "query", "q", "", "JQ expression to filter JSON output")
	// Alias --jq to --query for discoverability
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Alias for --query")
	_ = rootCmd.PersistentFlags().MarkHidden("jq")
	rootCmd.PersistentFlags().StringVar(&fieldsFlag, "fields", "", "Project fields (comma-separated paths, use key=path to rename)")
	rootCmd.PersistentFlags().StringVar(&pickFlag, "pick", "", "Alias for --fields")
	_ = rootCmd.PersistentFlags().MarkHidden("pick")
	rootCmd.PersistentFlags().StringVar(&jsonPathFlag, "jsonpath", "", "Extract a value using JSONPath (e.g. $.results[0].id)")
	rootCmd.PersistentFlags().StringVar(&queryFile, "query-file", "", "Read JQ expression from file ('-' for stdin)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output (shows HTTP requests/responses)")
	rootCmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace to use (overrides NOTION_WORKSPACE env var)")
	rootCmd.PersistentFlags().StringVar(&errorFormat, "error-format", "auto", "Error output format (auto|text|json)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&failEmptyFlag, "fail-empty", false, "Exit with error when results are empty")
	rootCmd.PersistentFlags().BoolVar(&compactJSON, "compact-json", false, "Output compact JSON (single-line) instead of pretty JSON")
	rootCmd.PersistentFlags().BoolVar(&resultsOnlyFlag, "items-only", false, "Output only the items/results array when present (JSON output)")
	rootCmd.PersistentFlags().BoolVar(&resultsOnlyFlag, "results-only", false, "Alias for --items-only")
	_ = rootCmd.PersistentFlags().MarkHidden("results-only")

	// Machine-readable help (hidden; intercepted in App.Execute before arg validation)
	rootCmd.PersistentFlags().Bool("help-json", false, "Output command help as JSON (for agent discovery)")
	_ = rootCmd.PersistentFlags().MarkHidden("help-json")

	// Agent-friendly flags
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&yesFlag, "no-input", false, "Disable interactive prompts (alias for --yes)")
	_ = rootCmd.PersistentFlags().MarkHidden("no-input")
	rootCmd.PersistentFlags().IntVar(&limitFlag, "limit", 0, "Limit number of results (0 = no limit)")
	rootCmd.PersistentFlags().StringVar(&sortBy, "sort-by", "", "Sort results by field")
	rootCmd.PersistentFlags().BoolVar(&descFlag, "desc", false, "Sort in descending order")
	rootCmd.PersistentFlags().BoolVar(&latestFlag, "latest", false, "Shortcut for --sort-by created_time --desc --limit 1")
	rootCmd.PersistentFlags().IntVar(&recentFlag, "recent", 0, "Shortcut for --sort-by created_time --desc --limit N")

	// Flag aliases for agent ergonomics
	// Note: "json" already has -j via BoolP; no need for flagAlias.
	flagAlias(rootCmd.PersistentFlags(), "output", "out")
	flagAlias(rootCmd.PersistentFlags(), "query", "qr")
	flagAlias(rootCmd.PersistentFlags(), "fields", "fds")
	flagAlias(rootCmd.PersistentFlags(), "results-only", "ro")
	flagAlias(rootCmd.PersistentFlags(), "items-only", "io")
	flagAlias(rootCmd.PersistentFlags(), "items-only", "i")
	flagAlias(rootCmd.PersistentFlags(), "fail-empty", "fe")
	flagAlias(rootCmd.PersistentFlags(), "sort-by", "sb")
	flagAlias(rootCmd.PersistentFlags(), "query-file", "qf")
	flagAlias(rootCmd.PersistentFlags(), "compact-json", "cj")

	// Register subcommands
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newWorkspaceCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newBlockCmd())
	rootCmd.AddCommand(newDBCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newResolveCmd())
	rootCmd.AddCommand(newCommentCmd())
	rootCmd.AddCommand(newFileCmd())
	rootCmd.AddCommand(newDataSourceCmd())
	rootCmd.AddCommand(newFetchCmd())
	rootCmd.AddCommand(newWebhookCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newAPICmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newWorkersCmd())
	rootCmd.AddCommand(newBulkCmd())
	rootCmd.AddCommand(newSkillCmd())

	// Top-level convenience commands (desire-path aliases)
	var loginNoBrowser bool
	loginAliasCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Notion (alias for 'auth login')",
		Long: `Authenticate with Notion using OAuth.

This is a convenience alias for 'ntn auth login'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOAuthLogin(cmd.Context(), loginNoBrowser)
		},
	}
	loginAliasCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "Do not auto-open browser; print auth URL instead")
	rootCmd.AddCommand(loginAliasCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials (alias for 'auth logout')",
		Long: `Remove the stored Notion credentials from the system keyring.

This is a convenience alias for 'ntn auth logout'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := auth.DeleteToken(); err != nil {
				return fmt.Errorf("failed to remove token: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, map[string]interface{}{
				"status":  "success",
				"message": "Logged out successfully",
			})
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "whoami",
		Short: "Show current user (alias for 'user me')",
		Long: `Retrieve the bot user associated with the API token.

This is a convenience alias for 'ntn user me'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}
			user, err := client.GetSelf(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, user)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:     "open <page-id-or-name>",
		Aliases: []string{"o"},
		Short:   "Open a Notion page in the browser",
		Long: `Open a Notion page in your default web browser.

Accepts a page ID, skill file alias, or page name.

Example:
  ntn open abc123
  ntn open my-page-alias
  ntn open "Meeting Notes"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Resolve ID with search fallback
			pageID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			pageID, err = cmdutil.NormalizeNotionID(pageID)
			if err != nil {
				return err
			}

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return wrapAPIError(err, "get page", "page", args[0])
			}

			// Open in browser
			var openCmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				openCmd = exec.Command("open", page.URL)
			case "linux":
				openCmd = exec.Command("xdg-open", page.URL)
			case "windows":
				openCmd = exec.Command("cmd", "/c", "start", page.URL)
			default:
				return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
			}
			if err := openCmd.Run(); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			_, _ = fmt.Fprintf(stderrFromContext(ctx), "Opened %s\n", page.URL)
			return nil
		},
	})

	// Action-first top-level commands (agent-friendly desire paths)

	// `ntn list` → search for pages
	rootCmd.AddCommand(newPageListCmd())

	// `ntn get <id>` → auto-detect entity type
	rootCmd.AddCommand(&cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Get any Notion object by ID or name (auto-detects type)",
		Long: `Retrieve a Notion page, database, or block by its ID or name.

If you provide a name instead of an ID, the CLI will search for matching objects.
Automatically detects the object type by trying page first, then database,
then block. This is useful when you have an ID but don't know its type.

Example:
  ntn get abc123
  ntn get my-page-alias
  ntn get "Meeting Notes"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}
			printer := printerForContext(ctx)

			// Resolve ID with search fallback (no filter - could be page or database)
			id, err := resolveIDWithSearch(ctx, client, sf, args[0], "")
			if err != nil {
				return err
			}
			id, err = cmdutil.NormalizeNotionID(id)
			if err != nil {
				return err
			}

			// Try page first (most common)
			page, pageErr := client.GetPage(ctx, id)
			if pageErr == nil {
				return printer.Print(ctx, page)
			}

			// Try database
			db, dbErr := client.GetDatabase(ctx, id)
			if dbErr == nil {
				return printer.Print(ctx, db)
			}

			// Try block
			block, blockErr := client.GetBlock(ctx, id)
			if blockErr == nil {
				return printer.Print(ctx, block)
			}

			// All failed - show helpful error with suggestions
			return errors.WrapUserError(
				fmt.Errorf("tried page, database, and block APIs"),
				fmt.Sprintf("could not find object %q", args[0]),
				fmt.Sprintf("Suggestions:\n  • Run 'ntn search %s' to find matching pages or databases\n  • Check the ID or name is correct\n  • Verify your integration has access to this object", args[0]),
			)
		},
	})

	// `ntn create <title>` → create page with smart defaults
	rootCmd.AddCommand(&cobra.Command{
		Use:   "create <title>",
		Short: "Create a page with smart defaults",
		Long: `Create a new page using the first database from your skill file.

This is a convenience command for quick page creation. It uses the first
database configured in your skill file (~/.claude/skills/notion-cli/notion-cli.md).

If no databases are configured, run 'ntn skill init' first.

Example:
  ntn create "My new page"
  ntn create "Meeting notes for today"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			title := args[0]

			// Get the first database from skill file
			if sf == nil || len(sf.Databases) == 0 {
				return errors.NoDatabaseConfiguredError()
			}

			// Get first database (sorted alphabetically for consistency)
			var firstDB *skill.DatabaseAlias
			for _, db := range sf.Databases {
				if firstDB == nil || db.Alias < firstDB.Alias {
					dbCopy := db
					firstDB = &dbCopy
				}
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Build properties based on database config
			titleProp := "Name"
			if firstDB.TitleProperty != "" {
				titleProp = firstDB.TitleProperty
			}

			properties := map[string]interface{}{
				titleProp: map[string]interface{}{
					"title": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]interface{}{
								"content": title,
							},
						},
					},
				},
			}

			// Add default status if configured
			if firstDB.DefaultStatus != "" {
				properties["Status"] = map[string]interface{}{
					"status": map[string]interface{}{
						"name": firstDB.DefaultStatus,
					},
				}
			}

			// Resolve the data source ID for the database
			resolvedDataSourceID, err := resolveDataSourceID(ctx, client, firstDB.ID, "")
			if err != nil {
				return err
			}

			req := &notion.CreatePageRequest{
				Parent: map[string]interface{}{
					"type":           "data_source_id",
					"data_source_id": resolvedDataSourceID,
				},
				Properties: properties,
			}

			page, err := client.CreatePage(ctx, req)
			if err != nil {
				return wrapAPIError(err, "create page", "database", firstDB.Alias)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	})

	// `ntn delete <id>` → delete (archive) a page
	rootCmd.AddCommand(&cobra.Command{
		Use:     "delete <page-id-or-alias>",
		Aliases: []string{"rm", "d"},
		Short:   "Archive a page (alias for 'page delete')",
		Long: `Archive a Notion page by its ID or skill file alias.

This is a convenience alias for 'ntn page delete'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deleteCmd := newPageDeleteCmd()
			deleteCmd.SetContext(cmd.Context())
			return deleteCmd.RunE(deleteCmd, args)
		},
	})

	// Canonical additive verb aliases for cross-CLI consistency.
	applyCanonicalVerbAliases(rootCmd)
	// Ensure every non-root command has at least one short alias (without sibling collisions).
	applyDefaultCommandAliases(rootCmd)
	// Add safe shorthand aliases (-x) to visible flags where possible.
	applyDefaultFlagShorthands(rootCmd)
	// Use a curated root help menu optimized for humans and agents.
	installRootHelp(rootCmd)

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

	// Allows tests and proxies to override the Notion API base URL.
	// Precedence:
	// 1) NOTION_API_BASE_URL env var
	// 2) workspace api_url in config.yaml (selected workspace or default)
	if baseURL := strings.TrimSpace(os.Getenv("NOTION_API_BASE_URL")); baseURL != "" {
		client.WithBaseURL(baseURL)
	} else {
		cfg := ConfigFromContext(ctx)
		if cfg == nil {
			// Backward compatibility for tests/direct calls that bypass root pre-run.
			cfg, _ = config.Load()
		}
		if cfg != nil {
			wsName := WorkspaceFromContext(ctx)
			var ws *config.WorkspaceConfig
			if wsName != "" {
				ws, _ = cfg.GetWorkspace(wsName)
			} else {
				ws, _ = cfg.GetDefaultWorkspace()
			}
			if ws != nil && strings.TrimSpace(ws.APIURL) != "" {
				client.WithBaseURL(ws.APIURL)
			}
		}
	}
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

func installRootHelp(root *cobra.Command) {
	defaultHelp := root.HelpFunc()

	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != root {
			defaultHelp(cmd, args)
			return
		}

		_, _ = fmt.Fprint(cmd.OutOrStdout(), rootHelpText)
	})
}
