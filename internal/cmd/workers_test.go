package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/iocontext"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

type fakeWorkersRunCall struct {
	bin  string
	args []string
}

type fakeWorkersRuntime struct {
	lookPaths map[string]string
	runErr    error
	runs      []fakeWorkersRunCall
	outputs   map[string]struct {
		out string
		err error
	}
}

func (f *fakeWorkersRuntime) lookPath(file string) (string, error) {
	if f.lookPaths == nil {
		return "", errors.New("not found")
	}
	if p, ok := f.lookPaths[file]; ok {
		return p, nil
	}
	return "", fmt.Errorf("%s not found", file)
}

func (f *fakeWorkersRuntime) run(_ context.Context, bin string, args []string, _ io.Reader, _ io.Writer, _ io.Writer) error {
	f.runs = append(f.runs, fakeWorkersRunCall{
		bin:  bin,
		args: append([]string(nil), args...),
	})
	return f.runErr
}

func (f *fakeWorkersRuntime) output(_ context.Context, bin string, args []string) (string, error) {
	key := bin + "\x00" + strings.Join(args, "\x00")
	if r, ok := f.outputs[key]; ok {
		return r.out, r.err
	}
	return "", fmt.Errorf("missing output for %q", key)
}

func TestBuildWorkersCommandInvocation_Default(t *testing.T) {
	t.Setenv(workersCLIVersionEnvVar, "")
	t.Setenv(workersNPXBinEnvVar, "")

	inv := buildWorkersCommandInvocation([]string{"deploy", "--dry-run"})
	if inv.bin != "npx" {
		t.Fatalf("bin = %q, want %q", inv.bin, "npx")
	}
	want := []string{"--yes", "ntn@" + workersCLIVersionDefault(), "workers", "deploy", "--dry-run"}
	if !reflect.DeepEqual(inv.args, want) {
		t.Fatalf("args = %#v, want %#v", inv.args, want)
	}
}

func TestBuildWorkersCommandInvocation_EnvOverride(t *testing.T) {
	t.Setenv(workersCLIVersionEnvVar, "0.2.0")
	t.Setenv(workersNPXBinEnvVar, "my-npx")

	inv := buildWorkersCommandInvocation([]string{"runs", "list"})
	if inv.bin != "my-npx" {
		t.Fatalf("bin = %q, want %q", inv.bin, "my-npx")
	}
	want := []string{"--yes", "ntn@0.2.0", "workers", "runs", "list"}
	if !reflect.DeepEqual(inv.args, want) {
		t.Fatalf("args = %#v, want %#v", inv.args, want)
	}
}

func TestRunWorkersPassthrough(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/bin/npx"},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	ctx := iocontext.WithIO(context.Background(), &out, &errBuf)

	if err := runWorkersPassthrough(ctx, []string{"deploy", "--verbose"}); err != nil {
		t.Fatalf("runWorkersPassthrough() error = %v", err)
	}
	if len(fake.runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(fake.runs))
	}
	got := fake.runs[0]
	if got.bin != "npx" {
		t.Fatalf("run bin = %q, want %q", got.bin, "npx")
	}
	wantArgs := []string{"--yes", "ntn@" + workersCLIVersionDefault(), "workers", "deploy", "--verbose"}
	if !reflect.DeepEqual(got.args, wantArgs) {
		t.Fatalf("run args = %#v, want %#v", got.args, wantArgs)
	}
}

func TestRunWorkersPassthrough_MissingNPX(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	workersRuntimeImpl = &fakeWorkersRuntime{}

	err := runWorkersPassthrough(context.Background(), []string{"deploy"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clierrors.IsUserError(err) {
		t.Fatalf("expected user error, got %T (%v)", err, err)
	}
}

func TestRunWorkersPassthrough_ProxyExitErrorPassesThrough(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	wantErr := &proxiedCommandExitError{Code: 17}
	workersRuntimeImpl = &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/bin/npx"},
		runErr:    wantErr,
	}

	err := runWorkersPassthrough(context.Background(), []string{"deploy"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var proxiedErr *proxiedCommandExitError
	if !errors.As(err, &proxiedErr) {
		t.Fatalf("expected proxied exit error, got %T (%v)", err, err)
	}
	if proxiedErr.Code != wantErr.Code {
		t.Fatalf("exit code = %d, want %d", proxiedErr.Code, wantErr.Code)
	}
}

func TestRunWorkersDoctor_JSON(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/local/bin/npx"},
		outputs: map[string]struct {
			out string
			err error
		}{
			"node\x00--version": {"v22.12.0", nil},
			"npm\x00--version":  {"10.9.2", nil},
			"npx\x00--yes\x00ntn@" + workersCLIVersionDefault() + "\x00--version": {"0.1.35", nil},
		},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	ctx := iocontext.WithIO(context.Background(), &out, &errBuf)
	ctx = output.WithFormat(ctx, output.FormatJSON)

	if err := runWorkersDoctor(ctx, nil); err != nil {
		t.Fatalf("runWorkersDoctor() error = %v", err)
	}

	var report workersDoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("doctor output not valid json: %v\n%s", err, out.String())
	}
	if !report.OK {
		t.Fatalf("report.OK = false, want true: %+v", report)
	}
	if report.CLIVersion != workersCLIVersionDefault() {
		t.Fatalf("report.CLIVersion = %q, want %q", report.CLIVersion, workersCLIVersionDefault())
	}
	if len(report.Checks) != 4 {
		t.Fatalf("checks = %d, want 4", len(report.Checks))
	}
}

func TestWorkersCmdRoutesDoctor(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/local/bin/npx"},
		outputs: map[string]struct {
			out string
			err error
		}{
			"node\x00--version": {"v22.12.0", nil},
			"npm\x00--version":  {"10.9.2", nil},
			"npx\x00--yes\x00ntn@" + workersCLIVersionDefault() + "\x00--version": {"0.1.35", nil},
		},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	ctx := iocontext.WithIO(context.Background(), &out, &errBuf)
	ctx = output.WithFormat(ctx, output.FormatJSON)

	cmd := newWorkersCmd()
	cmd.SetContext(ctx)
	if err := cmd.RunE(cmd, []string{"doctor"}); err != nil {
		t.Fatalf("RunE doctor error = %v", err)
	}
	if len(fake.runs) != 0 {
		t.Fatalf("doctor should not invoke passthrough run, got %d calls", len(fake.runs))
	}
}

func TestWorkersCmdRoutesDoctor_WithOutputFlag(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/local/bin/npx"},
		outputs: map[string]struct {
			out string
			err error
		}{
			"node\x00--version": {"v22.12.0", nil},
			"npm\x00--version":  {"10.9.2", nil},
			"npx\x00--yes\x00ntn@" + workersCLIVersionDefault() + "\x00--version": {"0.1.35", nil},
		},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"doctor", "-o", "json"}); err != nil {
		t.Fatalf("RunE doctor -o json error = %v", err)
	}
	var report workersDoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("doctor output should be json: %v\n%s", err, out.String())
	}
}

func TestWorkersCmdRoutesDoctor_WithGlobalStyleOutputPrefix(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/local/bin/npx"},
		outputs: map[string]struct {
			out string
			err error
		}{
			"node\x00--version": {"v22.12.0", nil},
			"npm\x00--version":  {"10.9.2", nil},
			"npx\x00--yes\x00ntn@" + workersCLIVersionDefault() + "\x00--version": {"0.1.35", nil},
		},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"-o", "json", "doctor"}); err != nil {
		t.Fatalf("RunE -o json doctor error = %v", err)
	}
	var report workersDoctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("doctor output should be json: %v\n%s", err, out.String())
	}
}

func TestWorkersCmdRoutesStatus(t *testing.T) {
	oldStatus := workersStatusFunc
	t.Cleanup(func() { workersStatusFunc = oldStatus })

	var gotOpts workersStatusOptions
	workersStatusFunc = func(_ context.Context, opts workersStatusOptions) (workersStatusResult, error) {
		gotOpts = opts
		return workersStatusResult{
			Path:                 "/tmp/my-worker",
			MetadataFile:         "/tmp/my-worker/.ntn-workers-template.json",
			MetadataExists:       true,
			PinnedCLIVersion:     workersCLIVersionDefault(),
			EffectiveCLIVersion:  workersCLIVersionDefault(),
			PinnedTemplateRepo:   workersTemplateRepoDefault(),
			PinnedTemplateCommit: workersTemplateCommitDefault(),
			SyncState:            "in_sync",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"status", "my-worker", "--no-compare", "-o", "json"}); err != nil {
		t.Fatalf("RunE workers status error = %v", err)
	}
	if gotOpts.Path != "my-worker" || !gotOpts.NoCompare {
		t.Fatalf("unexpected status options: %+v", gotOpts)
	}

	var payload workersStatusResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("status output should be json: %v\n%s", err, out.String())
	}
	if payload.SyncState != "in_sync" {
		t.Fatalf("sync_state = %q, want in_sync", payload.SyncState)
	}
}

func TestWorkersCmdRoutesStatus_WithOutputPrefix(t *testing.T) {
	oldStatus := workersStatusFunc
	t.Cleanup(func() { workersStatusFunc = oldStatus })

	workersStatusFunc = func(_ context.Context, opts workersStatusOptions) (workersStatusResult, error) {
		return workersStatusResult{
			Path:                 opts.Path,
			MetadataFile:         "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			MetadataExists:       false,
			PinnedCLIVersion:     workersCLIVersionDefault(),
			EffectiveCLIVersion:  workersCLIVersionDefault(),
			PinnedTemplateRepo:   workersTemplateRepoDefault(),
			PinnedTemplateCommit: workersTemplateCommitDefault(),
			SyncState:            "uninitialized",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"-o", "json", "status", "from-prefix"}); err != nil {
		t.Fatalf("RunE -o json status error = %v", err)
	}

	var payload workersStatusResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("status output should be json: %v\n%s", err, out.String())
	}
	if payload.Path != "from-prefix" {
		t.Fatalf("path = %v, want from-prefix", payload.Path)
	}
}

func TestWorkersCmdForwardsSubcommandHelp(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })

	fake := &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/local/bin/npx"},
	}
	workersRuntimeImpl = fake

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"deploy", "--help"}); err != nil {
		t.Fatalf("RunE deploy --help error = %v", err)
	}

	if len(fake.runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(fake.runs))
	}
	want := []string{"--yes", "ntn@" + workersCLIVersionDefault(), "workers", "deploy", "--help"}
	if !reflect.DeepEqual(fake.runs[0].args, want) {
		t.Fatalf("run args = %#v, want %#v", fake.runs[0].args, want)
	}
}

func TestWorkersCmdRoutesNew(t *testing.T) {
	oldScaffold := workersScaffoldFunc
	t.Cleanup(func() { workersScaffoldFunc = oldScaffold })

	var gotOpts workersNewOptions
	workersScaffoldFunc = func(_ context.Context, opts workersNewOptions) (workersScaffoldResult, error) {
		gotOpts = opts
		return workersScaffoldResult{
			Path:            "/tmp/my-worker",
			TemplateRepo:    "https://github.com/makenotion/workers-template",
			TemplateCommit:  "abc123",
			OfficialVersion: "0.1.35",
			MetadataFile:    "/tmp/my-worker/.ntn-workers-template.json",
			TarballURL:      "https://codeload.github.com/makenotion/workers-template/tar.gz/abc123",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"new", "my-worker", "--force", "--repo", "https://github.com/makenotion/workers-template", "--commit", "abc123", "-o", "json"}); err != nil {
		t.Fatalf("RunE workers new error = %v", err)
	}

	if gotOpts.Path != "my-worker" || !gotOpts.Force || gotOpts.Repo == "" || gotOpts.Commit != "abc123" {
		t.Fatalf("unexpected new options: %+v", gotOpts)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("new output should be json: %v\n%s", err, out.String())
	}
	if payload["status"] != "success" {
		t.Fatalf("status = %v, want success", payload["status"])
	}
}

func TestWorkersCmdRoutesNew_WithOutputPrefix(t *testing.T) {
	oldScaffold := workersScaffoldFunc
	t.Cleanup(func() { workersScaffoldFunc = oldScaffold })

	workersScaffoldFunc = func(_ context.Context, opts workersNewOptions) (workersScaffoldResult, error) {
		return workersScaffoldResult{
			Path:            opts.Path,
			TemplateRepo:    "https://github.com/makenotion/workers-template",
			TemplateCommit:  "abc123",
			OfficialVersion: workersCLIVersionDefault(),
			MetadataFile:    "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			TarballURL:      "https://codeload.github.com/makenotion/workers-template/tar.gz/abc123",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"-o", "json", "new", "from-prefix"}); err != nil {
		t.Fatalf("RunE -o json new error = %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("new output should be json: %v\n%s", err, out.String())
	}
	if payload["path"] != "from-prefix" {
		t.Fatalf("path = %v, want from-prefix", payload["path"])
	}
}

func TestWorkersCmdRoutesUpgrade(t *testing.T) {
	oldUpgrade := workersUpgradeFunc
	t.Cleanup(func() { workersUpgradeFunc = oldUpgrade })

	var gotOpts workersUpgradeOptions
	workersUpgradeFunc = func(_ context.Context, opts workersUpgradeOptions) (workersUpgradeResult, error) {
		gotOpts = opts
		return workersUpgradeResult{
			Path:               "/tmp/my-worker",
			TemplateRepo:       "https://github.com/makenotion/workers-template",
			TemplateCommit:     "def456",
			PreviousCommit:     "abc123",
			OfficialVersion:    "0.1.35",
			MetadataFile:       "/tmp/my-worker/.ntn-workers-template.json",
			TarballURL:         "https://codeload.github.com/makenotion/workers-template/tar.gz/def456",
			MetadataSourceFile: "/tmp/my-worker/.ntn-workers-template.json",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"upgrade", "my-worker", "--force", "--repo", "https://github.com/makenotion/workers-template", "--commit", "def456", "-o", "json"}); err != nil {
		t.Fatalf("RunE workers upgrade error = %v", err)
	}

	if gotOpts.Path != "my-worker" || !gotOpts.Force || gotOpts.Repo == "" || gotOpts.Commit != "def456" {
		t.Fatalf("unexpected upgrade options: %+v", gotOpts)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("upgrade output should be json: %v\n%s", err, out.String())
	}
	if payload["status"] != "success" {
		t.Fatalf("status = %v, want success", payload["status"])
	}
	if payload["previous_commit"] != "abc123" {
		t.Fatalf("previous_commit = %v, want abc123", payload["previous_commit"])
	}
}

func TestWorkersCmdRoutesUpgrade_DryRunFromMetadata(t *testing.T) {
	oldUpgrade := workersUpgradeFunc
	t.Cleanup(func() { workersUpgradeFunc = oldUpgrade })

	var gotOpts workersUpgradeOptions
	workersUpgradeFunc = func(_ context.Context, opts workersUpgradeOptions) (workersUpgradeResult, error) {
		gotOpts = opts
		return workersUpgradeResult{
			Path:               opts.Path,
			TemplateRepo:       "https://github.com/makenotion/workers-template",
			TemplateCommit:     "def456",
			PreviousCommit:     "abc123",
			OfficialVersion:    workersCLIVersionDefault(),
			MetadataFile:       "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			TarballURL:         "https://codeload.github.com/makenotion/workers-template/tar.gz/def456",
			MetadataSourceFile: "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			DryRun:             true,
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"upgrade", "from-meta", "--dry-run", "--from-metadata", "-o", "json"}); err != nil {
		t.Fatalf("RunE workers upgrade --dry-run --from-metadata error = %v", err)
	}
	if !gotOpts.DryRun || !gotOpts.FromMetadata {
		t.Fatalf("expected dry-run + from-metadata flags, got %+v", gotOpts)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("upgrade output should be json: %v\n%s", err, out.String())
	}
	if payload["status"] != "dry-run" {
		t.Fatalf("status = %v, want dry-run", payload["status"])
	}
	if payload["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", payload["dry_run"])
	}
}

func TestWorkersCmdRoutesUpgrade_Plan(t *testing.T) {
	oldUpgrade := workersUpgradeFunc
	t.Cleanup(func() { workersUpgradeFunc = oldUpgrade })

	var gotOpts workersUpgradeOptions
	workersUpgradeFunc = func(_ context.Context, opts workersUpgradeOptions) (workersUpgradeResult, error) {
		gotOpts = opts
		return workersUpgradeResult{
			Path:               opts.Path,
			TemplateRepo:       "https://github.com/makenotion/workers-template",
			TemplateCommit:     "def456",
			PreviousCommit:     "abc123",
			OfficialVersion:    workersCLIVersionDefault(),
			MetadataFile:       "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			TarballURL:         "https://codeload.github.com/makenotion/workers-template/tar.gz/def456",
			MetadataSourceFile: "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			DryRun:             true,
			Plan: &workersUpgradePlan{
				AddCount:    1,
				ModifyCount: 2,
			},
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"upgrade", "plan-proj", "--plan", "-o", "json"}); err != nil {
		t.Fatalf("RunE workers upgrade --plan error = %v", err)
	}
	if !gotOpts.Plan {
		t.Fatalf("expected plan flag to be set: %+v", gotOpts)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("upgrade plan output should be json: %v\n%s", err, out.String())
	}
	if payload["status"] != "plan" {
		t.Fatalf("status = %v, want plan", payload["status"])
	}
	if _, ok := payload["plan"]; !ok {
		t.Fatalf("expected plan payload, got: %v", payload)
	}
}

func TestWorkersCmdRoutesUpgrade_WithOutputPrefix(t *testing.T) {
	oldUpgrade := workersUpgradeFunc
	t.Cleanup(func() { workersUpgradeFunc = oldUpgrade })

	workersUpgradeFunc = func(_ context.Context, opts workersUpgradeOptions) (workersUpgradeResult, error) {
		return workersUpgradeResult{
			Path:               opts.Path,
			TemplateRepo:       "https://github.com/makenotion/workers-template",
			TemplateCommit:     "def456",
			PreviousCommit:     "abc123",
			OfficialVersion:    workersCLIVersionDefault(),
			MetadataFile:       "/tmp/" + opts.Path + "/.ntn-workers-template.json",
			TarballURL:         "https://codeload.github.com/makenotion/workers-template/tar.gz/def456",
			MetadataSourceFile: "/tmp/" + opts.Path + "/.ntn-workers-template.json",
		}, nil
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd := newWorkersCmd()
	cmd.SetContext(iocontext.WithIO(context.Background(), &out, &errBuf))
	if err := cmd.RunE(cmd, []string{"-o", "json", "upgrade", "from-prefix"}); err != nil {
		t.Fatalf("RunE -o json upgrade error = %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("upgrade output should be json: %v\n%s", err, out.String())
	}
	if payload["path"] != "from-prefix" {
		t.Fatalf("path = %v, want from-prefix", payload["path"])
	}
}

func TestUpgradeWorkersTemplate_DryRunDefaultsToCompatCommit(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	result, err := upgradeWorkersTemplate(context.Background(), workersUpgradeOptions{
		Path:   tmpDir,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("upgradeWorkersTemplate() dry-run error = %v", err)
	}
	if !result.DryRun {
		t.Fatal("DryRun should be true")
	}
	if result.TemplateCommit != workersTemplateCommitDefault() {
		t.Fatalf("template commit = %q, want compat %q", result.TemplateCommit, workersTemplateCommitDefault())
	}
	if result.PreviousCommit != "meta-commit" {
		t.Fatalf("previous commit = %q, want meta-commit", result.PreviousCommit)
	}
}

func TestUpgradeWorkersTemplate_DryRunFromMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	result, err := upgradeWorkersTemplate(context.Background(), workersUpgradeOptions{
		Path:         tmpDir,
		DryRun:       true,
		FromMetadata: true,
	})
	if err != nil {
		t.Fatalf("upgradeWorkersTemplate() dry-run from metadata error = %v", err)
	}
	if result.TemplateCommit != "meta-commit" {
		t.Fatalf("template commit = %q, want metadata commit", result.TemplateCommit)
	}
}

func TestUpgradeWorkersTemplate_RequiresForce(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	_, err = upgradeWorkersTemplate(context.Background(), workersUpgradeOptions{
		Path: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clierrors.IsUserError(err) {
		t.Fatalf("expected user error, got %T (%v)", err, err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v, expected --force hint", err)
	}
}

func TestPrepareTemplateEntryDestination_ForceReplacesConflictingDir(t *testing.T) {
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "entry")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "nested.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested: %v", err)
	}

	if err := prepareTemplateEntryDestination(dest, tar.TypeReg, true); err != nil {
		t.Fatalf("prepareTemplateEntryDestination() error = %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dest should be removed, stat err = %v", err)
	}
}

func TestPrepareTemplateEntryDestination_NonForceRejectsExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "entry.txt")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := prepareTemplateEntryDestination(dest, tar.TypeReg, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clierrors.IsUserError(err) {
		t.Fatalf("expected user error, got %T (%v)", err, err)
	}
}

func TestExtractWorkersTemplateTarGZ_SymlinkConflictWithoutForce(t *testing.T) {
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "link")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatalf("write conflict file: %v", err)
	}

	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "repo/link",
		Typeflag: tar.TypeSymlink,
		Linkname: "./target",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	err := extractWorkersTemplateTarGZ(bytes.NewReader(archive.Bytes()), tmpDir, false)
	if err == nil {
		t.Fatal("expected extraction error, got nil")
	}
	if !clierrors.IsUserError(err) {
		t.Fatalf("expected user error, got %T (%v)", err, err)
	}
}

func TestExtractWorkersTemplateTarGZ_RejectsEscapingSymlinkTarget(t *testing.T) {
	tmpDir := t.TempDir()

	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "repo/escape",
		Typeflag: tar.TypeSymlink,
		Linkname: "../../etc/passwd",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	err := extractWorkersTemplateTarGZ(bytes.NewReader(archive.Bytes()), tmpDir, true)
	if err == nil {
		t.Fatal("expected error for escaping symlink target, got nil")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error should mention escaping, got: %v", err)
	}
}

func TestExtractWorkersTemplateTarGZ_RejectsAbsoluteSymlinkTarget(t *testing.T) {
	tmpDir := t.TempDir()

	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "repo/abs",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	err := extractWorkersTemplateTarGZ(bytes.NewReader(archive.Bytes()), tmpDir, true)
	if err == nil {
		t.Fatal("expected error for absolute symlink target, got nil")
	}
}

func TestSafeJoin_RejectsUnsafePaths(t *testing.T) {
	base := t.TempDir()

	if _, err := safeJoin(base, "../escape.txt"); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := safeJoin(base, "/etc/passwd"); err == nil {
		t.Fatal("expected absolute path error")
	}
}

func TestSummarizeWorkersUpgradePlan(t *testing.T) {
	targetDir := t.TempDir()
	templateDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(targetDir, "keep.txt"), []byte("same"), 0o644); err != nil {
		t.Fatalf("write keep target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "keep.txt"), []byte("same"), 0o644); err != nil {
		t.Fatalf("write keep template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "modify.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write modify target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "modify.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write modify template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "add.txt"), []byte("add"), 0o644); err != nil {
		t.Fatalf("write add template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "extra.txt"), []byte("extra"), 0o644); err != nil {
		t.Fatalf("write extra target: %v", err)
	}

	plan, err := summarizeWorkersUpgradePlan(targetDir, templateDir)
	if err != nil {
		t.Fatalf("summarizeWorkersUpgradePlan() error = %v", err)
	}
	if plan.AddCount != 1 || plan.ModifyCount != 1 || plan.UnchangedCount != 1 || plan.ExtraCount != 1 {
		t.Fatalf("unexpected plan counts: %+v", plan)
	}
	if len(plan.AddPaths) == 0 || plan.AddPaths[0] != "add.txt" {
		t.Fatalf("unexpected add paths: %+v", plan.AddPaths)
	}
	if len(plan.ModifyPaths) == 0 || plan.ModifyPaths[0] != "modify.txt" {
		t.Fatalf("unexpected modify paths: %+v", plan.ModifyPaths)
	}
}

func TestCollectWorkersStatus_Uninitialized(t *testing.T) {
	res, err := collectWorkersStatus(context.Background(), workersStatusOptions{
		Path:      t.TempDir(),
		NoCompare: true,
	})
	if err != nil {
		t.Fatalf("collectWorkersStatus() error = %v", err)
	}
	if res.SyncState != "uninitialized" {
		t.Fatalf("sync_state = %q, want uninitialized", res.SyncState)
	}
}

func TestCollectWorkersStatus_OutOfSyncNoCompare(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	res, err := collectWorkersStatus(context.Background(), workersStatusOptions{
		Path:      tmpDir,
		NoCompare: true,
	})
	if err != nil {
		t.Fatalf("collectWorkersStatus() error = %v", err)
	}
	if res.SyncState != "out_of_sync" {
		t.Fatalf("sync_state = %q, want out_of_sync", res.SyncState)
	}
}

func TestCollectWorkersStatus_NormalizesRepoComparison(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       "makenotion/workers-template",
		TemplateCommit:     workersTemplateCommitDefault(),
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	res, err := collectWorkersStatus(context.Background(), workersStatusOptions{
		Path:      tmpDir,
		NoCompare: true,
	})
	if err != nil {
		t.Fatalf("collectWorkersStatus() error = %v", err)
	}
	if res.SyncState != "in_sync" {
		t.Fatalf("sync_state = %q, want in_sync", res.SyncState)
	}
}

func TestCollectWorkersStatus_CompareMapping(t *testing.T) {
	oldCompare := workersCompareFunc
	t.Cleanup(func() { workersCompareFunc = oldCompare })

	tmpDir := t.TempDir()
	if _, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	tests := []struct {
		name       string
		compare    workersGitHubCompareResponse
		wantState  string
		wantAhead  int
		wantBehind int
	}{
		{
			name:       "ahead maps to behind",
			compare:    workersGitHubCompareResponse{Status: "ahead", AheadBy: 3, BehindBy: 0},
			wantState:  "behind",
			wantAhead:  0,
			wantBehind: 3,
		},
		{
			name:       "behind maps to ahead",
			compare:    workersGitHubCompareResponse{Status: "behind", AheadBy: 0, BehindBy: 2},
			wantState:  "ahead",
			wantAhead:  2,
			wantBehind: 0,
		},
		{
			name:       "diverged maps to diverged",
			compare:    workersGitHubCompareResponse{Status: "diverged", AheadBy: 5, BehindBy: 4},
			wantState:  "diverged",
			wantAhead:  4,
			wantBehind: 5,
		},
		{
			name:       "identical maps to in_sync",
			compare:    workersGitHubCompareResponse{Status: "identical"},
			wantState:  "in_sync",
			wantAhead:  0,
			wantBehind: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workersCompareFunc = func(_ context.Context, _, _, _ string) (workersGitHubCompareResponse, error) {
				return tc.compare, nil
			}
			res, err := collectWorkersStatus(context.Background(), workersStatusOptions{
				Path: tmpDir,
			})
			if err != nil {
				t.Fatalf("collectWorkersStatus() error = %v", err)
			}
			if res.SyncState != tc.wantState {
				t.Fatalf("sync_state = %q, want %q", res.SyncState, tc.wantState)
			}
			if res.AheadBy != tc.wantAhead || res.BehindBy != tc.wantBehind {
				t.Fatalf("ahead/behind = %d/%d, want %d/%d", res.AheadBy, res.BehindBy, tc.wantAhead, tc.wantBehind)
			}
		})
	}
}

func TestCollectWorkersStatus_CompareError(t *testing.T) {
	oldCompare := workersCompareFunc
	t.Cleanup(func() { workersCompareFunc = oldCompare })

	tmpDir := t.TempDir()
	if _, err := writeWorkersTemplateMetadata(tmpDir, workersTemplateMetadata{
		TemplateRepo:       workersTemplateRepoDefault(),
		TemplateCommit:     "meta-commit",
		OfficialCLIVersion: "0.1.0",
		ScaffoldedAt:       "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("writeWorkersTemplateMetadata() error = %v", err)
	}

	workersCompareFunc = func(_ context.Context, _, _, _ string) (workersGitHubCompareResponse, error) {
		return workersGitHubCompareResponse{}, errors.New("compare unavailable")
	}
	res, err := collectWorkersStatus(context.Background(), workersStatusOptions{
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("collectWorkersStatus() error = %v", err)
	}
	if res.SyncState != "out_of_sync" {
		t.Fatalf("sync_state = %q, want out_of_sync", res.SyncState)
	}
	if res.CompareError == "" {
		t.Fatal("expected compare_error to be populated")
	}
}

func TestExtractWorkersTemplateTarGZ_RejectsOversizedFile(t *testing.T) {
	tmpDir := t.TempDir()

	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	bigSize := int64(workersTemplateMaxFileSize + 1)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "repo/huge.bin",
		Typeflag: tar.TypeReg,
		Size:     bigSize,
		Mode:     0o644,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	fakeData := make([]byte, bigSize)
	if _, err := tw.Write(fakeData); err != nil {
		t.Fatalf("write data: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	err := extractWorkersTemplateTarGZ(bytes.NewReader(archive.Bytes()), tmpDir, true)
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention size limit, got: %v", err)
	}
}

func TestAppExecute_SuppressesProxyExitErrorPrinting(t *testing.T) {
	oldRuntime := workersRuntimeImpl
	t.Cleanup(func() { workersRuntimeImpl = oldRuntime })
	t.Setenv(auth.EnvVarName, "test-token")

	workersRuntimeImpl = &fakeWorkersRuntime{
		lookPaths: map[string]string{"npx": "/usr/bin/npx"},
		runErr:    &proxiedCommandExitError{Code: 9},
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	app := &App{
		Stdout: &out,
		Stderr: &errBuf,
	}

	err := app.Execute(context.Background(), []string{"workers", "deploy"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := ExitCode(err); code != 9 {
		t.Fatalf("ExitCode(err) = %d, want 9", code)
	}
	if got := strings.TrimSpace(errBuf.String()); got != "" {
		t.Fatalf("stderr should be empty for proxied error, got %q", got)
	}
}
