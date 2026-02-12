package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newImportCSVCmd() *cobra.Command {
	var (
		filePath  string
		columnMap string
		batchSize int
		dryRun    bool
		skipRows  int
	)

	cmd := &cobra.Command{
		Use:     "csv <database-id>",
		Aliases: []string{"c"},
		Short:   "Import a CSV file into a Notion database",
		Long: `Import rows from a CSV file as pages into a Notion database.

The first row of the CSV is used as column headers. Headers are matched to
database properties by name (case-sensitive first, then case-insensitive).

Use --column-map to override the mapping: "CSV Header=Notion Property,Name=Title"

Supported property types:
  - title, rich_text, number, select, multi_select (semicolon-separated),
    date, checkbox, url, email, phone_number

Examples:
  ntn import csv abc123 --file data.csv
  ntn import csv abc123 --file data.csv --column-map "CSV Header=Notion Property,Name=Title"
  ntn import csv abc123 --file - < data.csv
  ntn import csv abc123 --file data.csv --dry-run
  ntn import csv abc123 --file data.csv --batch-size 5
  ntn import csv abc123 --file data.csv --skip-rows 2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return errors.NewUserError("--file is required", "Provide a CSV file path or use - for stdin")
			}

			dbID, err := cmdutil.NormalizeNotionID(args[0])
			if err != nil {
				return err
			}

			// Read CSV
			records, err := readCSVFile(filePath)
			if err != nil {
				return errors.WrapUserError(err, "failed to read CSV file", "Check that the file exists and is valid CSV")
			}

			if len(records) == 0 {
				return errors.NewUserError("CSV file has no headers", "The first row must contain column headers")
			}

			headers := records[0]
			dataRows := records[1:]

			// Skip rows after header
			if skipRows > 0 {
				if skipRows >= len(dataRows) {
					dataRows = nil
				} else {
					dataRows = dataRows[skipRows:]
				}
			}

			if len(dataRows) == 0 && !dryRun {
				return errors.NewUserError("CSV file has no data rows", "Add data rows after the header row")
			}

			// Parse column-map overrides
			colMap, err := parseColumnMap(columnMap)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			stderr := stderrFromContext(ctx)

			// Fetch database schema
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			db, err := client.GetDatabase(ctx, dbID)
			if err != nil {
				return fmt.Errorf("failed to get database schema: %w", err)
			}

			// Map CSV headers to database properties
			mappings, warnings := buildColumnMappings(headers, db.Properties, colMap)

			// Print warnings for unmapped columns
			for _, w := range warnings {
				_, _ = fmt.Fprintf(stderr, "Warning: %s\n", w)
			}

			// Check that at least one title property is mapped
			hasTitleMapping := false
			for _, m := range mappings {
				if m.propType == "title" {
					hasTitleMapping = true
					break
				}
			}
			if !hasTitleMapping {
				return errors.NewUserError(
					"no CSV column maps to a title property",
					"Use --column-map to map a column to the database's title property",
				)
			}

			// Dry run
			if dryRun {
				printer := NewDryRunPrinter(stderr)
				printer.Header("import", "csv", filePath)

				dbName := extractDatabaseTitle(*db)
				if dbName != "" {
					printer.Field("Target database", fmt.Sprintf("%s (%s)", dbName, dbID))
				} else {
					printer.Field("Target database", dbID)
				}
				printer.Field("Total rows", fmt.Sprintf("%d", len(dataRows)))
				printer.Field("Batch size", fmt.Sprintf("%d", batchSize))
				if skipRows > 0 {
					printer.Field("Skipped rows", fmt.Sprintf("%d", skipRows))
				}

				printer.Section("Column mappings:")
				for _, m := range mappings {
					_, _ = fmt.Fprintf(stderr, "  %s -> %s (%s)\n", m.csvHeader, m.notionProp, m.propType)
				}

				// Show preview of first 3 rows
				previewCount := 3
				if len(dataRows) < previewCount {
					previewCount = len(dataRows)
				}
				if previewCount > 0 {
					printer.Section("Preview (first rows):")
					for i := 0; i < previewCount; i++ {
						_, _ = fmt.Fprintf(stderr, "  Row %d:\n", i+1)
						for _, m := range mappings {
							val := ""
							if m.csvIndex < len(dataRows[i]) {
								val = dataRows[i][m.csvIndex]
							}
							_, _ = fmt.Fprintf(stderr, "    %s: %s\n", m.notionProp, val)
						}
					}
				}

				printer.Footer()
				return nil
			}

			// Import rows in batches
			if batchSize <= 0 {
				batchSize = 10
			}

			var totalCreated int
			for i := 0; i < len(dataRows); i += batchSize {
				end := i + batchSize
				if end > len(dataRows) {
					end = len(dataRows)
				}

				batch := dataRows[i:end]
				for rowIdx, row := range batch {
					props := buildPageProperties(row, mappings)
					if props == nil {
						continue
					}

					req := &notion.CreatePageRequest{
						Parent: map[string]interface{}{
							"database_id": dbID,
						},
						Properties: props,
					}

					_, err := client.CreatePage(ctx, req)
					if err != nil {
						return fmt.Errorf("failed to create page for row %d: %w", i+rowIdx+1, err)
					}
					totalCreated++
				}
			}

			// Print summary
			dbName := extractDatabaseTitle(*db)
			if dbName != "" {
				_, _ = fmt.Fprintf(stderr, "Imported %d pages into database %s\n", totalCreated, dbName)
			} else {
				_, _ = fmt.Fprintf(stderr, "Imported %d pages into database %s\n", totalCreated, dbID)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "CSV file to import (use - for stdin)")
	cmd.Flags().StringVar(&columnMap, "column-map", "", "Column mapping overrides: CSVHeader=NotionProp,...")
	cmd.Flags().IntVar(&batchSize, "batch-size", 10, "Number of pages to create per batch")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview import without creating pages")
	cmd.Flags().IntVar(&skipRows, "skip-rows", 0, "Number of rows to skip after header")

	return cmd
}

// columnMapping holds the mapping from a CSV column to a Notion property.
type columnMapping struct {
	csvIndex   int
	csvHeader  string
	notionProp string
	propType   string
}

// readCSVFile reads all records from a CSV file or stdin.
func readCSVFile(path string) ([][]string, error) {
	var reader io.Reader

	if path == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		reader = f
	}

	r := csv.NewReader(reader)
	return r.ReadAll()
}

// parseColumnMap parses comma-separated "CSVHeader=NotionProp" pairs.
func parseColumnMap(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}

	result := make(map[string]string)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, errors.NewUserError(
				fmt.Sprintf("invalid column-map entry: %q", pair),
				"Use format: CSVHeader=NotionProp,CSVHeader2=NotionProp2",
			)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

// buildColumnMappings maps CSV headers to database property names and types.
// Returns the mappings and a list of warnings for unmapped columns.
func buildColumnMappings(
	headers []string,
	dbProps map[string]map[string]interface{},
	overrides map[string]string,
) ([]columnMapping, []string) {
	var mappings []columnMapping
	var warnings []string

	// Build a case-insensitive lookup for database properties
	propLower := make(map[string]string) // lowered name -> actual name
	for name := range dbProps {
		propLower[strings.ToLower(name)] = name
	}

	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}

		// Check overrides first
		targetProp := ""
		if overrides != nil {
			if override, ok := overrides[header]; ok {
				targetProp = override
			}
		}

		if targetProp == "" {
			// Exact match (case-sensitive)
			if _, ok := dbProps[header]; ok {
				targetProp = header
			} else if actual, ok := propLower[strings.ToLower(header)]; ok {
				// Case-insensitive fallback
				targetProp = actual
			}
		}

		if targetProp == "" {
			warnings = append(warnings, fmt.Sprintf("column %q does not match any database property (skipping)", header))
			continue
		}

		// Verify the property exists in the database
		prop, ok := dbProps[targetProp]
		if !ok {
			// The override might point to a non-existent property
			warnings = append(warnings, fmt.Sprintf("column %q mapped to %q but property not found in database (skipping)", header, targetProp))
			continue
		}

		propType, _ := prop["type"].(string)
		mappings = append(mappings, columnMapping{
			csvIndex:   i,
			csvHeader:  header,
			notionProp: targetProp,
			propType:   propType,
		})
	}

	return mappings, warnings
}

// buildPageProperties converts a CSV row to Notion page properties based on mappings.
func buildPageProperties(row []string, mappings []columnMapping) map[string]interface{} {
	props := make(map[string]interface{})

	for _, m := range mappings {
		value := ""
		if m.csvIndex < len(row) {
			value = strings.TrimSpace(row[m.csvIndex])
		}

		// Skip empty values
		if value == "" {
			continue
		}

		prop := csvValueToProperty(value, m.propType)
		if prop != nil {
			props[m.notionProp] = prop
		}
	}

	if len(props) == 0 {
		return nil
	}

	return props
}

// csvValueToProperty converts a CSV string value to a Notion property value
// based on the property type.
func csvValueToProperty(value string, propType string) map[string]interface{} {
	switch propType {
	case "title":
		return map[string]interface{}{
			"title": []map[string]interface{}{
				{"text": map[string]interface{}{"content": value}},
			},
		}

	case "rich_text":
		return map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"text": map[string]interface{}{"content": value}},
			},
		}

	case "number":
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil
		}
		return map[string]interface{}{
			"number": n,
		}

	case "select":
		return map[string]interface{}{
			"select": map[string]interface{}{"name": value},
		}

	case "multi_select":
		parts := strings.Split(value, ";")
		var options []map[string]interface{}
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				options = append(options, map[string]interface{}{"name": p})
			}
		}
		if len(options) == 0 {
			return nil
		}
		return map[string]interface{}{
			"multi_select": options,
		}

	case "date":
		return map[string]interface{}{
			"date": map[string]interface{}{"start": value},
		}

	case "checkbox":
		lower := strings.ToLower(value)
		checked := lower == "true" || lower == "yes" || lower == "1"
		return map[string]interface{}{
			"checkbox": checked,
		}

	case "url":
		return map[string]interface{}{
			"url": value,
		}

	case "email":
		return map[string]interface{}{
			"email": value,
		}

	case "phone_number":
		return map[string]interface{}{
			"phone_number": value,
		}

	default:
		// Unsupported property type — skip
		return nil
	}
}

// extractDatabaseTitle is defined in skill.go
