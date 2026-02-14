package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

type exportBlock struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Content  map[string]interface{} `json:"content,omitempty"`
	Children []exportBlock          `json:"children"`
}

func newPageExportCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "export <page-id>",
		Aliases: []string{"ex"},
		Short:   "Export a page's content",
		Long:    "Export a Notion page as Markdown or JSON block tree.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			pageID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return fmt.Errorf("failed to fetch page: %w", err)
			}

			blocks, err := fetchExportBlocks(ctx, client, pageID)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "markdown", "md":
				markdown := renderMarkdown(blocks, 0)
				if title := pageTitleFromProperties(page.Properties); title != "" {
					markdown = "# " + title + "\n\n" + markdown
				}
				_, _ = fmt.Fprintln(stdoutFromContext(ctx), markdown)
				return nil
			case "json":
				printer := output.NewPrinter(stdoutFromContext(ctx), output.FormatJSON)
				return printer.Print(ctx, map[string]interface{}{
					"page":   page,
					"blocks": blocks,
				})
			default:
				return fmt.Errorf("invalid --format %q (expected markdown or json)", format)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "Export format (markdown or json)")

	return cmd
}

func fetchExportBlocks(ctx context.Context, client blockChildrenReader, blockID string) ([]exportBlock, error) {
	var blocks []notion.Block
	cursor := ""

	for {
		opts := &notion.BlockChildrenOptions{StartCursor: cursor, PageSize: 100}
		list, err := client.GetBlockChildren(ctx, blockID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch block children: %w", err)
		}
		blocks = append(blocks, list.Results...)
		if !list.HasMore || list.NextCursor == nil || *list.NextCursor == "" {
			break
		}
		cursor = *list.NextCursor
	}

	var nodes []exportBlock
	for i := range blocks {
		block := blocks[i]
		node := exportBlock{
			ID:      block.ID,
			Type:    block.Type,
			Content: block.Content,
		}
		if block.HasChildren {
			children, err := fetchExportBlocks(ctx, client, block.ID)
			if err != nil {
				return nil, err
			}
			node.Children = children
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func renderMarkdown(blocks []exportBlock, indent int) string {
	return strings.TrimRight(strings.Join(renderMarkdownLines(blocks, indent), "\n"), "\n")
}

func renderMarkdownLines(blocks []exportBlock, indent int) []string {
	var lines []string
	for _, block := range blocks {
		lines = append(lines, renderBlockMarkdown(block, indent)...)
	}
	return lines
}

func renderBlockMarkdown(block exportBlock, indent int) []string {
	prefix := strings.Repeat("  ", indent)
	switch block.Type {
	case "paragraph":
		return []string{prefix + richTextContentToMarkdown(block.Content, "rich_text")}
	case "heading_1":
		return []string{"# " + richTextContentToMarkdown(block.Content, "rich_text")}
	case "heading_2":
		return []string{"## " + richTextContentToMarkdown(block.Content, "rich_text")}
	case "heading_3":
		return []string{"### " + richTextContentToMarkdown(block.Content, "rich_text")}
	case "bulleted_list_item":
		lines := []string{prefix + "- " + richTextContentToMarkdown(block.Content, "rich_text")}
		if len(block.Children) > 0 {
			lines = append(lines, renderMarkdownLines(block.Children, indent+1)...)
		}
		return lines
	case "numbered_list_item":
		lines := []string{prefix + "1. " + richTextContentToMarkdown(block.Content, "rich_text")}
		if len(block.Children) > 0 {
			lines = append(lines, renderMarkdownLines(block.Children, indent+1)...)
		}
		return lines
	case "to_do":
		checked, _ := block.Content["checked"].(bool)
		box := " "
		if checked {
			box = "x"
		}
		lines := []string{prefix + "- [" + box + "] " + richTextContentToMarkdown(block.Content, "rich_text")}
		if len(block.Children) > 0 {
			lines = append(lines, renderMarkdownLines(block.Children, indent+1)...)
		}
		return lines
	case "toggle":
		lines := []string{prefix + "- " + richTextContentToMarkdown(block.Content, "rich_text")}
		if len(block.Children) > 0 {
			lines = append(lines, renderMarkdownLines(block.Children, indent+1)...)
		}
		return lines
	case "quote":
		return []string{prefix + "> " + richTextContentToMarkdown(block.Content, "rich_text")}
	case "code":
		language, _ := block.Content["language"].(string)
		code := richTextFromContent(block.Content, "rich_text")
		return []string{prefix + "```" + language, code, prefix + "```"}
	case "callout":
		return []string{prefix + "> " + richTextContentToMarkdown(block.Content, "rich_text")}
	case "divider":
		return []string{prefix + "---"}
	case "image":
		if url := extractFileURL(block.Content); url != "" {
			return []string{prefix + "![](" + url + ")"}
		}
		return []string{prefix + "![](unsupported-image)"}
	default:
		if len(block.Children) > 0 {
			lines := []string{prefix + "<!-- unsupported block type: " + block.Type + " -->"}
			lines = append(lines, renderMarkdownLines(block.Children, indent+1)...)
			return lines
		}
		return []string{prefix + "<!-- unsupported block type: " + block.Type + " -->"}
	}
}

func richTextFromContent(content map[string]interface{}, key string) string {
	value, ok := content[key]
	if !ok {
		return ""
	}

	items, ok := value.([]interface{})
	if !ok {
		return ""
	}

	var parts []string
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if plain, ok := entry["plain_text"].(string); ok {
			parts = append(parts, plain)
			continue
		}
		if text, ok := entry["text"].(map[string]interface{}); ok {
			if content, ok := text["content"].(string); ok {
				parts = append(parts, content)
			}
		}
	}
	return strings.Join(parts, "")
}

// richTextToMarkdown converts a Notion rich_text array ([]interface{}) to a markdown string,
// preserving bold, italic, code, strikethrough, and links.
func richTextToMarkdown(items []interface{}) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		text := ""
		if plain, ok := entry["plain_text"].(string); ok {
			text = plain
		} else if textObj, ok := entry["text"].(map[string]interface{}); ok {
			text, _ = textObj["content"].(string)
		}
		if text == "" {
			continue
		}

		linkURL := ""
		if href, ok := entry["href"].(string); ok && href != "" {
			linkURL = href
		} else if textObj, ok := entry["text"].(map[string]interface{}); ok {
			if linkObj, ok := textObj["link"].(map[string]interface{}); ok {
				linkURL, _ = linkObj["url"].(string)
			}
		}

		var bold, italic, code, strikethrough bool
		if ann, ok := entry["annotations"].(map[string]interface{}); ok {
			bold, _ = ann["bold"].(bool)
			italic, _ = ann["italic"].(bool)
			code, _ = ann["code"].(bool)
			strikethrough, _ = ann["strikethrough"].(bool)
		}

		segment := text
		if linkURL != "" {
			segment = "[" + segment + "](" + linkURL + ")"
		}
		if code {
			segment = "`" + segment + "`"
		}
		if strikethrough {
			segment = "~~" + segment + "~~"
		}
		if bold && italic {
			segment = "***" + segment + "***"
		} else if bold {
			segment = "**" + segment + "**"
		} else if italic {
			segment = "*" + segment + "*"
		}

		b.WriteString(segment)
	}

	return b.String()
}

// richTextContentToMarkdown extracts a rich_text array from a block's content map
// and converts it to markdown with formatting preserved.
func richTextContentToMarkdown(content map[string]interface{}, key string) string {
	value, ok := content[key]
	if !ok {
		return ""
	}
	items, ok := value.([]interface{})
	if !ok {
		return ""
	}
	return richTextToMarkdown(items)
}

func extractFileURL(content map[string]interface{}) string {
	fileType, _ := content["type"].(string)
	if fileType == "" {
		if file, ok := content["file"].(map[string]interface{}); ok {
			if url, ok := file["url"].(string); ok {
				return url
			}
		}
		if external, ok := content["external"].(map[string]interface{}); ok {
			if url, ok := external["url"].(string); ok {
				return url
			}
		}
		return ""
	}

	switch fileType {
	case "file":
		if file, ok := content["file"].(map[string]interface{}); ok {
			if url, ok := file["url"].(string); ok {
				return url
			}
		}
	case "external":
		if external, ok := content["external"].(map[string]interface{}); ok {
			if url, ok := external["url"].(string); ok {
				return url
			}
		}
	}

	return ""
}

func pageTitleFromProperties(properties map[string]interface{}) string {
	for _, val := range properties {
		prop, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := prop["title"]; ok {
			return richTextContentToMarkdown(prop, "title")
		}
	}
	return ""
}
