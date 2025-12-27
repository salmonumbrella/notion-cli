package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "page",
		Aliases: []string{"pages", "p"},
		Short:   "Manage Notion pages",
		Long:    `Create, retrieve, and update Notion pages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// When invoked without subcommand, default to search --filter page
			searchCmd := newSearchCmd()
			searchCmd.SetContext(cmd.Context())
			// Set the filter flag to "page" to only show pages
			if err := searchCmd.Flags().Set("filter", "page"); err != nil {
				return err
			}
			return searchCmd.RunE(searchCmd, args)
		},
	}

	// Desire-path alias: agents often try `notion page list ...`
	cmd.AddCommand(newPageListCmd())
	cmd.AddCommand(newPageGetCmd())
	cmd.AddCommand(newPagePropertiesCmd())
	cmd.AddCommand(newPageCreateCmd())
	cmd.AddCommand(newPageUpdateCmd())
	cmd.AddCommand(newPageCreateBatchCmd())
	cmd.AddCommand(newPageUpdateBatchCmd())
	cmd.AddCommand(newPageDuplicateCmd())
	cmd.AddCommand(newPageExportCmd())
	cmd.AddCommand(newPagePropertyCmd())
	cmd.AddCommand(newPageMoveCmd())
	cmd.AddCommand(newPageDeleteCmd())
	cmd.AddCommand(newPageSyncCmd())

	return cmd
}

func newPageListCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "list [query]",
		Aliases: []string{"ls"},
		Short:   "List pages (alias for 'search --filter page')",
		Long: `Search for pages in Notion.

This is a convenience alias for 'ntn search --filter page'.
Use --light (or --li) for compact output (id, object, title, url).

Example:
  ntn page list
  ntn page list --li
  ntn page list "project"
  ntn page ls meetings`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchCmd := newSearchCmd()
			searchCmd.SetContext(cmd.Context())
			if err := searchCmd.Flags().Set("filter", "page"); err != nil {
				return err
			}
			if light {
				if err := searchCmd.Flags().Set("light", "true"); err != nil {
					return err
				}
			}
			return searchCmd.RunE(searchCmd, args)
		},
	}

	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, object, title, url)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func resolveAndNormalizePageID(ctx context.Context, client *notion.Client, sf *skill.SkillFile, input string) (string, error) {
	pageID, err := resolveIDWithSearch(ctx, client, sf, input, "page")
	if err != nil {
		return "", err
	}
	return cmdutil.NormalizeNotionID(pageID)
}

func newPageDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <page-id-or-name>",
		Aliases: []string{"rm", "d"},
		Short:   "Archive a page",
		Long: `Archive a Notion page by its ID or name.

This command archives (soft-deletes) a page. Archived pages can be restored
from the Notion UI.

If you provide a name instead of an ID, the CLI will search for matching pages.

Example:
  notion page delete abc123
  notion page delete "Old Meeting Notes"
  notion page archive abc123
  notion page rm abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			pageID, err := resolveAndNormalizePageID(ctx, client, sf, args[0])
			if err != nil {
				return err
			}

			page, err := client.UpdatePage(ctx, pageID, &notion.UpdatePageRequest{
				Archived: ptrBool(true),
			})
			if err != nil {
				return wrapAPIError(err, "archive page", "page", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}
}

func newPageGetCmd() *cobra.Command {
	var editableOnly bool
	var enrich bool
	var includeChildren bool
	var childrenDepth int
	var light bool

	cmd := &cobra.Command{
		Use:     "get <page-id-or-name>",
		Aliases: []string{"g"},
		Short:   "Get a page by ID or name",
		Long: `Retrieve a Notion page by its ID or name.

If you provide a name instead of an ID, the CLI will search for matching pages
and return the one that matches. If multiple pages match, you'll see suggestions.

Use --editable to filter out read-only computed properties (formula, rollup,
created_by, created_time, last_edited_by, last_edited_time, unique_id, button).
This is useful when you want to copy properties to create or update a page.

Use --enrich to include additional metadata:
  - parent_title: the resolved title of the parent database or page
  - child_count: the number of immediate child blocks
Note: --enrich requires 1-2 extra API calls per page (1 for parent title, 1 for child count).

Use --include-children to include page body blocks in the response.
Use --children-depth to control recursive block fetching depth (1 = direct children only).
Use --light (or --li) for compact lookup output (id, object, title, url).
Use --light to emit compact JSON output for fast lookups.

Example:
  notion page get 12345678-1234-1234-1234-123456789012
  notion page get "Meeting Notes"
  notion page get 12345678-1234-1234-1234-123456789012 --li -j
  notion page get 12345678-1234-1234-1234-123456789012 --editable -o json
  notion page get 12345678-1234-1234-1234-123456789012 --enrich
  notion page get 12345678-1234-1234-1234-123456789012 --include-children --children-depth 2 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			pageID, err := resolveAndNormalizePageID(ctx, client, sf, args[0])
			if err != nil {
				return err
			}

			if light {
				if editableOnly {
					return fmt.Errorf("--light cannot be combined with --editable")
				}
				if enrich {
					return fmt.Errorf("--light cannot be combined with --enrich")
				}
				if includeChildren {
					return fmt.Errorf("--light cannot be combined with --include-children")
				}
			}

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return wrapAPIError(err, "get page", "page", args[0])
			}

			if light {
				printer := printerForContext(ctx)
				return printer.Print(ctx, toLightPage(page))
			}

			// Filter out read-only properties if requested
			if editableOnly {
				page.Properties = filterEditableProperties(page.Properties)
			}

			if includeChildren && childrenDepth < 1 {
				return fmt.Errorf("--children-depth must be >= 1")
			}

			var result interface{} = page
			if enrich {
				result = enrichPage(ctx, client, page)
			}

			if includeChildren {
				var children []notion.Block
				if childrenDepth == 1 {
					children, err = fetchAllBlockChildren(ctx, client, pageID)
				} else {
					children, err = client.GetBlockChildrenRecursive(ctx, pageID, childrenDepth, nil)
				}
				if err != nil {
					return wrapAPIError(err, "access block", "block", args[0])
				}

				resultWithChildren, err := withChildren(result, children)
				if err != nil {
					return err
				}
				result = resultWithChildren
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, result)
		},
	}

	cmd.Flags().BoolVar(&editableOnly, "editable", false, "Filter out read-only computed properties")
	cmd.Flags().BoolVar(&enrich, "enrich", false, "Include parent title and child count (extra API calls)")
	cmd.Flags().BoolVar(&includeChildren, "include-children", false, "Include page body blocks in output")
	cmd.Flags().IntVar(&childrenDepth, "children-depth", 1, "Depth for --include-children (1 = direct children only)")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact payload (id, object, title, url)")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func withChildren(data interface{}, children []notion.Block) (map[string]interface{}, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode page output: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode page output: %w", err)
	}

	payload["children"] = children
	return payload, nil
}

func newPagePropertiesCmd() *cobra.Command {
	var typesCSV string
	var onlySet bool
	var includeValues bool
	var simple bool

	cmd := &cobra.Command{
		Use:     "properties <page-id-or-name>",
		Aliases: []string{"props"},
		Short:   "List page properties",
		Long: `List page properties with optional filtering.

If you provide a name instead of an ID, the CLI will search for matching pages.

Examples:
  notion page properties 12345678-1234-1234-1234-123456789012
  notion page properties "Meeting Notes"
  notion page properties 12345678-1234-1234-1234-123456789012 --types title,select --only-set`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			pageID, err := resolveAndNormalizePageID(ctx, client, sf, args[0])
			if err != nil {
				return err
			}

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return wrapAPIError(err, "get page", "page", args[0])
			}

			typeFilter := make(map[string]bool)
			if strings.TrimSpace(typesCSV) != "" {
				for _, part := range strings.Split(typesCSV, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						typeFilter[part] = true
					}
				}
			}

			rows := make([]map[string]interface{}, 0, len(page.Properties))
			for name, raw := range page.Properties {
				prop, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				propType, _ := prop["type"].(string)
				if propType == "" {
					continue
				}
				if len(typeFilter) > 0 && !typeFilter[propType] {
					continue
				}

				value := prop[propType]
				if onlySet && !propertyHasValue(value) {
					continue
				}

				entry := map[string]interface{}{
					"name": name,
					"type": propType,
				}
				if includeValues {
					entry["value"] = value
				}
				if simple {
					entry["simple"] = simplifyPropertyValue(propType, value)
				}
				rows = append(rows, entry)
			}

			sort.Slice(rows, func(i, j int) bool {
				return rows[i]["name"].(string) < rows[j]["name"].(string)
			})

			printer := printerForContext(ctx)
			return printer.Print(ctx, rows)
		},
	}

	cmd.Flags().StringVar(&typesCSV, "types", "", "Comma-separated property types to include (e.g., title,select)")
	cmd.Flags().BoolVar(&onlySet, "only-set", false, "Include only properties with a value")
	cmd.Flags().BoolVar(&includeValues, "with-values", false, "Include property values in output")
	cmd.Flags().BoolVar(&simple, "simple", false, "Include a best-effort simplified value (agent-friendly)")

	return cmd
}

// filterEditableProperties removes read-only computed property types from properties map.
func filterEditableProperties(properties map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})
	for name, val := range properties {
		prop, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		propType, _ := prop["type"].(string)
		if propType == "" || readOnlyPropertyTypes[propType] {
			continue
		}
		filtered[name] = val
	}
	return filtered
}

func newPageCreateCmd() *cobra.Command {
	var parentID string
	var parentType string
	var propertiesJSON string
	var propertiesFile string
	var dataSourceID string
	var statusFlag string
	var priorityFlag string
	var assigneeFlag string
	var titleFlag string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a new page",
		Long: `Create a new Notion page with the specified parent and properties.

The --properties flag accepts a JSON string with the page properties.
You can also pass @file or - (stdin) to avoid shell-escaping JSON.
The simplest format for a basic page is:
  {"title": [{"text": {"content": "Page Title"}}]}

The --parent-type flag specifies whether the parent is a "page" (default), "database", or "datasource".
Use --datasource to target a specific data source (overrides --parent-type database).

PROPERTY SHORTHAND FLAGS:
For common properties, you can use shorthand flags instead of JSON:
  --title <value>     Set the page title (finds title property by type)
  --status <value>    Set Status property (status type)
  --priority <value>  Set Priority property (select type)
  --assignee <value>  Set Assignee property (people type, accepts user ID or alias)

These flags merge with --properties; shorthand flags take precedence if both specify the same property.

Examples:
  # Create page under another page
  notion page create --parent 12345678-1234-1234-1234-123456789012 \
    --title "My New Page"

  # Create page under a database with shorthand flags
  notion page create --parent 87654321-4321-4321-4321-210987654321 \
    --parent-type database \
    --title "Task Name" \
    --status "In Progress" --priority High --assignee user-id-or-alias`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Validate required flags
			if parentID == "" && dataSourceID == "" {
				return fmt.Errorf("--parent flag is required (or use --datasource)")
			}
			// Allow --title to be used without --properties
			hasShorthandFlags := titleFlag != "" || statusFlag != "" || priorityFlag != "" || assigneeFlag != ""
			if propertiesJSON == "" && propertiesFile == "" && !hasShorthandFlags {
				return fmt.Errorf("--properties, --properties-file, or shorthand flags (--title, --status, etc.) required")
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

			var properties map[string]interface{}
			if hasJSONInput(propertiesJSON, propertiesFile) {
				parsed, resolved, err := resolveAndDecodeJSON[map[string]interface{}](propertiesJSON, propertiesFile, "failed to parse properties JSON")
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				properties = parsed
			}

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Merge shorthand flags into properties (flags take precedence)
			properties = buildPropertiesFromFlags(sf, properties, statusFlag, priorityFlag, assigneeFlag)

			// Handle --title flag: find title property name based on parent type
			if titleFlag != "" {
				titlePropName, err := resolveTitlePropertyNameForPageCreate(ctx, client, parentID, parentType, dataSourceID)
				if err != nil {
					return err
				}
				properties = setTitleProperty(properties, titlePropName, titleFlag)
			}

			parent, err := resolvePageParent(ctx, client, parentID, parentType, dataSourceID)
			if err != nil {
				return err
			}

			// Build request
			req := &notion.CreatePageRequest{
				Parent:     parent,
				Properties: properties,
			}

			// Create page
			page, err := client.CreatePage(ctx, req)
			if err != nil {
				return wrapAPIError(err, "create page", "parent", parentID)
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'datasource'")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().StringVar(&dataSourceID, "datasource", "", "Data source ID (optional, overrides --parent-type database)")
	cmd.Flags().StringVar(&titleFlag, "title", "", "Set page title (finds title property by type)")
	cmd.Flags().StringVar(&statusFlag, "status", "", "Set Status property value")
	cmd.Flags().StringVar(&priorityFlag, "priority", "", "Set Priority property value")
	cmd.Flags().StringVar(&assigneeFlag, "assignee", "", "Set Assignee property (user ID or alias)")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")
	flagAlias(cmd.Flags(), "datasource", "ds")
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")

	return cmd
}

func newPageUpdateCmd() *cobra.Command {
	var propertiesJSON string
	var propertiesFile string
	var archived bool
	var setArchived bool
	var dryRun bool
	var mentions []string
	var verbose bool
	var richText bool
	var statusFlag string
	var priorityFlag string
	var assigneeFlag string
	var titleFlag string

	cmd := &cobra.Command{
		Use:     "update <page-id-or-name>",
		Aliases: []string{"u"},
		Short:   "Update a page",
		Long: `Update a Notion page's properties.

If you provide a name instead of an ID, the CLI will search for matching pages.

The --properties flag accepts a JSON string with the properties to update.
Only the properties specified will be updated; others remain unchanged.
You can also pass @file or - (stdin) to avoid shell-escaping JSON.

PROPERTY SHORTHAND FLAGS:
For common properties, you can use shorthand flags instead of JSON:
  --title <value>     Set the page title (finds title property by type)
  --status <value>    Set Status property (status type)
  --priority <value>  Set Priority property (select type)
  --assignee <value>  Set Assignee property (people type, accepts user ID or alias)

These flags merge with --properties; shorthand flags take precedence if both specify the same property.

Example - Using shorthand flags:
  notion page update PAGE_ID --title "New Title"
  notion page update PAGE_ID --status Done
  notion page update PAGE_ID --status "In Progress" --priority High
  notion page update PAGE_ID --assignee user-id-or-alias

RICH TEXT WITH MARKDOWN:
For rich_text properties, you can use a string shorthand with markdown formatting:
  **bold**, *italic*, ` + "`code`" + `, ***bold italic***

Use --rich-text to enable markdown parsing for string values:
  notion page update PAGE_ID \
    --properties '{"Notes": "This is **bold** text"}' \
    --rich-text

RICH TEXT WITH MENTIONS:
For rich_text properties, you can also use @Name patterns for mentions:
  {"Summary": "@Reviewer should film this"}

When combined with --mention flags, @Name patterns are replaced with proper
mention objects that notify users. Mentions are matched to user IDs in order,
with properties processed alphabetically by name.

Note: Using --mention automatically enables markdown parsing (--rich-text is
implied when --mention is provided).

Example - Simple update (no transformation):
  notion page update 12345678-1234-1234-1234-123456789012 \
    --properties '{"title": [{"text": {"content": "Updated Title"}}]}'

Example - Rich text with markdown only:
  notion page update PAGE_ID \
    --properties '{"Notes": "This is **bold** and *italic*"}' \
    --rich-text

Example - Rich text with mention:
  notion page update PAGE_ID \
    --properties '{"Summary": "@Reviewer should film this"}' \
    --mention 1a907419-f65d-46cd-8c74-7e597605b832

Example - Multiple mentions:
  notion page update PAGE_ID \
    --properties '{"Notes": "@Alice and @Bob please review"}' \
    --mention alice-user-id --mention bob-user-id

Use --verbose to see how markdown is parsed and mentions are matched:
  notion page update PAGE_ID \
    --properties '{"Summary": "@Reviewer should **review**"}' \
    --mention user-id --verbose

Combined example (all flags together):
  notion page update PAGE_ID \
    --properties '{"Summary": "@Alice please **review** this ` + "`" + `code` + "`" + ` change", "Notes": "Additional context here"}' \
    --mention alice-user-id \
    --rich-text \
    --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			pageID, err := resolveAndNormalizePageID(ctx, client, sf, args[0])
			if err != nil {
				return err
			}

			var properties map[string]interface{}
			if hasJSONInput(propertiesJSON, propertiesFile) {
				parsed, resolved, err := resolveAndDecodeJSON[map[string]interface{}](propertiesJSON, propertiesFile, "failed to parse properties JSON")
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				properties = parsed
			}

			// Merge shorthand flags into properties (flags take precedence)
			properties = buildPropertiesFromFlags(sf, properties, statusFlag, priorityFlag, assigneeFlag)

			// Handle --title flag: find title property name from page's current properties
			if titleFlag != "" {
				page, err := client.GetPage(ctx, pageID)
				if err != nil {
					return wrapAPIError(err, "get page for title property lookup", "page", args[0])
				}
				titlePropName := findTitlePropertyNameFromPage(page.Properties)
				properties = setTitleProperty(properties, titlePropName, titleFlag)
			}

			// Transform string shorthand values to rich_text arrays with mentions
			// Applies when --mention flags are provided OR --rich-text flag is set
			if (len(mentions) > 0 || richText) && properties != nil {
				properties, _ = transformPropertiesWithMentionsVerbose(stderrFromContext(ctx), properties, mentions, verbose, len(mentions) > 0)
			}

			if dryRun {
				// Fetch current page to show what would be updated
				currentPage, err := client.GetPage(ctx, pageID)
				if err != nil {
					return wrapAPIError(err, "get page", "page", args[0])
				}

				printer := NewDryRunPrinter(stderrFromContext(ctx))
				printer.Header("update", "page", pageID)

				// Show archived status change if applicable
				if setArchived {
					if archived != currentPage.Archived {
						printer.Change("Archived", fmt.Sprintf("%t", currentPage.Archived), fmt.Sprintf("%t", archived))
					} else {
						printer.Unchanged("Archived")
					}
				}

				// Show properties to update
				if propertiesJSON != "" {
					printer.Section("Properties to update:")
					for propName := range properties {
						_, _ = fmt.Fprintf(stderrFromContext(ctx), "  - %s\n", propName)
					}

					// Show current values for properties being updated
					printer.Section("\nCurrent property values:")
					for propName := range properties {
						if currentVal, ok := currentPage.Properties[propName]; ok {
							currentBytes, _ := json.Marshal(currentVal)
							_, _ = fmt.Fprintf(stderrFromContext(ctx), "  %s: %s\n", propName, string(currentBytes))
						} else {
							_, _ = fmt.Fprintf(stderrFromContext(ctx), "  %s: (not set)\n", propName)
						}
					}

					printer.Section("New property values:")
					for propName, propVal := range properties {
						newBytes, _ := json.Marshal(propVal)
						_, _ = fmt.Fprintf(stderrFromContext(ctx), "  %s: %s\n", propName, string(newBytes))
					}
				}

				printer.Footer()
				return nil
			}

			// Build request
			req := &notion.UpdatePageRequest{
				Properties: properties,
			}

			// Set archived flag if specified
			if setArchived {
				req.Archived = &archived
			}

			// Update page
			page, err := client.UpdatePage(ctx, pageID, req)
			if err != nil {
				enhanced := notion.EnhanceStatusError(ctx, client, pageID, err)
				return wrapAPIError(enhanced, "update page", "page", args[0])
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the page")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "User ID(s) to @-mention in rich_text properties (repeatable)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show parsed markdown details for property transformations")
	cmd.Flags().BoolVar(&richText, "rich-text", false, "Enable markdown parsing for string values without requiring --mention")
	cmd.Flags().StringVar(&titleFlag, "title", "", "Set page title (finds title property by type)")
	cmd.Flags().StringVar(&statusFlag, "status", "", "Set Status property value")
	cmd.Flags().StringVar(&priorityFlag, "priority", "", "Set Priority property value")
	cmd.Flags().StringVar(&assigneeFlag, "assignee", "", "Set Assignee property (user ID or alias)")
	// Track if archived flag was explicitly set
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		setArchived = cmd.Flags().Changed("archived")
		return nil
	}

	// Flag aliases
	flagAlias(cmd.Flags(), "properties", "props")
	flagAlias(cmd.Flags(), "properties-file", "props-file")
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}

func newPagePropertyCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "property <page-id> <property-id>",
		Aliases: []string{"prop"},
		Short:   "Get a specific page property",
		Long: `Retrieve a specific property value from a page.

This is useful for retrieving paginated properties or getting
just a specific property without the entire page.

Example:
  notion page property 12345678-1234-1234-1234-123456789012 title`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			pageID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}
			propertyID := args[1]

			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			// Get property
			property, err := client.GetPageProperty(ctx, pageID, propertyID)
			if err != nil {
				return wrapAPIError(err, "get page property", "page", args[0])
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, property)
		},
	}
}

func newPageMoveCmd() *cobra.Command {
	var parentID string
	var parentType string
	var after string

	cmd := &cobra.Command{
		Use:     "move <page-id>",
		Aliases: []string{"mv"},
		Short:   "Move a page to a new parent",
		Long: `Move a page to a different parent page or database.

Example - Move page under another page:
  notion page move abc123 --parent def456

Example - Move page to database:
  notion page move abc123 --parent db789 --parent-type database`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			pageID, err := cmdutil.NormalizeNotionID(resolveID(sf, args[0]))
			if err != nil {
				return err
			}

			if parentID == "" {
				return fmt.Errorf("--parent is required")
			}

			normalizedParent, err := cmdutil.NormalizeNotionID(resolveID(sf, parentID))
			if err != nil {
				return err
			}
			parentID = normalizedParent

			if after != "" {
				normalizedAfter, err := cmdutil.NormalizeNotionID(resolveID(sf, after))
				if err != nil {
					return err
				}
				after = normalizedAfter
			}

			var parentKey string
			switch parentType {
			case "page":
				parentKey = "page_id"
			case "database":
				parentKey = "database_id"
			default:
				return fmt.Errorf("invalid --parent-type: %s (expected 'page' or 'database')", parentType)
			}

			// Get token from context (respects workspace selection)
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			req := &notion.MovePageRequest{
				Parent: map[string]interface{}{parentKey: parentID},
				After:  after,
			}

			page, err := client.MovePage(ctx, pageID, req)
			if err != nil {
				return wrapAPIError(err, "move page", "page", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "New parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type: 'page' or 'database'")
	cmd.Flags().StringVar(&after, "after", "", "Block ID to position after")

	// Flag aliases
	flagAlias(cmd.Flags(), "parent", "pa")

	return cmd
}
