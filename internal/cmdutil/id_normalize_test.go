package cmdutil

import "testing"

func TestNormalizeNotionID(t *testing.T) {
	const id = "12345678123412341234123456789012"

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty",
			input:   "   ",
			wantErr: true,
		},
		{
			name:  "raw id",
			input: id,
			want:  id,
		},
		{
			name:  "notion url",
			input: "https://www.notion.so/Some-Page-" + id,
			want:  id,
		},
		{
			name:  "trimmed",
			input: "  " + id + "  ",
			want:  id,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeNotionID(tt.input)
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

func TestLooksLikeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"http scheme", "https://notion.so/page", true},
		{"notion domain", "notion.so/page", true},
		{"slash", "page/with/slash", true},
		{"plain id", "12345678123412341234123456789012", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeURL(tt.input); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDesirePath_IDFormats verifies all ID formats agents naturally try.
func TestDesirePath_IDFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Dashed UUID
		{"dashed UUID", "12345678-1234-1234-1234-123456789012", false},

		// Undashed UUID
		{"undashed UUID", "12345678123412341234123456789012", false},

		// Notion URL (full)
		{"full notion URL", "https://www.notion.so/workspace/Page-Title-12345678123412341234123456789012", false},

		// Notion URL (short)
		{"short notion URL", "https://notion.so/12345678123412341234123456789012", false},

		// Notion site URL
		{"notion.site URL", "https://myworkspace.notion.site/Page-12345678123412341234123456789012", false},

		// URL without scheme
		{"URL without https", "notion.so/Page-12345678123412341234123456789012", false},

		// Empty input
		{"empty", "", true},

		// Whitespace
		{"whitespace only", "   ", true},

		// UUID with surrounding whitespace
		{"UUID with whitespace", "  12345678-1234-1234-1234-123456789012  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeNotionID(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func BenchmarkNormalizeNotionID(b *testing.B) {
	input := "https://www.notion.so/Page-12345678123412341234123456789012"
	for i := 0; i < b.N; i++ {
		_, _ = NormalizeNotionID(input)
	}
}
