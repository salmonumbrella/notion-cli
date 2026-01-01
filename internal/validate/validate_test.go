package validate

import (
	"strings"
	"testing"
)

func TestUUID(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       string
		wantError   bool
		errContains string
	}{
		{
			name:      "valid UUID with dashes",
			field:     "page_id",
			value:     "12345678-1234-1234-1234-123456789abc",
			wantError: false,
		},
		{
			name:      "valid UUID without dashes",
			field:     "database_id",
			value:     "123456781234123412341234567890ab",
			wantError: false,
		},
		{
			name:        "valid UUID mixed case",
			field:       "id",
			value:       "12345678-1234-1234-1234-123456789ABC",
			wantError:   true, // regex requires lowercase
			errContains: "must be a valid UUID",
		},
		{
			name:        "empty UUID",
			field:       "page_id",
			value:       "",
			wantError:   true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid UUID too short",
			field:       "page_id",
			value:       "12345678-1234-1234-1234-12345678",
			wantError:   true,
			errContains: "must be a valid UUID",
		},
		{
			name:        "invalid UUID with invalid chars",
			field:       "page_id",
			value:       "12345678-1234-1234-1234-12345678ghij",
			wantError:   true,
			errContains: "must be a valid UUID",
		},
		{
			name:        "invalid UUID wrong format",
			field:       "page_id",
			value:       "not-a-uuid",
			wantError:   true,
			errContains: "must be a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UUID(tt.field, tt.value)
			if tt.wantError {
				if err == nil {
					t.Errorf("UUID() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UUID() error = %v, should contain %q", err, tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("UUID() error = %v, should contain field name %q", err, tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("UUID() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestPageSize(t *testing.T) {
	tests := []struct {
		name        string
		size        int
		wantError   bool
		errContains string
	}{
		{
			name:      "valid minimum size",
			size:      1,
			wantError: false,
		},
		{
			name:      "valid maximum size",
			size:      100,
			wantError: false,
		},
		{
			name:      "valid middle size",
			size:      50,
			wantError: false,
		},
		{
			name:        "invalid zero",
			size:        0,
			wantError:   true,
			errContains: "must be at least 1",
		},
		{
			name:        "invalid negative",
			size:        -5,
			wantError:   true,
			errContains: "must be at least 1",
		},
		{
			name:        "invalid too large",
			size:        101,
			wantError:   true,
			errContains: "must be at most 100",
		},
		{
			name:        "invalid way too large",
			size:        1000,
			wantError:   true,
			errContains: "must be at most 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PageSize(tt.size)
			if tt.wantError {
				if err == nil {
					t.Errorf("PageSize() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("PageSize() error = %v, should contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("PageSize() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       string
		wantError   bool
		errContains string
	}{
		{
			name:      "valid non-empty string",
			field:     "title",
			value:     "some text",
			wantError: false,
		},
		{
			name:      "valid single character",
			field:     "char",
			value:     "a",
			wantError: false,
		},
		{
			name:      "valid whitespace (not trimmed)",
			field:     "text",
			value:     "   ",
			wantError: false,
		},
		{
			name:        "invalid empty string",
			field:       "name",
			value:       "",
			wantError:   true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NonEmpty(tt.field, tt.value)
			if tt.wantError {
				if err == nil {
					t.Errorf("NonEmpty() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NonEmpty() error = %v, should contain %q", err, tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("NonEmpty() error = %v, should contain field name %q", err, tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("NonEmpty() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestJSONObject(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		data        string
		wantError   bool
		errContains string
	}{
		{
			name:      "valid empty object",
			field:     "properties",
			data:      "{}",
			wantError: false,
		},
		{
			name:      "valid object with fields",
			field:     "properties",
			data:      `{"name": "test", "count": 42}`,
			wantError: false,
		},
		{
			name:      "valid nested object",
			field:     "properties",
			data:      `{"user": {"name": "test", "age": 30}}`,
			wantError: false,
		},
		{
			name:        "invalid empty string",
			field:       "properties",
			data:        "",
			wantError:   true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid not JSON",
			field:       "properties",
			data:        "not json",
			wantError:   true,
			errContains: "must be valid JSON object",
		},
		{
			name:        "invalid JSON array",
			field:       "properties",
			data:        `["array", "not", "object"]`,
			wantError:   true,
			errContains: "must be valid JSON object",
		},
		{
			name:        "invalid JSON string",
			field:       "properties",
			data:        `"just a string"`,
			wantError:   true,
			errContains: "must be valid JSON object",
		},
		{
			name:        "invalid JSON number",
			field:       "properties",
			data:        `42`,
			wantError:   true,
			errContains: "must be valid JSON object",
		},
		{
			name:        "invalid malformed JSON",
			field:       "properties",
			data:        `{"key": "value"`,
			wantError:   true,
			errContains: "must be valid JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONObject(tt.field, tt.data)
			if tt.wantError {
				if err == nil {
					t.Errorf("JSONObject() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("JSONObject() error = %v, should contain %q", err, tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("JSONObject() error = %v, should contain field name %q", err, tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("JSONObject() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestDate(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		dateStr     string
		wantError   bool
		errContains string
	}{
		{
			name:      "valid ISO date",
			field:     "created_time",
			dateStr:   "2024-12-19",
			wantError: false,
		},
		{
			name:      "valid RFC3339 datetime",
			field:     "created_time",
			dateStr:   "2024-12-19T10:30:00Z",
			wantError: false,
		},
		{
			name:      "valid RFC3339 with timezone",
			field:     "created_time",
			dateStr:   "2024-12-19T10:30:00-08:00",
			wantError: false,
		},
		{
			name:        "invalid empty string",
			field:       "created_time",
			dateStr:     "",
			wantError:   true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid format",
			field:       "created_time",
			dateStr:     "12/19/2024",
			wantError:   true,
			errContains: "must be a valid ISO 8601 date",
		},
		{
			name:        "invalid partial date",
			field:       "created_time",
			dateStr:     "2024-12",
			wantError:   true,
			errContains: "must be a valid ISO 8601 date",
		},
		{
			name:        "invalid not a date",
			field:       "created_time",
			dateStr:     "not-a-date",
			wantError:   true,
			errContains: "must be a valid ISO 8601 date",
		},
		{
			name:        "invalid date values",
			field:       "created_time",
			dateStr:     "2024-13-45",
			wantError:   true,
			errContains: "must be a valid ISO 8601 date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Date(tt.field, tt.dateStr)
			if tt.wantError {
				if err == nil {
					t.Errorf("Date() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Date() error = %v, should contain %q", err, tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("Date() error = %v, should contain field name %q", err, tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("Date() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		urlStr      string
		wantError   bool
		errContains string
	}{
		{
			name:      "valid HTTP URL",
			field:     "icon_url",
			urlStr:    "http://example.com",
			wantError: false,
		},
		{
			name:      "valid HTTPS URL",
			field:     "icon_url",
			urlStr:    "https://example.com",
			wantError: false,
		},
		{
			name:      "valid URL with path",
			field:     "icon_url",
			urlStr:    "https://example.com/path/to/resource",
			wantError: false,
		},
		{
			name:      "valid URL with query",
			field:     "icon_url",
			urlStr:    "https://example.com/path?key=value",
			wantError: false,
		},
		{
			name:      "valid URL with fragment",
			field:     "icon_url",
			urlStr:    "https://example.com/path#section",
			wantError: false,
		},
		{
			name:      "valid custom scheme",
			field:     "callback_url",
			urlStr:    "notion://page/123",
			wantError: false,
		},
		{
			name:        "invalid empty string",
			field:       "icon_url",
			urlStr:      "",
			wantError:   true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid no scheme",
			field:       "icon_url",
			urlStr:      "example.com",
			wantError:   true,
			errContains: "must have a scheme",
		},
		{
			name:        "invalid no host",
			field:       "icon_url",
			urlStr:      "http://",
			wantError:   true,
			errContains: "must have a host",
		},
		{
			name:        "invalid malformed URL",
			field:       "icon_url",
			urlStr:      "ht!tp://example.com",
			wantError:   true,
			errContains: "must be a valid URL",
		},
		{
			name:        "invalid just a path",
			field:       "icon_url",
			urlStr:      "/just/a/path",
			wantError:   true,
			errContains: "must have a scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URL(tt.field, tt.urlStr)
			if tt.wantError {
				if err == nil {
					t.Errorf("URL() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("URL() error = %v, should contain %q", err, tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("URL() error = %v, should contain field name %q", err, tt.field)
				}
			} else {
				if err != nil {
					t.Errorf("URL() unexpected error = %v", err)
				}
			}
		})
	}
}
