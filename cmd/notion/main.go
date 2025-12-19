package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/salmonumbrella/notion-cli/internal/cmd"
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

	cmd.SetVersionInfo(Version, Commit, BuildTime)
	if err := cmd.Execute(ctx, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
