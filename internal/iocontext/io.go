// Package iocontext provides context-based stdout/stderr injection for testable I/O.
package iocontext

import (
	"context"
	"io"
)

type ctxKey int

const (
	stdoutKey ctxKey = iota
	stderrKey
)

// WithIO injects stdout and stderr writers into context.
func WithIO(ctx context.Context, stdout, stderr io.Writer) context.Context {
	ctx = context.WithValue(ctx, stdoutKey, stdout)
	ctx = context.WithValue(ctx, stderrKey, stderr)
	return ctx
}

// Stdout returns the stdout writer from context, or nil if not set.
func Stdout(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stdoutKey).(io.Writer); ok {
		return w
	}
	return nil
}

// Stderr returns the stderr writer from context, or nil if not set.
func Stderr(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(stderrKey).(io.Writer); ok {
		return w
	}
	return nil
}

// StdoutOrDefault returns stdout from context or the provided default.
func StdoutOrDefault(ctx context.Context, def io.Writer) io.Writer {
	if w := Stdout(ctx); w != nil {
		return w
	}
	return def
}

// StderrOrDefault returns stderr from context or the provided default.
func StderrOrDefault(ctx context.Context, def io.Writer) io.Writer {
	if w := Stderr(ctx); w != nil {
		return w
	}
	return def
}
