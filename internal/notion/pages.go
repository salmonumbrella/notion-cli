package notion

import (
	"context"
	"fmt"
	"net/url"
)

// Page represents a Notion page.
// See: https://developers.notion.com/reference/page
type Page struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	CreatedBy      map[string]interface{} `json:"created_by,omitempty"`
	LastEditedBy   map[string]interface{} `json:"last_edited_by,omitempty"`
	Cover          map[string]interface{} `json:"cover,omitempty"`
	Icon           map[string]interface{} `json:"icon,omitempty"`
	Parent         map[string]interface{} `json:"parent,omitempty"`
	Archived       bool                   `json:"archived"`
	InTrash        bool                   `json:"in_trash,omitempty"`
	Properties     map[string]interface{} `json:"properties"`
	URL            string                 `json:"url,omitempty"`
	PublicURL      string                 `json:"public_url,omitempty"`
}

// CreatePageRequest represents the request body for creating a page.
type CreatePageRequest struct {
	Parent     map[string]interface{} `json:"parent"`
	Properties map[string]interface{} `json:"properties"`
	Children   []interface{}          `json:"children,omitempty"`
	Icon       map[string]interface{} `json:"icon,omitempty"`
	Cover      map[string]interface{} `json:"cover,omitempty"`
}

// UpdatePageRequest represents the request body for updating a page.
type UpdatePageRequest struct {
	Properties map[string]interface{} `json:"properties,omitempty"`
	Archived   *bool                  `json:"archived,omitempty"`
	InTrash    *bool                  `json:"in_trash,omitempty"`
	Icon       map[string]interface{} `json:"icon,omitempty"`
	Cover      map[string]interface{} `json:"cover,omitempty"`
}

// PageProperty represents a page property value.
// Properties can be paginated so this might contain pagination info.
type PageProperty struct {
	Object  string                 `json:"object"`
	ID      string                 `json:"id,omitempty"`
	Type    string                 `json:"type"`
	Results []interface{}          `json:"results"`
	Data    map[string]interface{} `json:"-"` // Catch-all for the actual property data
}

// GetPage retrieves a page by ID.
// See: https://developers.notion.com/reference/retrieve-a-page
func (c *Client) GetPage(ctx context.Context, pageID string) (*Page, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page ID is required")
	}

	path := fmt.Sprintf("/pages/%s", pageID)
	var page Page

	if err := c.doGet(ctx, path, nil, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// CreatePage creates a new page.
// See: https://developers.notion.com/reference/post-page
func (c *Client) CreatePage(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	if req == nil {
		return nil, fmt.Errorf("create page request is required")
	}
	if req.Parent == nil {
		return nil, fmt.Errorf("parent is required")
	}
	if req.Properties == nil {
		return nil, fmt.Errorf("properties are required")
	}

	var page Page
	if err := c.doPost(ctx, "/pages", req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// UpdatePage updates a page's properties.
// See: https://developers.notion.com/reference/patch-page
func (c *Client) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update page request is required")
	}

	path := fmt.Sprintf("/pages/%s", pageID)
	var page Page

	if err := c.doPatch(ctx, path, req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// GetPageProperty retrieves a specific page property value.
// See: https://developers.notion.com/reference/retrieve-a-page-property
func (c *Client) GetPageProperty(ctx context.Context, pageID, propertyID string) (map[string]interface{}, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page ID is required")
	}
	if propertyID == "" {
		return nil, fmt.Errorf("property ID is required")
	}

	path := fmt.Sprintf("/pages/%s/properties/%s", pageID, url.PathEscape(propertyID))
	var result map[string]interface{}

	if err := c.doGet(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// MovePageRequest represents a request to move a page.
type MovePageRequest struct {
	Parent map[string]interface{} `json:"parent"`
	After  string                 `json:"after,omitempty"` // Block ID to position after
}

// MovePage moves a page to a new parent.
// See: https://developers.notion.com/reference/move-page
func (c *Client) MovePage(ctx context.Context, pageID string, req *MovePageRequest) (*Page, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("move page request is required")
	}
	if req.Parent == nil {
		return nil, fmt.Errorf("parent is required")
	}

	path := fmt.Sprintf("/pages/%s", pageID)
	var page Page

	if err := c.doPatch(ctx, path, req, &page); err != nil {
		return nil, err
	}

	return &page, nil
}
