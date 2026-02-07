package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/richtext"
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

	return cmd
}

func newPageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [query]",
		Aliases: []string{"ls"},
		Short:   "List pages (alias for 'search --filter page')",
		Long: `Search for pages in Notion.

This is a convenience alias for 'notion search --filter page'.

Example:
  notion page list
  notion page list "project"
  notion page ls meetings`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchCmd := newSearchCmd()
			searchCmd.SetContext(cmd.Context())
			if err := searchCmd.Flags().Set("filter", "page"); err != nil {
				return err
			}
			return searchCmd.RunE(searchCmd, args)
		},
	}
}

func newPageDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <page-id-or-name>",
		Aliases: []string{"archive", "rm"},
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

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			// Resolve ID with search fallback
			pageID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			pageID, err = cmdutil.NormalizeNotionID(pageID)
			if err != nil {
				return err
			}

			page, err := client.UpdatePage(ctx, pageID, &notion.UpdatePageRequest{
				Archived: ptrBool(true),
			})
			if err != nil {
				return errors.APINotFoundError(err, "page", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}
}

// PathEntry represents one ancestor in a page's breadcrumb path.
type PathEntry struct {
	Type  string `json:"type"` // "page" or "database"
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// EnrichedPage wraps a Page with additional metadata resolved via extra API calls.
// parent_title is the resolved title of the parent database or page.
// child_count is the number of immediate child blocks.
// path is the breadcrumb path from root to immediate parent.
type EnrichedPage struct {
	*notion.Page
	ParentTitle string      `json:"parent_title,omitempty"`
	ChildCount  int         `json:"child_count"`
	Path        []PathEntry `json:"path,omitempty"`
}

// enrichPage fetches additional metadata for a page: parent title and child count.
// This requires extra API calls, so it's opt-in via the --enrich flag.
func enrichPage(ctx context.Context, client *notion.Client, page *notion.Page) *EnrichedPage {
	enriched := &EnrichedPage{Page: page}

	// Build breadcrumb path (root-first) and derive parent title from it.
	// The last entry in the path is the immediate parent, so we extract
	// ParentTitle from there instead of making a separate API call.
	if page.Parent != nil {
		enriched.Path = buildBreadcrumbPath(ctx, client, page.Parent)
		if len(enriched.Path) > 0 {
			enriched.ParentTitle = enriched.Path[len(enriched.Path)-1].Title
		}
	}

	// Get child count (immediate children only)
	children, err := client.GetBlockChildren(ctx, page.ID, nil)
	if err == nil && children != nil {
		enriched.ChildCount = len(children.Results)
	}

	return enriched
}

// buildBreadcrumbPath walks the parent chain and returns the path from root to
// the immediate parent. Stops at workspace root or after maxDepth hops to avoid
// runaway chains. Returns entries in root-first order.
func buildBreadcrumbPath(ctx context.Context, client *notion.Client, parent map[string]interface{}) []PathEntry {
	const maxDepth = 10
	var entries []PathEntry

	current := parent
	for i := 0; i < maxDepth && current != nil; i++ {
		if dbID, ok := current["database_id"].(string); ok && dbID != "" {
			db, err := client.GetDatabase(ctx, dbID)
			if err != nil {
				break
			}
			entries = append(entries, PathEntry{
				Type:  "database",
				ID:    dbID,
				Title: extractDatabaseTitle(*db),
			})
			break // Stop at database boundary to limit API calls
		}

		if pageID, ok := current["page_id"].(string); ok && pageID != "" {
			parentPage, err := client.GetPage(ctx, pageID)
			if err != nil {
				break
			}
			entries = append(entries, PathEntry{
				Type:  "page",
				ID:    pageID,
				Title: extractPageTitleFromProperties(parentPage.Properties),
			})
			current = parentPage.Parent
			continue
		}

		break // workspace or unknown parent type
	}

	// Reverse: we collected child-first, but want root-first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

// extractPageTitleFromProperties extracts the plain text title from a page's properties.
func extractPageTitleFromProperties(properties map[string]interface{}) string {
	if properties == nil {
		return ""
	}
	// Look for the title property (type: "title")
	for _, v := range properties {
		prop, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if prop["type"] == "title" {
			if titleArr, ok := prop["title"].([]interface{}); ok {
				return extractTitlePlainText(titleArr)
			}
		}
	}
	return ""
}

func newPageGetCmd() *cobra.Command {
	var editableOnly bool
	var enrich bool

	cmd := &cobra.Command{
		Use:   "get <page-id-or-name>",
		Short: "Get a page by ID or name",
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

Example:
  notion page get 12345678-1234-1234-1234-123456789012
  notion page get "Meeting Notes"
  notion page get 12345678-1234-1234-1234-123456789012 --editable -o json
  notion page get 12345678-1234-1234-1234-123456789012 --enrich`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sf := SkillFileFromContext(ctx)

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Resolve ID with search fallback
			pageID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			pageID, err = cmdutil.NormalizeNotionID(pageID)
			if err != nil {
				return err
			}

			// Get page
			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return errors.APINotFoundError(err, "page", args[0])
			}

			// Filter out read-only properties if requested
			if editableOnly {
				page.Properties = filterEditableProperties(page.Properties)
			}

			// Print result (enriched or plain)
			printer := printerForContext(ctx)
			if enrich {
				enrichedPage := enrichPage(ctx, client, page)
				return printer.Print(ctx, enrichedPage)
			}
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().BoolVar(&editableOnly, "editable", false, "Filter out read-only computed properties")
	cmd.Flags().BoolVar(&enrich, "enrich", false, "Include parent title and child count (extra API calls)")

	return cmd
}

func newPagePropertiesCmd() *cobra.Command {
	var typesCSV string
	var onlySet bool
	var includeValues bool
	var simple bool

	cmd := &cobra.Command{
		Use:   "properties <page-id-or-name>",
		Short: "List page properties",
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

			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			// Resolve ID with search fallback
			pageID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			pageID, err = cmdutil.NormalizeNotionID(pageID)
			if err != nil {
				return err
			}

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return errors.APINotFoundError(err, "page", args[0])
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

// propertyHasValue checks if a property has any value set. Note that false and 0
// intentionally return true because a checkbox set to unchecked or a number set
// to zero is still explicitly set (as opposed to being empty/unset).
func propertyHasValue(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
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
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Notion page with the specified parent and properties.

The --properties flag accepts a JSON string with the page properties.
You can also pass @file or - (stdin) to avoid shell-escaping JSON.
The simplest format for a basic page is:
  {"title": [{"text": {"content": "Page Title"}}]}

The --parent-type flag specifies whether the parent is a "page" (default), "database", or "data-source".
Use --data-source to target a specific data source (overrides --parent-type database).

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
				return fmt.Errorf("--parent flag is required (or use --data-source)")
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

			// Resolve and parse properties JSON (may be empty if only using shorthand flags)
			var properties map[string]interface{}
			if propertiesJSON != "" || propertiesFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Merge shorthand flags into properties (flags take precedence)
			properties = buildPropertiesFromFlags(sf, properties, statusFlag, priorityFlag, assigneeFlag)

			// Handle --title flag: find title property name based on parent type
			if titleFlag != "" {
				titlePropName := "title" // default for page parents
				// For database parents, look up the schema to find the title property name
				dbID := ""
				if dataSourceID != "" {
					dbID = dataSourceID
				} else if parentType == "database" || parentType == "data-source" {
					dbID = parentID
				}
				if dbID != "" {
					db, err := client.GetDatabase(ctx, dbID)
					if err != nil {
						return fmt.Errorf("failed to get database schema for title property: %w", err)
					}
					titlePropName = findTitlePropertyName(db.Properties)
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
				// Provide better error for parent not found
				return errors.APINotFoundError(err, "parent", parentID)
			}

			// Print result
			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'data-source'")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON (@file or - for stdin)")
	cmd.Flags().StringVar(&propertiesFile, "properties-file", "", "Read properties JSON from file (- for stdin)")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID (optional, overrides --parent-type database)")
	cmd.Flags().StringVar(&titleFlag, "title", "", "Set page title (finds title property by type)")
	cmd.Flags().StringVar(&statusFlag, "status", "", "Set Status property value")
	cmd.Flags().StringVar(&priorityFlag, "priority", "", "Set Priority property value")
	cmd.Flags().StringVar(&assigneeFlag, "assignee", "", "Set Assignee property (user ID or alias)")

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
		Use:   "update <page-id-or-name>",
		Short: "Update a page",
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
  {"Summary": "@Georges should film this"}

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
    --properties '{"Summary": "@Georges should film this"}' \
    --mention 1a907419-f65d-46cd-8c74-7e597605b832

Example - Multiple mentions:
  notion page update PAGE_ID \
    --properties '{"Notes": "@Alice and @Bob please review"}' \
    --mention alice-user-id --mention bob-user-id

Use --verbose to see how markdown is parsed and mentions are matched:
  notion page update PAGE_ID \
    --properties '{"Summary": "@Georges should **review**"}' \
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

			// Get token early so we can use client for search resolution
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}
			client := NewNotionClient(ctx, token)

			// Resolve ID with search fallback
			pageID, err := resolveIDWithSearch(ctx, client, sf, args[0], "page")
			if err != nil {
				return err
			}
			pageID, err = cmdutil.NormalizeNotionID(pageID)
			if err != nil {
				return err
			}

			// Resolve and parse properties JSON if provided
			var properties map[string]interface{}
			if propertiesJSON != "" || propertiesFile != "" {
				resolved, err := cmdutil.ResolveJSONInput(propertiesJSON, propertiesFile)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := cmdutil.UnmarshalJSONInput(propertiesJSON, &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Merge shorthand flags into properties (flags take precedence)
			properties = buildPropertiesFromFlags(sf, properties, statusFlag, priorityFlag, assigneeFlag)

			// Handle --title flag: find title property name from page's current properties
			if titleFlag != "" {
				page, err := client.GetPage(ctx, pageID)
				if err != nil {
					return fmt.Errorf("failed to get page for title property lookup: %w", err)
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
					return fmt.Errorf("failed to fetch page: %w", err)
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
				// Try to enhance status validation errors with valid options
				enhanced := notion.EnhanceStatusError(ctx, client, pageID, err)
				return fmt.Errorf("failed to update page: %w", enhanced)
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

	return cmd
}

// transformPropertiesWithMentions transforms string shorthand values in properties
// to rich_text arrays with mentions. Only string values are transformed; other
// property types (arrays, objects) are passed through unchanged.
// Properties are processed in alphabetical order by name for deterministic
// user ID assignment when multiple properties contain @Name patterns.
// Returns the transformed properties and the number of user IDs that were actually used.
func transformPropertiesWithMentions(properties map[string]interface{}, userIDs []string) (map[string]interface{}, int) {
	return transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, false, false)
}

// transformPropertiesWithMentionsVerbose is like transformPropertiesWithMentions but
// optionally prints verbose output about markdown parsing and mention matching.
// The w parameter specifies where verbose and warning output is written (typically os.Stderr in production).
// When emitWarnings is true, warnings about unused --mention flags are also written to w.
func transformPropertiesWithMentionsVerbose(w io.Writer, properties map[string]interface{}, userIDs []string, verbose bool, emitWarnings bool) (map[string]interface{}, int) {
	result := make(map[string]interface{}, len(properties))
	userIDIndex := 0

	// Sort property names for deterministic iteration order
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value := properties[name]
		// Only transform string values (shorthand for rich_text)
		if strVal, ok := value.(string); ok {
			// Parse markdown once - used for both verbose output and building rich text
			tokens := richtext.ParseMarkdown(strVal)

			if verbose {
				_, _ = fmt.Fprintf(w, "Property %q:\n", name)
				summary := richtext.SummarizeTokens(tokens)
				_, _ = fmt.Fprintf(w, "  %s\n", richtext.FormatSummary(summary))
			}

			// Count @Name patterns in this string to consume the right number of user IDs
			mentionsNeeded := richtext.CountMentions(strVal)

			// Allocate user IDs for this property
			var propertyUserIDs []string
			if userIDIndex < len(userIDs) {
				end := userIDIndex + mentionsNeeded
				if end > len(userIDs) {
					end = len(userIDs)
				}
				propertyUserIDs = userIDs[userIDIndex:end]
				userIDIndex = end
			}

			if verbose && mentionsNeeded > 0 {
				richtext.FormatMentionMappingsIndented(w, strVal, propertyUserIDs, "  ")
			}

			// Build rich text array from pre-parsed tokens (avoids redundant parsing)
			richTextContent := richtext.BuildWithMentionsFromTokens(tokens, propertyUserIDs, nil)

			// Convert to the format expected by Notion API
			richTextArray := make([]interface{}, len(richTextContent))
			for i, rt := range richTextContent {
				rtMap := map[string]interface{}{
					"type": rt.Type,
				}
				if rt.Text != nil {
					rtMap["text"] = map[string]interface{}{
						"content": rt.Text.Content,
					}
				}
				if rt.Mention != nil {
					mentionMap := map[string]interface{}{
						"type": rt.Mention.Type,
					}
					if rt.Mention.User != nil {
						mentionMap["user"] = map[string]interface{}{
							"id": rt.Mention.User.ID,
						}
					}
					rtMap["mention"] = mentionMap
				}
				if rt.Annotations != nil {
					rtMap["annotations"] = map[string]interface{}{
						"bold":          rt.Annotations.Bold,
						"italic":        rt.Annotations.Italic,
						"strikethrough": rt.Annotations.Strikethrough,
						"underline":     rt.Annotations.Underline,
						"code":          rt.Annotations.Code,
						"color":         rt.Annotations.Color,
					}
				}
				richTextArray[i] = rtMap
			}

			// Wrap in rich_text property structure
			result[name] = map[string]interface{}{
				"rich_text": richTextArray,
			}
		} else {
			// Pass through non-string values unchanged
			result[name] = value
		}
	}

	// Emit warnings about unused --mention flags if requested
	if emitWarnings && len(userIDs) > 0 {
		if userIDIndex == 0 {
			_, _ = fmt.Fprintf(w, "warning: %d --mention flag(s) provided but no @Name patterns found in property values\n", len(userIDs))
		} else if userIDIndex < len(userIDs) {
			_, _ = fmt.Fprintf(w, "warning: %d of %d --mention flag(s) unused (not enough @Name patterns)\n", len(userIDs)-userIDIndex, len(userIDs))
		}
	}

	return result, userIDIndex
}

// buildPropertiesFromFlags merges shorthand property flags into a properties map.
// The shorthand flags take precedence over properties already in the map.
// If properties is nil, a new map is created.
// Supports: --status (status property), --priority (select property), --assignee (people property).
func buildPropertiesFromFlags(sf *skill.SkillFile, properties map[string]interface{}, status, priority, assignee string) map[string]interface{} {
	if properties == nil {
		properties = make(map[string]interface{})
	}

	if status != "" {
		properties["Status"] = map[string]interface{}{
			"status": map[string]interface{}{
				"name": status,
			},
		}
	}

	if priority != "" {
		properties["Priority"] = map[string]interface{}{
			"select": map[string]interface{}{
				"name": priority,
			},
		}
	}

	if assignee != "" {
		// Resolve user ID from skill file or use as-is
		resolvedID := resolveUserID(sf, assignee)
		properties["Assignee"] = map[string]interface{}{
			"people": []map[string]interface{}{
				{"object": "user", "id": resolvedID},
			},
		}
	}

	return properties
}

// findTitlePropertyName finds the title property name in a database schema.
// Returns "title" as the default if no title property is found.
func findTitlePropertyName(properties map[string]map[string]interface{}) string {
	for propName, propDef := range properties {
		if propType, ok := propDef["type"].(string); ok && propType == "title" {
			return propName
		}
	}
	return "title"
}

// findTitlePropertyNameFromPage finds the title property name from a page's properties.
// Returns "title" as the default if no title property is found.
func findTitlePropertyNameFromPage(properties map[string]interface{}) string {
	for propName, propVal := range properties {
		prop, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		if prop["type"] == "title" {
			return propName
		}
	}
	return "title"
}

// setTitleProperty sets the title property in a properties map using the given property name.
// The title is formatted as a rich text array with a single text element.
func setTitleProperty(properties map[string]interface{}, propName, title string) map[string]interface{} {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties[propName] = map[string]interface{}{
		"title": []map[string]interface{}{
			{
				"text": map[string]interface{}{
					"content": title,
				},
			},
		},
	}
	return properties
}

func newPagePropertyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "property <page-id> <property-id>",
		Short: "Get a specific page property",
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

			// Get token from context (respects workspace selection)
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			// Create client
			client := NewNotionClient(ctx, token)

			// Get property
			property, err := client.GetPageProperty(ctx, pageID, propertyID)
			if err != nil {
				return fmt.Errorf("failed to get page property: %w", err)
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
		Use:   "move <page-id>",
		Short: "Move a page to a new parent",
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
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return errors.AuthRequiredError(err)
			}

			client := NewNotionClient(ctx, token)

			req := &notion.MovePageRequest{
				Parent: map[string]interface{}{parentKey: parentID},
				After:  after,
			}

			page, err := client.MovePage(ctx, pageID, req)
			if err != nil {
				return errors.APINotFoundError(err, "page", args[0])
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "New parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type: 'page' or 'database'")
	cmd.Flags().StringVar(&after, "after", "", "Block ID to position after")

	return cmd
}
