package main

import (
	"os"

	"github.com/salmonumbrella/notion-cli/internal/cmd"
)

// Version information set via ldflags during build
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, Commit, BuildTime)
	if err := cmd.Execute(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
