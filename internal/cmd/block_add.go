package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newBlockAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add common block types",
		Long:  `Convenience commands for adding common block types like paragraphs, headings, lists, etc.`,
	}

	cmd.AddCommand(newBlockAddParagraphCmd())
	cmd.AddCommand(newBlockAddHeadingCmd())
	cmd.AddCommand(newBlockAddBulletCmd())
	cmd.AddCommand(newBlockAddNumberCmd())
	cmd.AddCommand(newBlockAddToggleCmd())
	cmd.AddCommand(newBlockAddQuoteCmd())
	cmd.AddCommand(newBlockAddCalloutCmd())
	cmd.AddCommand(newBlockAddCodeCmd())
	cmd.AddCommand(newBlockAddToDoCmd())
	cmd.AddCommand(newBlockAddImageCmd())

	return cmd
}

func newBlockAddParagraphCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paragraph <parent-id> <text>",
		Short: "Add a paragraph block",
		Long: `Add a paragraph block with text content.

Example:
  notion block add paragraph abc123 "This is a paragraph"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewParagraph(text)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add paragraph: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddHeadingCmd() *cobra.Command {
	var level int

	cmd := &cobra.Command{
		Use:   "heading <parent-id> <text>",
		Short: "Add a heading block",
		Long: `Add a heading block with specified level (1-3).

Examples:
  notion block add heading abc123 "Main Title" --level 1
  notion block add heading abc123 "Subtitle" --level 2
  notion block add heading abc123 "Section" --level 3`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			if level < 1 || level > 3 {
				return fmt.Errorf("heading level must be between 1 and 3, got %d", level)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			var block map[string]interface{}
			switch level {
			case 1:
				block = notion.NewHeading1(text)
			case 2:
				block = notion.NewHeading2(text)
			case 3:
				block = notion.NewHeading3(text)
			}

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add heading: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().IntVar(&level, "level", 2, "Heading level (1-3)")
	return cmd
}

func newBlockAddBulletCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bullet <parent-id> <text>",
		Short: "Add a bulleted list item",
		Long: `Add a bulleted list item block.

Example:
  notion block add bullet abc123 "First item"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewBulletedListItem(text)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add bulleted list item: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddNumberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "number <parent-id> <text>",
		Short: "Add a numbered list item",
		Long: `Add a numbered list item block.

Example:
  notion block add number abc123 "First item"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewNumberedListItem(text)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add numbered list item: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddToggleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle <parent-id> <text>",
		Short: "Add a toggle block",
		Long: `Add a toggle block (collapsible section).

Note: Toggle blocks in Notion are implemented as bulleted list items with children.
You can add children to the created toggle block using its block ID.

Example:
  notion block add toggle abc123 "Click to expand"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			// Note: Toggle blocks in Notion API are toggle_list_item blocks
			block := map[string]interface{}{
				"type": "toggle",
				"toggle": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]interface{}{
								"content": text,
							},
						},
					},
				},
			}

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add toggle: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddQuoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quote <parent-id> <text>",
		Short: "Add a quote block",
		Long: `Add a quote block.

Example:
  notion block add quote abc123 "To be or not to be"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewQuote(text)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add quote: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}
}

func newBlockAddCalloutCmd() *cobra.Command {
	var emoji string

	cmd := &cobra.Command{
		Use:   "callout <parent-id> <text>",
		Short: "Add a callout block",
		Long: `Add a callout block with an emoji icon.

Examples:
  notion block add callout abc123 "Important note" --emoji "💡"
  notion block add callout abc123 "Warning!" --emoji "⚠️"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewCallout(text, emoji)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add callout: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
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
  notion block add code abc123 'func main() {}' --language go
  notion block add code abc123 'console.log("hello")' --language javascript
  notion block add code abc123 'def hello():\n    print("hello")' --language python`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			code := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewCode(code, language)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add code block: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
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
  notion block add todo abc123 "Buy milk"
  notion block add todo abc123 "Completed task" --checked`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}
			text := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			block := notion.NewToDo(text, checked)
			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{block},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add to-do: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
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
  notion block add image abc123 --file ./photo.jpg

Example with caption:
  notion block add image abc123 --file ./photo.jpg --caption "Team offsite"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			parentID, err := cmdutil.NormalizeNotionID(args[0])
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

			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			createReq := &notion.CreateFileUploadRequest{
				FileName:    filename,
				ContentType: contentType,
			}
			upload, err := client.CreateFileUpload(ctx, createReq)
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
					"type": "file_upload",
					"file_upload": map[string]interface{}{
						"id": upload.ID,
					},
				},
			}

			if caption != "" {
				imageProps := image["image"].(map[string]interface{})
				imageProps["caption"] = []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": caption,
						},
					},
				}
			}

			req := &notion.AppendBlockChildrenRequest{
				Children: []map[string]interface{}{image},
			}

			result, err := client.AppendBlockChildren(ctx, parentID, req)
			if err != nil {
				return fmt.Errorf("failed to add image: %w", err)
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Local image file path")
	cmd.Flags().StringVar(&caption, "caption", "", "Caption text for the image")

	return cmd
}
