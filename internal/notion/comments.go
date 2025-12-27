package notion

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Comment represents a Notion comment.
// See: https://developers.notion.com/reference/comment-object
type Comment struct {
	Object         string     `json:"object"`
	ID             string     `json:"id"`
	Parent         Parent     `json:"parent"`
	DiscussionID   string     `json:"discussion_id"`
	CreatedTime    string     `json:"created_time"`
	LastEditedTime string     `json:"last_edited_time"`
	CreatedBy      User       `json:"created_by"`
	RichText       []RichText `json:"rich_text"`
}

// Parent represents the parent of a comment (page or block).
type Parent struct {
	Type   string `json:"type"`
	PageID string `json:"page_id,omitempty"`
}

// RichText represents rich text content.
type RichText struct {
	Type        string       `json:"type"`
	Text        *TextContent `json:"text,omitempty"`
	Mention     *Mention     `json:"mention,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string       `json:"plain_text,omitempty"`
	Href        string       `json:"href,omitempty"`
}

// TextContent represents the text portion of rich text.
type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a hyperlink.
type Link struct {
	URL string `json:"url"`
}

// Mention represents a mention in rich text.
type Mention struct {
	Type string       `json:"type"`
	User *UserMention `json:"user,omitempty"`
	Page *PageMention `json:"page,omitempty"`
}

// UserMention represents a user mention.
type UserMention struct {
	ID string `json:"id"`
}

// PageMention represents a page mention.
type PageMention struct {
	ID string `json:"id"`
}

// Annotations represents text formatting annotations.
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// CommentList represents a paginated list of comments.
type CommentList struct {
	Object     string     `json:"object"`
	Results    []*Comment `json:"results"`
	NextCursor *string    `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
}

// ListCommentsOptions contains options for listing comments.
type ListCommentsOptions struct {
	StartCursor string
	PageSize    int
}

// CreateCommentRequest represents a request to create a comment.
type CreateCommentRequest struct {
	Parent       *CommentParent `json:"parent,omitempty"`
	DiscussionID string         `json:"discussion_id,omitempty"`
	RichText     []RichText     `json:"rich_text"`
}

// CommentParent represents the parent for creating a comment.
type CommentParent struct {
	PageID string `json:"page_id"`
}

// ListComments retrieves a list of un-resolved comments from a page or block.
// See: https://developers.notion.com/reference/retrieve-a-comment
func (c *Client) ListComments(ctx context.Context, blockID string, opts *ListCommentsOptions) (*CommentList, error) {
	if blockID == "" {
		return nil, fmt.Errorf("block ID is required")
	}

	query := url.Values{}
	query.Set("block_id", blockID)

	if opts != nil {
		if opts.StartCursor != "" {
			query.Set("start_cursor", opts.StartCursor)
		}
		if opts.PageSize > 0 {
			if opts.PageSize > 100 {
				return nil, fmt.Errorf("page_size must be <= 100")
			}
			query.Set("page_size", strconv.Itoa(opts.PageSize))
		}
	}

	var commentList CommentList
	if err := c.doGet(ctx, "/comments", query, &commentList); err != nil {
		return nil, err
	}

	return &commentList, nil
}

// GetComment retrieves a comment by ID.
// See: https://developers.notion.com/reference/retrieve-a-comment
func (c *Client) GetComment(ctx context.Context, commentID string) (*Comment, error) {
	if commentID == "" {
		return nil, fmt.Errorf("comment ID is required")
	}

	path := fmt.Sprintf("/comments/%s", commentID)
	var comment Comment

	if err := c.doGet(ctx, path, nil, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// CreateComment creates a comment in a page or existing discussion thread.
// See: https://developers.notion.com/reference/create-a-comment
func (c *Client) CreateComment(ctx context.Context, req *CreateCommentRequest) (*Comment, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Validate that either parent or discussion_id is provided, but not both
	if req.Parent == nil && req.DiscussionID == "" {
		return nil, fmt.Errorf("either parent or discussion_id is required")
	}
	if req.Parent != nil && req.DiscussionID != "" {
		return nil, fmt.Errorf("cannot specify both parent and discussion_id")
	}

	// Validate rich text
	if len(req.RichText) == 0 {
		return nil, fmt.Errorf("rich_text is required")
	}

	var comment Comment
	if err := c.doPost(ctx, "/comments", req, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}
