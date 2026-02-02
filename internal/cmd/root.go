package cmd

import (
	"context"
	"fmt"
	"io"
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

	// Top-level convenience commands (desire-path aliases)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Authenticate with Notion (alias for 'auth login')",
		Long: `Authenticate with Notion using OAuth.

This is a convenience alias for 'notion auth login'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOAuthLogin(cmd.Context())
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials (alias for 'auth logout')",
		Long: `Remove the stored Notion credentials from the system keyring.

This is a convenience alias for 'notion auth logout'.`,
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

This is a convenience alias for 'notion user me'.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)
			user, err := client.GetSelf(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, user)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "open <page-id-or-alias>",
		Short: "Open a Notion page in the browser",
		Long: `Open a Notion page in your default web browser.

Accepts a page ID or a skill file alias.

Example:
  notion open abc123
  notion open my-page-alias`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			pageID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)
			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return fmt.Errorf("failed to get page: %w", err)
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

	// `notion list` → search for pages
	rootCmd.AddCommand(&cobra.Command{
		Use:     "list [query]",
		Aliases: []string{"ls"},
		Short:   "List pages (alias for 'search --filter page')",
		Long: `Search for pages in Notion.

This is a convenience alias for 'notion search --filter page'.

Example:
  notion list
  notion list "project"
  notion ls meetings`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Delegate to search with page filter
			searchCmd := newSearchCmd()
			searchCmd.SetContext(cmd.Context())
			if err := searchCmd.Flags().Set("filter", "page"); err != nil {
				return err
			}
			return searchCmd.RunE(searchCmd, args)
		},
	})

	// `notion get <id>` → auto-detect entity type
	rootCmd.AddCommand(&cobra.Command{
		Use:   "get <id-or-alias>",
		Short: "Get any Notion object by ID (auto-detects type)",
		Long: `Retrieve a Notion page, database, or block by its ID.

Automatically detects the object type by trying page first, then database,
then block. This is useful when you have an ID but don't know its type.

Example:
  notion get abc123
  notion get my-page-alias`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			id, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)
			printer := printerForContext(ctx)

			// Try page first (most common)
			page, err := client.GetPage(ctx, id)
			if err == nil {
				return printer.Print(ctx, page)
			}

			// Try database
			db, err := client.GetDatabase(ctx, id)
			if err == nil {
				return printer.Print(ctx, db)
			}

			// Try block
			block, err := client.GetBlock(ctx, id)
			if err == nil {
				return printer.Print(ctx, block)
			}

			// All attempts failed
			return fmt.Errorf("could not find page, database, or block with ID %q", id)
		},
	})

	// `notion create <title>` → create page with smart defaults
	rootCmd.AddCommand(&cobra.Command{
		Use:   "create <title>",
		Short: "Create a page with smart defaults",
		Long: `Create a new page using the first database from your skill file.

This is a convenience command for quick page creation. It uses the first
database configured in your skill file (~/.claude/skills/notion-cli/notion-cli.md).

If no databases are configured, run 'notion skill init' first.

Example:
  notion create "My new page"
  notion create "Meeting notes for today"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			title := args[0]

			// Get the first database from skill file
			if sf == nil || len(sf.Databases) == 0 {
				return fmt.Errorf("no databases configured in skill file\n\nRun 'notion skill init' to set up database aliases, or use:\n  notion page create --parent <database-id> --parent-type database --properties '{...}'")
			}

			// Get first database (sorted alphabetically for consistency)
			var firstDB *skill.DatabaseAlias
			for _, db := range sf.Databases {
				if firstDB == nil || db.Alias < firstDB.Alias {
					dbCopy := db
					firstDB = &dbCopy
				}
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

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
				return fmt.Errorf("failed to create page: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	})

	// `notion delete <id>` → delete (archive) a page
	rootCmd.AddCommand(&cobra.Command{
		Use:     "delete <page-id-or-alias>",
		Aliases: []string{"rm", "archive"},
		Short:   "Archive a page (alias for 'page delete')",
		Long: `Archive a Notion page by its ID.

This is a convenience alias for 'notion page delete'.
Archived pages can be restored from the Notion UI.

Example:
  notion delete abc123
  notion rm my-page-alias
  notion archive old-page`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			pageID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			page, err := client.UpdatePage(ctx, pageID, &notion.UpdatePageRequest{
				Archived: ptrBool(true),
			})
			if err != nil {
				return fmt.Errorf("failed to archive page: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	})

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
