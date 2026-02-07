package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Block represents a Notion block.
// See: https://developers.notion.com/reference/block
type Block struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	Parent         map[string]interface{} `json:"parent,omitempty"`
	Type           string                 `json:"type"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	CreatedBy      map[string]interface{} `json:"created_by,omitempty"`
	LastEditedBy   map[string]interface{} `json:"last_edited_by,omitempty"`
	HasChildren    bool                   `json:"has_children"`
	Archived       bool                   `json:"archived"`
	InTrash        bool                   `json:"in_trash,omitempty"`
	// Type-specific content (e.g., paragraph, heading_1, etc.)
	// Using map to handle different block types flexibly
	Content map[string]interface{} `json:"-"` // Will be unmarshaled from type field
	// Children contains nested child blocks when fetched with depth > 1
	Children []Block `json:"children"`
}

// MarshalJSON implements custom JSON marshaling to include type-specific content.
func (b Block) MarshalJSON() ([]byte, error) {
	// Build a map with all the standard fields
	m := map[string]interface{}{
		"object":           b.Object,
		"id":               b.ID,
		"type":             b.Type,
		"created_time":     b.CreatedTime,
		"last_edited_time": b.LastEditedTime,
		"has_children":     b.HasChildren,
		"archived":         b.Archived,
	}

	// Add optional fields if present
	if b.Parent != nil {
		m["parent"] = b.Parent
	}
	if b.CreatedBy != nil {
		m["created_by"] = b.CreatedBy
	}
	if b.LastEditedBy != nil {
		m["last_edited_by"] = b.LastEditedBy
	}
	if b.InTrash {
		m["in_trash"] = b.InTrash
	}

	// Add type-specific content under the type key
	if b.Type != "" && b.Content != nil && len(b.Content) > 0 {
		m[b.Type] = b.Content
	}

	// Always emit children as an array (empty [] when nil) to avoid null in JSON output
	if b.Children != nil {
		m["children"] = b.Children
	} else {
		m["children"] = []Block{}
	}

	return json.Marshal(m)
}

// BlockList represents a paginated list of blocks.
type BlockList struct {
	Object     string   `json:"object"`
	Results    []Block  `json:"results"`
	NextCursor *string  `json:"next_cursor"`
	HasMore    bool     `json:"has_more"`
	Type       string   `json:"type,omitempty"`
	Block      struct{} `json:"block,omitempty"`
}

// AppendBlockChildrenRequest represents the request body for appending block children.
type AppendBlockChildrenRequest struct {
	Children []map[string]interface{} `json:"children"`
	After    string                   `json:"after,omitempty"`
}

// UpdateBlockRequest represents the request body for updating a block.
type UpdateBlockRequest struct {
	// The block type-specific content
	Content  map[string]interface{} `json:"-"` // Will be set to the type field key
	Archived *bool                  `json:"archived,omitempty"`
}

// BlockChildrenOptions holds pagination options for getting block children.
type BlockChildrenOptions struct {
	StartCursor string
	PageSize    int
}

// GetBlock retrieves a block by ID.
// See: https://developers.notion.com/reference/retrieve-a-block
func (c *Client) GetBlock(ctx context.Context, blockID string) (*Block, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)
	var result map[string]interface{}

	if err := c.doGet(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	// Parse the block from the raw response
	block, err := parseBlock(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return block, nil
}

// GetBlockChildren retrieves the children of a block.
// See: https://developers.notion.com/reference/get-block-children
func (c *Client) GetBlockChildren(ctx context.Context, blockID string, opts *BlockChildrenOptions) (*BlockList, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}

	path := fmt.Sprintf("/blocks/%s/children", blockID)
	query := url.Values{}

	if opts != nil {
		if opts.StartCursor != "" {
			query.Set("start_cursor", opts.StartCursor)
		}
		if opts.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(opts.PageSize))
		}
	}

	var result map[string]interface{}
	if err := c.doGet(ctx, path, query, &result); err != nil {
		return nil, err
	}

	// Parse the block list from the raw response
	blockList, err := parseBlockList(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block list: %w", err)
	}

	return blockList, nil
}

// AppendBlockChildren appends children blocks to a parent block.
// See: https://developers.notion.com/reference/patch-block-children
func (c *Client) AppendBlockChildren(ctx context.Context, blockID string, req *AppendBlockChildrenRequest) (*BlockList, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("append block children request is required")
	}
	if len(req.Children) == 0 {
		return nil, fmt.Errorf("children are required")
	}

	path := fmt.Sprintf("/blocks/%s/children", blockID)
	var result map[string]interface{}

	if err := c.doPatch(ctx, path, req, &result); err != nil {
		return nil, err
	}

	// Parse the block list from the raw response
	blockList, err := parseBlockList(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block list: %w", err)
	}

	return blockList, nil
}

// UpdateBlock updates a block's content.
// See: https://developers.notion.com/reference/update-a-block
func (c *Client) UpdateBlock(ctx context.Context, blockID string, req *UpdateBlockRequest) (*Block, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update block request is required")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)

	// Build request body - merge content into the request
	requestBody := make(map[string]interface{})
	for k, v := range req.Content {
		requestBody[k] = v
	}
	if req.Archived != nil {
		requestBody["archived"] = *req.Archived
	}

	var result map[string]interface{}
	if err := c.doPatch(ctx, path, requestBody, &result); err != nil {
		return nil, err
	}

	// Parse the block from the raw response
	block, err := parseBlock(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return block, nil
}

// GetBlockChildrenRecursive retrieves children of a block recursively up to the specified depth.
// depth=0 returns no blocks, depth=1 returns direct children only, depth=2 includes grandchildren, etc.
func (c *Client) GetBlockChildrenRecursive(ctx context.Context, blockID string, depth int, opts *BlockChildrenOptions) ([]Block, error) {
	if depth <= 0 {
		return []Block{}, nil
	}

	// Fetch direct children
	var allBlocks []Block
	cursor := ""
	if opts != nil {
		cursor = opts.StartCursor
	}

	for {
		fetchOpts := &BlockChildrenOptions{
			StartCursor: cursor,
		}
		if opts != nil && opts.PageSize > 0 {
			fetchOpts.PageSize = opts.PageSize
		}

		blockList, err := c.GetBlockChildren(ctx, blockID, fetchOpts)
		if err != nil {
			return nil, err
		}

		allBlocks = append(allBlocks, blockList.Results...)

		if !blockList.HasMore || blockList.NextCursor == nil || *blockList.NextCursor == "" {
			break
		}
		cursor = *blockList.NextCursor
	}

	// If depth > 1, recursively fetch children for blocks that have children
	if depth > 1 {
		// Create new options for child blocks - don't pass parent's StartCursor
		childOpts := &BlockChildrenOptions{}
		if opts != nil {
			childOpts.PageSize = opts.PageSize
		}

		for i := range allBlocks {
			if allBlocks[i].HasChildren {
				children, err := c.GetBlockChildrenRecursive(ctx, allBlocks[i].ID, depth-1, childOpts)
				if err != nil {
					return nil, fmt.Errorf("failed to get children of block %s: %w", allBlocks[i].ID, err)
				}
				allBlocks[i].Children = children
			}
		}
	}

	return allBlocks, nil
}

// DeleteBlock deletes (archives) a block.
// See: https://developers.notion.com/reference/delete-a-block
func (c *Client) DeleteBlock(ctx context.Context, blockID string) (*Block, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)
	var result map[string]interface{}

	if err := c.doDelete(ctx, path, &result); err != nil {
		return nil, err
	}

	// Parse the block from the raw response
	block, err := parseBlock(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return block, nil
}

// parseBlock converts a raw JSON response into a Block struct.
func parseBlock(data map[string]interface{}) (*Block, error) {
	block := &Block{
		Content: make(map[string]interface{}),
	}

	// Extract standard fields
	if v, ok := data["object"].(string); ok {
		block.Object = v
	}
	if v, ok := data["id"].(string); ok {
		block.ID = v
	}
	if v, ok := data["type"].(string); ok {
		block.Type = v
		// Store type-specific content
		if content, ok := data[v].(map[string]interface{}); ok {
			block.Content = content
		}
	}
	if v, ok := data["created_time"].(string); ok {
		block.CreatedTime = v
	}
	if v, ok := data["last_edited_time"].(string); ok {
		block.LastEditedTime = v
	}
	if v, ok := data["has_children"].(bool); ok {
		block.HasChildren = v
	}
	if v, ok := data["archived"].(bool); ok {
		block.Archived = v
	}
	if v, ok := data["in_trash"].(bool); ok {
		block.InTrash = v
	}
	if v, ok := data["parent"].(map[string]interface{}); ok {
		block.Parent = v
	}
	if v, ok := data["created_by"].(map[string]interface{}); ok {
		block.CreatedBy = v
	}
	if v, ok := data["last_edited_by"].(map[string]interface{}); ok {
		block.LastEditedBy = v
	}

	return block, nil
}

// parseBlockList converts a raw JSON response into a BlockList struct.
func parseBlockList(data map[string]interface{}) (*BlockList, error) {
	blockList := &BlockList{}

	// Extract standard fields
	if v, ok := data["object"].(string); ok {
		blockList.Object = v
	}
	if v, ok := data["type"].(string); ok {
		blockList.Type = v
	}
	if v, ok := data["has_more"].(bool); ok {
		blockList.HasMore = v
	}
	if v, ok := data["next_cursor"].(string); ok && v != "" {
		blockList.NextCursor = &v
	}

	// Parse results array
	if results, ok := data["results"].([]interface{}); ok {
		blockList.Results = make([]Block, 0, len(results))
		for _, item := range results {
			if blockData, ok := item.(map[string]interface{}); ok {
				block, err := parseBlock(blockData)
				if err != nil {
					return nil, fmt.Errorf("failed to parse block in list: %w", err)
				}
				blockList.Results = append(blockList.Results, *block)
			}
		}
	}

	return blockList, nil
}
