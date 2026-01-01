package notion

import (
	"fmt"
	"regexp"
	"strings"
)

// statusOptionRegex matches "Status option "X" does not exist"
var statusOptionRegex = regexp.MustCompile(`Status option "([^"]+)" does not exist`)

// IsStatusValidationError checks if an error message indicates an invalid status option.
func IsStatusValidationError(message string) bool {
	return statusOptionRegex.MatchString(message)
}

// ExtractInvalidStatusValue extracts the invalid status value from an error message.
// Returns empty string if not found.
func ExtractInvalidStatusValue(message string) string {
	matches := statusOptionRegex.FindStringSubmatch(message)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// StatusProperty represents a status property with its options.
type StatusProperty struct {
	Name    string
	Options []StatusOption
}

// ExtractStatusOptions extracts all status properties and their options from datasource properties.
func ExtractStatusOptions(properties map[string]interface{}) []StatusProperty {
	var result []StatusProperty

	for propName, propValue := range properties {
		propMap, ok := propValue.(map[string]interface{})
		if !ok {
			continue
		}

		propType, _ := propMap["type"].(string)
		if propType != "status" {
			continue
		}

		statusConfig, ok := propMap["status"].(map[string]interface{})
		if !ok {
			continue
		}

		optionsRaw, ok := statusConfig["options"].([]interface{})
		if !ok {
			continue
		}

		var options []StatusOption
		for _, optRaw := range optionsRaw {
			optMap, ok := optRaw.(map[string]interface{})
			if !ok {
				continue
			}

			opt := StatusOption{
				Name:        getString(optMap, "name"),
				Description: getString(optMap, "description"),
				Color:       getString(optMap, "color"),
			}
			options = append(options, opt)
		}

		result = append(result, StatusProperty{
			Name:    propName,
			Options: options,
		})
	}

	return result
}

// getString safely extracts a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// EnhancedStatusError provides a user-friendly error for invalid status values.
type EnhancedStatusError struct {
	InvalidValue     string
	StatusProperties []StatusProperty
	OriginalError    error
}

// Error implements the error interface with a formatted message showing valid options.
func (e *EnhancedStatusError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Invalid status value %q\n", e.InvalidValue))

	for _, prop := range e.StatusProperties {
		sb.WriteString(fmt.Sprintf("\nProperty: %s\n", prop.Name))
		sb.WriteString("Valid options:\n")
		for _, opt := range prop.Options {
			if opt.Description != "" {
				sb.WriteString(fmt.Sprintf("  - %s (%s)\n", opt.Name, opt.Description))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s\n", opt.Name))
			}
		}
	}

	return sb.String()
}

// Unwrap returns the original error.
func (e *EnhancedStatusError) Unwrap() error {
	return e.OriginalError
}
