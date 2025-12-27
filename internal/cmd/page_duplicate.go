package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

var readOnlyPropertyTypes = map[string]bool{
	"formula":          true,
	"rollup":           true,
	"created_time":     true,
	"created_by":       true,
	"last_edited_time": true,
	"last_edited_by":   true,
	"unique_id":        true,
}

var unsupportedBlockTypes = map[string]bool{
	"unsupported":    true,
	"child_database": true,
	"child_page":     true,
}

func newPageDuplicateCmd() *cobra.Command {
	var parentID string
	var parentType string
	var dataSourceID string
	var titleOverride string
	var noChildren bool

	cmd := &cobra.Command{
		Use:     "duplicate <page-id>",
		Aliases: []string{"dup"},
		Short:   "Duplicate a page",
		Long: `Duplicate a Notion page, optionally changing the parent or title.

By default, the duplicate is created under the same parent and includes children blocks.
Use --no-children to skip block duplication.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			sourceID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			if parentID != "" {
				normalized, err := cmdutil.NormalizeNotionID(resolveID(sf, parentID))
				if err != nil {
					return err
				}
				parentID = normalized
			}
			if dataSourceID != "" {
				normalized, err := cmdutil.NormalizeNotionID(resolveID(sf, dataSourceID))
				if err != nil {
					return err
				}
				dataSourceID = normalized
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			sourcePage, err := client.GetPage(ctx, sourceID)
			if err != nil {
				return fmt.Errorf("failed to fetch source page: %w", err)
			}

			properties := sanitizePageProperties(sourcePage.Properties)
			if titleOverride != "" {
				overrideTitleProperty(properties, titleOverride)
			}

			var parent map[string]interface{}
			if parentID != "" || dataSourceID != "" {
				parent, err = resolvePageParent(ctx, client, parentID, parentType, dataSourceID)
				if err != nil {
					return err
				}
			} else {
				parent = sourcePage.Parent
				if parentType, ok := parent["type"].(string); ok && parentType == "database_id" {
					if dbID, ok := parent["database_id"].(string); ok && dbID != "" {
						resolved, err := resolveDataSourceID(ctx, client, dbID, "")
						if err != nil {
							return err
						}
						parent = map[string]interface{}{"data_source_id": resolved}
					}
				}
			}

			req := &notion.CreatePageRequest{
				Parent:     parent,
				Properties: properties,
				Icon:       sourcePage.Icon,
				Cover:      sourcePage.Cover,
			}

			createdPage, err := client.CreatePage(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to create duplicate page: %w", err)
			}

			if !noChildren {
				children, err := buildBlockTree(ctx, client, sourceID)
				if err != nil {
					return err
				}
				if err := appendChildrenInBatches(ctx, client, createdPage.ID, children); err != nil {
					return err
				}
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, createdPage)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (optional)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'datasource'")
	cmd.Flags().StringVar(&dataSourceID, "datasource", "", "Data source ID (optional, overrides --parent-type database)")
	cmd.Flags().StringVar(&titleOverride, "title", "", "Override the duplicate page title")
	cmd.Flags().BoolVar(&noChildren, "no-children", false, "Skip duplicating page content blocks")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "datasource", "ds")

	return cmd
}

func sanitizePageProperties(raw map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{})
	for name, val := range raw {
		prop, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		propType, _ := prop["type"].(string)
		if propType == "" || readOnlyPropertyTypes[propType] {
			continue
		}
		if value, ok := prop[propType]; ok {
			cleaned[name] = map[string]interface{}{
				propType: value,
			}
		}
	}
	return cleaned
}

func overrideTitleProperty(properties map[string]interface{}, title string) {
	if title == "" {
		return
	}
	for name, val := range properties {
		prop, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := prop["title"]; ok {
			prop["title"] = []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": title,
					},
				},
			}
			properties[name] = prop
			return
		}
	}
	// Fallback for pages without explicit title property name
	properties["title"] = map[string]interface{}{
		"title": []map[string]interface{}{
			{
				"type": "text",
				"text": map[string]interface{}{
					"content": title,
				},
			},
		},
	}
}

func buildBlockTree(ctx context.Context, client blockChildrenReader, blockID string) ([]map[string]interface{}, error) {
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

	var payloads []map[string]interface{}
	for i := range blocks {
		block := blocks[i]
		if unsupportedBlockTypes[block.Type] {
			_, _ = fmt.Fprintf(stderrFromContext(ctx), "warning: skipping unsupported block type %q (%s)\n", block.Type, block.ID)
			continue
		}

		var children []map[string]interface{}
		if block.HasChildren {
			childPayloads, err := buildBlockTree(ctx, client, block.ID)
			if err != nil {
				return nil, err
			}
			children = childPayloads
		}

		content := make(map[string]interface{}, len(block.Content))
		for k, v := range block.Content {
			content[k] = v
		}

		payload := map[string]interface{}{
			"object":   "block",
			"type":     block.Type,
			block.Type: content,
		}
		if len(children) > 0 {
			payload["children"] = children
		}

		payloads = append(payloads, payload)
	}

	return payloads, nil
}

func appendChildrenInBatches(ctx context.Context, client blockChildrenWriter, parentID string, children []map[string]interface{}) error {
	const batchSize = 100

	for i := 0; i < len(children); i += batchSize {
		end := i + batchSize
		if end > len(children) {
			end = len(children)
		}

		req := &notion.AppendBlockChildrenRequest{
			Children: children[i:end],
		}
		if _, err := client.AppendBlockChildren(ctx, parentID, req); err != nil {
			return fmt.Errorf("failed to append block children: %w", err)
		}
	}

	return nil
}
