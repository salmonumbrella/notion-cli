package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type batchPageSpec struct {
	Properties map[string]interface{} `json:"properties"`
	Children   []interface{}          `json:"children,omitempty"`
	Icon       map[string]interface{} `json:"icon,omitempty"`
	Cover      map[string]interface{} `json:"cover,omitempty"`
}

type batchPageUpdateSpec struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Archived   *bool                  `json:"archived,omitempty"`
	InTrash    *bool                  `json:"in_trash,omitempty"`
	Icon       map[string]interface{} `json:"icon,omitempty"`
	Cover      map[string]interface{} `json:"cover,omitempty"`
}

func newPageCreateBatchCmd() *cobra.Command {
	var parentID string
	var parentType string
	var dataSourceID string
	var pagesJSON string
	var pagesFile string
	var continueOnError bool

	cmd := &cobra.Command{
		Use:     "create-batch",
		Aliases: []string{"cb"},
		Short:   "Create multiple pages in a single batch",
		Long: `Create multiple Notion pages from a JSON array.

The --pages flag accepts a JSON array of page objects. Each object should include
"properties" and may include "children", "icon", or "cover".
Use --file to read the JSON array from a file instead of passing it inline.

Example:
  ntn page create-batch --parent <id> --pages '[{"properties":{"Name":{"title":[{"text":{"content":"One"}}]}}},{"properties":{"Name":{"title":[{"text":{"content":"Two"}}]}}}]'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			if parentID == "" && dataSourceID == "" {
				return fmt.Errorf("--parent flag is required (or use --datasource)")
			}
			if pagesJSON == "" && pagesFile == "" {
				return fmt.Errorf("--pages or --file is required")
			}
			if pagesJSON != "" && pagesFile != "" {
				return fmt.Errorf("use only one of --pages or --file")
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

			if pagesFile != "" {
				data, err := os.ReadFile(pagesFile)
				if err != nil {
					return fmt.Errorf("failed to read pages file: %w", err)
				}
				pagesJSON = string(data)
			}
			if pagesJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(pagesJSON)
				if err != nil {
					return err
				}
				pagesJSON = resolved
			}

			var specs []batchPageSpec
			if err := cmdutil.UnmarshalJSONInput(pagesJSON, &specs); err != nil {
				return fmt.Errorf("failed to parse pages JSON: %w", err)
			}
			if len(specs) == 0 {
				return fmt.Errorf("no pages provided")
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}
			parent, err := resolvePageParent(ctx, client, parentID, parentType, dataSourceID)
			if err != nil {
				return err
			}

			var created []*notion.Page
			var errors []map[string]interface{}

			for i, spec := range specs {
				if spec.Properties == nil {
					err := fmt.Errorf("page %d: properties are required", i)
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return err
				}

				req := &notion.CreatePageRequest{
					Parent:     parent,
					Properties: spec.Properties,
					Children:   spec.Children,
					Icon:       spec.Icon,
					Cover:      spec.Cover,
				}

				page, err := client.CreatePage(ctx, req)
				if err != nil {
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return fmt.Errorf("failed to create page %d: %w", i, err)
				}

				created = append(created, page)
			}

			printer := printerForContext(ctx)
			if continueOnError && len(errors) > 0 {
				return printer.Print(ctx, map[string]interface{}{
					"pages":  created,
					"errors": errors,
				})
			}
			return printer.Print(ctx, created)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'datasource'")
	cmd.Flags().StringVar(&dataSourceID, "datasource", "", "Data source ID (optional, overrides --parent-type database)")
	cmd.Flags().StringVar(&pagesJSON, "pages", "", "Pages as JSON array")
	cmd.Flags().StringVar(&pagesFile, "file", "", "Read pages JSON array from file")
	cmd.Flags().BoolVar(&continueOnError, "continue-on-error", false, "Continue creating pages even if one fails")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "datasource", "ds")

	return cmd
}

func newPageUpdateBatchCmd() *cobra.Command {
	var pagesJSON string
	var pagesFile string
	var continueOnError bool

	cmd := &cobra.Command{
		Use:     "update-batch",
		Aliases: []string{"ub"},
		Short:   "Update multiple pages in a single batch",
		Long: `Update multiple Notion pages from a JSON array.

The --pages flag accepts a JSON array of page update objects. Each object must include
"id" and may include "properties", "archived", "in_trash", "icon", or "cover".
Use --file to read the JSON array from a file instead of passing it inline.

Example:
  ntn page update-batch --pages '[{"id":"<page-id>","properties":{"Status":{"status":{"name":"Done"}}}}]'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pagesJSON == "" && pagesFile == "" {
				return fmt.Errorf("--pages or --file is required")
			}
			if pagesJSON != "" && pagesFile != "" {
				return fmt.Errorf("use only one of --pages or --file")
			}

			if pagesFile != "" {
				data, err := os.ReadFile(pagesFile)
				if err != nil {
					return fmt.Errorf("failed to read pages file: %w", err)
				}
				pagesJSON = string(data)
			}
			if pagesJSON != "" {
				resolved, err := cmdutil.ReadJSONInput(pagesJSON)
				if err != nil {
					return err
				}
				pagesJSON = resolved
			}

			var specs []batchPageUpdateSpec
			if err := cmdutil.UnmarshalJSONInput(pagesJSON, &specs); err != nil {
				return fmt.Errorf("failed to parse pages JSON: %w", err)
			}
			if len(specs) == 0 {
				return fmt.Errorf("no pages provided")
			}

			ctx := cmd.Context()
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			var updated []*notion.Page
			var errors []map[string]interface{}

			for i, spec := range specs {
				if spec.ID == "" {
					err := fmt.Errorf("page %d: id is required", i)
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return err
				}

				normalizedID, err := cmdutil.NormalizeNotionID(spec.ID)
				if err != nil {
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return err
				}

				if spec.Properties == nil && spec.Archived == nil && spec.InTrash == nil && spec.Icon == nil && spec.Cover == nil {
					err := fmt.Errorf("page %d: no update fields provided", i)
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return err
				}

				req := &notion.UpdatePageRequest{
					Properties: spec.Properties,
					Archived:   spec.Archived,
					InTrash:    spec.InTrash,
					Icon:       spec.Icon,
					Cover:      spec.Cover,
				}

				page, err := client.UpdatePage(ctx, normalizedID, req)
				if err != nil {
					enhanced := notion.EnhanceStatusError(ctx, client, normalizedID, err)
					err = fmt.Errorf("failed to update page %d (%s): %w", i, normalizedID, enhanced)
					if continueOnError {
						errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
						continue
					}
					return err
				}

				updated = append(updated, page)
			}

			printer := printerForContext(ctx)
			if continueOnError && len(errors) > 0 {
				return printer.Print(ctx, map[string]interface{}{
					"pages":  updated,
					"errors": errors,
				})
			}
			return printer.Print(ctx, updated)
		},
	}

	cmd.Flags().StringVar(&pagesJSON, "pages", "", "Pages as JSON array")
	cmd.Flags().StringVar(&pagesFile, "file", "", "Read pages JSON array from file")
	cmd.Flags().BoolVar(&continueOnError, "continue-on-error", false, "Continue updating pages even if one fails")

	return cmd
}
