package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type blockTextCommandSpec struct {
	Use    string
	Short  string
	Long   string
	Action string
	Build  func(text string) map[string]interface{}
}

func newBlockAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add common block types",
		Long:  `Convenience commands for adding common block types like paragraphs, headings, lists, etc.`,
	}

	simpleTextSpecs := []blockTextCommandSpec{
		{
			Use:   "paragraph <parent-id> <text>",
			Short: "Add a paragraph block",
			Long: `Add a paragraph block with text content.

Example:
  ntn block add paragraph abc123 "This is a paragraph"`,
			Action: "add paragraph",
			Build:  notion.NewParagraph,
		},
		{
			Use:   "bullet <parent-id> <text>",
			Short: "Add a bulleted list item",
			Long: `Add a bulleted list item block.

Example:
  ntn block add bullet abc123 "First item"`,
			Action: "add bulleted list item",
			Build:  notion.NewBulletedListItem,
		},
		{
			Use:   "number <parent-id> <text>",
			Short: "Add a numbered list item",
			Long: `Add a numbered list item block.

Example:
  ntn block add number abc123 "First item"`,
			Action: "add numbered list item",
			Build:  notion.NewNumberedListItem,
		},
		{
			Use:   "toggle <parent-id> <text>",
			Short: "Add a toggle block",
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
			Use:   "quote <parent-id> <text>",
			Short: "Add a quote block",
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

	return cmd
}

func newSimpleTextBlockAddCmd(spec blockTextCommandSpec) *cobra.Command {
	return &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long:  spec.Long,
		Args:  cobra.ExactArgs(2),
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
		Use:   "heading <parent-id> <text>",
		Short: "Add a heading block",
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
		Use:   "callout <parent-id> <text>",
		Short: "Add a callout block",
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
		Use:   "code <parent-id> <code>",
		Short: "Add a code block",
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
		Use:   "todo <parent-id> <text>",
		Short: "Add a to-do item",
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
		Use:   "image <parent-id>",
		Short: "Add an image block from a local file",
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
			n, _ := file.Read(buffer)
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
