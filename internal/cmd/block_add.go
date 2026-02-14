package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// notionSupportedFileExtensions lists all file extensions accepted by the Notion API.
// See: https://developers.notion.com/docs/working-with-files-and-media
var notionSupportedFileExtensions = map[string]bool{
	// Audio
	".aac": true, ".adts": true, ".mid": true, ".midi": true, ".mp3": true,
	".mpga": true, ".m4a": true, ".m4b": true, ".oga": true, ".ogg": true,
	".wav": true, ".wma": true,
	// Document
	".pdf": true, ".txt": true, ".json": true, ".doc": true, ".dot": true,
	".docx": true, ".dotx": true, ".xls": true, ".xlt": true, ".xla": true,
	".xlsx": true, ".xltx": true, ".ppt": true, ".pot": true, ".pps": true,
	".ppa": true, ".pptx": true, ".potx": true,
	// Image
	".gif": true, ".heic": true, ".jpeg": true, ".jpg": true, ".png": true,
	".svg": true, ".tif": true, ".tiff": true, ".webp": true, ".ico": true,
	// Video
	".amv": true, ".asf": true, ".wmv": true, ".avi": true, ".f4v": true,
	".flv": true, ".gifv": true, ".m4v": true, ".mp4": true, ".mkv": true,
	".webm": true, ".mov": true, ".qt": true, ".mpeg": true,
}

// unsupportedFileWorkarounds maps common unsupported extensions to suggested alternatives.
var unsupportedFileWorkarounds = map[string]string{
	".md":       "rename to .txt",
	".markdown": "rename to .txt",
	".yml":      "rename to .txt",
	".yaml":     "rename to .txt",
	".xml":      "rename to .txt",
	".html":     "rename to .txt",
	".htm":      "rename to .txt",
	".csv":      "rename to .txt",
	".tsv":      "rename to .txt",
	".log":      "rename to .txt",
	".ini":      "rename to .txt",
	".toml":     "rename to .txt",
	".cfg":      "rename to .txt",
	".conf":     "rename to .txt",
	".sh":       "rename to .txt",
	".py":       "rename to .txt",
	".go":       "rename to .txt",
	".rs":       "rename to .txt",
	".js":       "rename to .txt",
	".ts":       "rename to .txt",
}

// validateFileExtension checks if a file extension is supported by the Notion API.
// Returns nil if supported, or an error with a helpful message including workarounds.
func validateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fmt.Errorf("file %q has no extension; Notion requires a supported file extension", filename)
	}
	if notionSupportedFileExtensions[ext] {
		return nil
	}
	msg := fmt.Sprintf("Notion API does not support %q files", ext)
	if workaround, ok := unsupportedFileWorkarounds[ext]; ok {
		msg += fmt.Sprintf(" (%s before uploading)", workaround)
	}
	msg += fmt.Sprintf("\nSupported: %s", supportedExtensionsList())
	return fmt.Errorf("%s", msg)
}

func supportedExtensionsList() string {
	exts := make([]string, 0, len(notionSupportedFileExtensions))
	for ext := range notionSupportedFileExtensions {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return strings.Join(exts, ", ")
}

type blockTextCommandSpec struct {
	Use     string
	Aliases []string
	Short   string
	Long    string
	Action  string
	Build   func(text string) map[string]interface{}
}

func newBlockAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Short:   "Add common block types",
		Long:    `Convenience commands for adding common block types like paragraphs, headings, lists, etc.`,
	}

	simpleTextSpecs := []blockTextCommandSpec{
		{
			Use:     "paragraph <parent-id> <text>",
			Aliases: []string{"para", "p"},
			Short:   "Add a paragraph block",
			Long: `Add a paragraph block with text content.

Example:
  ntn block add paragraph abc123 "This is a paragraph"`,
			Action: "add paragraph",
			Build:  notion.NewParagraph,
		},
		{
			Use:     "bullet <parent-id> <text>",
			Aliases: []string{"bl"},
			Short:   "Add a bulleted list item",
			Long: `Add a bulleted list item block.

Example:
  ntn block add bullet abc123 "First item"`,
			Action: "add bulleted list item",
			Build:  notion.NewBulletedListItem,
		},
		{
			Use:     "number <parent-id> <text>",
			Aliases: []string{"num", "nl"},
			Short:   "Add a numbered list item",
			Long: `Add a numbered list item block.

Example:
  ntn block add number abc123 "First item"`,
			Action: "add numbered list item",
			Build:  notion.NewNumberedListItem,
		},
		{
			Use:     "toggle <parent-id> <text>",
			Aliases: []string{"tg"},
			Short:   "Add a toggle block",
			Long: `Add a toggle block (collapsible section).

Note: Toggle blocks in Notion are implemented as bulleted list items with children.
You can add children to the created toggle block using its block ID.

Example:
  ntn block add toggle abc123 "Click to expand"`,
			Action: "add toggle",
			Build: func(text string) map[string]interface{} {
				return map[string]interface{}{
					"type": "toggle",
					"toggle": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"type": "text",
								"text": map[string]interface{}{"content": text},
							},
						},
					},
				}
			},
		},
		{
			Use:     "quote <parent-id> <text>",
			Aliases: []string{"qt"},
			Short:   "Add a quote block",
			Long: `Add a quote block.

Example:
  ntn block add quote abc123 "To be or not to be"`,
			Action: "add quote",
			Build:  notion.NewQuote,
		},
	}

	for _, spec := range simpleTextSpecs {
		cmd.AddCommand(newSimpleTextBlockAddCmd(spec))
	}

	cmd.AddCommand(newBlockAddHeadingCmd())
	cmd.AddCommand(newBlockAddCalloutCmd())
	cmd.AddCommand(newBlockAddCodeCmd())
	cmd.AddCommand(newBlockAddToDoCmd())
	cmd.AddCommand(newBlockAddImageCmd())
	cmd.AddCommand(newBlockAddFileCmd())

	return cmd
}

func newSimpleTextBlockAddCmd(spec blockTextCommandSpec) *cobra.Command {
	return &cobra.Command{
		Use:     spec.Use,
		Aliases: spec.Aliases,
		Short:   spec.Short,
		Long:    spec.Long,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			result, err := appendSingleBlock(ctx, client, parentID, spec.Build(args[1]), spec.Action, args[0])
			if err != nil {
				return err
			}

			return printerForContext(ctx).Print(ctx, result)
		},
	}
}

func resolveBlockAddParentID(ctx context.Context, raw string) (string, error) {
	sf := SkillFileFromContext(ctx)
	return cmdutil.NormalizeNotionID(resolveID(sf, raw))
}

func appendSingleBlock(ctx context.Context, client *notion.Client, parentID string, block map[string]interface{}, action, identifier string) (*notion.BlockList, error) {
	req := &notion.AppendBlockChildrenRequest{Children: []map[string]interface{}{block}}
	result, err := client.AppendBlockChildren(ctx, parentID, req)
	if err != nil {
		return nil, wrapAPIError(err, action, "block", identifier)
	}
	return result, nil
}

func newBlockAddHeadingCmd() *cobra.Command {
	var level int

	cmd := &cobra.Command{
		Use:     "heading <parent-id> <text>",
		Aliases: []string{"h"},
		Short:   "Add a heading block",
		Long: `Add a heading block with specified level (1-3).

Examples:
  ntn block add heading abc123 "Main Title" --level 1
  ntn block add heading abc123 "Subtitle" --level 2
  ntn block add heading abc123 "Section" --level 3`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}
			if level < 1 || level > 3 {
				return fmt.Errorf("heading level must be between 1 and 3, got %d", level)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			text := args[1]
			var block map[string]interface{}
			switch level {
			case 1:
				block = notion.NewHeading1(text)
			case 2:
				block = notion.NewHeading2(text)
			default:
				block = notion.NewHeading3(text)
			}

			result, err := appendSingleBlock(ctx, client, parentID, block, "add heading", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&level, "level", 2, "Heading level (1-3)")
	return cmd
}

func newBlockAddCalloutCmd() *cobra.Command {
	var emoji string

	cmd := &cobra.Command{
		Use:     "callout <parent-id> <text>",
		Aliases: []string{"co"},
		Short:   "Add a callout block",
		Long: `Add a callout block with an emoji icon.

Examples:
  ntn block add callout abc123 "Important note" --emoji "💡"
  ntn block add callout abc123 "Warning!" --emoji "⚠️"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			result, err := appendSingleBlock(ctx, client, parentID, notion.NewCallout(args[1], emoji), "add callout", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&emoji, "emoji", "💡", "Emoji icon for the callout")
	return cmd
}

func newBlockAddCodeCmd() *cobra.Command {
	var language string

	cmd := &cobra.Command{
		Use:     "code <parent-id> <code>",
		Aliases: []string{"cd"},
		Short:   "Add a code block",
		Long: `Add a code block with syntax highlighting.

Examples:
  ntn block add code abc123 'func main() {}' --language go
  ntn block add code abc123 'console.log("hello")' --language javascript
  ntn block add code abc123 'def hello():\n    print("hello")' --language python`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			result, err := appendSingleBlock(ctx, client, parentID, notion.NewCode(args[1], language), "add code block", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&language, "language", "plain text", "Programming language for syntax highlighting")
	return cmd
}

func newBlockAddToDoCmd() *cobra.Command {
	var checked bool

	cmd := &cobra.Command{
		Use:     "todo <parent-id> <text>",
		Aliases: []string{"td"},
		Short:   "Add a to-do item",
		Long: `Add a to-do (checkbox) item.

Examples:
  ntn block add todo abc123 "Buy milk"
  ntn block add todo abc123 "Completed task" --checked`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			result, err := appendSingleBlock(ctx, client, parentID, notion.NewToDo(args[1], checked), "add to-do", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().BoolVar(&checked, "checked", false, "Mark the to-do as checked")
	return cmd
}

func newBlockAddImageCmd() *cobra.Command {
	var filePath string
	var caption string

	cmd := &cobra.Command{
		Use:     "image <parent-id>",
		Aliases: []string{"img", "i"},
		Short:   "Add an image block from a local file",
		Long: `Add an image block by uploading a local file.

Example:
  ntn block add image abc123 --file ./photo.jpg

Example with caption:
  ntn block add image abc123 --file ./photo.jpg --caption "Team offsite"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			if err := validateFileExtension(filePath); err != nil {
				return err
			}

			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}

			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer func() { _ = file.Close() }()

			filename := filepath.Base(filePath)
			buffer := make([]byte, 512)
			n, readErr := file.Read(buffer)
			if readErr != nil {
				return fmt.Errorf("failed to read file header: %w", readErr)
			}
			contentType := http.DetectContentType(buffer[:n])
			if _, err := file.Seek(0, 0); err != nil {
				return fmt.Errorf("failed to reset file position: %w", err)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			upload, err := client.CreateFileUpload(ctx, &notion.CreateFileUploadRequest{
				FileName:    filename,
				ContentType: contentType,
			})
			if err != nil {
				return fmt.Errorf("failed to create file upload: %w", err)
			}

			upload, err = client.SendFileUpload(ctx, upload.UploadURL, file, filename, contentType)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			image := map[string]interface{}{
				"type": "image",
				"image": map[string]interface{}{
					"type":        "file_upload",
					"file_upload": map[string]interface{}{"id": upload.ID},
				},
			}
			if caption != "" {
				imageProps := image["image"].(map[string]interface{})
				imageProps["caption"] = []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{"content": caption},
					},
				}
			}

			result, err := appendSingleBlock(ctx, client, parentID, image, "add image", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Local image file path")
	cmd.Flags().StringVar(&caption, "caption", "", "Caption text for the image")
	return cmd
}

func newBlockAddFileCmd() *cobra.Command {
	var filePath string
	var caption string

	cmd := &cobra.Command{
		Use:     "file <parent-id>",
		Aliases: []string{"f"},
		Short:   "Add a file block from a local file",
		Long: `Add a file block by uploading a local file to a page body.

Supported extensions:
  Document: .pdf .txt .json .doc .docx .dotx .xls .xlsx .xltx .ppt .pptx .potx
  Image:    .gif .heic .jpeg .jpg .png .svg .tif .tiff .webp .ico
  Audio:    .aac .mp3 .m4a .m4b .mp4 .ogg .wav .wma
  Video:    .mp4 .mov .avi .mkv .webm .mpeg .flv .wmv

Note: .md files are NOT supported by Notion. Rename to .txt before uploading.

Example:
  ntn block add file abc123 --file ./report.pdf

Example with caption:
  ntn block add file abc123 --file ./report.pdf --caption "Q4 Report"

Workaround for .md files:
  cp notes.md notes.txt && ntn block add file abc123 --file ./notes.txt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			if err := validateFileExtension(filePath); err != nil {
				return err
			}

			parentID, err := resolveBlockAddParentID(ctx, args[0])
			if err != nil {
				return err
			}

			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer func() { _ = file.Close() }()

			filename := filepath.Base(filePath)
			buffer := make([]byte, 512)
			n, readErr := file.Read(buffer)
			if readErr != nil {
				return fmt.Errorf("failed to read file header: %w", readErr)
			}
			contentType := http.DetectContentType(buffer[:n])
			if _, err := file.Seek(0, 0); err != nil {
				return fmt.Errorf("failed to reset file position: %w", err)
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			upload, err := client.CreateFileUpload(ctx, &notion.CreateFileUploadRequest{
				FileName:    filename,
				ContentType: contentType,
			})
			if err != nil {
				return fmt.Errorf("failed to create file upload: %w", err)
			}

			upload, err = client.SendFileUpload(ctx, upload.UploadURL, file, filename, contentType)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			fileBlock := map[string]interface{}{
				"type": "file",
				"file": map[string]interface{}{
					"type":        "file_upload",
					"file_upload": map[string]interface{}{"id": upload.ID},
					"name":        filename,
				},
			}
			if caption != "" {
				fileProps := fileBlock["file"].(map[string]interface{})
				fileProps["caption"] = []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{"content": caption},
					},
				}
			}

			result, err := appendSingleBlock(ctx, client, parentID, fileBlock, "add file", args[0])
			if err != nil {
				return err
			}
			return printerForContext(ctx).Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Local file path to upload")
	cmd.Flags().StringVar(&caption, "caption", "", "Caption text for the file")
	return cmd
}
