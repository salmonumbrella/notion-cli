package notion

import (
	"regexp"
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
