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

func TestExtractStatusOptions(t *testing.T) {
	// Simulate datasource properties with a status property
	properties := map[string]interface{}{
		"電匯已發送 | Sent": map[string]interface{}{
			"type": "status",
			"status": map[string]interface{}{
				"options": []interface{}{
					map[string]interface{}{
						"name":        "未發送",
						"description": "Not Sent",
						"color":       "yellow",
					},
					map[string]interface{}{
						"name":        "已完成",
						"description": "Completed",
						"color":       "green",
					},
				},
			},
		},
		"Name": map[string]interface{}{
			"type": "title",
		},
	}

	options := ExtractStatusOptions(properties)

	if len(options) != 1 {
		t.Fatalf("expected 1 status property, got %d", len(options))
	}

	prop := options[0]
	if prop.Name != "電匯已發送 | Sent" {
		t.Errorf("expected property name '電匯已發送 | Sent', got %q", prop.Name)
	}
	if len(prop.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(prop.Options))
	}
	if prop.Options[0].Name != "未發送" {
		t.Errorf("expected first option '未發送', got %q", prop.Options[0].Name)
	}
}
