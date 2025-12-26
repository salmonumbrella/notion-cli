// Package logging provides structured logging configuration using slog.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// handlerType specifies the output format for the logger.
type handlerType int

const (
	handlerText handlerType = iota
	handlerJSON
)

// setup is the internal helper that configures the global slog logger.
// It reduces duplication between Setup and SetupJSON.
func setup(debug bool, w io.Writer, ht handlerType) {
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

	var handler slog.Handler
	switch ht {
	case handlerJSON:
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Setup configures the global slog logger with text output.
// If debug is true, sets level to Debug; otherwise Info.
// Output goes to the provided writer (defaults to os.Stderr if nil).
func Setup(debug bool, w io.Writer) {
	setup(debug, w, handlerText)
}

// SetupJSON configures the global slog logger with JSON output.
// If debug is true, sets level to Debug; otherwise Info.
// Output goes to the provided writer (defaults to os.Stderr if nil).
func SetupJSON(debug bool, w io.Writer) {
	setup(debug, w, handlerJSON)
}
