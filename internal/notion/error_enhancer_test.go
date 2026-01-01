package notion

import (
	"testing"
)

func TestIsStatusValidationError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "status validation error",
			message:  `Invalid status option. Status option "已發送" does not exist".`,
			expected: true,
		},
		{
			name:     "status validation error english",
			message:  `Invalid status option. Status option "Sent" does not exist".`,
			expected: true,
		},
		{
			name:     "other validation error",
			message:  "Invalid property type",
			expected: false,
		},
		{
			name:     "empty message",
			message:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStatusValidationError(tt.message)
			if result != tt.expected {
				t.Errorf("IsStatusValidationError(%q) = %v, want %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestExtractInvalidStatusValue(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "chinese value",
			message:  `Invalid status option. Status option "已發送" does not exist".`,
			expected: "已發送",
		},
		{
			name:     "english value",
			message:  `Invalid status option. Status option "Sent" does not exist".`,
			expected: "Sent",
		},
		{
			name:     "no match",
			message:  "Invalid property type",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractInvalidStatusValue(tt.message)
			if result != tt.expected {
				t.Errorf("ExtractInvalidStatusValue(%q) = %q, want %q", tt.message, result, tt.expected)
			}
		})
	}
}
