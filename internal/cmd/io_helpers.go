package cmd

import (
	"context"
	"io"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/iocontext"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func stdoutFromContext(ctx context.Context) io.Writer {
	return iocontext.StdoutOrDefault(ctx, os.Stdout)
}

func stderrFromContext(ctx context.Context) io.Writer {
	return iocontext.StderrOrDefault(ctx, os.Stderr)
}

func printerForContext(ctx context.Context) *output.Printer {
	return output.NewPrinter(stdoutFromContext(ctx), output.FormatFromContext(ctx))
}
