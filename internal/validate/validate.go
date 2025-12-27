package validate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"
)

// uuidRegex matches UUIDs with or without dashes
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`)

// UUID validates that the value is a valid Notion UUID (with or without dashes).
// UUIDs can be in the format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx or xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
func UUID(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}
	if !uuidRegex.MatchString(value) {
		return fmt.Errorf("%s: must be a valid UUID, got %q", field, value)
	}
	return nil
}

// PageSize validates that the size is within the valid range for Notion API pagination (1-100).
func PageSize(size int) error {
	if size < 1 {
		return fmt.Errorf("page_size: must be at least 1, got %d", size)
	}
	if size > 100 {
		return fmt.Errorf("page_size: must be at most 100, got %d", size)
	}
	return nil
}

// NonEmpty validates that a required string field is not empty.
func NonEmpty(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}
	return nil
}

// JSONObject validates that the data is valid JSON and is an object (not array, string, etc.).
func JSONObject(field, data string) error {
	if data == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return fmt.Errorf("%s: must be valid JSON object, got error: %v", field, err)
	}

	return nil
}

// Date validates that the dateStr is in ISO 8601 date format (YYYY-MM-DD).
// Also accepts full ISO 8601 datetime formats.
func Date(field, dateStr string) error {
	if dateStr == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}

	// Try parsing as date only (YYYY-MM-DD)
	if _, err := time.Parse("2006-01-02", dateStr); err == nil {
		return nil
	}

	// Try parsing as RFC3339 (ISO 8601 datetime)
	if _, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return nil
	}

	return fmt.Errorf("%s: must be a valid ISO 8601 date (YYYY-MM-DD) or datetime, got %q", field, dateStr)
}

// URL validates that the urlStr is a valid URL with a scheme and host.
func URL(field, urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%s: must be a valid URL, got error: %v", field, err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("%s: must have a scheme (http, https, etc.), got %q", field, urlStr)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%s: must have a host, got %q", field, urlStr)
	}

	return nil
}
