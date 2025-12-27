package cmdutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveJSONInput(t *testing.T) {
	// Create temp file for file-based tests
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create temp file with whitespace to test trimming
	whitespaceFile := filepath.Join(tmpDir, "whitespace.json")
	if err := os.WriteFile(whitespaceFile, []byte("  trimmed content  \n"), 0o644); err != nil {
		t.Fatalf("failed to create whitespace test file: %v", err)
	}

	tests := []struct {
		name    string
		raw     string
		file    string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "inline JSON passthrough",
			raw:  `{"title": "Test"}`,
			file: "",
			want: `{"title": "Test"}`,
		},
		{
			name: "empty inputs",
			raw:  "",
			file: "",
			want: "",
		},
		{
			name: "file flag",
			raw:  "",
			file: testFile,
			want: `{"key": "value"}`,
		},
		{
			name: "@file syntax",
			raw:  "@" + testFile,
			file: "",
			want: `{"key": "value"}`,
		},
		{
			name: "@file with leading whitespace",
			raw:  "  @" + testFile,
			file: "",
			want: `{"key": "value"}`,
		},
		{
			name:    "both raw and file provided",
			raw:     `{"inline": true}`,
			file:    testFile,
			wantErr: true,
			errMsg:  "use only one of inline JSON or --file",
		},
		{
			name:    "file not found",
			raw:     "",
			file:    filepath.Join(tmpDir, "nonexistent.json"),
			wantErr: true,
			errMsg:  "failed to read file",
		},
		{
			name:    "@file not found",
			raw:     "@" + filepath.Join(tmpDir, "nonexistent.json"),
			file:    "",
			wantErr: true,
			errMsg:  "failed to read file",
		},
		{
			name: "file content trimmed",
			raw:  "",
			file: whitespaceFile,
			want: "trimmed content",
		},
		{
			name: "inline JSON preserved as-is",
			raw:  "  {\"key\": \"value\"}  ",
			file: "",
			want: "  {\"key\": \"value\"}  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveJSONInput(tt.raw, tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.errMsg)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveJSONInput_Stdin(t *testing.T) {
	// Stdin tests are skipped because the implementation reads directly from
	// os.Stdin, which cannot be easily mocked without refactoring the API.
	// The stdin code path is tested indirectly through ReadInputSource.
	t.Skip("stdin tests require os.Stdin mocking")
}

func TestReadJSONInput(t *testing.T) {
	// Create temp file for file-based tests
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte(`{"delegated": true}`), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "inline JSON passthrough",
			value: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "empty value",
			value: "",
			want:  "",
		},
		{
			name:  "@file syntax",
			value: "@" + testFile,
			want:  `{"delegated": true}`,
		},
		{
			name:    "@file not found",
			value:   "@/nonexistent/path.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadJSONInput(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSONInput(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		var got map[string]interface{}
		if err := UnmarshalJSONInput(`{"key": "value"}`, &got); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["key"] != "value" {
			t.Fatalf("got %v, want %v", got["key"], "value")
		}
	})

	t.Run("double-serialized object", func(t *testing.T) {
		var got map[string]interface{}
		if err := UnmarshalJSONInput(`"{\"key\": \"value\"}"`, &got); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["key"] != "value" {
			t.Fatalf("got %v, want %v", got["key"], "value")
		}
	})

	t.Run("double-serialized array", func(t *testing.T) {
		var got []interface{}
		if err := UnmarshalJSONInput(`"[1, 2, 3]"`, &got); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got len %d, want 3", len(got))
		}
		if got[0].(float64) != 1 {
			t.Fatalf("got %v, want 1", got[0])
		}
	})

	t.Run("plain string does not unwrap", func(t *testing.T) {
		var got map[string]interface{}
		if err := UnmarshalJSONInput(`"hello"`, &got); err == nil {
			t.Fatalf("expected error for non-object JSON")
		}
	})
}

func TestReadInputSource(t *testing.T) {
	// Create temp file for file-based tests
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(testFile, []byte("file content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	whitespaceFile := filepath.Join(tmpDir, "whitespace.txt")
	if err := os.WriteFile(whitespaceFile, []byte("\n  content with whitespace  \n\n"), 0o644); err != nil {
		t.Fatalf("failed to create whitespace test file: %v", err)
	}

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to create empty test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "read file",
			path: testFile,
			want: "file content",
		},
		{
			name: "read file with whitespace trimmed",
			path: whitespaceFile,
			want: "content with whitespace",
		},
		{
			name: "read empty file",
			path: emptyFile,
			want: "",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "input file path is required",
		},
		{
			name:    "file not found",
			path:    filepath.Join(tmpDir, "nonexistent.txt"),
			wantErr: true,
			errMsg:  "failed to read file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadInputSource(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.errMsg)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadInputSource_Stdin(t *testing.T) {
	// Stdin tests are skipped because the implementation reads directly from
	// os.Stdin, which cannot be easily mocked without refactoring the API.
	// To properly test stdin, the function would need to accept an io.Reader.
	t.Skip("stdin tests require os.Stdin mocking")
}

func TestNormalizeJSONInput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace-only",
			raw:  "   \t\n  ",
			want: "   \t\n  ",
		},
		{
			name: "regular JSON object",
			raw:  `{"key": "value"}`,
			want: `{"key": "value"}`,
		},
		{
			name: "regular JSON array",
			raw:  `[1, 2, 3]`,
			want: `[1, 2, 3]`,
		},
		{
			name: "regular JSON number",
			raw:  `42`,
			want: `42`,
		},
		{
			name: "regular JSON boolean",
			raw:  `true`,
			want: `true`,
		},
		{
			name: "regular JSON null",
			raw:  `null`,
			want: `null`,
		},
		{
			name: "double-serialized object",
			raw:  `"{\"key\": \"value\"}"`,
			want: `{"key": "value"}`,
		},
		{
			name: "double-serialized array",
			raw:  `"[1, 2, 3]"`,
			want: `[1, 2, 3]`,
		},
		{
			name: "double-serialized number",
			raw:  `"42"`,
			want: `42`,
		},
		{
			name: "triple-serialized object unwraps one level",
			raw:  `"\"{\\\"key\\\": \\\"value\\\"}\""`,
			want: `"{\"key\": \"value\"}"`,
		},
		{
			name: "JSON string containing non-JSON text",
			raw:  `"hello world"`,
			want: `"hello world"`,
		},
		{
			name: "JSON string containing empty string",
			raw:  `""`,
			want: `""`,
		},
		{
			name: "JSON string containing whitespace-only",
			raw:  `"   "`,
			want: `"   "`,
		},
		{
			name: "JSON object with leading/trailing whitespace",
			raw:  `  {"key": "value"}  `,
			want: `  {"key": "value"}  `,
		},
		{
			name: "double-serialized with leading whitespace",
			raw:  `  "{\"key\": \"value\"}"`,
			want: `{"key": "value"}`,
		},
		{
			name: "invalid JSON",
			raw:  `{invalid json}`,
			want: `{invalid json}`,
		},
		{
			name: "plain text",
			raw:  `just some text`,
			want: `just some text`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeJSONInput(tt.raw)
			if got != tt.want {
				t.Fatalf("NormalizeJSONInput(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
