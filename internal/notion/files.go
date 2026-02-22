package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// FileUpload represents a Notion file upload object.
// See: https://developers.notion.com/reference/file-upload-object
type FileUpload struct {
	Object      string `json:"object"`
	ID          string `json:"id"`
	CreatedTime string `json:"created_time"`
	ExpiryTime  string `json:"expiry_time"`
	FileName    string `json:"file_name,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Status      string `json:"status"`
	UploadURL   string `json:"upload_url,omitempty"`
}

// FileUploadList represents a paginated list of file uploads.
type FileUploadList struct {
	Object     string        `json:"object"`
	Results    []*FileUpload `json:"results"`
	NextCursor *string       `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

// CreateFileUploadRequest represents a request to create a file upload.
type CreateFileUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type,omitempty"`
	Mode        string `json:"mode,omitempty"` // "single_part" (default) or "multi_part"
}

// DefaultChunkSize is the default size for multi-part upload chunks (5MB)
const DefaultChunkSize = 5 * 1024 * 1024

// FileUploadPart represents a single part of a multi-part upload.
type FileUploadPart struct {
	PartNumber int    `json:"part_number"`
	Status     string `json:"status"`
	ETag       string `json:"etag,omitempty"`
}

// CreateFileUpload creates a new file upload object.
// See: https://developers.notion.com/reference/create-a-file-upload
func (c *Client) CreateFileUpload(ctx context.Context, req *CreateFileUploadRequest) (*FileUpload, error) {
	if req == nil || req.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}

	var upload FileUpload
	if err := c.doPost(ctx, "/file_uploads", req, &upload); err != nil {
		return nil, err
	}

	return &upload, nil
}

// SendFileUpload uploads the file content to Notion.
// See: https://developers.notion.com/reference/send-a-file-upload
func (c *Client) SendFileUpload(ctx context.Context, uploadURL string, file io.Reader, filename, contentType string) (*FileUpload, error) {
	if uploadURL == "" {
		return nil, fmt.Errorf("upload URL is required")
	}

	var upload FileUpload
	if err := c.doMultipartRequest(ctx, uploadURL, "file", file, filename, contentType, &upload); err != nil {
		return nil, err
	}

	return &upload, nil
}

// CompleteFileUpload marks a file upload as complete.
// This is required for multi-part uploads after all parts have been sent.
// See: https://developers.notion.com/reference/complete-a-file-upload
func (c *Client) CompleteFileUpload(ctx context.Context, fileUploadID string) (*FileUpload, error) {
	if fileUploadID == "" {
		return nil, fmt.Errorf("file upload ID is required")
	}

	path := fmt.Sprintf("/file_uploads/%s/complete", fileUploadID)
	var upload FileUpload

	if err := c.doPost(ctx, path, nil, &upload); err != nil {
		return nil, err
	}

	return &upload, nil
}

// GetFileUpload retrieves a file upload by ID.
// See: https://developers.notion.com/reference/retrieve-a-file-upload
func (c *Client) GetFileUpload(ctx context.Context, fileUploadID string) (*FileUpload, error) {
	if fileUploadID == "" {
		return nil, fmt.Errorf("file upload ID is required")
	}

	path := fmt.Sprintf("/file_uploads/%s", fileUploadID)
	var upload FileUpload

	if err := c.doGet(ctx, path, nil, &upload); err != nil {
		return nil, err
	}

	return &upload, nil
}

// ListFileUploadsOptions contains options for listing file uploads.
type ListFileUploadsOptions struct {
	StartCursor string
	PageSize    int
}

// ListFileUploads lists file uploads for the integration.
// See: https://developers.notion.com/reference/list-file-uploads
func (c *Client) ListFileUploads(ctx context.Context, opts *ListFileUploadsOptions) (*FileUploadList, error) {
	query := url.Values{}

	if opts != nil {
		if opts.StartCursor != "" {
			query.Set("start_cursor", opts.StartCursor)
		}
		if opts.PageSize > 0 {
			query.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
	}

	var list FileUploadList
	if err := c.doGet(ctx, "/file_uploads", query, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// SendFilePart uploads a single part of a multi-part file upload.
func (c *Client) SendFilePart(ctx context.Context, uploadURL string, part io.Reader, partNumber int) (*FileUploadPart, error) {
	if uploadURL == "" {
		return nil, fmt.Errorf("upload URL is required")
	}
	if partNumber < 1 {
		return nil, fmt.Errorf("part number must be >= 1")
	}

	resp, err := c.doRequestOnceWithReader(
		ctx,
		http.MethodPost,
		uploadURL,
		part,
		"application/octet-stream",
		map[string]string{"X-Part-Number": strconv.Itoa(partNumber)},
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result FileUploadPart
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UploadLargeFile handles the complete multi-part upload workflow.
// It automatically chunks the file and uploads each part.
func (c *Client) UploadLargeFile(ctx context.Context, filename string, file io.Reader) (*FileUpload, error) {
	// Step 1: Create multi-part upload
	createReq := &CreateFileUploadRequest{
		FileName: filename,
		Mode:     "multi_part",
	}

	upload, err := c.CreateFileUpload(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create file upload: %w", err)
	}

	if upload.UploadURL == "" {
		return nil, fmt.Errorf("no upload URL returned")
	}

	// Step 2: Upload parts
	partNumber := 1
	buffer := make([]byte, DefaultChunkSize)

	for {
		n, err := io.ReadFull(file, buffer)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Copy each chunk before upload because the HTTP transport may still be
		// reading from the previous part while we fill the buffer for the next one.
		chunk := append([]byte(nil), buffer[:n]...)
		_, err = c.SendFilePart(ctx, upload.UploadURL, bytes.NewReader(chunk), partNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		partNumber++

		if n < DefaultChunkSize {
			break
		}
	}

	// Step 3: Complete upload
	return c.CompleteFileUpload(ctx, upload.ID)
}
