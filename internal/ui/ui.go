// Package ui provides terminal color support and UX polish for notion-cli.
package ui

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

// ColorMode determines when to use colored output.
type ColorMode int

const (
	// ColorAuto automatically detects whether to use colors based on terminal capabilities.
	ColorAuto ColorMode = iota
	// ColorAlways forces colored output regardless of terminal capabilities.
	ColorAlways
	// ColorNever disables all colored output.
	ColorNever
)

type contextKey string

const uiContextKey contextKey = "ui"

// UI provides methods for formatted terminal output with color support.
// All output goes to stderr by default, leaving stdout for data.
type UI struct {
	out   *termenv.Output
	color ColorMode
}

// New creates a new UI instance with the specified color mode.
// It respects the NO_COLOR environment variable (POSIX standard).
func New(mode ColorMode) *UI {
	// Respect NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		mode = ColorNever
	}

	profile := termenv.ColorProfile()
	switch mode {
	case ColorNever:
		profile = termenv.Ascii
	case ColorAlways:
		// Use at least ANSI256 if forcing colors
		if profile == termenv.Ascii {
			profile = termenv.ANSI256
		}
	}

	return &UI{
		out:   termenv.NewOutput(os.Stderr, termenv.WithProfile(profile)),
		color: mode,
	}
}

// WithUI returns a new context with the UI instance attached.
func WithUI(ctx context.Context, ui *UI) context.Context {
	return context.WithValue(ctx, uiContextKey, ui)
}

// FromContext retrieves the UI instance from the context.
// If no UI is found, it returns a default UI with ColorAuto mode.
func FromContext(ctx context.Context) *UI {
	if ui, ok := ctx.Value(uiContextKey).(*UI); ok {
		return ui
	}
	return New(ColorAuto)
}

// Success prints a success message in green to stderr.
func (u *UI) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(u.out, u.out.String("✓ "+msg).Foreground(termenv.ANSIGreen))
}

// Warning prints a warning message in yellow to stderr.
func (u *UI) Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(u.out, u.out.String("⚠ "+msg).Foreground(termenv.ANSIYellow))
}

// Error prints an error message in red to stderr.
func (u *UI) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(u.out, u.out.String("✗ "+msg).Foreground(termenv.ANSIRed))
}

// Info prints an informational message in blue to stderr.
func (u *UI) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(u.out, u.out.String("ℹ "+msg).Foreground(termenv.ANSIBlue))
}

// Writer returns the underlying writer for the UI (stderr).
func (u *UI) Writer() io.Writer {
	return u.out
}
