package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	if msg := update.Check(ctx, Version); msg != "" {
		fmt.Fprintln(os.Stderr, "\n"+msg)
	}

	if err != nil {
		os.Exit(1)
	}
}
