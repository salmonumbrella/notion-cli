package output_test

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

// Example demonstrates basic usage of the output package.
func Example() {
	ctx := context.Background()

	// Define sample data
	type Page struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		URL   string `json:"url"`
	}

	pages := []Page{
		{ID: "page-1", Title: "Getting Started", URL: "https://example.invalid/page-1"},
		{ID: "page-2", Title: "API Reference", URL: "https://example.invalid/page-2"},
	}

	// Text format (default)
	fmt.Println("=== Text Format ===")
	textPrinter := output.NewPrinter(os.Stdout, output.FormatText)
	_ = textPrinter.Print(ctx, pages[0])

	// JSON format
	fmt.Println("\n=== JSON Format ===")
	jsonPrinter := output.NewPrinter(os.Stdout, output.FormatJSON)
	_ = jsonPrinter.Print(ctx, pages[0])

	// Table format
	fmt.Println("=== Table Format ===")
	tablePrinter := output.NewPrinter(os.Stdout, output.FormatTable)
	_ = tablePrinter.Print(ctx, pages)
}

// ExampleParseFormat demonstrates parsing format strings.
func ExampleParseFormat() {
	formats := []string{"text", "json", "table", "TEXT", ""}

	for _, f := range formats {
		format, err := output.ParseFormat(f)
		if err != nil {
			fmt.Printf("Error parsing '%s': %v\n", f, err)
			continue
		}
		fmt.Printf("Parsed '%s' -> %s\n", f, format)
	}

	// Output:
	// Parsed 'text' -> text
	// Parsed 'json' -> json
	// Parsed 'table' -> table
	// Parsed 'TEXT' -> text
	// Parsed '' -> text
}

// ExamplePrinter_Print_singleObject shows printing a single object.
func ExamplePrinter_Print_singleObject() {
	ctx := context.Background()

	type Database struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	db := Database{
		ID:          "db-123",
		Title:       "Tasks",
		Description: "My task database",
	}

	// Print as text
	printer := output.NewPrinter(os.Stdout, output.FormatText)
	_ = printer.Print(ctx, db)

	// Output:
	// id: db-123
	// title: Tasks
	// description: My task database
}

// ExamplePrinter_Print_list shows printing a list as a table.
func ExamplePrinter_Print_list() {
	ctx := context.Background()

	type Task struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}

	tasks := []Task{
		{ID: "1", Status: "todo", Title: "Write docs"},
		{ID: "2", Status: "done", Title: "Write tests"},
		{ID: "3", Status: "todo", Title: "Deploy"},
	}

	// Print as table
	printer := output.NewPrinter(os.Stdout, output.FormatTable)
	_ = printer.Print(ctx, tasks)

	// Output will be a formatted table (exact spacing depends on tabwriter):
	// ID  STATUS  TITLE
	// 1   todo    Write docs
	// 2   done    Write tests
	// 3   todo    Deploy
}
