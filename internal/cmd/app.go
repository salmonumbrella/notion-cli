package cmd

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// App owns CLI wiring and execution configuration.
type App struct {
	Stdout    io.Writer
	Stderr    io.Writer
	Version   string
	Commit    string
	BuildTime string
}

// NewApp constructs an App with default settings.
func NewApp() *App {
	return &App{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Version:   "dev",
		Commit:    "unknown",
		BuildTime: "unknown",
	}
}

// Execute runs the CLI with the provided args.
func (a *App) Execute(ctx context.Context, args []string) error {
	root := newRootCmd(a)
	root.SetArgs(args)

	// Handle --help-json before Execute so it bypasses arg validation.
	if cmd, ok := findHelpJSONTarget(root, args); ok {
		return printHelpJSON(cmd)
	}

	if err := root.ExecuteContext(ctx); err != nil {
		if _, proxied := proxiedCommandExitStatus(err); !proxied {
			printCommandError(root.Context(), err)
		}
		return err
	}
	return nil
}

// RootCommand exposes the root Cobra command for embedding/tests.
func (a *App) RootCommand() *cobra.Command {
	return newRootCmd(a)
}
