package cmd

import (
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type lightComment struct {
	ID           string `json:"id"`
	DiscussionID string `json:"discussion_id,omitempty"`
	ParentID     string `json:"parent_id,omitempty"`
	CreatedByID  string `json:"created_by_id,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
	CreatedTime  string `json:"created_time,omitempty"`
	Text         string `json:"text,omitempty"`
}

func toLightComments(comments []*notion.Comment) []lightComment {
	light := make([]lightComment, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}

		entry := lightComment{
			ID:           comment.ID,
			DiscussionID: comment.DiscussionID,
			ParentID:     comment.Parent.PageID,
			CreatedByID:  comment.CreatedBy.ID,
			CreatedBy:    comment.CreatedBy.Name,
			CreatedTime:  comment.CreatedTime,
			Text:         commentPlainText(comment.RichText),
		}
		light = append(light, entry)
	}
	return light
}

func commentPlainText(items []notion.RichText) string {
	var n int
	for i := range items {
		if items[i].PlainText != "" {
			n += len(items[i].PlainText)
		} else if items[i].Text != nil {
			n += len(items[i].Text.Content)
		}
	}
	var b strings.Builder
	b.Grow(n)
	for i := range items {
		item := items[i]
		if item.PlainText != "" {
			b.WriteString(item.PlainText)
			continue
		}
		if item.Text != nil && item.Text.Content != "" {
			b.WriteString(item.Text.Content)
		}
	}
	return b.String()
}

type lightFileUpload struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name,omitempty"`
	Status      string `json:"status,omitempty"`
	Size        int64  `json:"size,omitempty"`
	CreatedTime string `json:"created_time,omitempty"`
	ExpiryTime  string `json:"expiry_time,omitempty"`
}

func toLightFileUploads(uploads []*notion.FileUpload) []lightFileUpload {
	light := make([]lightFileUpload, 0, len(uploads))
	for _, upload := range uploads {
		if upload == nil {
			continue
		}
		light = append(light, lightFileUpload{
			ID:          upload.ID,
			FileName:    upload.FileName,
			Status:      upload.Status,
			Size:        upload.Size,
			CreatedTime: upload.CreatedTime,
			ExpiryTime:  upload.ExpiryTime,
		})
	}
	return light
}
