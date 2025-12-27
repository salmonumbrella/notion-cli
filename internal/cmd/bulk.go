package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func newBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations on database pages",
		Long: `Perform bulk operations (update, archive) on pages matching a filter condition.

Use --where to specify filter conditions on database properties.
Multiple --where flags are ANDed together.

Examples:
  ntn bulk update <db-id> --where "Status=Done" --set "Status=Archived"
  ntn bulk archive <db-id> --where "Status=Cancelled" --yes
  ntn bulk update <db-id> --where "Priority=High" --set "DRI=user-id" --dry-run`,
	}

	cmd.AddCommand(newBulkUpdateCmd())
	cmd.AddCommand(newBulkArchiveCmd())

	return cmd
}

func newBulkUpdateCmd() *cobra.Command {
	var whereClauses []string
	var setClauses []string
	var dryRun bool
	var limitFlag int

	cmd := &cobra.Command{
		Use:   "update <database-id-or-name>",
		Short: "Bulk update pages matching a condition",
		Long: `Update multiple pages in a database that match the given --where conditions.

The --where flag specifies filter conditions as PropertyName=Value pairs.
Multiple --where flags are ANDed together.
Property types are auto-detected from the database schema.

The --set flag specifies property updates as PropertyName=Value pairs.
Multiple --set flags update multiple properties.

Examples:
  ntn bulk update <db-id> --where "Status=Done" --set "Status=Archived"
  ntn bulk update <db-id> --where "Priority=High" --set "DRI=user-id" --set "Status=In Progress"
  ntn bulk update <db-id> --where "Status=Todo" --set "Status=In Progress" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(whereClauses) == 0 {
				return errors.NewUserError(
					"--where is required for bulk update",
					"Provide at least one --where flag to prevent accidental updates to all pages.",
				)
			}
			if len(setClauses) == 0 {
				return errors.NewUserError(
					"--set is required for bulk update",
					"Provide at least one --set flag specifying which properties to update.",
				)
			}

			ctx := cmd.Context()
			return runBulkOperation(ctx, args[0], whereClauses, setClauses, dryRun, limitFlag, "update")
		},
	}

	cmd.Flags().StringArrayVar(&whereClauses, "where", nil, "Filter condition as Property=Value (repeatable, ANDed)")
	cmd.Flags().StringArrayVar(&setClauses, "set", nil, "Property update as Property=Value (repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show matching pages without modifying")
	cmd.Flags().IntVar(&limitFlag, "limit", 0, "Cap number of pages affected (0 = no limit)")

	// Flag aliases
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}

func newBulkArchiveCmd() *cobra.Command {
	var whereClauses []string
	var dryRun bool
	var limitFlag int

	cmd := &cobra.Command{
		Use:   "archive <database-id-or-name>",
		Short: "Bulk archive pages matching a condition",
		Long: `Archive multiple pages in a database that match the given --where conditions.

The --where flag specifies filter conditions as PropertyName=Value pairs.
Multiple --where flags are ANDed together.
Property types are auto-detected from the database schema.

Examples:
  ntn bulk archive <db-id> --where "Status=Cancelled" --yes
  ntn bulk archive <db-id> --where "Status=Done" --where "Priority=Low" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(whereClauses) == 0 {
				return errors.NewUserError(
					"--where is required for bulk archive",
					"Provide at least one --where flag to prevent accidental archiving of all pages.",
				)
			}

			ctx := cmd.Context()
			return runBulkOperation(ctx, args[0], whereClauses, nil, dryRun, limitFlag, "archive")
		},
	}

	cmd.Flags().StringArrayVar(&whereClauses, "where", nil, "Filter condition as Property=Value (repeatable, ANDed)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show matching pages without modifying")
	cmd.Flags().IntVar(&limitFlag, "limit", 0, "Cap number of pages affected (0 = no limit)")

	// Flag aliases
	flagAlias(cmd.Flags(), "dry-run", "dr")

	return cmd
}

// runBulkOperation performs the core bulk update or archive flow.
func runBulkOperation(ctx context.Context, dbArg string, whereClauses, setClauses []string, dryRun bool, limit int, operation string) error {
	sf := SkillFileFromContext(ctx)
	stderr := stderrFromContext(ctx)

	client, err := clientFromContext(ctx)
	if err != nil {
		return err
	}

	// Resolve database ID with search fallback
	databaseID, err := resolveIDWithSearch(ctx, client, sf, dbArg, "database")
	if err != nil {
		return err
	}
	databaseID, err = cmdutil.NormalizeNotionID(databaseID)
	if err != nil {
		return err
	}

	// Resolve data source ID from the database
	resolvedDataSourceID, err := resolveDataSourceID(ctx, client, databaseID, "")
	if err != nil {
		return err
	}

	// Fetch data source schema for property type detection
	ds, err := client.GetDataSource(ctx, resolvedDataSourceID)
	if err != nil {
		return errors.WrapUserError(
			err,
			"failed to fetch data source schema",
			"Check the database ID is correct and your integration has access.",
		)
	}
	schema := ds.Properties

	// Parse --where clauses into Notion filter
	filter, err := parseWhereClauses(whereClauses, schema)
	if err != nil {
		return err
	}

	// Parse --set clauses into property update payload (for update operation only)
	var updateProps map[string]interface{}
	if operation == "update" {
		updateProps, err = parseSetClauses(setClauses, schema)
		if err != nil {
			return err
		}
	}

	// Query all matching pages (paginated)
	allPages, _, _, err := fetchAllPages(ctx, "", NotionMaxPageSize, 0, func(ctx context.Context, cursor string, pageSize int) ([]notion.Page, *string, bool, error) {
		req := &notion.QueryDataSourceRequest{
			Filter:      filter,
			StartCursor: cursor,
			PageSize:    pageSize,
		}
		result, err := client.QueryDataSource(ctx, resolvedDataSourceID, req)
		if err != nil {
			return nil, nil, false, err
		}
		return result.Results, result.NextCursor, result.HasMore, nil
	})
	if err != nil {
		return wrapAPIError(err, "query database for bulk operation", "database", dbArg)
	}

	// Apply limit if set
	if limit > 0 && len(allPages) > limit {
		allPages = allPages[:limit]
	}

	if len(allPages) == 0 {
		_, _ = fmt.Fprintf(stderr, "No pages matched the filter.\n")
		return nil
	}

	// Dry-run: show what would happen and exit
	if dryRun {
		_, _ = fmt.Fprintf(stderr, "[DRY-RUN] Found %d page(s) matching filter.\n", len(allPages))
		previewCount := len(allPages)
		if previewCount > 5 {
			previewCount = 5
		}
		for i := 0; i < previewCount; i++ {
			title := extractPageTitleFromProperties(allPages[i].Properties)
			if title == "" {
				title = "(untitled)"
			}
			_, _ = fmt.Fprintf(stderr, "  - %s (%s)\n", title, allPages[i].ID)
		}
		if len(allPages) > 5 {
			_, _ = fmt.Fprintf(stderr, "  ... and %d more\n", len(allPages)-5)
		}
		if operation == "update" {
			_, _ = fmt.Fprintf(stderr, "\nWould update properties:\n")
			for propName := range updateProps {
				_, _ = fmt.Fprintf(stderr, "  - %s\n", propName)
			}
		} else {
			_, _ = fmt.Fprintf(stderr, "\nWould archive %d page(s).\n", len(allPages))
		}
		_, _ = fmt.Fprintf(stderr, "\n[DRY-RUN] No changes made.\n")
		return nil
	}

	// Confirm unless --yes is set
	yes := output.YesFromContext(ctx)
	if !yes {
		_, _ = fmt.Fprintf(stderr, "Found %d page(s) matching filter.\n", len(allPages))
		previewCount := len(allPages)
		if previewCount > 5 {
			previewCount = 5
		}
		for i := 0; i < previewCount; i++ {
			title := extractPageTitleFromProperties(allPages[i].Properties)
			if title == "" {
				title = "(untitled)"
			}
			_, _ = fmt.Fprintf(stderr, "  - %s (%s)\n", title, allPages[i].ID)
		}
		if len(allPages) > 5 {
			_, _ = fmt.Fprintf(stderr, "  ... and %d more\n", len(allPages)-5)
		}

		// Check if stdin is a terminal (non-interactive requires --yes)
		if !isTerminal(os.Stdin) {
			return errors.NewUserError(
				"confirmation required but stdin is not a terminal",
				"Use --yes to skip confirmation in non-interactive mode.",
			)
		}

		_, _ = fmt.Fprintf(stderr, "Continue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		answer := strings.TrimSpace(strings.ToLower(line))
		if answer != "y" && answer != "yes" {
			_, _ = fmt.Fprintf(stderr, "Cancelled.\n")
			return nil
		}
	}

	// Execute the bulk operation
	start := time.Now()
	var updated int
	var operationErrors []map[string]interface{}

	for _, page := range allPages {
		var req *notion.UpdatePageRequest
		if operation == "update" {
			req = &notion.UpdatePageRequest{
				Properties: updateProps,
			}
		} else {
			// archive
			req = &notion.UpdatePageRequest{
				Archived: ptrBool(true),
			}
		}

		_, err := client.UpdatePage(ctx, page.ID, req)
		if err != nil {
			operationErrors = append(operationErrors, map[string]interface{}{
				"page_id": page.ID,
				"error":   err.Error(),
			})
			continue
		}
		updated++
	}

	elapsed := time.Since(start)
	verb := "Updated"
	if operation == "archive" {
		verb = "Archived"
	}
	_, _ = fmt.Fprintf(stderr, "%s %d page(s) in %s\n", verb, updated, formatDuration(elapsed))

	if len(operationErrors) > 0 {
		_, _ = fmt.Fprintf(stderr, "%d error(s) occurred:\n", len(operationErrors))
		for _, e := range operationErrors {
			_, _ = fmt.Fprintf(stderr, "  - %s: %s\n", e["page_id"], e["error"])
		}
	}

	// Print summary to stdout
	printer := printerForContext(ctx)
	return printer.Print(ctx, map[string]interface{}{
		"operation": operation,
		"matched":   len(allPages),
		"succeeded": updated,
		"failed":    len(operationErrors),
		"elapsed":   elapsed.String(),
	})
}

// parseWhereClause parses a single "PropertyName=Value" clause.
func parseWhereClause(clause string) (propName, value string, err error) {
	idx := strings.Index(clause, "=")
	if idx < 0 {
		return "", "", errors.NewUserError(
			fmt.Sprintf("invalid --where clause %q", clause),
			"Expected format: PropertyName=Value (e.g., Status=Done)",
		)
	}
	propName = strings.TrimSpace(clause[:idx])
	value = strings.TrimSpace(clause[idx+1:])
	if propName == "" {
		return "", "", errors.NewUserError(
			fmt.Sprintf("empty property name in --where clause %q", clause),
			"Expected format: PropertyName=Value (e.g., Status=Done)",
		)
	}
	return propName, value, nil
}

// parseWhereClauses parses --where flags into a Notion filter object.
func parseWhereClauses(clauses []string, schema map[string]interface{}) (map[string]interface{}, error) {
	var filters []map[string]interface{}

	for _, clause := range clauses {
		propName, value, err := parseWhereClause(clause)
		if err != nil {
			return nil, err
		}

		canonicalName, propType, err := resolveSchemaProperty(schema, propName)
		if err != nil {
			return nil, err
		}

		f, err := buildFilterForType(canonicalName, propType, value)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}

	if len(filters) == 1 {
		return filters[0], nil
	}

	// Multiple filters: AND them
	and := make([]interface{}, len(filters))
	for i, f := range filters {
		and[i] = f
	}
	return map[string]interface{}{"and": and}, nil
}

// buildFilterForType creates a Notion filter object for a given property type and value.
func buildFilterForType(propName, propType, value string) (map[string]interface{}, error) {
	switch propType {
	case "status":
		return map[string]interface{}{
			"property": propName,
			"status":   map[string]interface{}{"equals": value},
		}, nil
	case "select":
		return map[string]interface{}{
			"property": propName,
			"select":   map[string]interface{}{"equals": value},
		}, nil
	case "checkbox":
		boolVal := strings.EqualFold(value, "true")
		return map[string]interface{}{
			"property": propName,
			"checkbox": map[string]interface{}{"equals": boolVal},
		}, nil
	case "title":
		return map[string]interface{}{
			"property": propName,
			"title":    map[string]interface{}{"contains": value},
		}, nil
	case "rich_text":
		return map[string]interface{}{
			"property":  propName,
			"rich_text": map[string]interface{}{"contains": value},
		}, nil
	case "number":
		numVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, errors.NewUserError(
				fmt.Sprintf("cannot parse %q as a number for property %q", value, propName),
				"Provide a valid numeric value.",
			)
		}
		return map[string]interface{}{
			"property": propName,
			"number":   map[string]interface{}{"equals": numVal},
		}, nil
	default:
		return nil, errors.NewUserError(
			fmt.Sprintf("unsupported property type %q for --where on property %q", propType, propName),
			"Supported types: status, select, checkbox, title, rich_text, number.",
		)
	}
}

// parseSetClauses parses --set flags into a Notion property update payload.
func parseSetClauses(clauses []string, schema map[string]interface{}) (map[string]interface{}, error) {
	props := make(map[string]interface{})

	for _, clause := range clauses {
		idx := strings.Index(clause, "=")
		if idx < 0 {
			return nil, errors.NewUserError(
				fmt.Sprintf("invalid --set clause %q", clause),
				"Expected format: PropertyName=Value (e.g., Status=Done)",
			)
		}
		propName := strings.TrimSpace(clause[:idx])
		value := strings.TrimSpace(clause[idx+1:])
		if propName == "" {
			return nil, errors.NewUserError(
				fmt.Sprintf("empty property name in --set clause %q", clause),
				"Expected format: PropertyName=Value (e.g., Status=Done)",
			)
		}

		canonicalName, propType, err := resolveSchemaProperty(schema, propName)
		if err != nil {
			return nil, err
		}

		propPayload, err := buildPropertyPayloadForType(canonicalName, propType, value)
		if err != nil {
			return nil, err
		}
		props[canonicalName] = propPayload
	}

	return props, nil
}

// buildPropertyPayloadForType creates a Notion property update payload for a given property type.
func buildPropertyPayloadForType(propName, propType, value string) (interface{}, error) {
	switch propType {
	case "status":
		return map[string]interface{}{
			"status": map[string]interface{}{"name": value},
		}, nil
	case "select":
		return map[string]interface{}{
			"select": map[string]interface{}{"name": value},
		}, nil
	case "checkbox":
		boolVal := strings.EqualFold(value, "true")
		return map[string]interface{}{
			"checkbox": boolVal,
		}, nil
	case "title":
		return map[string]interface{}{
			"title": []map[string]interface{}{
				{"text": map[string]interface{}{"content": value}},
			},
		}, nil
	case "rich_text":
		return map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"text": map[string]interface{}{"content": value}},
			},
		}, nil
	case "number":
		numVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, errors.NewUserError(
				fmt.Sprintf("cannot parse %q as a number for property %q", value, propName),
				"Provide a valid numeric value.",
			)
		}
		return map[string]interface{}{
			"number": numVal,
		}, nil
	case "people":
		return map[string]interface{}{
			"people": []map[string]interface{}{
				{"id": value},
			},
		}, nil
	default:
		return nil, errors.NewUserError(
			fmt.Sprintf("unsupported property type %q for --set on property %q", propType, propName),
			"Supported types: status, select, checkbox, title, rich_text, number, people.",
		)
	}
}

// resolveSchemaProperty finds the canonical property name and type from the data source schema.
// Uses normalized matching for agent-friendly property name resolution.
func resolveSchemaProperty(schema map[string]interface{}, propName string) (canonicalName, propType string, err error) {
	if schema == nil {
		return "", "", errors.NewUserError(
			"data source schema is empty",
			"Check the database exists and has properties defined.",
		)
	}

	norm := normalizePropName(propName)
	for k, v := range schema {
		if normalizePropName(k) != norm {
			continue
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return "", "", errors.NewUserError(
				fmt.Sprintf("property %q has unexpected schema format", k),
				"Check the database schema.",
			)
		}
		t, _ := m["type"].(string)
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			return "", "", errors.NewUserError(
				fmt.Sprintf("property %q has no type in schema", k),
				"Check the database schema.",
			)
		}
		return k, t, nil
	}

	return "", "", errors.NewUserError(
		fmt.Sprintf("unknown property %q in database schema", propName),
		"Check property names with: ntn db get <database-id>",
	)
}

// formatDuration formats a duration into a human-readable string like "1.3s".
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
