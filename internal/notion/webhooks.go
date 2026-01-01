package notion

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// WebhookEvent represents a webhook event from Notion.
type WebhookEvent struct {
	Type      string           `json:"type"`
	Timestamp string           `json:"timestamp"`
	Data      WebhookEventData `json:"data"`
}

// WebhookEventData contains the event payload data.
type WebhookEventData struct {
	PageID       string `json:"page_id,omitempty"`
	DatabaseID   string `json:"database_id,omitempty"`
	DataSourceID string `json:"data_source_id,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
	CommentID    string `json:"comment_id,omitempty"`
	DiscussionID string `json:"discussion_id,omitempty"`
}

// WebhookEventType constants for known event types.
const (
	WebhookEventPageContentUpdated      = "page.content_updated"
	WebhookEventPageLocked              = "page.locked"
	WebhookEventCommentCreated          = "comment.created"
	WebhookEventDataSourceCreated       = "data_source.created"
	WebhookEventDataSourceDeleted       = "data_source.deleted"
	WebhookEventDataSourceSchemaUpdated = "data_source.schema_updated"
	WebhookEventDatabaseSchemaUpdated   = "database.schema_updated" // deprecated
)

// WebhookVerificationRequest is sent when creating a webhook subscription.
type WebhookVerificationRequest struct {
	VerificationToken string `json:"verification_token"`
}

// ComputeWebhookSignature computes the HMAC-SHA256 signature for webhook verification.
func ComputeWebhookSignature(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// VerifyWebhookSignature verifies the X-Notion-Signature header.
// Uses constant-time comparison to prevent timing attacks.
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	expected := ComputeWebhookSignature(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ParseWebhookEvent parses a webhook event payload.
func ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}
	return &event, nil
}

// ParseWebhookVerification parses a verification request payload.
func ParseWebhookVerification(payload []byte) (*WebhookVerificationRequest, error) {
	var req WebhookVerificationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("failed to parse verification request: %w", err)
	}
	return &req, nil
}

// IsVerificationRequest checks if the payload is a verification request.
func IsVerificationRequest(payload []byte) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return false
	}
	_, hasToken := data["verification_token"]
	return hasToken
}
