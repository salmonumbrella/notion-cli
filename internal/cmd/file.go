package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "file",
		Aliases: []string{"f", "files"},
		Short:   "Manage Notion file uploads",
		Long:    `Upload files to Notion and manage file uploads.`,
	}

	cmd.AddCommand(newFileUploadCmd())
	cmd.AddCommand(newFileGetCmd())
	cmd.AddCommand(newFileListCmd())

	return cmd
}

func newFileUploadCmd() *cobra.Command {
	var pageID string
	var propertyName string

	cmd := &cobra.Command{
		Use:     "upload <filepath>",
		Aliases: []string{"up"},
		Short:   "Upload a file to Notion",
		Long: `Upload a file to Notion storage.

If --page and --property are provided, the file will be attached
to the specified page property after upload.

Example - Upload a file:
  ntn file upload ./document.pdf

Example - Upload and attach to page property:
  ntn file upload ./receipt.pdf --page abc123 --property "Attachments"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Open file
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer func() { _ = file.Close() }()

			// Get file info
			fileInfo, err := file.Stat()
			if err != nil {
				return fmt.Errorf("failed to stat file: %w", err)
			}

			filename := filepath.Base(filePath)

			// Detect content type from file content
			buffer := make([]byte, 512)
			n, _ := file.Read(buffer)
			contentType := http.DetectContentType(buffer[:n])
			// Reset file position for upload
			if _, err := file.Seek(0, 0); err != nil {
				return fmt.Errorf("failed to reset file position: %w", err)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			if pageID != "" {
				normalized, err := cmdutil.NormalizeNotionID(pageID)
				if err != nil {
					return err
				}
				pageID = normalized
			}
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Step 1: Create file upload
			createReq := &notion.CreateFileUploadRequest{
				FileName:    filename,
				ContentType: contentType,
			}
			upload, err := client.CreateFileUpload(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create file upload: %w", err)
			}

			// Step 2: Send file
			upload, err = client.SendFileUpload(ctx, upload.UploadURL, file, filename, contentType)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			// Step 3: Attach to page property if specified
			if pageID != "" && propertyName != "" {
				updateReq := &notion.UpdatePageRequest{
					Properties: map[string]interface{}{
						propertyName: map[string]interface{}{
							"files": []map[string]interface{}{
								{
									"type": "file_upload",
									"file_upload": map[string]interface{}{
										"id": upload.ID,
									},
									"name": filename,
								},
							},
						},
					},
				}
				_, err = client.UpdatePage(ctx, pageID, updateReq)
				if err != nil {
					return fmt.Errorf("failed to attach file to page: %w", err)
				}
			}

			// Print result
			result := map[string]interface{}{
				"id":        upload.ID,
				"status":    upload.Status,
				"file_name": filename,
				"size":      fileInfo.Size(),
			}
			if pageID != "" {
				result["attached_to"] = map[string]string{
					"page_id":  pageID,
					"property": propertyName,
				}
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVarP(&pageID, "page", "p", "", "Page ID to attach file to")
	cmd.Flags().StringVar(&propertyName, "property", "", "Property name to attach file to")

	// Flag aliases
	flagAlias(cmd.Flags(), "page", "pg")
	flagAlias(cmd.Flags(), "property", "prop")

	return cmd
}

func newFileGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <file-upload-id>",
		Aliases: []string{"g"},
		Short:   "Get file upload status",
		Long: `Get the status of a file upload by ID.

Example:
  ntn file get abc123-def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileUploadID := args[0]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			upload, err := client.GetFileUpload(ctx, fileUploadID)
			if err != nil {
				return fmt.Errorf("failed to get file upload: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, upload)
		},
	}
}

func newFileListCmd() *cobra.Command {
	var startCursor string
	var pageSize int
	var light bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List file uploads",
		Long: `List file uploads for the current integration.
Use --light (or --li) for compact output (id, file_name, status, size, timestamps).

Example:
  ntn file list --page-size 10
  ntn file list --li`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			opts := &notion.ListFileUploadsOptions{
				StartCursor: startCursor,
				PageSize:    pageSize,
			}

			list, err := client.ListFileUploads(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list file uploads: %w", err)
			}

			if limit > 0 && len(list.Results) > limit {
				list.Results = list.Results[:limit]
				list.HasMore = true
			}

			printer := printerForContext(ctx)
			if light {
				return printer.Print(ctx, map[string]interface{}{
					"object":      "list",
					"results":     toLightFileUploads(list.Results),
					"has_more":    list.HasMore,
					"next_cursor": list.NextCursor,
				})
			}
			return printer.Print(ctx, list)
		},
	}

	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Results per page")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, file_name, status, size, timestamps)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
