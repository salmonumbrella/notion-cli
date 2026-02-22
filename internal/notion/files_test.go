package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCreateFileUpload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/file_uploads" {
			t.Errorf("expected path /file_uploads, got %s", r.URL.Path)
		}

		var req CreateFileUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.FileName != "test.pdf" {
			t.Errorf("expected file_name 'test.pdf', got %q", req.FileName)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUpload{
			Object:    "file_upload",
			ID:        "upload123",
			Status:    "pending",
			UploadURL: "https://upload.example.com/abc",
			FileName:  req.FileName,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateFileUploadRequest{
		FileName:    "test.pdf",
		ContentType: "application/pdf",
	}

	upload, err := client.CreateFileUpload(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.ID != "upload123" {
		t.Errorf("expected ID 'upload123', got %q", upload.ID)
	}
	if upload.UploadURL != "https://upload.example.com/abc" {
		t.Errorf("expected upload URL, got %q", upload.UploadURL)
	}
}

func TestCreateFileUpload_NilRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CreateFileUpload(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	expected := "file_name is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateFileUpload_EmptyFileName(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req := &CreateFileUploadRequest{
		FileName: "",
	}

	_, err := client.CreateFileUpload(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty file name")
	}

	expected := "file_name is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestSendFileUpload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Verify it's multipart
		contentType := r.Header.Get("Content-Type")
		if contentType == "" || len(contentType) < 19 || contentType[:19] != "multipart/form-data" {
			t.Errorf("expected multipart/form-data content type, got %s", contentType)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUpload{
			Object: "file_upload",
			ID:     "upload123",
			Status: "uploaded",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	fileContent := bytes.NewReader([]byte("test file content"))
	upload, err := client.SendFileUpload(ctx, server.URL+"/upload", fileContent, "test.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "uploaded" {
		t.Errorf("expected status 'uploaded', got %q", upload.Status)
	}
}

func TestSendFileUpload_EmptyURL(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	fileContent := bytes.NewReader([]byte("test"))
	_, err := client.SendFileUpload(ctx, "", fileContent, "test.pdf", "application/pdf")
	if err == nil {
		t.Fatal("expected error for empty upload URL")
	}

	expected := "upload URL is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCompleteFileUpload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/file_uploads/upload123/complete" {
			t.Errorf("expected path /file_uploads/upload123/complete, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUpload{
			Object: "file_upload",
			ID:     "upload123",
			Status: "complete",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	upload, err := client.CompleteFileUpload(ctx, "upload123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", upload.Status)
	}
}

func TestCompleteFileUpload_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.CompleteFileUpload(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty file upload ID")
	}

	expected := "file upload ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetFileUpload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/file_uploads/upload123" {
			t.Errorf("expected path /file_uploads/upload123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUpload{
			Object:   "file_upload",
			ID:       "upload123",
			Status:   "complete",
			FileName: "test.pdf",
			MimeType: "application/pdf",
			Size:     12345,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	upload, err := client.GetFileUpload(ctx, "upload123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.ID != "upload123" {
		t.Errorf("expected ID 'upload123', got %q", upload.ID)
	}
	if upload.FileName != "test.pdf" {
		t.Errorf("expected file_name 'test.pdf', got %q", upload.FileName)
	}
	if upload.Size != 12345 {
		t.Errorf("expected size 12345, got %d", upload.Size)
	}
}

func TestGetFileUpload_EmptyID(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	_, err := client.GetFileUpload(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty file upload ID")
	}

	expected := "file upload ID is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGetFileUpload_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Object:  "error",
			Status:  404,
			Code:    "object_not_found",
			Message: "Could not find file upload.",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	_, err := client.GetFileUpload(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent file upload")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestListFileUploads_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/file_uploads" {
			t.Errorf("expected path /file_uploads, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUploadList{
			Object: "list",
			Results: []*FileUpload{
				{Object: "file_upload", ID: "upload1"},
				{Object: "file_upload", ID: "upload2"},
			},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	list, err := client.ListFileUploads(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(list.Results))
	}
}

func TestListFileUploads_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "10" {
			t.Errorf("expected page_size=10, got %s", r.URL.Query().Get("page_size"))
		}
		if r.URL.Query().Get("start_cursor") != "cursor123" {
			t.Errorf("expected start_cursor=cursor123, got %s", r.URL.Query().Get("start_cursor"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUploadList{
			Object:  "list",
			Results: []*FileUpload{},
			HasMore: false,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	opts := &ListFileUploadsOptions{
		StartCursor: "cursor123",
		PageSize:    10,
	}

	_, err := client.ListFileUploads(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListFileUploads_Pagination(t *testing.T) {
	nextCursor := "next123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUploadList{
			Object: "list",
			Results: []*FileUpload{
				{Object: "file_upload", ID: "upload1"},
			},
			NextCursor: &nextCursor,
			HasMore:    true,
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	list, err := client.ListFileUploads(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !list.HasMore {
		t.Error("expected HasMore to be true")
	}
	if list.NextCursor == nil || *list.NextCursor != "next123" {
		t.Error("expected NextCursor to be 'next123'")
	}
}

func TestCreateFileUpload_MultiPart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateFileUploadRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Mode != "multi_part" {
			t.Errorf("expected mode 'multi_part', got %q", req.Mode)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUpload{
			Object:    "file_upload",
			ID:        "upload123",
			Status:    "pending",
			UploadURL: "https://upload.example.com/part",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	req := &CreateFileUploadRequest{
		FileName: "large-file.zip",
		Mode:     "multi_part",
	}

	upload, err := client.CreateFileUpload(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.ID != "upload123" {
		t.Errorf("expected ID 'upload123', got %q", upload.ID)
	}
}

func TestSendFilePart_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify part number header
		partNum := r.Header.Get("X-Part-Number")
		if partNum == "" {
			t.Error("expected X-Part-Number header")
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FileUploadPart{
			PartNumber: 1,
			Status:     "uploaded",
		})
	}))
	defer server.Close()

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	part := bytes.NewReader([]byte("chunk data"))
	result, err := client.SendFilePart(ctx, server.URL+"/upload", part, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PartNumber != 1 {
		t.Errorf("expected part number 1, got %d", result.PartNumber)
	}
}

func TestUploadLargeFile_Integration(t *testing.T) {
	// Test the full multi-part upload workflow
	var partCount atomic.Int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file_uploads" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{
				ID:        "upload123",
				Status:    "pending",
				UploadURL: serverURL + "/upload",
			})
			return
		}
		if r.URL.Path == "/upload" && r.Method == "POST" {
			current := partCount.Add(1)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUploadPart{PartNumber: int(current), Status: "uploaded"})
			return
		}
		if r.URL.Path == "/file_uploads/upload123/complete" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{ID: "upload123", Status: "complete"})
			return
		}
	}))
	defer server.Close()

	// Set the server URL after the server is created
	serverURL = server.URL

	client := NewClient("test-token").WithBaseURL(server.URL)

	// Create large fake data (3 chunks worth)
	data := make([]byte, DefaultChunkSize*3)
	for i := range data {
		data[i] = byte(i % 256)
	}

	ctx := context.Background()
	upload, err := client.UploadLargeFile(ctx, "test.bin", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", upload.Status)
	}
	if got := partCount.Load(); got != 3 {
		t.Errorf("expected 3 parts uploaded, got %d", got)
	}
}

func TestSendFilePart_EmptyURL(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	part := bytes.NewReader([]byte("test data"))
	_, err := client.SendFilePart(ctx, "", part, 1)
	if err == nil {
		t.Fatal("expected error for empty upload URL")
	}

	expected := "upload URL is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestSendFilePart_InvalidPartNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with invalid part number")
	}))
	defer server.Close()

	client := NewClient("test-token")
	ctx := context.Background()

	part := bytes.NewReader([]byte("test data"))

	// Test with part number 0
	_, err := client.SendFilePart(ctx, server.URL+"/upload", part, 0)
	if err == nil {
		t.Fatal("expected error for part number 0")
	}
	expected := "part number must be >= 1"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}

	// Test with negative part number
	part = bytes.NewReader([]byte("test data"))
	_, err = client.SendFilePart(ctx, server.URL+"/upload", part, -5)
	if err == nil {
		t.Fatal("expected error for negative part number")
	}
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestUploadLargeFile_EmptyFile(t *testing.T) {
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file_uploads" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{
				ID:        "upload123",
				Status:    "pending",
				UploadURL: serverURL + "/upload",
			})
			return
		}
		if r.URL.Path == "/file_uploads/upload123/complete" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{ID: "upload123", Status: "complete"})
			return
		}
	}))
	defer server.Close()

	serverURL = server.URL

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// Create empty file (0 bytes)
	emptyFile := bytes.NewReader([]byte{})

	upload, err := client.UploadLargeFile(ctx, "empty.bin", emptyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", upload.Status)
	}
}

func TestUploadLargeFile_SingleChunk(t *testing.T) {
	partCount := 0
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file_uploads" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{
				ID:        "upload123",
				Status:    "pending",
				UploadURL: serverURL + "/upload",
			})
			return
		}
		if r.URL.Path == "/upload" && r.Method == "POST" {
			partCount++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUploadPart{PartNumber: partCount, Status: "uploaded"})
			return
		}
		if r.URL.Path == "/file_uploads/upload123/complete" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{ID: "upload123", Status: "complete"})
			return
		}
	}))
	defer server.Close()

	serverURL = server.URL

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// Create small file (1KB, much smaller than DefaultChunkSize of 5MB)
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	upload, err := client.UploadLargeFile(ctx, "small.bin", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", upload.Status)
	}
	if partCount != 1 {
		t.Errorf("expected 1 part uploaded, got %d", partCount)
	}
}

func TestUploadLargeFile_ExactChunkSize(t *testing.T) {
	partCount := 0
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file_uploads" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{
				ID:        "upload123",
				Status:    "pending",
				UploadURL: serverURL + "/upload",
			})
			return
		}
		if r.URL.Path == "/upload" && r.Method == "POST" {
			partCount++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUploadPart{PartNumber: partCount, Status: "uploaded"})
			return
		}
		if r.URL.Path == "/file_uploads/upload123/complete" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(FileUpload{ID: "upload123", Status: "complete"})
			return
		}
	}))
	defer server.Close()

	serverURL = server.URL

	client := NewClient("test-token").WithBaseURL(server.URL)
	ctx := context.Background()

	// Create file exactly DefaultChunkSize bytes (5MB)
	data := make([]byte, DefaultChunkSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	upload, err := client.UploadLargeFile(ctx, "exact.bin", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", upload.Status)
	}
	if partCount != 1 {
		t.Errorf("expected 1 part uploaded, got %d", partCount)
	}
}
