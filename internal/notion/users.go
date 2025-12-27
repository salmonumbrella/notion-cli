package notion

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// User represents a Notion user.
// See: https://developers.notion.com/reference/user
type User struct {
	Object    string      `json:"object"`
	ID        string      `json:"id"`
	Type      string      `json:"type,omitempty"`
	Name      string      `json:"name,omitempty"`
	AvatarURL string      `json:"avatar_url,omitempty"`
	Person    *Person     `json:"person,omitempty"`
	Bot       interface{} `json:"bot,omitempty"`
}

// Person represents a person user's details.
type Person struct {
	Email string `json:"email,omitempty"`
}

// UserList represents a paginated list of users.
type UserList struct {
	Object     string  `json:"object"`
	Results    []*User `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// ListUsersOptions contains options for listing users.
type ListUsersOptions struct {
	StartCursor string
	PageSize    int
}

// GetUser retrieves a user by ID.
// See: https://developers.notion.com/reference/get-user
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	path := fmt.Sprintf("/users/%s", userID)
	var user User

	if err := c.doGet(ctx, path, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ListUsers lists all users in the workspace.
// See: https://developers.notion.com/reference/get-users
func (c *Client) ListUsers(ctx context.Context, opts *ListUsersOptions) (*UserList, error) {
	query := url.Values{}

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

	var userList UserList
	if err := c.doGet(ctx, "/users", query, &userList); err != nil {
		return nil, err
	}

	return &userList, nil
}

// GetSelf retrieves the bot user associated with the API token.
// See: https://developers.notion.com/reference/get-self
func (c *Client) GetSelf(ctx context.Context) (*User, error) {
	var user User
	if err := c.doGet(ctx, "/users/me", nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
