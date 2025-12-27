package notion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUser_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/users/user123" {
			t.Errorf("expected path /users/user123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			Object: "user",
			ID:     "user123",
			Type:   "person",
			Name:   "Test User",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	user, err := client.GetUser(ctx, "user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "user123" {
		t.Errorf("expected ID 'user123', got %q", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name 'Test User', got %q", user.Name)
	}
}

func TestGetUser_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetUser(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}

	expected := "user ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetUser_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "User not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.GetUser(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestListUsers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/users" {
			t.Errorf("expected path /users, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(UserList{
			Object: "list",
			Results: []*User{
				{
					Object: "user",
					ID:     "user1",
					Name:   "User 1",
				},
				{
					Object: "user",
					ID:     "user2",
					Name:   "User 2",
				},
			},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	userList, err := client.ListUsers(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if userList.Object != "list" {
		t.Errorf("expected object 'list', got %q", userList.Object)
	}
	if len(userList.Results) != 2 {
		t.Errorf("expected 2 users, got %d", len(userList.Results))
	}
}

func TestListUsers_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "50" {
			t.Errorf("expected page_size=50, got %s", r.URL.Query().Get("page_size"))
		}
		if r.URL.Query().Get("start_cursor") != "cursor123" {
			t.Errorf("expected start_cursor=cursor123, got %s", r.URL.Query().Get("start_cursor"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(UserList{
			Object:  "list",
			Results: []*User{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &ListUsersOptions{
		StartCursor: "cursor123",
		PageSize:    50,
	}

	_, err := client.ListUsers(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListUsers_PageSizeLimit(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	opts := &ListUsersOptions{
		PageSize: 101, // Exceeds max of 100
	}

	_, err := client.ListUsers(ctx, opts)
	if err == nil {
		t.Fatal("expected error for page_size > 100")
	}

	expected := "page_size must be <= 100"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestListUsers_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		nextCursor := "nextcursor456"
		_ = json.NewEncoder(w).Encode(UserList{
			Object: "list",
			Results: []*User{
				{
					Object: "user",
					ID:     "user1",
				},
			},
			HasMore:    true,
			NextCursor: &nextCursor,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &ListUsersOptions{
		PageSize: 10,
	}

	userList, err := client.ListUsers(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !userList.HasMore {
		t.Error("expected has_more to be true")
	}
	if userList.NextCursor == nil || *userList.NextCursor != "nextcursor456" {
		t.Errorf("expected next_cursor 'nextcursor456', got %v", userList.NextCursor)
	}
}

func TestGetSelf_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/users/me" {
			t.Errorf("expected path /users/me, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			Object: "user",
			ID:     "bot123",
			Type:   "bot",
			Name:   "Test Bot",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	user, err := client.GetSelf(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "bot123" {
		t.Errorf("expected ID 'bot123', got %q", user.ID)
	}
	if user.Type != "bot" {
		t.Errorf("expected type 'bot', got %q", user.Type)
	}
}

func TestGetSelf_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  401,
			Code:    "unauthorized",
			Message: "Invalid API token",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.GetSelf(ctx)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestGetUser_PersonType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			Object: "user",
			ID:     "person123",
			Type:   "person",
			Name:   "Example User",
			Person: &Person{
				Email: "user@example.invalid",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	user, err := client.GetUser(ctx, "person123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Type != "person" {
		t.Errorf("expected type 'person', got %q", user.Type)
	}
	if user.Person == nil {
		t.Fatal("expected person to be set")
	}
	if user.Person.Email != "user@example.invalid" {
		t.Errorf("expected email 'user@example.invalid', got %q", user.Person.Email)
	}
}

func TestGetUser_BotType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			Object: "user",
			ID:     "bot456",
			Type:   "bot",
			Name:   "My Bot",
			Bot:    map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	user, err := client.GetUser(ctx, "bot456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Type != "bot" {
		t.Errorf("expected type 'bot', got %q", user.Type)
	}
	if user.Bot == nil {
		t.Error("expected bot to be set")
	}
}
