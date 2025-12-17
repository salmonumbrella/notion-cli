package notion

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Database represents a Notion database.
// See: https://developers.notion.com/reference/database
type Database struct {
	Object         string                            `json:"object"`
	ID             string                            `json:"id"`
	CreatedTime    string                            `json:"created_time"`
	LastEditedTime string                            `json:"last_edited_time"`
	CreatedBy      map[string]interface{}            `json:"created_by,omitempty"`
	LastEditedBy   map[string]interface{}            `json:"last_edited_by,omitempty"`
	Title          []map[string]interface{}          `json:"title"`
	Description    []map[string]interface{}          `json:"description,omitempty"`
	Icon           map[string]interface{}            `json:"icon,omitempty"`
	Cover          map[string]interface{}            `json:"cover,omitempty"`
	Properties     map[string]map[string]interface{} `json:"properties"`
	Parent         map[string]interface{}            `json:"parent,omitempty"`
	URL            string                            `json:"url,omitempty"`
	Archived       bool                              `json:"archived"`
	IsInline       bool                              `json:"is_inline,omitempty"`
	PublicURL      string                            `json:"public_url,omitempty"`
}

// DatabaseQueryRequest represents the request body for querying a database.
type DatabaseQueryRequest struct {
	Filter      map[string]interface{}   `json:"filter,omitempty"`
	Sorts       []map[string]interface{} `json:"sorts,omitempty"`
	StartCursor string                   `json:"start_cursor,omitempty"`
	PageSize    int                      `json:"page_size,omitempty"`
}

// DatabaseQueryResult represents the result of a database query.
// The results are pages from the database.
type DatabaseQueryResult struct {
	Object     string  `json:"object"`
	Results    []Page  `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Type       string  `json:"type,omitempty"`
}

// CreateDatabaseRequest represents the request body for creating a database.
type CreateDatabaseRequest struct {
	Parent      map[string]interface{}            `json:"parent"`
	Title       []map[string]interface{}          `json:"title,omitempty"`
	Description []map[string]interface{}          `json:"description,omitempty"`
	Properties  map[string]map[string]interface{} `json:"properties"`
	Icon        map[string]interface{}            `json:"icon,omitempty"`
	Cover       map[string]interface{}            `json:"cover,omitempty"`
	IsInline    bool                              `json:"is_inline,omitempty"`
}

// UpdateDatabaseRequest represents the request body for updating a database.
type UpdateDatabaseRequest struct {
	Title       []map[string]interface{}          `json:"title,omitempty"`
	Description []map[string]interface{}          `json:"description,omitempty"`
	Properties  map[string]map[string]interface{} `json:"properties,omitempty"`
	Icon        map[string]interface{}            `json:"icon,omitempty"`
	Cover       map[string]interface{}            `json:"cover,omitempty"`
	Archived    *bool                             `json:"archived,omitempty"`
}

// GetDatabase retrieves a database by ID.
// See: https://developers.notion.com/reference/retrieve-a-database
func (c *Client) GetDatabase(ctx context.Context, databaseID string) (*Database, error) {
	if databaseID == "" {
		return nil, fmt.Errorf("database ID is required")
	}

	path := fmt.Sprintf("/databases/%s", databaseID)
	var database Database

	if err := c.doGet(ctx, path, nil, &database); err != nil {
		return nil, err
	}

	return &database, nil
}

// QueryDatabase queries a database with optional filters, sorts, and pagination.
// See: https://developers.notion.com/reference/post-database-query
func (c *Client) QueryDatabase(ctx context.Context, databaseID string, req *DatabaseQueryRequest) (*DatabaseQueryResult, error) {
	if databaseID == "" {
		return nil, fmt.Errorf("database ID is required")
	}

	path := fmt.Sprintf("/databases/%s/query", databaseID)

	// Use empty request if none provided
	if req == nil {
		req = &DatabaseQueryRequest{}
	}

	var result DatabaseQueryResult
	if err := c.doPost(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateDatabase creates a new database.
// See: https://developers.notion.com/reference/create-a-database
func (c *Client) CreateDatabase(ctx context.Context, req *CreateDatabaseRequest) (*Database, error) {
	if req == nil {
		return nil, fmt.Errorf("create database request is required")
	}
	if req.Parent == nil {
		return nil, fmt.Errorf("parent is required")
	}
	if len(req.Properties) == 0 {
		return nil, fmt.Errorf("properties are required")
	}

	var database Database
	if err := c.doPost(ctx, "/databases", req, &database); err != nil {
		return nil, err
	}

	return &database, nil
}

// UpdateDatabase updates a database's metadata and properties.
// See: https://developers.notion.com/reference/update-a-database
func (c *Client) UpdateDatabase(ctx context.Context, databaseID string, req *UpdateDatabaseRequest) (*Database, error) {
	if databaseID == "" {
		return nil, fmt.Errorf("database ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update database request is required")
	}

	path := fmt.Sprintf("/databases/%s", databaseID)
	var database Database

	if err := c.doPatch(ctx, path, req, &database); err != nil {
		return nil, err
	}

	return &database, nil
}

// BuildDatabaseQueryOptions is a helper struct for building database queries with URL parameters.
// This is useful when you want to construct query options from CLI flags.
type BuildDatabaseQueryOptions struct {
	FilterJSON  string
	SortsJSON   string
	StartCursor string
	PageSize    int
}

// BuildQueryURL constructs a database query URL with the given options.
// This is a helper function for CLI commands.
func BuildQueryURL(baseURL string, opts *BuildDatabaseQueryOptions) (string, error) {
	if opts == nil {
		return baseURL, nil
	}

	query := url.Values{}

	if opts.StartCursor != "" {
		query.Set("start_cursor", opts.StartCursor)
	}
	if opts.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(opts.PageSize))
	}

	if len(query) > 0 {
		return baseURL + "?" + query.Encode(), nil
	}

	return baseURL, nil
}
