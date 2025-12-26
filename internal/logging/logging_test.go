// internal/logging/logging_test.go
package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSetup_DebugMode(t *testing.T) {
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
	// Should not panic when writer is nil (defaults to stderr)
	Setup(false, nil)
	slog.Info("test") // Should not panic
}

func TestSetupJSON(t *testing.T) {
	var buf bytes.Buffer
	SetupJSON(true, &buf)

	slog.Info("json test", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"json test"`) {
		t.Errorf("expected JSON format, got: %s", output)
	}
}
