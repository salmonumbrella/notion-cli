package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListComments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/comments" {
			t.Errorf("expected path /comments, got %s", r.URL.Path)
		}

		// Verify block_id query parameter
		if r.URL.Query().Get("block_id") != "block123" {
			t.Errorf("expected block_id=block123, got %s", r.URL.Query().Get("block_id"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CommentList{
			Object: "list",
			Results: []*Comment{
				{
					Object: "comment",
					ID:     "comment123",
				},
			},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	commentList, err := client.ListComments(ctx, "block123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commentList.Object != "list" {
		t.Errorf("expected object 'list', got %q", commentList.Object)
	}
	if len(commentList.Results) != 1 {
		t.Errorf("expected 1 comment, got %d", len(commentList.Results))
	}
}

func TestListComments_EmptyBlockID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.ListComments(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty block ID")
	}

	expected := "block ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestListComments_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "50" {
			t.Errorf("expected page_size=50, got %s", r.URL.Query().Get("page_size"))
		}
		if r.URL.Query().Get("start_cursor") != "cursor123" {
			t.Errorf("expected start_cursor=cursor123, got %s", r.URL.Query().Get("start_cursor"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CommentList{
			Object:  "list",
			Results: []*Comment{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &ListCommentsOptions{
		StartCursor: "cursor123",
		PageSize:    50,
	}

	_, err := client.ListComments(ctx, "block123", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListComments_PageSizeLimit(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	opts := &ListCommentsOptions{
		PageSize: 101, // Exceeds max of 100
	}

	_, err := client.ListComments(ctx, "block123", opts)
	if err == nil {
		t.Fatal("expected error for page_size > 100")
	}

	expected := "page_size must be <= 100"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateComment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/comments" {
			t.Errorf("expected path /comments, got %s", r.URL.Path)
		}

		var req CreateCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Parent == nil {
			t.Error("expected parent to be set")
		}
		if len(req.RichText) == 0 {
			t.Error("expected rich_text to be set")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Comment{
			Object:   "comment",
			ID:       "comment123",
			RichText: req.RichText,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateCommentRequest{
		Parent: &CommentParent{
			PageID: "page123",
		},
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Test comment",
				},
			},
		},
	}

	comment, err := client.CreateComment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comment.ID != "comment123" {
		t.Errorf("expected ID 'comment123', got %q", comment.ID)
	}
}

func TestCreateComment_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CreateComment(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "request is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateComment_MissingParentAndDiscussionID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateCommentRequest{
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Test comment",
				},
			},
		},
	}

	_, err := client.CreateComment(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing parent and discussion_id")
	}

	expected := "either parent or discussion_id is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateComment_BothParentAndDiscussionID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateCommentRequest{
		Parent: &CommentParent{
			PageID: "page123",
		},
		DiscussionID: "discussion123",
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Test comment",
				},
			},
		},
	}

	_, err := client.CreateComment(ctx, req)
	if err == nil {
		t.Fatal("expected error for both parent and discussion_id")
	}

	expected := "cannot specify both parent and discussion_id"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateComment_EmptyRichText(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateCommentRequest{
		Parent: &CommentParent{
			PageID: "page123",
		},
		RichText: []RichText{},
	}

	_, err := client.CreateComment(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty rich_text")
	}

	expected := "rich_text is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateComment_WithDiscussionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.DiscussionID != "discussion123" {
			t.Errorf("expected discussion_id 'discussion123', got %q", req.DiscussionID)
		}
		if req.Parent != nil {
			t.Error("expected parent to be nil")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Comment{
			Object:       "comment",
			ID:           "comment123",
			DiscussionID: req.DiscussionID,
			RichText:     req.RichText,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateCommentRequest{
		DiscussionID: "discussion123",
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Reply to discussion",
				},
			},
		},
	}

	comment, err := client.CreateComment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comment.DiscussionID != "discussion123" {
		t.Errorf("expected discussion_id 'discussion123', got %q", comment.DiscussionID)
	}
}

func TestCreateComment_WithUserMention(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Verify we have both text and mention
		if len(req.RichText) != 2 {
			t.Errorf("expected 2 rich text items, got %d", len(req.RichText))
		}

		// Check the mention
		mentionFound := false
		for _, rt := range req.RichText {
			if rt.Type == "mention" && rt.Mention != nil {
				if rt.Mention.Type != "user" {
					t.Errorf("expected mention type 'user', got %q", rt.Mention.Type)
				}
				if rt.Mention.User == nil || rt.Mention.User.ID != "user123" {
					t.Error("expected user mention with ID 'user123'")
				}
				mentionFound = true
			}
		}
		if !mentionFound {
			t.Error("expected to find a user mention in rich text")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Comment{
			Object:   "comment",
			ID:       "comment123",
			RichText: req.RichText,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateCommentRequest{
		Parent: &CommentParent{
			PageID: "page123",
		},
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Please review ",
				},
			},
			{
				Type: "mention",
				Mention: &Mention{
					Type: "user",
					User: &UserMention{
						ID: "user123",
					},
				},
			},
		},
	}

	comment, err := client.CreateComment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comment.ID != "comment123" {
		t.Errorf("expected ID 'comment123', got %q", comment.ID)
	}
}

func TestCreateComment_WithMultipleMentions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Count mentions
		mentionCount := 0
		for _, rt := range req.RichText {
			if rt.Type == "mention" {
				mentionCount++
			}
		}

		if mentionCount != 2 {
			t.Errorf("expected 2 mentions, got %d", mentionCount)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Comment{
			Object:   "comment",
			ID:       "comment123",
			RichText: req.RichText,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateCommentRequest{
		Parent: &CommentParent{
			PageID: "page123",
		},
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: "Hey ",
				},
			},
			{
				Type: "mention",
				Mention: &Mention{
					Type: "user",
					User: &UserMention{ID: "user1"},
				},
			},
			{
				Type: "text",
				Text: &TextContent{
					Content: " and ",
				},
			},
			{
				Type: "mention",
				Mention: &Mention{
					Type: "user",
					User: &UserMention{ID: "user2"},
				},
			},
			{
				Type: "text",
				Text: &TextContent{
					Content: ", please review",
				},
			},
		},
	}

	_, err := client.CreateComment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMentionType_Serialization(t *testing.T) {
	mention := &Mention{
		Type: "user",
		User: &UserMention{
			ID: "test-user-id",
		},
	}

	data, err := json.Marshal(mention)
	if err != nil {
		t.Fatalf("failed to marshal mention: %v", err)
	}

	expected := `{"type":"user","user":{"id":"test-user-id"}}`
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestRichText_WithMention_Serialization(t *testing.T) {
	rt := RichText{
		Type: "mention",
		Mention: &Mention{
			Type: "user",
			User: &UserMention{
				ID: "user123",
			},
		},
	}

	data, err := json.Marshal(rt)
	if err != nil {
		t.Fatalf("failed to marshal rich text: %v", err)
	}

	// Verify the structure contains the mention
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["type"] != "mention" {
		t.Errorf("expected type 'mention', got %v", result["type"])
	}

	mention, ok := result["mention"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mention to be a map")
	}

	if mention["type"] != "user" {
		t.Errorf("expected mention type 'user', got %v", mention["type"])
	}
}

func TestGetComment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/comments/comment123" {
			t.Errorf("expected path /comments/comment123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Comment{
			Object:       "comment",
			ID:           "comment123",
			DiscussionID: "discussion456",
			RichText: []RichText{
				{
					Type: "text",
					Text: &TextContent{
						Content: "Test comment content",
					},
					PlainText: "Test comment content",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	comment, err := client.GetComment(ctx, "comment123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comment.Object != "comment" {
		t.Errorf("expected object 'comment', got %q", comment.Object)
	}
	if comment.ID != "comment123" {
		t.Errorf("expected ID 'comment123', got %q", comment.ID)
	}
	if comment.DiscussionID != "discussion456" {
		t.Errorf("expected discussion_id 'discussion456', got %q", comment.DiscussionID)
	}
	if len(comment.RichText) != 1 {
		t.Errorf("expected 1 rich text item, got %d", len(comment.RichText))
	}
}

func TestGetComment_EmptyCommentID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetComment(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty comment ID")
	}

	expected := "comment ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}
