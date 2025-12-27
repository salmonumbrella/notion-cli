package notion

import (
	"context"
)

// SearchRequest represents the request body for the search endpoint.
// See: https://developers.notion.com/reference/post-search
type SearchRequest struct {
	Query       string                 `json:"query,omitempty"`
	Sort        map[string]interface{} `json:"sort,omitempty"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	StartCursor string                 `json:"start_cursor,omitempty"`
	PageSize    int                    `json:"page_size,omitempty"`
}

// SearchResult represents the result of a search query.
// Results can be either pages or databases depending on the filter.
// See: https://developers.notion.com/reference/post-search
type SearchResult struct {
	Object     string                   `json:"object"`
	Results    []map[string]interface{} `json:"results"`
	NextCursor *string                  `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
	Type       string                   `json:"type,omitempty"`
}

// Search searches for pages and databases by title.
// See: https://developers.notion.com/reference/post-search
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	// Use empty request if none provided
	if req == nil {
		req = &SearchRequest{}
	}

	var result SearchResult
	if err := c.doPost(ctx, "/search", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
