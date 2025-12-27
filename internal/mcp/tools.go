package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SearchMode controls the type of search performed by the Notion MCP server.
type SearchMode string

const (
	SearchModeWorkspace SearchMode = "workspace_search"
	SearchModeAI        SearchMode = "ai_search"
)

// SearchParams holds parameters for the notion-search MCP tool.
type SearchParams struct {
	Query             string
	ContentSearchMode SearchMode
	QueryType         string
	PageURL           string
	DataSourceURL     string
	TeamspaceID       string
	Filters           map[string]interface{}
}

// Search invokes the notion-search MCP tool.
func (c *Client) Search(ctx context.Context, params SearchParams) (string, error) {
	args := buildSearchArgs(params)
	return c.CallTool(ctx, "notion-search", args)
}

// Fetch invokes the notion-fetch MCP tool to retrieve a page or database as markdown.
func (c *Client) Fetch(ctx context.Context, id string, includeDiscussions bool) (string, error) {
	args := buildFetchArgs(id, includeDiscussions)
	return c.CallTool(ctx, "notion-fetch", args)
}

// CreatePageInput holds the parameters for a single page in the MCP create-pages tool.
type CreatePageInput struct {
	Properties map[string]interface{} `json:"properties,omitempty"`
	Content    string                 `json:"content,omitempty"`
	TemplateID string                 `json:"template_id,omitempty"`
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
		if p.TemplateID != "" {
			page["template_id"] = p.TemplateID
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
	UpdateCmdApplyTemplate       UpdateCommand = "apply_template"
)

// UpdatePageParams holds parameters for the notion-update-page MCP tool.
type UpdatePageParams struct {
	PageID                string
	Command               UpdateCommand
	NewStr                string                 // for replace_content, replace_content_range, insert_content_after
	SelectionWithEllipsis string                 // for replace_content_range (the text to match with "..." for omitted parts)
	Properties            map[string]interface{} // for update_properties
	TemplateID            string                 // for apply_template
	AllowDeletingContent  *bool                  // optional, only passed when explicitly set
}

// CreateDatabaseParent identifies the parent page for a new database.
type CreateDatabaseParent struct {
	PageID string `json:"page_id,omitempty"`
}

// CreateDatabaseParams holds parameters for the notion-create-database MCP tool.
type CreateDatabaseParams struct {
	Parent      *CreateDatabaseParent  `json:"parent,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      string                 `json:"schema,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"` // legacy JSON schema support
}

// UpdateDataSourceParams holds parameters for the notion-update-data-source MCP tool.
type UpdateDataSourceParams struct {
	DataSourceID string                 `json:"data_source_id"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Statements   string                 `json:"statements,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"` // legacy JSON schema support
	InTrash      *bool                  `json:"in_trash,omitempty"`
	IsInline     *bool                  `json:"is_inline,omitempty"`
}

// QueryDataSourcesSQLParams holds parameters for SQL mode in notion-query-data-sources.
type QueryDataSourcesSQLParams struct {
	DataSourceURLs []string
	Query          string
	Params         []interface{}
}

// QueryDataSourcesViewParams holds parameters for view mode in notion-query-data-sources.
type QueryDataSourcesViewParams struct {
	ViewURL string
}

// QueryMeetingNotesParams holds parameters for notion-query-meeting-notes.
type QueryMeetingNotesParams struct {
	Filter map[string]interface{}
}

// UpdatePage invokes the notion-update-page MCP tool.
func (c *Client) UpdatePage(ctx context.Context, params UpdatePageParams) (string, error) {
	args, err := buildUpdatePageArgs(params)
	if err != nil {
		return "", err
	}

	return c.CallTool(ctx, "notion-update-page", args)
}

// GetCommentsParams holds parameters for the notion-get-comments MCP tool.
type GetCommentsParams struct {
	TargetID         string // preferred: page or block target identifier
	PageID           string // deprecated alias for TargetID
	DiscussionID     string
	IncludeAllBlocks bool
	IncludeResolved  bool
}

// GetComments invokes the notion-get-comments MCP tool.
func (c *Client) GetComments(ctx context.Context, params GetCommentsParams) (string, error) {
	args := buildGetCommentsArgs(params)
	return c.CallTool(ctx, "notion-get-comments", args)
}

// CreateCommentParams holds parameters for the notion-create-comment MCP tool.
type CreateCommentParams struct {
	TargetID              string // preferred: page or block target identifier
	PageID                string // deprecated alias for TargetID
	Text                  string
	DiscussionID          string
	SelectionWithEllipsis string
}

// CreateComment invokes the notion-create-comment MCP tool.
func (c *Client) CreateComment(ctx context.Context, params CreateCommentParams) (string, error) {
	args := buildCreateCommentArgs(params)
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

// CreateDatabase invokes the notion-create-database MCP tool.
func (c *Client) CreateDatabase(ctx context.Context, params CreateDatabaseParams) (string, error) {
	args, err := buildCreateDatabaseArgs(params)
	if err != nil {
		return "", err
	}
	return c.CallTool(ctx, "notion-create-database", args)
}

// UpdateDataSource invokes the notion-update-data-source MCP tool.
func (c *Client) UpdateDataSource(ctx context.Context, params UpdateDataSourceParams) (string, error) {
	args, err := buildUpdateDataSourceArgs(params)
	if err != nil {
		return "", err
	}
	return c.CallTool(ctx, "notion-update-data-source", args)
}

// QueryDataSourcesSQL invokes the notion-query-data-sources MCP tool in SQL mode.
func (c *Client) QueryDataSourcesSQL(ctx context.Context, params QueryDataSourcesSQLParams) (string, error) {
	args, err := buildQueryDataSourcesSQLArgs(params)
	if err != nil {
		return "", err
	}
	return c.CallTool(ctx, "notion-query-data-sources", args)
}

// QueryDataSourcesView invokes the notion-query-data-sources MCP tool in view mode.
func (c *Client) QueryDataSourcesView(ctx context.Context, params QueryDataSourcesViewParams) (string, error) {
	args, err := buildQueryDataSourcesViewArgs(params)
	if err != nil {
		return "", err
	}
	return c.CallTool(ctx, "notion-query-data-sources", args)
}

// QueryMeetingNotes invokes the notion-query-meeting-notes MCP tool.
func (c *Client) QueryMeetingNotes(ctx context.Context, params QueryMeetingNotesParams) (string, error) {
	args, err := buildQueryMeetingNotesArgs(params)
	if err != nil {
		return "", err
	}
	return c.CallTool(ctx, "notion-query-meeting-notes", args)
}

func buildSearchArgs(params SearchParams) map[string]interface{} {
	args := map[string]interface{}{
		"query": params.Query,
	}
	if params.ContentSearchMode != "" {
		args["content_search_mode"] = string(params.ContentSearchMode)
	}
	if params.QueryType != "" {
		args["query_type"] = params.QueryType
	}
	if params.PageURL != "" {
		args["page_url"] = params.PageURL
	}
	if params.DataSourceURL != "" {
		args["data_source_url"] = params.DataSourceURL
	}
	if params.TeamspaceID != "" {
		args["teamspace_id"] = params.TeamspaceID
	}
	if len(params.Filters) > 0 {
		args["filters"] = params.Filters
	}
	return args
}

func buildFetchArgs(id string, includeDiscussions bool) map[string]interface{} {
	args := map[string]interface{}{
		"id": id,
	}
	if includeDiscussions {
		args["include_discussions"] = true
	}
	return args
}

func buildGetCommentsArgs(params GetCommentsParams) map[string]interface{} {
	targetID := strings.TrimSpace(params.TargetID)
	if targetID == "" {
		targetID = strings.TrimSpace(params.PageID)
	}
	args := map[string]interface{}{
		"page_id": targetID,
	}
	if params.DiscussionID != "" {
		args["discussion_id"] = params.DiscussionID
	}
	if params.IncludeAllBlocks {
		args["include_all_blocks"] = true
	}
	if params.IncludeResolved {
		args["include_resolved"] = true
	}
	return args
}

func buildCreateCommentArgs(params CreateCommentParams) map[string]interface{} {
	targetID := strings.TrimSpace(params.TargetID)
	if targetID == "" {
		targetID = strings.TrimSpace(params.PageID)
	}
	args := map[string]interface{}{
		"page_id": targetID,
		"rich_text": []interface{}{
			map[string]interface{}{
				"text": map[string]interface{}{
					"content": params.Text,
				},
			},
		},
	}
	if params.DiscussionID != "" {
		args["discussion_id"] = params.DiscussionID
	}
	if params.SelectionWithEllipsis != "" {
		args["selection_with_ellipsis"] = params.SelectionWithEllipsis
	}
	return args
}

func buildUpdatePageArgs(params UpdatePageParams) (map[string]interface{}, error) {
	req := updatePageRequest{
		PageID:               params.PageID,
		AllowDeletingContent: params.AllowDeletingContent,
	}

	switch params.Command {
	case UpdateCmdReplaceContent:
		req.Command = string(UpdateCmdReplaceContent)
		req.NewStr = params.NewStr
	case UpdateCmdReplaceContentRange:
		req.Command = string(UpdateCmdReplaceContentRange)
		req.NewStr = params.NewStr
		req.SelectionWithEllipsis = params.SelectionWithEllipsis
	case UpdateCmdInsertContentAfter:
		req.Command = string(UpdateCmdInsertContentAfter)
		req.NewStr = params.NewStr
		req.SelectionWithEllipsis = params.SelectionWithEllipsis
	case UpdateCmdUpdateProperties:
		req.Command = string(UpdateCmdUpdateProperties)
		req.Properties = params.Properties
	case UpdateCmdApplyTemplate:
		req.Command = string(UpdateCmdApplyTemplate)
		req.TemplateID = params.TemplateID
	default:
		return nil, fmt.Errorf("unknown update command: %q", params.Command)
	}

	return encodeToolArgs(req)
}

func buildCreateDatabaseArgs(params CreateDatabaseParams) (map[string]interface{}, error) {
	if strings.TrimSpace(params.Schema) == "" {
		return nil, fmt.Errorf("schema is required")
	}
	return encodeToolArgs(params)
}

func buildUpdateDataSourceArgs(params UpdateDataSourceParams) (map[string]interface{}, error) {
	if strings.TrimSpace(params.DataSourceID) == "" {
		return nil, fmt.Errorf("data source id is required")
	}
	return encodeToolArgs(params)
}

func buildQueryDataSourcesSQLArgs(params QueryDataSourcesSQLParams) (map[string]interface{}, error) {
	if len(params.DataSourceURLs) == 0 {
		return nil, fmt.Errorf("at least one data source URL is required")
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("SQL query is required")
	}

	req := queryDataSourcesRequest{
		Data: queryDataSourcesData{
			DataSourceURLs: params.DataSourceURLs,
			Query:          params.Query,
			Params:         params.Params,
		},
	}

	return encodeToolArgs(req)
}

func buildQueryDataSourcesViewArgs(params QueryDataSourcesViewParams) (map[string]interface{}, error) {
	if strings.TrimSpace(params.ViewURL) == "" {
		return nil, fmt.Errorf("view URL is required")
	}

	req := queryDataSourcesRequest{
		Data: queryDataSourcesData{
			Mode:    "view",
			ViewURL: params.ViewURL,
		},
	}

	return encodeToolArgs(req)
}

func buildQueryMeetingNotesArgs(params QueryMeetingNotesParams) (map[string]interface{}, error) {
	req := queryMeetingNotesRequest(params)
	return encodeToolArgs(req)
}

func encodeToolArgs(v interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tool arguments: %w", err)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, fmt.Errorf("failed to decode tool arguments: %w", err)
	}
	if args == nil {
		args = map[string]interface{}{}
	}

	return args, nil
}

type updatePageRequest struct {
	PageID                string                 `json:"page_id"`
	Command               string                 `json:"command"`
	NewStr                string                 `json:"new_str,omitempty"`
	SelectionWithEllipsis string                 `json:"selection_with_ellipsis,omitempty"`
	Properties            map[string]interface{} `json:"properties,omitempty"`
	TemplateID            string                 `json:"template_id,omitempty"`
	AllowDeletingContent  *bool                  `json:"allow_deleting_content,omitempty"`
}

type queryDataSourcesRequest struct {
	Data queryDataSourcesData `json:"data"`
}

type queryDataSourcesData struct {
	Mode           string        `json:"mode,omitempty"`
	ViewURL        string        `json:"view_url,omitempty"`
	DataSourceURLs []string      `json:"data_source_urls,omitempty"`
	Query          string        `json:"query,omitempty"`
	Params         []interface{} `json:"params,omitempty"`
}

type queryMeetingNotesRequest struct {
	Filter map[string]interface{} `json:"filter,omitempty"`
}
