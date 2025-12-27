package notion

import (
	"context"
	"fmt"
	"net/url"
)

// DataSource represents a Notion data source (table within a database).
// See: https://developers.notion.com/reference/data-source-object
type DataSource struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Title          []RichText             `json:"title"`
	Properties     map[string]interface{} `json:"properties"`
	Parent         map[string]interface{} `json:"parent"`
}

// DataSourceList represents a paginated list of data sources.
type DataSourceList struct {
	Object     string        `json:"object"`
	Results    []*DataSource `json:"results"`
	NextCursor *string       `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

// CreateDataSourceRequest represents a request to create a data source.
type CreateDataSourceRequest struct {
	Parent     map[string]interface{} `json:"parent"`
	Title      []RichText             `json:"title,omitempty"`
	Properties map[string]interface{} `json:"properties"`
}

// UpdateDataSourceRequest represents a request to update a data source.
type UpdateDataSourceRequest struct {
	Title      []RichText             `json:"title,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Parent     map[string]interface{} `json:"parent,omitempty"`
}

// QueryDataSourceRequest represents a data source query.
type QueryDataSourceRequest struct {
	Filter      map[string]interface{}   `json:"filter,omitempty"`
	Sorts       []map[string]interface{} `json:"sorts,omitempty"`
	StartCursor string                   `json:"start_cursor,omitempty"`
	PageSize    int                      `json:"page_size,omitempty"`
}

// DataSourceTemplate represents a data source template.
type DataSourceTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// DataSourceTemplateList represents a list of templates.
type DataSourceTemplateList struct {
	Object  string                `json:"object"`
	Results []*DataSourceTemplate `json:"results"`
}

// DataSourceQueryResult represents the result of querying a data source.
type DataSourceQueryResult struct {
	Object     string  `json:"object"`
	Results    []Page  `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Type       string  `json:"type,omitempty"`
}

// CreateDataSource creates a new data source.
// See: https://developers.notion.com/reference/create-a-data-source
func (c *Client) CreateDataSource(ctx context.Context, req *CreateDataSourceRequest) (*DataSource, error) {
	if req == nil || req.Parent == nil {
		return nil, fmt.Errorf("parent is required")
	}

	var ds DataSource
	if err := c.doPost(ctx, "/data_sources", req, &ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// GetDataSource retrieves a data source by ID.
// See: https://developers.notion.com/reference/retrieve-a-data-source
func (c *Client) GetDataSource(ctx context.Context, dataSourceID string) (*DataSource, error) {
	if dataSourceID == "" {
		return nil, fmt.Errorf("data source ID is required")
	}

	path := fmt.Sprintf("/data_sources/%s", dataSourceID)
	var ds DataSource

	if err := c.doGet(ctx, path, nil, &ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// UpdateDataSource updates a data source.
// See: https://developers.notion.com/reference/update-a-data-source
func (c *Client) UpdateDataSource(ctx context.Context, dataSourceID string, req *UpdateDataSourceRequest) (*DataSource, error) {
	if dataSourceID == "" {
		return nil, fmt.Errorf("data source ID is required")
	}

	path := fmt.Sprintf("/data_sources/%s", dataSourceID)
	var ds DataSource

	if err := c.doPatch(ctx, path, req, &ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

// QueryDataSource queries a data source.
// See: https://developers.notion.com/reference/query-a-data-source
func (c *Client) QueryDataSource(ctx context.Context, dataSourceID string, req *QueryDataSourceRequest) (*DataSourceQueryResult, error) {
	if dataSourceID == "" {
		return nil, fmt.Errorf("data source ID is required")
	}

	path := fmt.Sprintf("/data_sources/%s/query", dataSourceID)
	var result DataSourceQueryResult

	if err := c.doPost(ctx, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListDataSourceTemplates lists available data source templates.
// See: https://developers.notion.com/reference/list-data-source-templates
func (c *Client) ListDataSourceTemplates(ctx context.Context) (*DataSourceTemplateList, error) {
	var list DataSourceTemplateList
	if err := c.doGet(ctx, "/data_sources/templates", url.Values{}, &list); err != nil {
		return nil, err
	}
	return &list, nil
}
