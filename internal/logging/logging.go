// Package logging provides structured logging configuration using slog.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// Setup configures the global slog logger with text output.
// If debug is true, sets level to Debug; otherwise Info.
// Output goes to the provided writer (defaults to os.Stderr if nil).
func Setup(debug bool, w io.Writer) {
	if w == nil {
		w = os.Stderr
	}

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(w, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// SetupJSON configures the global slog logger with JSON output.
// If debug is true, sets level to Debug; otherwise Info.
// Output goes to the provided writer (defaults to os.Stderr if nil).
func SetupJSON(debug bool, w io.Writer) {
	if w == nil {
		w = os.Stderr
	}

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(w, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
