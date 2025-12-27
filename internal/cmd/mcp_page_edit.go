package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/mcp"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func newMCPCreateCmd() *cobra.Command {
	var (
		parentID       string
		dataSourceID   string
		title          string
		content        string
		contentFile    string
		propertiesJSON string
		templateID     string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a Notion page via MCP",
		Long: `Create a new Notion page with markdown content using the MCP
notion-create-pages tool.

Use --parent for a parent page ID, or --data-source for a data source ID.
If neither is provided, the page is created as a standalone workspace page.

Use --title as a shorthand for setting properties.title. For full control
over page properties, use --properties with a JSON object.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			if title == "" && propertiesJSON == "" && templateID == "" {
				return fmt.Errorf("--title, --properties, or --template-id is required")
			}

			// Read content from file if specified.
			body := content
			if contentFile != "" {
				data, err := readTextFileForFlag(contentFile, "content file")
				if err != nil {
					return err
				}
				body = data
			}
			if templateID != "" && strings.TrimSpace(body) != "" {
				return fmt.Errorf("--content/--file cannot be used with --template-id")
			}
			if templateID != "" {
				// Avoid sending whitespace-only content when a template is selected.
				body = ""
			}

			// Build properties map.
			props, err := parseMCPJSONObject(propertiesJSON, "properties")
			if err != nil {
				return err
			}
			if title != "" {
				if props == nil {
					props = map[string]interface{}{}
				}
				props["title"] = title
			}
			if len(props) > 0 {
				props = resolveMCPCreatePeopleIDs(sf, props)
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			// Build parent if specified.
			var parent *mcp.CreatePagesParent
			if parentID != "" || dataSourceID != "" {
				parent = &mcp.CreatePagesParent{
					PageID:       parentID,
					DataSourceID: dataSourceID,
				}
			}

			page := mcp.CreatePageInput{
				Properties: props,
				Content:    body,
				TemplateID: templateID,
			}
			result, err := client.CreatePages(ctx, parent, []mcp.CreatePageInput{page})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID for the parent")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Page title (shorthand for properties.title)")
	cmd.Flags().StringVar(&content, "content", "", "Markdown content for the page body")
	cmd.Flags().StringVarP(&contentFile, "file", "f", "", "Read markdown content from a file path")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as a JSON object")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Template ID to apply when creating a page in a data source")

	return cmd
}

func resolveMCPCreatePeopleIDs(sf *skill.SkillFile, properties map[string]interface{}) map[string]interface{} {
	if sf == nil || len(properties) == 0 {
		return properties
	}

	for _, rawProp := range properties {
		propMap, ok := rawProp.(map[string]interface{})
		if !ok {
			continue
		}
		peopleRaw, ok := propMap["people"]
		if !ok {
			continue
		}
		peopleArr, ok := peopleRaw.([]interface{})
		if !ok {
			continue
		}
		for _, personRaw := range peopleArr {
			personMap, ok := personRaw.(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := personMap["id"].(string)
			if !ok || strings.TrimSpace(id) == "" {
				continue
			}
			personMap["id"] = resolveUserID(sf, id)
		}
	}
	return properties
}

func newMCPEditCmd() *cobra.Command {
	var (
		replaceContent       string
		replaceRange         string
		insertAfter          string
		newStr               string
		propertiesJSON       string
		applyTemplateID      string
		allowDeletingContent bool
	)

	cmd := &cobra.Command{
		Use:     "edit <page-id>",
		Aliases: []string{"e"},
		Short:   "Edit a Notion page via MCP",
		Long: `Edit a Notion page using the MCP notion-update-page tool.

Supports five operations:
  --replace <markdown>                          Replace entire page content
  --replace-range <selection> --new <markdown>  Replace a range identified by selection_with_ellipsis
  --insert-after <selection> --new <markdown>   Insert content after the matched selection
  --properties <json>                           Update page properties (JSON object)
  --apply-template <template-id>                Apply a template to the page`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pageID := args[0]

			// Count operations to ensure exactly one is specified.
			ops := 0
			if replaceContent != "" {
				ops++
			}
			if replaceRange != "" {
				ops++
			}
			if insertAfter != "" {
				ops++
			}
			if propertiesJSON != "" {
				ops++
			}
			if applyTemplateID != "" {
				ops++
			}
			if ops == 0 {
				return fmt.Errorf("specify one of --replace, --replace-range, --insert-after, --properties, or --apply-template")
			}
			if ops > 1 {
				return fmt.Errorf("specify only one of --replace, --replace-range, --insert-after, --properties, or --apply-template")
			}

			client, cleanup, err := mcpClientFromToken(ctx)
			if err != nil {
				return err
			}
			defer cleanup()

			var params mcp.UpdatePageParams
			params.PageID = pageID
			if cmd.Flags().Changed("allow-deleting-content") {
				params.AllowDeletingContent = &allowDeletingContent
			}

			switch {
			case replaceContent != "":
				params.Command = mcp.UpdateCmdReplaceContent
				params.NewStr = replaceContent
			case replaceRange != "":
				if newStr == "" {
					return fmt.Errorf("--new is required when using --replace-range")
				}
				params.Command = mcp.UpdateCmdReplaceContentRange
				params.SelectionWithEllipsis = replaceRange
				params.NewStr = newStr
			case insertAfter != "":
				if newStr == "" {
					return fmt.Errorf("--new is required when using --insert-after")
				}
				params.Command = mcp.UpdateCmdInsertContentAfter
				params.SelectionWithEllipsis = insertAfter
				params.NewStr = newStr
			case propertiesJSON != "":
				props, err := parseMCPJSONObject(propertiesJSON, "properties")
				if err != nil {
					return err
				}
				params.Command = mcp.UpdateCmdUpdateProperties
				params.Properties = props
			case applyTemplateID != "":
				params.Command = mcp.UpdateCmdApplyTemplate
				params.TemplateID = applyTemplateID
			}

			result, err := client.UpdatePage(ctx, params)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(stdoutFromContext(ctx), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&replaceContent, "replace", "", "Replace entire page content with markdown")
	cmd.Flags().StringVar(&replaceRange, "replace-range", "", "Selection with ellipsis to match a content range")
	cmd.Flags().StringVar(&insertAfter, "insert-after", "", "Selection with ellipsis to insert content after")
	cmd.Flags().StringVar(&newStr, "new", "", "New markdown content (used with --replace-range or --insert-after)")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties to update (JSON object)")
	cmd.Flags().StringVar(&applyTemplateID, "apply-template", "", "Template ID to apply to the page")
	cmd.Flags().BoolVar(&allowDeletingContent, "allow-deleting-content", false, "Allow operations that delete existing child content")

	return cmd
}
