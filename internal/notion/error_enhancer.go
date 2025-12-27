package notion

import (
	"context"
	"errors"
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

	fmt.Fprintf(&sb, "Invalid status value %q\n", e.InvalidValue)

	for _, prop := range e.StatusProperties {
		fmt.Fprintf(&sb, "\nProperty: %s\n", prop.Name)
		sb.WriteString("Valid options:\n")
		for _, opt := range prop.Options {
			if opt.Description != "" {
				fmt.Fprintf(&sb, "  - %s (%s)\n", opt.Name, opt.Description)
			} else {
				fmt.Fprintf(&sb, "  - %s\n", opt.Name)
			}
		}
	}

	return sb.String()
}

// Unwrap returns the original error.
func (e *EnhancedStatusError) Unwrap() error {
	return e.OriginalError
}

// EnhanceStatusError attempts to enhance a status validation error with valid options.
// If the error is not a status validation error or enhancement fails, returns the original error.
func EnhanceStatusError(ctx context.Context, client *Client, pageID string, err error) error {
	if err == nil {
		return nil
	}

	// Extract the error message
	var message string
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Response != nil {
		message = apiErr.Response.Message
	} else {
		message = err.Error()
	}

	// Check if it's a status validation error
	if !IsStatusValidationError(message) {
		return err
	}

	invalidValue := ExtractInvalidStatusValue(message)
	if invalidValue == "" {
		return err
	}

	// If no client provided, can't enhance
	if client == nil {
		return err
	}

	// Try to fetch schema and enhance the error
	statusProps, fetchErr := fetchStatusProperties(ctx, client, pageID)
	if fetchErr != nil || len(statusProps) == 0 {
		return err // Fall back to original error
	}

	return &EnhancedStatusError{
		InvalidValue:     invalidValue,
		StatusProperties: statusProps,
		OriginalError:    err,
	}
}

// fetchStatusProperties fetches status properties for a page's parent database.
func fetchStatusProperties(ctx context.Context, client *Client, pageID string) ([]StatusProperty, error) {
	// 1. Get page to find parent database
	page, err := client.GetPage(ctx, pageID)
	if err != nil {
		return nil, err
	}

	// 2. Extract database ID from parent
	parent := page.Parent
	if parent == nil {
		return nil, fmt.Errorf("page has no parent")
	}

	dbID, _ := parent["database_id"].(string)
	if dbID == "" {
		// Try data_source_id for newer API
		dsID, _ := parent["data_source_id"].(string)
		if dsID != "" {
			return fetchStatusPropertiesFromDataSource(ctx, client, dsID)
		}
		return nil, fmt.Errorf("page parent is not a database")
	}

	// 3. Get database to find data source
	db, err := client.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, err
	}

	if len(db.DataSources) == 0 {
		return nil, fmt.Errorf("database has no data sources")
	}

	// 4. Get data source schema
	return fetchStatusPropertiesFromDataSource(ctx, client, db.DataSources[0].ID)
}

// fetchStatusPropertiesFromDataSource fetches status properties from a data source.
func fetchStatusPropertiesFromDataSource(ctx context.Context, client *Client, dsID string) ([]StatusProperty, error) {
	ds, err := client.GetDataSource(ctx, dsID)
	if err != nil {
		return nil, err
	}

	return ExtractStatusOptions(ds.Properties), nil
}
