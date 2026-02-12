package mcp

import (
	"context"
	"fmt"
)

// SearchMode controls the type of search performed by the Notion MCP server.
type SearchMode string

const (
	SearchModeWorkspace SearchMode = "workspace_search"
	SearchModeAI        SearchMode = "ai_search"
)

// Search invokes the notion-search MCP tool.
func (c *Client) Search(ctx context.Context, query string, mode SearchMode) (string, error) {
	args := map[string]interface{}{
		"query":       query,
		"search_mode": string(mode),
	}
	return c.CallTool(ctx, "notion-search", args)
}

// Fetch invokes the notion-fetch MCP tool to retrieve a page or database as markdown.
func (c *Client) Fetch(ctx context.Context, resourceID string) (string, error) {
	args := map[string]interface{}{
		"resource_id": resourceID,
	}
	return c.CallTool(ctx, "notion-fetch", args)
}

// CreatePageInput holds the parameters for a single page in the MCP create-pages tool.
type CreatePageInput struct {
	Properties map[string]interface{} `json:"properties,omitempty"`
	Content    string                 `json:"content,omitempty"`
}

// CreatePagesParent identifies the parent for created pages.
// Exactly one of PageID or DataSourceID should be set.
type CreatePagesParent struct {
	PageID       string
	DataSourceID string
}

// CreatePages invokes the notion-create-pages MCP tool to create one or more pages.
// If parent is nil the pages are created as standalone workspace pages.
func (c *Client) CreatePages(ctx context.Context, parent *CreatePagesParent, pages []CreatePageInput) (string, error) {
	// Build each page object with properties and optional content.
	pagesArg := make([]interface{}, len(pages))
	for i, p := range pages {
		page := map[string]interface{}{}
		if p.Properties != nil {
			page["properties"] = p.Properties
		}
		if p.Content != "" {
			page["content"] = p.Content
		}
		pagesArg[i] = page
	}

	args := map[string]interface{}{
		"pages": pagesArg,
	}

	// Add parent as a top-level object when provided.
	if parent != nil {
		parentObj := map[string]interface{}{}
		if parent.PageID != "" {
			parentObj["page_id"] = parent.PageID
		}
		if parent.DataSourceID != "" {
			parentObj["data_source_id"] = parent.DataSourceID
		}
		args["parent"] = parentObj
	}

	return c.CallTool(ctx, "notion-create-pages", args)
}

// UpdateCommand describes the command type for notion-update-page.
type UpdateCommand string

const (
	UpdateCmdReplaceContent      UpdateCommand = "replace_content"
	UpdateCmdReplaceContentRange UpdateCommand = "replace_content_range"
	UpdateCmdInsertContentAfter  UpdateCommand = "insert_content_after"
	UpdateCmdUpdateProperties    UpdateCommand = "update_properties"
)

// UpdatePageParams holds parameters for the notion-update-page MCP tool.
type UpdatePageParams struct {
	PageID                string
	Command               UpdateCommand
	NewStr                string                 // for replace_content, replace_content_range, insert_content_after
	SelectionWithEllipsis string                 // for replace_content_range (the text to match with "..." for omitted parts)
	Properties            map[string]interface{} // for update_properties
}

// UpdatePage invokes the notion-update-page MCP tool.
func (c *Client) UpdatePage(ctx context.Context, params UpdatePageParams) (string, error) {
	args := map[string]interface{}{
		"page_id": params.PageID,
	}

	switch params.Command {
	case UpdateCmdReplaceContent:
		args["command"] = "replace_content"
		args["new_str"] = params.NewStr
	case UpdateCmdReplaceContentRange:
		args["command"] = "replace_content_range"
		args["new_str"] = params.NewStr
		args["selection_with_ellipsis"] = params.SelectionWithEllipsis
	case UpdateCmdInsertContentAfter:
		args["command"] = "insert_content_after"
		args["new_str"] = params.NewStr
		args["selection_with_ellipsis"] = params.SelectionWithEllipsis
	case UpdateCmdUpdateProperties:
		args["command"] = "update_properties"
		args["properties"] = params.Properties
	default:
		return "", fmt.Errorf("unknown update command: %q", params.Command)
	}

	return c.CallTool(ctx, "notion-update-page", args)
}

// GetComments invokes the notion-get-comments MCP tool.
func (c *Client) GetComments(ctx context.Context, resourceID string) (string, error) {
	args := map[string]interface{}{
		"resource_id": resourceID,
	}
	return c.CallTool(ctx, "notion-get-comments", args)
}

// CreateComment invokes the notion-create-comment MCP tool.
func (c *Client) CreateComment(ctx context.Context, pageID string, body string) (string, error) {
	args := map[string]interface{}{
		"page_id": pageID,
		"body":    body,
	}
	return c.CallTool(ctx, "notion-create-comment", args)
}

// MovePages invokes the notion-move-pages MCP tool to move pages/databases to a new parent.
func (c *Client) MovePages(ctx context.Context, pageIDs []string, parentPageID string) (string, error) {
	args := map[string]interface{}{
		"page_ids":       pageIDs,
		"parent_page_id": parentPageID,
	}
	return c.CallTool(ctx, "notion-move-pages", args)
}

// DuplicatePage invokes the notion-duplicate-page MCP tool.
func (c *Client) DuplicatePage(ctx context.Context, pageID string) (string, error) {
	args := map[string]interface{}{
		"page_id": pageID,
	}
	return c.CallTool(ctx, "notion-duplicate-page", args)
}

// GetTeams invokes the notion-get-teams MCP tool to list workspace teamspaces.
func (c *Client) GetTeams(ctx context.Context, query string) (string, error) {
	args := map[string]interface{}{}
	if query != "" {
		args["query"] = query
	}
	return c.CallTool(ctx, "notion-get-teams", args)
}

// GetUsers invokes the notion-get-users MCP tool to list workspace users.
func (c *Client) GetUsers(ctx context.Context, query string, userID string, startCursor string, pageSize int) (string, error) {
	args := map[string]interface{}{}
	if query != "" {
		args["query"] = query
	}
	if userID != "" {
		args["user_id"] = userID
	}
	if startCursor != "" {
		args["start_cursor"] = startCursor
	}
	if pageSize > 0 {
		args["page_size"] = pageSize
	}
	return c.CallTool(ctx, "notion-get-users", args)
}
