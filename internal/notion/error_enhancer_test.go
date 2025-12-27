package notion

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestEnhancedStatusError_Error(t *testing.T) {
	err := &EnhancedStatusError{
		InvalidValue: "已發送",
		StatusProperties: []StatusProperty{
			{
				Name: "電匯已發送 | Sent",
				Options: []StatusOption{
					{Name: "未發送", Description: "Not Sent"},
					{Name: "已完成", Description: "Completed"},
				},
			},
		},
	}

	msg := err.Error()

	// Check key parts are present
	if !strings.Contains(msg, "已發送") {
		t.Error("error should contain invalid value")
	}
	if !strings.Contains(msg, "電匯已發送 | Sent") {
		t.Error("error should contain property name")
	}
	if !strings.Contains(msg, "未發送") {
		t.Error("error should contain valid option")
	}
	if !strings.Contains(msg, "已完成") {
		t.Error("error should contain valid option")
	}
}

func TestEnhanceStatusError_NotStatusError(t *testing.T) {
	originalErr := fmt.Errorf("some other error")

	result := EnhanceStatusError(context.Background(), nil, "", originalErr)

	// Should return original error unchanged
	if result != originalErr {
		t.Error("non-status errors should pass through unchanged")
	}
}

func TestEnhanceStatusError_NilError(t *testing.T) {
	result := EnhanceStatusError(context.Background(), nil, "page123", nil)
	if result != nil {
		t.Errorf("expected nil for nil error, got %v", result)
	}
}

func TestEnhanceStatusError_StatusErrorNoClient(t *testing.T) {
	// When client is nil, should return original error unchanged
	originalErr := &APIError{
		Response: &ErrorResponse{
			Status:  400,
			Code:    "validation_error",
			Message: `Invalid status option. Status option "Test" does not exist".`,
		},
	}

	result := EnhanceStatusError(context.Background(), nil, "page123", originalErr)

	// Should return original error unchanged since client is nil
	if result != originalErr {
		t.Errorf("expected original error when client is nil, got %v", result)
	}
}

func TestExtractStatusOptions_EmptyProperties(t *testing.T) {
	options := ExtractStatusOptions(nil)
	if len(options) != 0 {
		t.Errorf("expected empty slice for nil properties, got %d items", len(options))
	}

	options = ExtractStatusOptions(map[string]interface{}{})
	if len(options) != 0 {
		t.Errorf("expected empty slice for empty properties, got %d items", len(options))
	}
}

func TestExtractStatusOptions_MalformedData(t *testing.T) {
	tests := []struct {
		name          string
		properties    map[string]interface{}
		expectedProps int // number of status properties returned
	}{
		{
			name: "property value not a map",
			properties: map[string]interface{}{
				"Status": "not a map",
			},
			expectedProps: 0,
		},
		{
			name: "status config not a map",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"type":   "status",
					"status": "not a map",
				},
			},
			expectedProps: 0,
		},
		{
			name: "options not a slice",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"type": "status",
					"status": map[string]interface{}{
						"options": "not a slice",
					},
				},
			},
			expectedProps: 0,
		},
		{
			name: "option not a map - returns property with empty options",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"type": "status",
					"status": map[string]interface{}{
						"options": []interface{}{"not a map"},
					},
				},
			},
			// This case returns a StatusProperty with empty Options slice
			// because the status property structure is valid, just the individual
			// option is malformed and gets skipped
			expectedProps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := ExtractStatusOptions(tt.properties)
			if len(options) != tt.expectedProps {
				t.Errorf("expected %d status properties, got %d", tt.expectedProps, len(options))
			}
			// For the case with 1 property but malformed options, verify options are empty
			if tt.expectedProps == 1 && len(options) == 1 && len(options[0].Options) != 0 {
				t.Errorf("expected 0 options for malformed option data, got %d", len(options[0].Options))
			}
		})
	}
}

func TestEnhancedStatusError_EmptyStatusProperties(t *testing.T) {
	err := &EnhancedStatusError{
		InvalidValue:     "Test",
		StatusProperties: nil,
		OriginalError:    errors.New("original"),
	}

	msg := err.Error()
	if !strings.Contains(msg, "Test") {
		t.Error("error message should contain invalid value")
	}
}

func TestEnhancedStatusError_OptionWithDescription(t *testing.T) {
	err := &EnhancedStatusError{
		InvalidValue: "Bad",
		StatusProperties: []StatusProperty{
			{
				Name: "Status",
				Options: []StatusOption{
					{Name: "Good", Description: "A good state"},
					{Name: "Better", Description: ""},
				},
			},
		},
	}

	msg := err.Error()
	if !strings.Contains(msg, "Good (A good state)") {
		t.Errorf("expected description in parentheses, got: %s", msg)
	}
	if strings.Contains(msg, "Better (") {
		t.Errorf("should not show parentheses for empty description, got: %s", msg)
	}
}
