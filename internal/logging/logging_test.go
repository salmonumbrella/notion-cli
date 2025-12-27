// internal/logging/logging_test.go
package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// saveAndRestoreLogger saves the current default logger and returns a cleanup function.
func saveAndRestoreLogger(t *testing.T) {
	t.Helper()
	original := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(original)
	})
}

func TestSetup_DebugMode(t *testing.T) {
	saveAndRestoreLogger(t)

	var buf bytes.Buffer
	Setup(true, &buf)

	slog.Debug("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected debug message in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected key=value in output, got: %s", output)
	}
}

func TestSetup_NormalMode(t *testing.T) {
	saveAndRestoreLogger(t)

	var buf bytes.Buffer
	Setup(false, &buf)

	slog.Debug("debug message")
	slog.Info("info message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Errorf("debug message should not appear in normal mode")
	}
	if !strings.Contains(output, "info message") {
		t.Errorf("info message should appear")
	}
}

func TestSetup_NilWriter(t *testing.T) {
	saveAndRestoreLogger(t)

	// Should not panic when writer is nil (defaults to stderr)
	Setup(false, nil)
	slog.Info("test") // Should not panic
}

func TestSetupJSON_DebugMode(t *testing.T) {
	saveAndRestoreLogger(t)

	var buf bytes.Buffer
	SetupJSON(true, &buf)

	slog.Debug("debug json", "key", "value")
	slog.Info("info json")

	output := buf.String()
	if !strings.Contains(output, `"msg":"debug json"`) {
		t.Errorf("expected debug message in JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"msg":"info json"`) {
		t.Errorf("expected info message in JSON output, got: %s", output)
	}
}

func TestSetupJSON_NormalMode(t *testing.T) {
	saveAndRestoreLogger(t)

	var buf bytes.Buffer
	SetupJSON(false, &buf)

	slog.Debug("debug message")
	slog.Info("info message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Errorf("debug message should not appear in normal JSON mode")
	}
	if !strings.Contains(output, `"msg":"info message"`) {
		t.Errorf("info message should appear in JSON format, got: %s", output)
	}
}
