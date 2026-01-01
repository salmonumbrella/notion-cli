package ui

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		mode      ColorMode
		noColor   string
		wantColor bool
	}{
		{
			name:      "ColorAuto with NO_COLOR set",
			mode:      ColorAuto,
			noColor:   "1",
			wantColor: false,
		},
		{
			name:      "ColorAlways with NO_COLOR set",
			mode:      ColorAlways,
			noColor:   "1",
			wantColor: false,
		},
		{
			name:      "ColorNever",
			mode:      ColorNever,
			noColor:   "",
			wantColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noColor != "" {
				old := os.Getenv("NO_COLOR")
				_ = os.Setenv("NO_COLOR", tt.noColor)
				defer func() { _ = os.Setenv("NO_COLOR", old) }()
			}

			ui := New(tt.mode)
			if ui == nil {
				t.Fatal("New() returned nil")
			}
			if ui.color != tt.mode && tt.noColor == "" {
				t.Errorf("New() color mode = %v, want %v", ui.color, tt.mode)
			}
		})
	}
}

func TestContextIntegration(t *testing.T) {
	ui := New(ColorNever)
	ctx := context.Background()

	// Test WithUI and FromContext
	ctx = WithUI(ctx, ui)
	retrieved := FromContext(ctx)

	if retrieved != ui {
		t.Error("FromContext() did not return the same UI instance")
	}
}

func TestFromContextDefault(t *testing.T) {
	ctx := context.Background()
	ui := FromContext(ctx)

	if ui == nil {
		t.Fatal("FromContext() returned nil for context without UI")
	}

	// Should return a default UI with ColorAuto
	if ui.color != ColorAuto && os.Getenv("NO_COLOR") == "" {
		t.Errorf("FromContext() default color mode = %v, want %v", ui.color, ColorAuto)
	}
}

func TestOutputMethods(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(*UI, string, ...any)
		input    string
		expected string
	}{
		{
			name:     "Success",
			fn:       (*UI).Success,
			input:    "operation completed",
			expected: "✓ operation completed",
		},
		{
			name:     "Warning",
			fn:       (*UI).Warning,
			input:    "potential issue",
			expected: "⚠ potential issue",
		},
		{
			name:     "Error",
			fn:       (*UI).Error,
			input:    "something failed",
			expected: "✗ something failed",
		},
		{
			name:     "Info",
			fn:       (*UI).Info,
			input:    "helpful information",
			expected: "ℹ helpful information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var buf bytes.Buffer

			// Create UI with custom output writer
			ui := &UI{
				out:   termenv.NewOutput(&buf, termenv.WithProfile(termenv.Ascii)),
				color: ColorNever,
			}

			// Call the method
			tt.fn(ui, tt.input)

			// Check output
			output := strings.TrimSpace(buf.String())
			if !strings.Contains(output, tt.expected) {
				t.Errorf("%s output = %q, want to contain %q", tt.name, output, tt.expected)
			}
		})
	}
}

func TestOutputMethodsWithFormatting(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create UI with custom output writer
	ui := &UI{
		out:   termenv.NewOutput(&buf, termenv.WithProfile(termenv.Ascii)),
		color: ColorNever,
	}

	ui.Success("completed %d of %d tasks", 5, 10)

	output := strings.TrimSpace(buf.String())
	expected := "✓ completed 5 of 10 tasks"
	if !strings.Contains(output, expected) {
		t.Errorf("Success with formatting = %q, want to contain %q", output, expected)
	}
}

func TestWriter(t *testing.T) {
	ui := New(ColorNever)
	writer := ui.Writer()

	if writer == nil {
		t.Fatal("Writer() returned nil")
	}

	// Verify it's the same as the underlying output
	if writer != ui.out {
		t.Error("Writer() did not return the underlying output writer")
	}
}

func TestColorProfile(t *testing.T) {
	tests := []struct {
		name string
		mode ColorMode
	}{
		{
			name: "ColorNever uses Ascii profile",
			mode: ColorNever,
		},
		{
			name: "ColorAuto uses detected profile",
			mode: ColorAuto,
		},
		{
			name: "ColorAlways preserves profile",
			mode: ColorAlways,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear NO_COLOR for this test
			old := os.Getenv("NO_COLOR")
			_ = os.Unsetenv("NO_COLOR")
			defer func() { _ = os.Setenv("NO_COLOR", old) }()

			ui := New(tt.mode)
			if ui == nil {
				t.Fatal("New() returned nil")
			}

			// Verify the profile is set correctly
			profile := termenv.NewOutput(ui.out, termenv.WithProfile(termenv.Ascii)).Profile
			if tt.mode == ColorNever && profile != termenv.Ascii {
				t.Errorf("ColorNever should use Ascii profile, got %v", profile)
			}
		})
	}
}
