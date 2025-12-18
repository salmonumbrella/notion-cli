# notion-cli

A command-line interface for Notion.

## Quick Start

### Step 1: Install

**Homebrew (recommended)**
```bash
brew tap salmonumbrella/notion-cli
brew install notion-cli
```

**Go install**
```bash
go install github.com/salmonumbrella/notion-cli/cmd/notion@latest
```

**Build from source** (requires Go 1.23+)
```bash
git clone https://github.com/salmonumbrella/notion-cli.git
cd notion-cli
make build
# Binary will be in bin/notion
```

### Step 2: Authenticate

Choose one of two authentication methods:

**Option A: Sign in with Notion (recommended)**

This opens your browser to authorize notion-cli with your Notion account. Actions will be attributed to you personally.

```bash
notion auth login
```

**Option B: Use an integration token**

For bot/integration access. Create a token at https://www.notion.so/my-integrations

```bash
notion auth add-token
```

### Step 3: Verify

```bash
notion auth status
```

You're ready to use notion-cli!

## Authentication Details

| Method | Command | Best for |
|--------|---------|----------|
| OAuth login | `notion auth login` | Personal use, actions attributed to you |
| Integration token | `notion auth add-token` | Bots, automations, shared integrations |

**Other auth commands:**
```bash
notion auth status   # Check authentication status
notion auth logout   # Remove stored credentials
```

**Environment variable (alternative):**
```bash
export NOTION_TOKEN="your_token_here"
```

## Commands

### Search

```bash
notion search                           # Search all pages and databases
notion search "project notes"           # Search with query
notion search "meeting" --filter page   # Search only pages
notion search "tasks" --filter database # Search only databases
```

### User

```bash
notion user me              # Get current user
notion user list            # List all workspace users
notion user get <user-id>   # Get user by ID
```

### Page

```bash
notion page get <page-id>                              # Get page
notion page create --parent <id> --properties <json>   # Create page
notion page update <page-id> --properties <json>       # Update page
notion page property <page-id> <property-id>           # Get property
```

### Database

```bash
notion db get <database-id>                            # Get database
notion db query <database-id>                          # Query database
notion db create --parent <id> --properties <json>     # Create database
notion db update <database-id> --properties <json>     # Update database
```

### Block

```bash
notion block get <block-id>                        # Get block
notion block children <block-id>                   # Get children
notion block append <parent-id> --children <json>  # Append blocks
notion block update <block-id> --content <json>    # Update block
notion block delete <block-id>                     # Delete block
```

**Quick block creation:**
```bash
notion block add-toc <parent-id>                   # Add table of contents
notion block add-toc <parent-id> --color blue      # With color
notion block add-breadcrumb <parent-id>            # Add breadcrumb navigation
notion block add-divider <parent-id>               # Add horizontal divider
notion block add-columns <parent-id> --columns 3   # Add 3-column layout (2-5)
```

### Comments

```bash
notion comment list <block-id>                  # List comments
notion comment add <block-id> --text "Comment"  # Add comment
```

## Output Formats

```bash
notion user me --output text   # Human-readable (default)
notion user me --output json   # JSON
notion user list --output table # Table format
```

## Go Library

The `notion` package can be used as a Go library for building Notion integrations.

```go
import "github.com/salmonumbrella/notion-cli/internal/notion"

client := notion.NewClient("your_token")
```

### Webhook Signature Verification

Verify incoming webhook requests from Notion:

```go
// Compute signature for comparison
signature := notion.ComputeWebhookSignature(secret, requestBody)

// Or verify directly
if notion.VerifyWebhookSignature(secret, requestBody, headerSignature) {
    // Valid webhook request
}

// Parse webhook events
event, err := notion.ParseWebhookEvent(requestBody)
if err != nil {
    // Handle error
}
fmt.Println(event.Type) // e.g., "page.content_updated"
```

### Block Type Helpers

Create blocks programmatically:

```go
// Text blocks
paragraph := notion.NewParagraph("Hello world")
heading := notion.NewHeading1("Section Title")
quote := notion.NewQuote("Important quote")
code := notion.NewCode("fmt.Println(\"hi\")", "go")
callout := notion.NewCallout("Note text", "💡")

// List items
bullet := notion.NewBulletedListItem("Item 1")
numbered := notion.NewNumberedListItem("Step 1")
todo := notion.NewToDo("Task", false)

// Layout blocks
divider := notion.NewDivider()
toc := notion.NewTableOfContents("default")
breadcrumb := notion.NewBreadcrumb()

// Columns (2-5 columns supported)
columns := notion.NewColumnList(
    []map[string]interface{}{notion.NewParagraph("Column 1")},
    []map[string]interface{}{notion.NewParagraph("Column 2")},
)

// Synced blocks
original := notion.NewSyncedBlock(nil, children)        // Original
synced := notion.NewSyncedBlock(&sourceID, nil)         // Reference

// Link preview
preview := notion.NewLinkPreview("https://example.com")
```

### Multi-part File Uploads

Upload large files in chunks:

```go
file, _ := os.Open("large-file.pdf")
defer file.Close()

// Automatically handles chunking (5MB chunks)
upload, err := client.UploadLargeFile(ctx, "large-file.pdf", file)
if err != nil {
    // Handle error
}
fmt.Println(upload.Status) // "complete"
```

Or handle parts manually:

```go
// 1. Create upload session
req := &notion.CreateFileUploadRequest{
    FileName:    "document.pdf",
    ContentType: "application/pdf",
    Mode:        "multi_part",
}
upload, _ := client.CreateFileUpload(ctx, req)

// 2. Upload each part
part, _ := client.SendFilePart(ctx, upload.UploadURL, chunkReader, partNumber)

// 3. Complete upload
result, _ := client.CompleteFileUpload(ctx, upload.ID)
```

### Formula & Rollup Properties

Read computed property values:

```go
// Formula properties
formula := notion.FormulaProperty{...}
str := formula.Formula.GetFormulaString()   // For string formulas
num := formula.Formula.GetFormulaNumber()   // For number formulas
ok := formula.Formula.GetFormulaBool()      // For boolean formulas

// Rollup properties
rollup := notion.RollupProperty{...}
count := rollup.Rollup.GetRollupNumber()        // For aggregations
length := rollup.Rollup.GetRollupArrayLength()  // For array rollups
```

## API Version

This CLI uses Notion API version `2025-09-03`.

## Development

After cloning, install git hooks:

```bash
make setup
```

This installs [lefthook](https://github.com/evilmartians/lefthook) pre-commit and pre-push hooks for linting and testing.

## License

MIT
