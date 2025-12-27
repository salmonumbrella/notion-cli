package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/salmonumbrella/notion-cli/internal/cmd"
	"github.com/salmonumbrella/notion-cli/internal/update"
)

// Version information set via ldflags during build
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	app := cmd.NewApp()
	app.Version = Version
	app.Commit = Commit
	app.BuildTime = BuildTime
	err := app.Execute(ctx, os.Args[1:])

	// Check for updates after command execution
	// Only do this for interactive humans, otherwise it pollutes agent output streams.
	if err == nil && os.Getenv("NOTION_NO_UPDATE_CHECK") == "" && term.IsTerminal(int(os.Stdout.Fd())) {
		if msg := update.Check(ctx, Version); msg != "" {
			fmt.Fprintln(os.Stderr, "\n"+msg)
		}
	}

	if err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
