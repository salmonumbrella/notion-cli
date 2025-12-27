package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/output"
	workerscfg "github.com/salmonumbrella/notion-cli/internal/workers"
)

const (
	fallbackWorkersCLIVersion      = "0.1.35"
	fallbackWorkersTemplateRepo    = "https://github.com/makenotion/workers-template"
	workersCLIVersionEnvVar        = "NTN_WORKERS_CLI_VERSION"
	workersNPXBinEnvVar            = "NTN_WORKERS_NPX_BIN"
	workersTemplateMetadataFile    = ".ntn-workers-template.json"
	defaultWorkersProjectDirectory = "my-worker"
	workersTemplateHTTPTimeout     = 30 * time.Second
	workersTemplateMaxFileSize     = 10 << 20 // 10 MiB per extracted file
)

type workersRuntime interface {
	lookPath(file string) (string, error)
	run(ctx context.Context, bin string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
	output(ctx context.Context, bin string, args []string) (string, error)
}

type defaultWorkersRuntime struct{}

func (defaultWorkersRuntime) lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (defaultWorkersRuntime) run(ctx context.Context, bin string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	c := exec.CommandContext(ctx, bin, args...)
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			if code <= 0 {
				code = ExitSystem
			}
			return &proxiedCommandExitError{Code: code}
		}
		return err
	}
	return nil
}

func (defaultWorkersRuntime) output(ctx context.Context, bin string, args []string) (string, error) {
	c := exec.CommandContext(ctx, bin, args...)
	out, err := c.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}

var workersRuntimeImpl workersRuntime = defaultWorkersRuntime{}

// workersScaffoldFunc is overridable in tests.
var workersScaffoldFunc = scaffoldWorkersTemplate

// workersUpgradeFunc is overridable in tests.
var workersUpgradeFunc = upgradeWorkersTemplate

// workersStatusFunc is overridable in tests.
var workersStatusFunc = collectWorkersStatus

// workersCompareFunc is overridable in tests.
var workersCompareFunc = compareGitHubCommits

type workersCommandInvocation struct {
	bin  string
	args []string
}

type workersDoctorCheck struct {
	Name   string `json:"name" yaml:"name"`
	Status string `json:"status" yaml:"status"`
	Detail string `json:"detail" yaml:"detail"`
}

type workersDoctorReport struct {
	OK             bool                 `json:"ok" yaml:"ok"`
	NPXBin         string               `json:"npx_bin" yaml:"npx_bin"`
	CLIVersion     string               `json:"official_cli_version" yaml:"official_cli_version"`
	TemplateRepo   string               `json:"template_repo" yaml:"template_repo"`
	TemplateCommit string               `json:"template_commit" yaml:"template_commit"`
	Checks         []workersDoctorCheck `json:"checks" yaml:"checks"`
}

type workersNewOptions struct {
	Path   string
	Force  bool
	Repo   string
	Commit string
}

type workersUpgradeOptions struct {
	Path         string
	Repo         string
	Commit       string
	Force        bool
	DryRun       bool
	Plan         bool
	FromMetadata bool
}

type workersStatusOptions struct {
	Path      string
	NoCompare bool
}

type workersScaffoldResult struct {
	Path            string
	TemplateRepo    string
	TemplateCommit  string
	OfficialVersion string
	MetadataFile    string
	TarballURL      string
}

type workersUpgradeResult struct {
	Path               string
	TemplateRepo       string
	TemplateCommit     string
	PreviousCommit     string
	OfficialVersion    string
	MetadataFile       string
	TarballURL         string
	MetadataSourceFile string
	DryRun             bool
	Plan               *workersUpgradePlan
}

type workersUpgradePlan struct {
	AddCount       int      `json:"add_count" yaml:"add_count"`
	ModifyCount    int      `json:"modify_count" yaml:"modify_count"`
	UnchangedCount int      `json:"unchanged_count" yaml:"unchanged_count"`
	ExtraCount     int      `json:"extra_count" yaml:"extra_count"`
	AddPaths       []string `json:"add_paths,omitempty" yaml:"add_paths,omitempty"`
	ModifyPaths    []string `json:"modify_paths,omitempty" yaml:"modify_paths,omitempty"`
	ExtraPaths     []string `json:"extra_paths,omitempty" yaml:"extra_paths,omitempty"`
}

type workersStatusResult struct {
	Path                  string `json:"path" yaml:"path"`
	MetadataFile          string `json:"metadata_file" yaml:"metadata_file"`
	MetadataExists        bool   `json:"metadata_exists" yaml:"metadata_exists"`
	PinnedCLIVersion      string `json:"pinned_cli_version" yaml:"pinned_cli_version"`
	EffectiveCLIVersion   string `json:"effective_cli_version" yaml:"effective_cli_version"`
	CLIVersionOverridden  bool   `json:"cli_version_overridden" yaml:"cli_version_overridden"`
	PinnedTemplateRepo    string `json:"pinned_template_repo" yaml:"pinned_template_repo"`
	PinnedTemplateCommit  string `json:"pinned_template_commit" yaml:"pinned_template_commit"`
	ProjectTemplateRepo   string `json:"project_template_repo,omitempty" yaml:"project_template_repo,omitempty"`
	ProjectTemplateCommit string `json:"project_template_commit,omitempty" yaml:"project_template_commit,omitempty"`
	ProjectScaffoldedAt   string `json:"project_scaffolded_at,omitempty" yaml:"project_scaffolded_at,omitempty"`
	SyncState             string `json:"sync_state" yaml:"sync_state"`
	AheadBy               int    `json:"ahead_by,omitempty" yaml:"ahead_by,omitempty"`
	BehindBy              int    `json:"behind_by,omitempty" yaml:"behind_by,omitempty"`
	CompareURL            string `json:"compare_url,omitempty" yaml:"compare_url,omitempty"`
	CompareError          string `json:"compare_error,omitempty" yaml:"compare_error,omitempty"`
}

type workersTemplateMetadata struct {
	TemplateRepo       string `json:"template_repo"`
	TemplateCommit     string `json:"template_commit"`
	OfficialCLIVersion string `json:"official_cli_version"`
	ScaffoldedAt       string `json:"scaffolded_at"`
	TarballURL         string `json:"tarball_url"`
}

func newWorkersCmd() *cobra.Command {
	defaultVersion := workersCLIVersionDefault()
	cmd := &cobra.Command{
		Use:     "workers [command/args...]",
		Aliases: []string{"wk"},
		Short:   "Proxy Notion Workers commands to the official CLI",
		Long: `Run Notion Workers commands via the official Notion CLI package
without replacing this binary.

By default, this command executes:
  npx --yes ntn@` + defaultVersion + ` workers ...

Local helpers:
  ntn workers doctor       Check local workers toolchain health
  ntn workers status [dir] Show pinned vs project template status
  ntn workers new [dir]    Scaffold a worker from pinned template commit
  ntn workers upgrade [dir] Plan with --plan, apply with --force

Overrides:
  ` + workersCLIVersionEnvVar + `  Override the pinned official CLI version
  ` + workersNPXBinEnvVar + `      Override the npx executable (default: npx)`,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if doctorArgs, ok, err := extractWorkersDoctorArgs(args); err != nil {
				return err
			} else if ok {
				return runWorkersDoctor(cmd.Context(), doctorArgs)
			}
			if statusArgs, ok, err := extractWorkersStatusArgs(args); err != nil {
				return err
			} else if ok {
				return runWorkersStatus(cmd.Context(), statusArgs)
			}
			if newArgs, ok, err := extractWorkersNewArgs(args); err != nil {
				return err
			} else if ok {
				return runWorkersNew(cmd.Context(), newArgs)
			}
			if upgradeArgs, ok, err := extractWorkersUpgradeArgs(args); err != nil {
				return err
			} else if ok {
				return runWorkersUpgrade(cmd.Context(), upgradeArgs)
			}
			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}
			return runWorkersPassthrough(cmd.Context(), args)
		},
	}
	return cmd
}

func runWorkersPassthrough(ctx context.Context, workersArgs []string) error {
	inv := buildWorkersCommandInvocation(workersArgs)
	if _, err := workersRuntimeImpl.lookPath(inv.bin); err != nil {
		return clierrors.NewUserError(
			fmt.Sprintf("%s not found in PATH", inv.bin),
			"Install Node.js (which includes npx) or set "+workersNPXBinEnvVar+" to a valid executable path.",
		)
	}

	if err := workersRuntimeImpl.run(ctx, inv.bin, inv.args, os.Stdin, stdoutFromContext(ctx), stderrFromContext(ctx)); err != nil {
		var proxiedErr *proxiedCommandExitError
		if errors.As(err, &proxiedErr) {
			return proxiedErr
		}
		return fmt.Errorf("failed to execute Notion Workers CLI: %w", err)
	}
	return nil
}

func runWorkersDoctor(ctx context.Context, args []string) error {
	ctxWithDoctorFlags, showHelp, err := parseWorkersDoctorArgs(ctx, args)
	if err != nil {
		return err
	}
	if showHelp {
		_, _ = fmt.Fprintln(stdoutFromContext(ctxWithDoctorFlags), "Usage: ntn workers doctor [--output json|yaml|text|table|ndjson|-j]")
		_, _ = fmt.Fprintln(stdoutFromContext(ctxWithDoctorFlags), "Checks local Node/npm/npx availability and pinned official CLI version.")
		return nil
	}

	templateRepo := workersTemplateRepoDefault()
	templateCommit := workersTemplateCommitDefault()

	report := workersDoctorReport{
		OK:             true,
		NPXBin:         workersNPXBin(),
		CLIVersion:     workersCLIVersion(),
		TemplateRepo:   templateRepo,
		TemplateCommit: templateCommit,
		Checks:         make([]workersDoctorCheck, 0, 4),
	}
	addCheck := func(name, status, detail string) {
		if strings.TrimSpace(detail) == "" {
			detail = "-"
		}
		report.Checks = append(report.Checks, workersDoctorCheck{
			Name:   name,
			Status: status,
			Detail: detail,
		})
		if status == "error" {
			report.OK = false
		}
	}

	npxPath, err := workersRuntimeImpl.lookPath(report.NPXBin)
	if err != nil {
		addCheck("npx", "error", err.Error())
	} else {
		addCheck("npx", "ok", npxPath)
	}

	checkCommandVersion := func(name, bin string, args []string) {
		out, err := workersRuntimeImpl.output(ctx, bin, args)
		if err != nil {
			detail := err.Error()
			if out != "" {
				detail = out
			}
			addCheck(name, "error", detail)
			return
		}
		if out == "" {
			out = "ok"
		}
		addCheck(name, "ok", out)
	}

	checkCommandVersion("node", "node", []string{"--version"})
	checkCommandVersion("npm", "npm", []string{"--version"})

	if npxPath == "" {
		addCheck("official-cli", "skipped", "npx unavailable")
	} else {
		checkCommandVersion(
			"official-cli",
			report.NPXBin,
			[]string{"--yes", fmt.Sprintf("ntn@%s", report.CLIVersion), "--version"},
		)
	}

	return printerForContext(ctxWithDoctorFlags).Print(ctxWithDoctorFlags, report)
}

func runWorkersStatus(ctx context.Context, args []string) error {
	ctxWithFlags, opts, showHelp, err := parseWorkersStatusArgs(ctx, args)
	if err != nil {
		return err
	}
	if showHelp {
		printWorkersStatusHelp(stdoutFromContext(ctxWithFlags))
		return nil
	}

	result, err := workersStatusFunc(ctxWithFlags, opts)
	if err != nil {
		return err
	}
	return printerForContext(ctxWithFlags).Print(ctxWithFlags, result)
}

func runWorkersNew(ctx context.Context, args []string) error {
	ctxWithFlags, opts, showHelp, err := parseWorkersNewArgs(ctx, args)
	if err != nil {
		return err
	}
	if showHelp {
		printWorkersNewHelp(stdoutFromContext(ctxWithFlags))
		return nil
	}

	result, err := workersScaffoldFunc(ctxWithFlags, opts)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"status":               "success",
		"path":                 result.Path,
		"template_repo":        result.TemplateRepo,
		"template_commit":      result.TemplateCommit,
		"official_cli_version": result.OfficialVersion,
		"metadata_file":        result.MetadataFile,
		"tarball_url":          result.TarballURL,
	}
	return printerForContext(ctxWithFlags).Print(ctxWithFlags, payload)
}

func runWorkersUpgrade(ctx context.Context, args []string) error {
	ctxWithFlags, opts, showHelp, err := parseWorkersUpgradeArgs(ctx, args)
	if err != nil {
		return err
	}
	if showHelp {
		printWorkersUpgradeHelp(stdoutFromContext(ctxWithFlags))
		return nil
	}

	result, err := workersUpgradeFunc(ctxWithFlags, opts)
	if err != nil {
		return err
	}

	status := "success"
	if result.Plan != nil {
		status = "plan"
	} else if result.DryRun {
		status = "dry-run"
	}
	payload := map[string]interface{}{
		"status":               status,
		"path":                 result.Path,
		"template_repo":        result.TemplateRepo,
		"template_commit":      result.TemplateCommit,
		"previous_commit":      result.PreviousCommit,
		"official_cli_version": result.OfficialVersion,
		"metadata_file":        result.MetadataFile,
		"metadata_source_file": result.MetadataSourceFile,
		"tarball_url":          result.TarballURL,
		"dry_run":              result.DryRun,
	}
	if result.Plan != nil {
		payload["plan"] = result.Plan
	}
	return printerForContext(ctxWithFlags).Print(ctxWithFlags, payload)
}

func consumeWorkersCommonFlag(ctx context.Context, args []string, i *int) (context.Context, bool, bool, error) {
	arg := args[*i]
	switch arg {
	case "-h", "--help":
		return ctx, true, true, nil
	case "-j", "--json":
		return output.WithFormat(ctx, output.FormatJSON), true, false, nil
	case "-o", "--output", "--format":
		if *i+1 >= len(args) {
			return ctx, false, false, clierrors.NewUserError(
				fmt.Sprintf("%s requires a value", arg),
				"Use one of: text, json, ndjson, jsonl, table, yaml",
			)
		}
		*i = *i + 1
		format, err := output.ParseFormat(args[*i])
		if err != nil {
			return ctx, false, false, err
		}
		return output.WithFormat(ctx, format), true, false, nil
	default:
		return ctx, false, false, nil
	}
}

func parseWorkersDoctorArgs(ctx context.Context, args []string) (context.Context, bool, error) {
	updated := ctx
	for i := 0; i < len(args); i++ {
		nextCtx, consumed, showHelp, err := consumeWorkersCommonFlag(updated, args, &i)
		if err != nil {
			return ctx, false, err
		}
		if consumed {
			updated = nextCtx
			if showHelp {
				return updated, true, nil
			}
			continue
		}
		return ctx, false, clierrors.NewUserError(
			fmt.Sprintf("unsupported workers doctor argument %q", args[i]),
			"Use only --help, -j, or --output/--format with 'ntn workers doctor'.",
		)
	}
	return updated, false, nil
}

func parseWorkersStatusArgs(ctx context.Context, args []string) (context.Context, workersStatusOptions, bool, error) {
	updated := ctx
	opts := workersStatusOptions{Path: "."}
	pathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		nextCtx, consumed, showHelp, err := consumeWorkersCommonFlag(updated, args, &i)
		if err != nil {
			return ctx, opts, false, err
		}
		if consumed {
			updated = nextCtx
			if showHelp {
				return updated, opts, true, nil
			}
			continue
		}

		switch arg {
		case "--no-compare":
			opts.NoCompare = true
		default:
			if strings.HasPrefix(arg, "-") {
				return ctx, opts, false, clierrors.NewUserError(
					fmt.Sprintf("unsupported workers status argument %q", arg),
					"Use --help to see supported workers status flags.",
				)
			}
			if pathSet {
				return ctx, opts, false, clierrors.NewUserError("too many path arguments for workers status", "Use only one path, e.g. 'ntn workers status .' or 'ntn workers status my-worker'.")
			}
			opts.Path = arg
			pathSet = true
		}
	}
	return updated, opts, false, nil
}

func parseWorkersNewArgs(ctx context.Context, args []string) (context.Context, workersNewOptions, bool, error) {
	updated := ctx
	opts := workersNewOptions{Path: defaultWorkersProjectDirectory}
	pathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		nextCtx, consumed, showHelp, err := consumeWorkersCommonFlag(updated, args, &i)
		if err != nil {
			return ctx, opts, false, err
		}
		if consumed {
			updated = nextCtx
			if showHelp {
				return updated, opts, true, nil
			}
			continue
		}

		switch arg {
		case "--force":
			opts.Force = true
		case "--repo":
			if i+1 >= len(args) {
				return ctx, opts, false, clierrors.NewUserError("--repo requires a value", "Example: --repo https://github.com/makenotion/workers-template")
			}
			i++
			opts.Repo = strings.TrimSpace(args[i])
		case "--commit":
			if i+1 >= len(args) {
				return ctx, opts, false, clierrors.NewUserError("--commit requires a value", "Example: --commit eaadb8490cd1b7540fd159a550b04409febfec95")
			}
			i++
			opts.Commit = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(arg, "-") {
				return ctx, opts, false, clierrors.NewUserError(
					fmt.Sprintf("unsupported workers new argument %q", arg),
					"Use --help to see supported workers new flags.",
				)
			}
			if pathSet {
				return ctx, opts, false, clierrors.NewUserError("too many path arguments for workers new", "Use only one target path, e.g. 'ntn workers new my-worker'.")
			}
			opts.Path = arg
			pathSet = true
		}
	}

	return updated, opts, false, nil
}

func parseWorkersUpgradeArgs(ctx context.Context, args []string) (context.Context, workersUpgradeOptions, bool, error) {
	updated := ctx
	opts := workersUpgradeOptions{Path: "."}
	pathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		nextCtx, consumed, showHelp, err := consumeWorkersCommonFlag(updated, args, &i)
		if err != nil {
			return ctx, opts, false, err
		}
		if consumed {
			updated = nextCtx
			if showHelp {
				return updated, opts, true, nil
			}
			continue
		}

		switch arg {
		case "--repo":
			if i+1 >= len(args) {
				return ctx, opts, false, clierrors.NewUserError("--repo requires a value", "Example: --repo https://github.com/makenotion/workers-template")
			}
			i++
			opts.Repo = strings.TrimSpace(args[i])
		case "--commit":
			if i+1 >= len(args) {
				return ctx, opts, false, clierrors.NewUserError("--commit requires a value", "Example: --commit eaadb8490cd1b7540fd159a550b04409febfec95")
			}
			i++
			opts.Commit = strings.TrimSpace(args[i])
		case "--force":
			opts.Force = true
		case "--dry-run":
			opts.DryRun = true
		case "--plan":
			opts.Plan = true
		case "--from-metadata":
			opts.FromMetadata = true
		default:
			if strings.HasPrefix(arg, "-") {
				return ctx, opts, false, clierrors.NewUserError(
					fmt.Sprintf("unsupported workers upgrade argument %q", arg),
					"Use --help to see supported workers upgrade flags.",
				)
			}
			if pathSet {
				return ctx, opts, false, clierrors.NewUserError("too many path arguments for workers upgrade", "Use only one target path, e.g. 'ntn workers upgrade .' or 'ntn workers upgrade my-worker'.")
			}
			opts.Path = arg
			pathSet = true
		}
	}

	return updated, opts, false, nil
}

func printWorkersNewHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: ntn workers new [path] [--force] [--repo <url>] [--commit <sha>] [-o json|-j]")
	_, _ = fmt.Fprintln(w, "Scaffold a Notion worker project from the pinned workers-template commit.")
	_, _ = fmt.Fprintf(w, "Default path: %s\n", defaultWorkersProjectDirectory)
}

func printWorkersStatusHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: ntn workers status [path] [--no-compare] [-o json|-j]")
	_, _ = fmt.Fprintln(w, "Show pinned compat versions and project metadata sync state.")
	_, _ = fmt.Fprintln(w, "Default path: .")
}

func printWorkersUpgradeHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: ntn workers upgrade [path] [--force] [--dry-run] [--plan] [--from-metadata] [--repo <url>] [--commit <sha>] [-o json|-j]")
	_, _ = fmt.Fprintln(w, "Upgrade an existing worker project using current pinned compat commit by default.")
	_, _ = fmt.Fprintln(w, "--plan computes a file-level upgrade preview against your current tree.")
	_, _ = fmt.Fprintln(w, "--from-metadata makes it re-sync exactly to the commit in .ntn-workers-template.json.")
	_, _ = fmt.Fprintln(w, "Default path: .")
}

func extractWorkersDoctorArgs(args []string) ([]string, bool, error) {
	return extractWorkersSubcommandArgs(args, "doctor")
}

func extractWorkersStatusArgs(args []string) ([]string, bool, error) {
	return extractWorkersSubcommandArgs(args, "status")
}

func extractWorkersNewArgs(args []string) ([]string, bool, error) {
	return extractWorkersSubcommandArgs(args, "new")
}

func extractWorkersUpgradeArgs(args []string) ([]string, bool, error) {
	return extractWorkersSubcommandArgs(args, "upgrade")
}

func extractWorkersSubcommandArgs(args []string, subcommand string) ([]string, bool, error) {
	prefix := make([]string, 0, 4)
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-j", "--json", "-h", "--help":
			prefix = append(prefix, args[i])
			i++
		case "-o", "--output", "--format":
			if i+1 >= len(args) {
				return nil, false, clierrors.NewUserError(
					fmt.Sprintf("%s requires a value", args[i]),
					"Use one of: text, json, ndjson, jsonl, table, yaml",
				)
			}
			prefix = append(prefix, args[i], args[i+1])
			i += 2
		default:
			goto donePrefix
		}
	}

donePrefix:
	if i >= len(args) || args[i] != subcommand {
		return nil, false, nil
	}
	subcommandArgs := append(prefix, args[i+1:]...)
	return subcommandArgs, true, nil
}

func buildWorkersCommandInvocation(workersArgs []string) workersCommandInvocation {
	args := []string{
		"--yes",
		fmt.Sprintf("ntn@%s", workersCLIVersion()),
		"workers",
	}
	args = append(args, workersArgs...)
	return workersCommandInvocation{
		bin:  workersNPXBin(),
		args: args,
	}
}

func workersCLIVersionDefault() string {
	cfg, err := workerscfg.Current()
	if err != nil || strings.TrimSpace(cfg.WorkersCLIVersion) == "" {
		return fallbackWorkersCLIVersion
	}
	return strings.TrimSpace(cfg.WorkersCLIVersion)
}

func workersTemplateRepoDefault() string {
	cfg, err := workerscfg.Current()
	if err != nil || strings.TrimSpace(cfg.TemplateRepo) == "" {
		return fallbackWorkersTemplateRepo
	}
	return strings.TrimSpace(cfg.TemplateRepo)
}

func workersTemplateCommitDefault() string {
	cfg, err := workerscfg.Current()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.TemplateCommit)
}

func workersCLIVersion() string {
	version := strings.TrimSpace(os.Getenv(workersCLIVersionEnvVar))
	if version == "" {
		return workersCLIVersionDefault()
	}
	return version
}

func workersNPXBin() string {
	bin := strings.TrimSpace(os.Getenv(workersNPXBinEnvVar))
	if bin == "" {
		return "npx"
	}
	return bin
}

func scaffoldWorkersTemplate(ctx context.Context, opts workersNewOptions) (workersScaffoldResult, error) {
	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		repo = workersTemplateRepoDefault()
	}
	commit := strings.TrimSpace(opts.Commit)
	if commit == "" {
		commit = workersTemplateCommitDefault()
	}
	if commit == "" {
		return workersScaffoldResult{}, clierrors.NewUserError(
			"workers template commit is not configured",
			"Update internal/workers/compat.json with a valid template_commit.",
		)
	}

	target := strings.TrimSpace(opts.Path)
	if target == "" {
		target = defaultWorkersProjectDirectory
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return workersScaffoldResult{}, fmt.Errorf("resolve target path: %w", err)
	}
	if err := prepareWorkersTargetDir(absTarget, opts.Force); err != nil {
		return workersScaffoldResult{}, err
	}

	tarballURL, err := workersTemplateTarballURL(repo, commit)
	if err != nil {
		return workersScaffoldResult{}, err
	}
	if err := downloadAndExtractWorkersTemplate(ctx, tarballURL, absTarget, opts.Force); err != nil {
		return workersScaffoldResult{}, err
	}

	meta := workersTemplateMetadata{
		TemplateRepo:       repo,
		TemplateCommit:     commit,
		OfficialCLIVersion: workersCLIVersion(),
		ScaffoldedAt:       time.Now().UTC().Format(time.RFC3339),
		TarballURL:         tarballURL,
	}
	metadataPath, err := writeWorkersTemplateMetadata(absTarget, meta)
	if err != nil {
		return workersScaffoldResult{}, err
	}

	return workersScaffoldResult{
		Path:            absTarget,
		TemplateRepo:    repo,
		TemplateCommit:  commit,
		OfficialVersion: workersCLIVersion(),
		MetadataFile:    metadataPath,
		TarballURL:      tarballURL,
	}, nil
}

func upgradeWorkersTemplate(ctx context.Context, opts workersUpgradeOptions) (workersUpgradeResult, error) {
	target := strings.TrimSpace(opts.Path)
	if target == "" {
		target = "."
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return workersUpgradeResult{}, fmt.Errorf("resolve target path: %w", err)
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return workersUpgradeResult{}, clierrors.NewUserError(
				fmt.Sprintf("target directory %q does not exist", absTarget),
				"Run 'ntn workers new <path>' first, or create the directory and scaffold it.",
			)
		}
		return workersUpgradeResult{}, fmt.Errorf("stat target dir: %w", err)
	}
	if !info.IsDir() {
		return workersUpgradeResult{}, clierrors.NewUserError(
			fmt.Sprintf("target path %q exists and is not a directory", absTarget),
			"Choose a worker project directory.",
		)
	}

	metadataSourcePath := filepath.Join(absTarget, workersTemplateMetadataFile)
	sourceMeta, err := readWorkersTemplateMetadata(metadataSourcePath)
	if err != nil {
		return workersUpgradeResult{}, err
	}

	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		if opts.FromMetadata {
			repo = sourceMeta.TemplateRepo
		} else {
			repo = workersTemplateRepoDefault()
		}
	}
	if repo == "" {
		repo = sourceMeta.TemplateRepo
	}

	commit := strings.TrimSpace(opts.Commit)
	if commit == "" {
		if opts.FromMetadata {
			commit = sourceMeta.TemplateCommit
		} else {
			commit = workersTemplateCommitDefault()
		}
	}
	if commit == "" {
		commit = sourceMeta.TemplateCommit
	}
	if commit == "" {
		return workersUpgradeResult{}, clierrors.NewUserError(
			"workers template commit is not configured",
			"Set --commit, or ensure .ntn-workers-template.json has template_commit.",
		)
	}

	tarballURL, err := workersTemplateTarballURL(repo, commit)
	if err != nil {
		return workersUpgradeResult{}, err
	}

	if opts.Plan {
		plan, err := buildWorkersUpgradePlan(ctx, tarballURL, absTarget)
		if err != nil {
			return workersUpgradeResult{}, err
		}
		return workersUpgradeResult{
			Path:               absTarget,
			TemplateRepo:       repo,
			TemplateCommit:     commit,
			PreviousCommit:     sourceMeta.TemplateCommit,
			OfficialVersion:    workersCLIVersion(),
			MetadataFile:       metadataSourcePath,
			TarballURL:         tarballURL,
			MetadataSourceFile: metadataSourcePath,
			DryRun:             true,
			Plan:               &plan,
		}, nil
	}

	if opts.DryRun {
		return workersUpgradeResult{
			Path:               absTarget,
			TemplateRepo:       repo,
			TemplateCommit:     commit,
			PreviousCommit:     sourceMeta.TemplateCommit,
			OfficialVersion:    workersCLIVersion(),
			MetadataFile:       metadataSourcePath,
			TarballURL:         tarballURL,
			MetadataSourceFile: metadataSourcePath,
			DryRun:             true,
			Plan:               nil,
		}, nil
	}
	if !opts.Force {
		return workersUpgradeResult{}, clierrors.NewUserError(
			"workers upgrade requires --force to apply template changes",
			"Run with --dry-run to preview target commit, then re-run with --force to apply.",
		)
	}

	if err := downloadAndExtractWorkersTemplate(ctx, tarballURL, absTarget, opts.Force); err != nil {
		return workersUpgradeResult{}, err
	}

	meta := workersTemplateMetadata{
		TemplateRepo:       repo,
		TemplateCommit:     commit,
		OfficialCLIVersion: workersCLIVersion(),
		ScaffoldedAt:       time.Now().UTC().Format(time.RFC3339),
		TarballURL:         tarballURL,
	}
	metadataPath, err := writeWorkersTemplateMetadata(absTarget, meta)
	if err != nil {
		return workersUpgradeResult{}, err
	}

	return workersUpgradeResult{
		Path:               absTarget,
		TemplateRepo:       repo,
		TemplateCommit:     commit,
		PreviousCommit:     sourceMeta.TemplateCommit,
		OfficialVersion:    workersCLIVersion(),
		MetadataFile:       metadataPath,
		TarballURL:         tarballURL,
		MetadataSourceFile: metadataSourcePath,
		DryRun:             false,
		Plan:               nil,
	}, nil
}

type workersGitHubCompareResponse struct {
	Status   string `json:"status"`
	AheadBy  int    `json:"ahead_by"`
	BehindBy int    `json:"behind_by"`
	HTMLURL  string `json:"html_url"`
}

type workersPlanEntry struct {
	Kind string
	Hash string
	Link string
}

func collectWorkersStatus(ctx context.Context, opts workersStatusOptions) (workersStatusResult, error) {
	target := strings.TrimSpace(opts.Path)
	if target == "" {
		target = "."
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return workersStatusResult{}, fmt.Errorf("resolve target path: %w", err)
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return workersStatusResult{}, clierrors.NewUserError(
				fmt.Sprintf("target directory %q does not exist", absTarget),
				"Provide a valid worker project path.",
			)
		}
		return workersStatusResult{}, fmt.Errorf("stat target dir: %w", err)
	}
	if !info.IsDir() {
		return workersStatusResult{}, clierrors.NewUserError(
			fmt.Sprintf("target path %q exists and is not a directory", absTarget),
			"Provide a worker project directory.",
		)
	}

	result := workersStatusResult{
		Path:                 absTarget,
		MetadataFile:         filepath.Join(absTarget, workersTemplateMetadataFile),
		MetadataExists:       false,
		PinnedCLIVersion:     workersCLIVersionDefault(),
		EffectiveCLIVersion:  workersCLIVersion(),
		PinnedTemplateRepo:   workersTemplateRepoDefault(),
		PinnedTemplateCommit: workersTemplateCommitDefault(),
		SyncState:            "uninitialized",
	}
	result.CLIVersionOverridden = result.EffectiveCLIVersion != result.PinnedCLIVersion

	if _, err := os.Stat(result.MetadataFile); err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return workersStatusResult{}, fmt.Errorf("stat workers metadata file: %w", err)
	}

	meta, err := readWorkersTemplateMetadata(result.MetadataFile)
	if err != nil {
		return workersStatusResult{}, err
	}
	result.MetadataExists = true
	result.ProjectTemplateRepo = meta.TemplateRepo
	result.ProjectTemplateCommit = meta.TemplateCommit
	result.ProjectScaffoldedAt = meta.ScaffoldedAt

	if strings.TrimSpace(result.ProjectTemplateRepo) == "" || strings.TrimSpace(result.ProjectTemplateCommit) == "" {
		result.SyncState = "unknown"
		return result, nil
	}

	sameRepo := result.ProjectTemplateRepo == result.PinnedTemplateRepo
	if !sameRepo {
		pinnedCanonical, pinnedErr := canonicalWorkersRepo(result.PinnedTemplateRepo)
		projectCanonical, projectErr := canonicalWorkersRepo(result.ProjectTemplateRepo)
		if pinnedErr == nil && projectErr == nil {
			sameRepo = pinnedCanonical == projectCanonical
		}
	}
	if !sameRepo {
		result.SyncState = "repo_mismatch"
		return result, nil
	}
	if result.ProjectTemplateCommit == result.PinnedTemplateCommit {
		result.SyncState = "in_sync"
		return result, nil
	}
	if opts.NoCompare {
		result.SyncState = "out_of_sync"
		return result, nil
	}

	compare, err := workersCompareFunc(ctx, result.PinnedTemplateRepo, result.ProjectTemplateCommit, result.PinnedTemplateCommit)
	if err != nil {
		result.SyncState = "out_of_sync"
		result.CompareError = err.Error()
		return result, nil
	}
	result.CompareURL = strings.TrimSpace(compare.HTMLURL)

	// Compare request is base=project...head=pinned, so ahead/behind are from pinned's perspective.
	switch strings.TrimSpace(compare.Status) {
	case "identical":
		result.SyncState = "in_sync"
	case "ahead":
		result.SyncState = "behind"
		result.BehindBy = compare.AheadBy
	case "behind":
		result.SyncState = "ahead"
		result.AheadBy = compare.BehindBy
	case "diverged":
		result.SyncState = "diverged"
		result.AheadBy = compare.BehindBy
		result.BehindBy = compare.AheadBy
	default:
		result.SyncState = "out_of_sync"
	}
	return result, nil
}

func canonicalWorkersRepo(repo string) (string, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return "", err
	}
	return strings.ToLower(owner + "/" + name), nil
}

func compareGitHubCommits(ctx context.Context, repo, baseCommit, headCommit string) (workersGitHubCompareResponse, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return workersGitHubCompareResponse{}, err
	}
	baseCommit = strings.TrimSpace(baseCommit)
	headCommit = strings.TrimSpace(headCommit)
	if baseCommit == "" || headCommit == "" {
		return workersGitHubCompareResponse{}, fmt.Errorf("compare requires non-empty commits")
	}

	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/compare/%s...%s",
		owner,
		name,
		url.PathEscape(baseCommit),
		url.PathEscape(headCommit),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return workersGitHubCompareResponse{}, fmt.Errorf("create compare request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ntn-workers-status")

	client := &http.Client{Timeout: workersTemplateHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return workersGitHubCompareResponse{}, fmt.Errorf("request compare API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
			return workersGitHubCompareResponse{}, fmt.Errorf("compare API status %d: %s", resp.StatusCode, trimmed)
		}
		return workersGitHubCompareResponse{}, fmt.Errorf("compare API status %d", resp.StatusCode)
	}

	var parsed workersGitHubCompareResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return workersGitHubCompareResponse{}, fmt.Errorf("decode compare API response: %w", err)
	}
	return parsed, nil
}

func buildWorkersUpgradePlan(ctx context.Context, tarballURL, targetDir string) (workersUpgradePlan, error) {
	tempDir, err := os.MkdirTemp("", "ntn-workers-plan-*")
	if err != nil {
		return workersUpgradePlan{}, fmt.Errorf("create temporary plan dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if err := downloadAndExtractWorkersTemplate(ctx, tarballURL, tempDir, true); err != nil {
		return workersUpgradePlan{}, err
	}

	return summarizeWorkersUpgradePlan(targetDir, tempDir)
}

func summarizeWorkersUpgradePlan(targetDir, templateDir string) (workersUpgradePlan, error) {
	targetEntries, err := collectWorkersPlanEntries(targetDir)
	if err != nil {
		return workersUpgradePlan{}, err
	}
	templateEntries, err := collectWorkersPlanEntries(templateDir)
	if err != nil {
		return workersUpgradePlan{}, err
	}

	addPaths := make([]string, 0, 32)
	modifyPaths := make([]string, 0, 32)
	unchangedCount := 0

	for rel, templateEntry := range templateEntries {
		targetEntry, ok := targetEntries[rel]
		if !ok {
			addPaths = append(addPaths, rel)
			continue
		}
		if workersPlanEntryEqual(templateEntry, targetEntry) {
			unchangedCount++
			continue
		}
		modifyPaths = append(modifyPaths, rel)
	}

	extraPaths := make([]string, 0, 32)
	for rel := range targetEntries {
		if _, ok := templateEntries[rel]; !ok {
			extraPaths = append(extraPaths, rel)
		}
	}

	sort.Strings(addPaths)
	sort.Strings(modifyPaths)
	sort.Strings(extraPaths)

	return workersUpgradePlan{
		AddCount:       len(addPaths),
		ModifyCount:    len(modifyPaths),
		UnchangedCount: unchangedCount,
		ExtraCount:     len(extraPaths),
		AddPaths:       sampleWorkersPlanPaths(addPaths, 20),
		ModifyPaths:    sampleWorkersPlanPaths(modifyPaths, 20),
		ExtraPaths:     sampleWorkersPlanPaths(extraPaths, 20),
	}, nil
}

func collectWorkersPlanEntries(root string) (map[string]workersPlanEntry, error) {
	entries := make(map[string]workersPlanEntry, 256)
	err := filepath.WalkDir(root, func(curr string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, curr)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == workersTemplateMetadataFile {
			return nil
		}

		entry := workersPlanEntry{}
		switch d.Type() {
		case os.ModeDir:
			entry.Kind = "dir"
		case os.ModeSymlink:
			entry.Kind = "symlink"
			link, err := os.Readlink(curr)
			if err != nil {
				return fmt.Errorf("read symlink %q: %w", curr, err)
			}
			entry.Link = link
		case 0: // regular file
			entry.Kind = "file"
			hash, err := hashWorkersPlanFile(curr)
			if err != nil {
				return err
			}
			entry.Hash = hash
		default:
			return nil
		}

		entries[rel] = entry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", root, err)
	}
	return entries, nil
}

func hashWorkersPlanFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file %q for hashing: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("hash file %q: %w", path, err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func workersPlanEntryEqual(a, b workersPlanEntry) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case "file":
		return a.Hash == b.Hash
	case "symlink":
		return a.Link == b.Link
	default:
		return true
	}
}

func sampleWorkersPlanPaths(paths []string, limit int) []string {
	if len(paths) <= limit {
		return paths
	}
	return paths[:limit]
}

func prepareWorkersTargetDir(target string, force bool) error {
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(target, 0o755)
		}
		return fmt.Errorf("stat target dir: %w", err)
	}
	if !info.IsDir() {
		return clierrors.NewUserError(
			fmt.Sprintf("target path %q exists and is not a directory", target),
			"Choose another path or remove the existing file.",
		)
	}
	hasEntries, err := dirHasEntries(target)
	if err != nil {
		return err
	}
	if hasEntries && !force {
		return clierrors.NewUserError(
			fmt.Sprintf("target directory %q is not empty", target),
			"Re-run with --force to allow overwriting existing files.",
		)
	}
	return nil
}

func dirHasEntries(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read target dir: %w", err)
	}
	return len(entries) > 0, nil
}

func workersTemplateTarballURL(repo, commit string) (string, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return "", err
	}
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return "", clierrors.NewUserError("workers template commit is empty", "Provide --commit or configure template_commit in internal/workers/compat.json.")
	}
	return fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, name, commit), nil
}

func parseGitHubRepo(repo string) (string, string, error) {
	raw := strings.TrimSpace(repo)
	if raw == "" {
		return "", "", clierrors.NewUserError("workers template repo is empty", "Provide --repo or configure template_repo in internal/workers/compat.json.")
	}

	if strings.Count(raw, "/") == 1 && !strings.Contains(raw, ":") && !strings.Contains(raw, ".") {
		parts := strings.SplitN(raw, "/", 2)
		return parts[0], parts[1], nil
	}
	if strings.HasPrefix(raw, "github.com/") {
		raw = "https://" + raw
	}
	if !strings.Contains(raw, "://") {
		return "", "", clierrors.NewUserError(
			fmt.Sprintf("unsupported workers template repo %q", repo),
			"Use GitHub format owner/repo or https://github.com/owner/repo.",
		)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", clierrors.NewUserError(fmt.Sprintf("invalid workers template repo %q", repo), "Use a valid GitHub repository URL.")
	}
	if !strings.EqualFold(u.Hostname(), "github.com") {
		return "", "", clierrors.NewUserError(
			fmt.Sprintf("unsupported workers template host %q", u.Hostname()),
			"Only GitHub repositories are currently supported for workers new.",
		)
	}
	repoPath := strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", clierrors.NewUserError(
			fmt.Sprintf("invalid workers template repo path %q", u.Path),
			"Expected format https://github.com/<owner>/<repo>.",
		)
	}
	return parts[0], parts[1], nil
}

func downloadAndExtractWorkersTemplate(ctx context.Context, tarballURL, targetDir string, force bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return fmt.Errorf("create template download request: %w", err)
	}
	client := &http.Client{Timeout: workersTemplateHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download workers template: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download workers template failed: status %d", resp.StatusCode)
	}
	return extractWorkersTemplateTarGZ(resp.Body, targetDir, force)
}

func extractWorkersTemplateTarGZ(r io.Reader, targetDir string, force bool) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open template archive: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read template archive: %w", err)
		}

		relPath, ok := stripArchiveTopLevel(hdr.Name)
		if !ok {
			continue
		}
		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
			continue
		}

		dest, err := safeJoin(targetDir, relPath)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := prepareTemplateEntryDestination(dest, hdr.Typeflag, force); err != nil {
				return err
			}
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("create directory %q: %w", dest, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return fmt.Errorf("create parent directory for %q: %w", dest, err)
			}
			if err := prepareTemplateEntryDestination(dest, hdr.Typeflag, force); err != nil {
				return err
			}
			perm := hdr.FileInfo().Mode().Perm()
			if perm == 0 {
				perm = 0o644
			}
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
			if err != nil {
				return fmt.Errorf("create file %q: %w", dest, err)
			}
			if _, err := io.Copy(f, io.LimitReader(tr, workersTemplateMaxFileSize+1)); err != nil {
				_ = f.Close()
				return fmt.Errorf("write file %q: %w", dest, err)
			}
			if fi, statErr := f.Stat(); statErr == nil && fi.Size() > workersTemplateMaxFileSize {
				_ = f.Close()
				_ = os.Remove(dest)
				return fmt.Errorf("file %q exceeds maximum size of %d bytes", dest, workersTemplateMaxFileSize)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("close file %q: %w", dest, err)
			}
		case tar.TypeSymlink:
			if err := validateSymlinkTarget(targetDir, dest, hdr.Linkname); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return fmt.Errorf("create parent directory for symlink %q: %w", dest, err)
			}
			if err := prepareTemplateEntryDestination(dest, hdr.Typeflag, force); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, dest); err != nil {
				return fmt.Errorf("create symlink %q -> %q: %w", dest, hdr.Linkname, err)
			}
		default:
			// Skip unsupported entry types.
			continue
		}
	}
	return nil
}

func prepareTemplateEntryDestination(dest string, entryType byte, force bool) error {
	info, err := os.Lstat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat template destination %q: %w", dest, err)
	}

	switch entryType {
	case tar.TypeDir:
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		if !force {
			return templatePathExistsError(dest)
		}
		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("remove conflicting path %q: %w", dest, err)
		}
		return nil
	case tar.TypeReg:
		if info.Mode().IsRegular() {
			if !force {
				return templatePathExistsError(dest)
			}
			return nil
		}
		if !force {
			return templatePathExistsError(dest)
		}
		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("remove conflicting path %q: %w", dest, err)
		}
		return nil
	case tar.TypeSymlink:
		if !force {
			return templatePathExistsError(dest)
		}
		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("remove conflicting path %q: %w", dest, err)
		}
		return nil
	default:
		return nil
	}
}

func templatePathExistsError(dest string) error {
	return clierrors.NewUserError(
		fmt.Sprintf("template path %q already exists", dest),
		"Use --force to overwrite existing files.",
	)
}

func stripArchiveTopLevel(name string) (string, bool) {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if clean == "." || clean == "/" {
		return "", false
	}
	parts := strings.SplitN(clean, "/", 2)
	if len(parts) < 2 {
		return "", false
	}
	rel := strings.TrimSpace(parts[1])
	if rel == "" || rel == "." {
		return "", false
	}
	return rel, true
}

func validateSymlinkTarget(baseDir, symlinkPath, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("symlink %q has absolute target %q which is not allowed", symlinkPath, target)
	}
	// Resolve target relative to the symlink's parent directory.
	resolved := filepath.Clean(filepath.Join(filepath.Dir(symlinkPath), target))
	base := filepath.Clean(baseDir)
	if resolved != base && !strings.HasPrefix(resolved, base+string(os.PathSeparator)) {
		return fmt.Errorf("symlink target %q escapes target directory from %q", target, symlinkPath)
	}
	return nil
}

func safeJoin(baseDir, rel string) (string, error) {
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe template path %q", rel)
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute template path %q is not allowed", rel)
	}
	base := filepath.Clean(baseDir)
	dest := filepath.Clean(filepath.Join(base, rel))
	if dest != base && !strings.HasPrefix(dest, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("template path %q escapes target directory", rel)
	}
	return dest, nil
}

func writeWorkersTemplateMetadata(targetDir string, meta workersTemplateMetadata) (string, error) {
	metaPath := filepath.Join(targetDir, workersTemplateMetadataFile)
	payload, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal workers metadata: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(metaPath, payload, 0o644); err != nil {
		return "", fmt.Errorf("write workers metadata file: %w", err)
	}
	return metaPath, nil
}

func readWorkersTemplateMetadata(metaPath string) (workersTemplateMetadata, error) {
	payload, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return workersTemplateMetadata{}, clierrors.NewUserError(
				fmt.Sprintf("workers metadata file not found at %q", metaPath),
				fmt.Sprintf("Run 'ntn workers new <path>' first to create %s.", workersTemplateMetadataFile),
			)
		}
		return workersTemplateMetadata{}, fmt.Errorf("read workers metadata file: %w", err)
	}

	var meta workersTemplateMetadata
	if err := json.Unmarshal(payload, &meta); err != nil {
		return workersTemplateMetadata{}, clierrors.NewUserError(
			fmt.Sprintf("workers metadata file is invalid JSON: %q", metaPath),
			"Re-run 'ntn workers new' to regenerate metadata, or fix the JSON file.",
		)
	}

	meta.TemplateRepo = strings.TrimSpace(meta.TemplateRepo)
	meta.TemplateCommit = strings.TrimSpace(meta.TemplateCommit)
	meta.OfficialCLIVersion = strings.TrimSpace(meta.OfficialCLIVersion)
	meta.ScaffoldedAt = strings.TrimSpace(meta.ScaffoldedAt)
	meta.TarballURL = strings.TrimSpace(meta.TarballURL)

	return meta, nil
}
