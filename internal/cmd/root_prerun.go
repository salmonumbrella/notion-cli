package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/config"
	"github.com/salmonumbrella/notion-cli/internal/debug"
	"github.com/salmonumbrella/notion-cli/internal/iocontext"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/ui"
)

type globalFlagInput struct {
	workspaceName   string
	queryFlag       string
	jqFlag          string
	fieldsFlag      string
	pickFlag        string
	jsonPathFlag    string
	quietFlag       bool
	failEmptyFlag   bool
	compactJSON     bool
	latestFlag      bool
	recentFlag      int
	yesFlag         bool
	limitFlag       int
	sortBy          string
	descFlag        bool
	resultsOnlyFlag bool
	errorFormat     string
}

type globalOptions struct {
	workspace       string
	format          output.Format
	query           string
	queryNormalized bool
	fieldsRaw       string
	jsonPathRaw     string
	quiet           bool
	failEmpty       bool
	compactJSON     bool
	light           bool
	yes             bool
	limit           int
	sortBy          string
	desc            bool
	resultsOnly     bool
	errorFormat     string

	queryFlagSet     bool
	jqFlagSet        bool
	queryFileFlagSet bool
	fieldsFlagSet    bool
	pickFlagSet      bool
	recentFlagSet    bool
	limitFlagSet     bool
	sortByFlagSet    bool
	descFlagSet      bool
	compactJSONSet   bool
	outputFlagSet    bool
	formatFlagSet    bool
	jsonFlagSet      bool
	latestFlag       bool
	recentFlag       int
}

func parseGlobalOptions(cmd *cobra.Command, cfg *config.Config, stdout io.Writer, flags globalFlagInput) (globalOptions, error) {
	opts := globalOptions{
		workspace:   flags.workspaceName,
		quiet:       flags.quietFlag,
		failEmpty:   flags.failEmptyFlag,
		compactJSON: flags.compactJSON,
		yes:         flags.yesFlag,
		limit:       flags.limitFlag,
		sortBy:      flags.sortBy,
		desc:        flags.descFlag,
		resultsOnly: flags.resultsOnlyFlag,
		errorFormat: flags.errorFormat,

		queryFlagSet:   strings.TrimSpace(flags.queryFlag) != "",
		jqFlagSet:      strings.TrimSpace(flags.jqFlag) != "",
		fieldsFlagSet:  strings.TrimSpace(flags.fieldsFlag) != "",
		pickFlagSet:    strings.TrimSpace(flags.pickFlag) != "",
		recentFlagSet:  cmd.Flags().Changed("recent"),
		limitFlagSet:   cmd.Flags().Changed("limit"),
		sortByFlagSet:  cmd.Flags().Changed("sort-by"),
		descFlagSet:    cmd.Flags().Changed("desc"),
		compactJSONSet: commandFlagChanged(cmd, "compact-json") || commandFlagChanged(cmd, "cj"),
		outputFlagSet:  commandFlagChanged(cmd, "output") || commandFlagChanged(cmd, "out"),
		formatFlagSet:  commandFlagChanged(cmd, "format"),
		jsonFlagSet:    commandFlagChanged(cmd, "json"),
		latestFlag:     flags.latestFlag,
		recentFlag:     flags.recentFlag,
	}

	lightValue, hasLightFlag := commandBoolFlagValue(cmd, "light")
	opts.light = hasLightFlag && lightValue

	if opts.workspace == "" {
		opts.workspace = os.Getenv("NOTION_WORKSPACE")
	}

	formatStr, _ := cmd.Flags().GetString("output")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		formatStr = "json"
	} else if opts.formatFlagSet {
		formatStr, _ = cmd.Flags().GetString("format")
	} else if !opts.outputFlagSet && strings.TrimSpace(os.Getenv("NOTION_OUTPUT")) != "" {
		formatStr = os.Getenv("NOTION_OUTPUT")
	} else if !opts.outputFlagSet && cfg.GetOutput() != "" {
		formatStr = cfg.GetOutput()
	} else if !opts.outputFlagSet && !isTerminal(stdout) {
		formatStr = string(output.FormatJSON)
	}

	format, err := output.ParseFormat(formatStr)
	if err != nil {
		return globalOptions{}, err
	}
	opts.format = format

	if opts.light && opts.format == output.FormatText && !opts.outputFlagSet && !opts.formatFlagSet && !opts.jsonFlagSet {
		opts.format = output.FormatJSON
	}
	if opts.light && !opts.compactJSONSet {
		opts.compactJSON = true
	}

	if !cmd.Flags().Changed("quiet") && !isTerminal(stdout) {
		switch opts.format {
		case output.FormatJSON, output.FormatNDJSON, output.FormatYAML:
			opts.quiet = true
		}
	}

	opts.query = flags.queryFlag
	if opts.query == "" {
		opts.query = flags.jqFlag
	}

	queryFileFlag, _ := cmd.Flags().GetString("query-file")
	opts.queryFileFlagSet = strings.TrimSpace(queryFileFlag) != ""
	if opts.queryFileFlagSet {
		loaded, err := cmdutil.ReadInputSource(queryFileFlag)
		if err != nil {
			return globalOptions{}, err
		}
		opts.query = loaded
	}

	opts.query, opts.queryNormalized = output.NormalizeQuery(opts.query)

	opts.fieldsRaw = strings.TrimSpace(flags.fieldsFlag)
	if opts.fieldsRaw == "" {
		opts.fieldsRaw = strings.TrimSpace(flags.pickFlag)
	}
	opts.jsonPathRaw = strings.TrimSpace(flags.jsonPathFlag)

	return opts, nil
}

func validateGlobalOptions(opts *globalOptions) error {
	if opts.jqFlagSet && opts.queryFlagSet {
		return errOnlyOne("--query", "--jq")
	}
	if opts.queryFileFlagSet && (opts.jqFlagSet || opts.queryFlagSet) {
		return errOnlyOne("--query/--jq", "--query-file")
	}
	if opts.fieldsFlagSet && opts.pickFlagSet {
		return errOnlyOne("--fields", "--pick")
	}
	if opts.fieldsRaw != "" {
		if err := output.ValidateFields(opts.fieldsRaw); err != nil {
			return err
		}
	}
	if opts.query != "" && (opts.fieldsRaw != "" || opts.jsonPathRaw != "") {
		return errOnlyOne("--query/--jq/--query-file", "--fields/--pick, or --jsonpath")
	}
	if opts.fieldsRaw != "" && opts.jsonPathRaw != "" {
		return errOnlyOne("--fields/--pick", "--jsonpath")
	}
	if opts.recentFlagSet && opts.recentFlag <= 0 {
		return fmt.Errorf("--recent must be >= 1")
	}
	if opts.latestFlag && opts.recentFlag > 0 {
		return errOnlyOne("--latest", "--recent")
	}
	if opts.latestFlag {
		opts.recentFlag = 1
	}
	if opts.recentFlag > 0 {
		if opts.limitFlagSet || opts.sortByFlagSet || opts.descFlagSet {
			return fmt.Errorf("--latest/--recent are shortcuts for --sort-by created_time --desc --limit N; do not combine with --sort-by/--desc/--limit")
		}
		opts.limit = opts.recentFlag
		opts.sortBy = "created_time"
		opts.desc = true
	}
	if err := validateErrorFormat(opts.errorFormat); err != nil {
		return err
	}
	return nil
}

func buildRootContext(ctx context.Context, app *App, cfg *config.Config, debugMode bool, opts globalOptions) context.Context {
	ctx = iocontext.WithIO(ctx, app.Stdout, app.Stderr)
	ctx = output.WithFormat(ctx, opts.format)
	ctx = output.WithQuery(ctx, opts.query)
	ctx = debug.WithDebug(ctx, debugMode)
	ctx = WithWorkspace(ctx, opts.workspace)
	ctx = WithConfig(ctx, cfg)

	ctx = output.WithYes(ctx, opts.yes)
	ctx = output.WithLimit(ctx, opts.limit)
	ctx = output.WithSort(ctx, opts.sortBy, opts.desc)
	ctx = output.WithQuiet(ctx, opts.quiet)
	ctx = output.WithFields(ctx, opts.fieldsRaw)
	ctx = output.WithJSONPath(ctx, opts.jsonPathRaw)
	ctx = output.WithFailEmpty(ctx, opts.failEmpty)
	ctx = output.WithResultsOnly(ctx, opts.resultsOnly)
	ctx = output.WithLight(ctx, opts.light)
	ctx = output.WithCompactJSON(ctx, opts.compactJSON)
	ctx = WithErrorFormat(ctx, opts.errorFormat)
	ctx = ui.WithUI(ctx, ui.New(parseColorMode(cfg.GetColor())))
	return ctx
}

func errOnlyOne(left, right string) error {
	return fmt.Errorf("use only one of %s or %s", left, right)
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}

	for current := cmd; current != nil; current = current.Parent() {
		if flag := current.Flags().Lookup(name); flag != nil && flag.Changed {
			return true
		}
		if flag := current.PersistentFlags().Lookup(name); flag != nil && flag.Changed {
			return true
		}
	}
	return false
}

func commandBoolFlagValue(cmd *cobra.Command, name string) (bool, bool) {
	if cmd == nil {
		return false, false
	}

	if cmd.Flags().Lookup(name) != nil {
		v, err := cmd.Flags().GetBool(name)
		if err == nil {
			return v, true
		}
	}

	if cmd.InheritedFlags().Lookup(name) != nil {
		v, err := cmd.InheritedFlags().GetBool(name)
		if err == nil {
			return v, true
		}
	}

	return false, false
}
