package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/richtext"
)

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Notion pages",
		Long:  `Create, retrieve, and update Notion pages.`,
	}

	cmd.AddCommand(newPageGetCmd())
	cmd.AddCommand(newPagePropertiesCmd())
	cmd.AddCommand(newPageCreateCmd())
	cmd.AddCommand(newPageUpdateCmd())
	cmd.AddCommand(newPageCreateBatchCmd())
	cmd.AddCommand(newPageDuplicateCmd())
	cmd.AddCommand(newPageExportCmd())
	cmd.AddCommand(newPagePropertyCmd())
	cmd.AddCommand(newPageMoveCmd())

	return cmd
}

func newPageGetCmd() *cobra.Command {
	var editableOnly bool

	cmd := &cobra.Command{
		Use:   "get <page-id>",
		Short: "Get a page by ID",
		Long: `Retrieve a Notion page by its ID.

Use --editable to filter out read-only computed properties (formula, rollup,
created_by, created_time, last_edited_by, last_edited_time, unique_id, button).
This is useful when you want to copy properties to create or update a page.

Example:
  notion page get 12345678-1234-1234-1234-123456789012
  notion page get 12345678-1234-1234-1234-123456789012 --editable -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get page
			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return fmt.Errorf("failed to get page: %w", err)
			}

			// Filter out read-only properties if requested
			if editableOnly {
				page.Properties = filterEditableProperties(page.Properties)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().BoolVar(&editableOnly, "editable", false, "Filter out read-only computed properties")

	return cmd
}

func newPagePropertiesCmd() *cobra.Command {
	var typesCSV string
	var onlySet bool
	var includeValues bool

	cmd := &cobra.Command{
		Use:   "properties <page-id>",
		Short: "List page properties",
		Long: `List page properties with optional filtering.

Examples:
  notion page properties 12345678-1234-1234-1234-123456789012
  notion page properties 12345678-1234-1234-1234-123456789012 --types title,select --only-set`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)

			page, err := client.GetPage(ctx, pageID)
			if err != nil {
				return fmt.Errorf("failed to get page: %w", err)
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
				rows = append(rows, entry)
			}

			sort.Slice(rows, func(i, j int) bool {
				return rows[i]["name"].(string) < rows[j]["name"].(string)
			})

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, rows)
		},
	}

	cmd.Flags().StringVar(&typesCSV, "types", "", "Comma-separated property types to include (e.g., title,select)")
	cmd.Flags().BoolVar(&onlySet, "only-set", false, "Include only properties with a value")
	cmd.Flags().BoolVar(&includeValues, "with-values", false, "Include property values in output")

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
	var dataSourceID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Notion page with the specified parent and properties.

The --properties flag accepts a JSON string with the page properties.
The simplest format for a basic page is:
  {"title": [{"text": {"content": "Page Title"}}]}

The --parent-type flag specifies whether the parent is a "page" (default), "database", or "data-source".
Use --data-source to target a specific data source (overrides --parent-type database).

Examples:
  # Create page under another page
  notion page create --parent 12345678-1234-1234-1234-123456789012 \
    --properties '{"title": [{"text": {"content": "My New Page"}}]}'

  # Create page under a database
  notion page create --parent 87654321-4321-4321-4321-210987654321 \
    --parent-type database \
    --properties '{"Name": {"title": [{"text": {"content": "Database Entry"}}]}}'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate required flags
			if parentID == "" && dataSourceID == "" {
				return fmt.Errorf("--parent flag is required (or use --data-source)")
			}
			if propertiesJSON == "" {
				return fmt.Errorf("--properties flag is required")
			}

			if parentID != "" {
				normalized, err := normalizeNotionID(parentID)
				if err != nil {
					return err
				}
				parentID = normalized
			}
			if dataSourceID != "" {
				normalized, err := normalizeNotionID(dataSourceID)
				if err != nil {
					return err
				}
				dataSourceID = normalized
			}

			// Resolve and parse properties JSON
			var properties map[string]interface{}
			resolved, err := readJSONInput(propertiesJSON)
			if err != nil {
				return err
			}
			propertiesJSON = resolved
			if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
				return fmt.Errorf("failed to parse properties JSON: %w", err)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

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
				return fmt.Errorf("failed to create page: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type of parent: 'page', 'database', or 'data-source'")
	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON (required)")
	cmd.Flags().StringVar(&dataSourceID, "data-source", "", "Data source ID (optional, overrides --parent-type database)")

	return cmd
}

func newPageUpdateCmd() *cobra.Command {
	var propertiesJSON string
	var archived bool
	var setArchived bool
	var dryRun bool
	var mentions []string
	var verbose bool
	var richText bool

	cmd := &cobra.Command{
		Use:   "update <page-id>",
		Short: "Update a page",
		Long: `Update a Notion page's properties.

The --properties flag accepts a JSON string with the properties to update.
Only the properties specified will be updated; others remain unchanged.

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
			pageID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Resolve and parse properties JSON if provided
			var properties map[string]interface{}
			if propertiesJSON != "" {
				resolved, err := readJSONInput(propertiesJSON)
				if err != nil {
					return err
				}
				propertiesJSON = resolved
				if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
					return fmt.Errorf("failed to parse properties JSON: %w", err)
				}
			}

			// Transform string shorthand values to rich_text arrays with mentions
			// Applies when --mention flags are provided OR --rich-text flag is set
			if (len(mentions) > 0 || richText) && properties != nil {
				properties, _ = transformPropertiesWithMentionsVerbose(os.Stderr, properties, mentions, verbose, len(mentions) > 0)
			}

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			if dryRun {
				// Fetch current page to show what would be updated
				currentPage, err := client.GetPage(ctx, pageID)
				if err != nil {
					return fmt.Errorf("failed to fetch page: %w", err)
				}

				printer := NewDryRunPrinter(os.Stderr)
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
						fmt.Fprintf(os.Stderr, "  - %s\n", propName)
					}

					// Show current values for properties being updated
					printer.Section("\nCurrent property values:")
					for propName := range properties {
						if currentVal, ok := currentPage.Properties[propName]; ok {
							currentBytes, _ := json.Marshal(currentVal)
							fmt.Fprintf(os.Stderr, "  %s: %s\n", propName, string(currentBytes))
						} else {
							fmt.Fprintf(os.Stderr, "  %s: (not set)\n", propName)
						}
					}

					printer.Section("New property values:")
					for propName, propVal := range properties {
						newBytes, _ := json.Marshal(propVal)
						fmt.Fprintf(os.Stderr, "  %s: %s\n", propName, string(newBytes))
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
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&propertiesJSON, "properties", "", "Page properties as JSON")
	cmd.Flags().BoolVar(&archived, "archived", false, "Archive the page")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "User ID(s) to @-mention in rich_text properties (repeatable)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show parsed markdown details for property transformations")
	cmd.Flags().BoolVar(&richText, "rich-text", false, "Enable markdown parsing for string values without requiring --mention")
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
				// Extract the actual @Name patterns for detailed output
				mentionMatches := richtext.FindMentions(strVal)
				_, _ = fmt.Fprintf(w, "  Mentions:\n")
				for i, mentionName := range mentionMatches {
					if i < len(propertyUserIDs) {
						_, _ = fmt.Fprintf(w, "    %s → %s\n", mentionName, propertyUserIDs[i])
					} else {
						_, _ = fmt.Fprintf(w, "    %s → (no user ID available)\n", mentionName)
					}
				}
			}

			// Build rich text array from pre-parsed tokens (avoids redundant parsing)
			richTextContent := richtext.BuildWithMentionsFromTokens(tokens, propertyUserIDs)

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
			pageID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}
			propertyID := args[1]

			// Get token from context (respects workspace selection)
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			// Create client
			client := NewNotionClient(token)

			// Get property
			property, err := client.GetPageProperty(ctx, pageID, propertyID)
			if err != nil {
				return fmt.Errorf("failed to get page property: %w", err)
			}

			// Print result
			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
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
			pageID, err := normalizeNotionID(args[0])
			if err != nil {
				return err
			}

			if parentID == "" {
				return fmt.Errorf("--parent is required")
			}

			normalizedParent, err := normalizeNotionID(parentID)
			if err != nil {
				return err
			}
			parentID = normalizedParent

			if after != "" {
				normalizedAfter, err := normalizeNotionID(after)
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
			ctx := cmd.Context()
			token, err := GetTokenFromContext(ctx)
			if err != nil {
				return fmt.Errorf("authentication required: %w\nRun 'notion auth login' or 'notion auth add-token' to configure", err)
			}

			client := NewNotionClient(token)

			req := &notion.MovePageRequest{
				Parent: map[string]interface{}{parentKey: parentID},
				After:  after,
			}

			page, err := client.MovePage(ctx, pageID, req)
			if err != nil {
				return fmt.Errorf("failed to move page: %w", err)
			}

			printer := output.NewPrinter(os.Stdout, GetOutputFormat())
			return printer.Print(ctx, page)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "New parent page or database ID (required)")
	cmd.Flags().StringVar(&parentType, "parent-type", "page", "Type: 'page' or 'database'")
	cmd.Flags().StringVar(&after, "after", "", "Block ID to position after")

	return cmd
}
