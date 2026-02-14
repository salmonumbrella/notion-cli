package cmd

import (
	"strings"
	"testing"
)

func TestValidateFileExtension(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
		errMsg  string
	}{
		{name: "pdf supported", file: "report.pdf", wantErr: false},
		{name: "txt supported", file: "notes.txt", wantErr: false},
		{name: "jpg supported", file: "photo.jpg", wantErr: false},
		{name: "png supported", file: "image.png", wantErr: false},
		{name: "mp4 supported", file: "video.mp4", wantErr: false},
		{name: "case insensitive", file: "image.PNG", wantErr: false},
		{name: "md unsupported with workaround", file: "notes.md", wantErr: true, errMsg: "rename to .txt"},
		{name: "yaml unsupported with workaround", file: "config.yaml", wantErr: true, errMsg: "rename to .txt"},
		{name: "no extension", file: "Makefile", wantErr: true, errMsg: "no extension"},
		{name: "unknown extension", file: "data.xyz", wantErr: true, errMsg: "does not support"},
		{name: "path with dirs", file: "/tmp/docs/report.pdf", wantErr: false},
		{name: "markdown unsupported", file: "README.markdown", wantErr: true, errMsg: "rename to .txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileExtension(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.file)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tt.file, err)
				}
			}
		})
	}
}

func TestSupportedExtensionsList(t *testing.T) {
	list := supportedExtensionsList()
	// Should contain known extensions
	if !strings.Contains(list, ".pdf") {
		t.Error("expected .pdf in supported list")
	}
	if !strings.Contains(list, ".txt") {
		t.Error("expected .txt in supported list")
	}
	// Should be sorted (spot check)
	pdfIdx := strings.Index(list, ".pdf")
	txtIdx := strings.Index(list, ".txt")
	if pdfIdx > txtIdx {
		t.Error("expected sorted output (.pdf before .txt)")
	}
}
