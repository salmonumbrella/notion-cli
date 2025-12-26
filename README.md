# notion-cli

Command-line interface for the Notion API with secure authentication, file uploads, and data source management.

## Features

- **OAuth & Token Authentication** - Browser-based OAuth login or manual integration token setup
- **Secure Credential Storage** - Uses OS keyring (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows)
- **Pages & Databases** - Create, query, update pages and databases with full property support
- **Blocks** - Manage content blocks with quick helpers for TOC, breadcrumbs, dividers, and column layouts
- **File Uploads** - Upload files to Notion with automatic chunking for large files (multi-part upload)
- **Data Sources** - Create and query Notion data sources (API v2025-09-03)
- **Search** - Full-text search across pages and databases with filtering and pagination
- **Comments** - List and add comments to pages
- **Multiple Output Formats** - Text, JSON, table, and YAML output for scripting and automation
- **Agent-Friendly Flags** - `--yes`, `--limit`, `--sort-by`, `--desc`, and `--query` flags for automation
- **Batch File Support** - Infrastructure for reading JSON/NDJSON batch files (for future batch commands)
- **jq Integration** - Filter JSON output with jq expressions via `--query`
- **Debug Mode** - Verbose HTTP request/response logging for troubleshooting

## Installation

### Homebrew

```bash
brew tap salmonumbrella/notion-cli
brew install notion-cli
```

### Go Install

```bash
go install github.com/salmonumbrella/notion-cli/cmd/notion@latest
```

### Build from Source

Requires Go 1.23+

```bash
git clone https://github.com/salmonumbrella/notion-cli.git
cd notion-cli
make build
# Binary will be in bin/notion
```

## Quick Start

### 1. Authenticate

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

### 2. Verify Authentication

```bash
notion auth status
```

### 3. Start Using

```bash
# Search your workspace
notion search "project notes"

# Get current user
notion user me

# List all users
notion user list
```

You're ready to use notion-cli!

## Configuration

### Environment Variables

- `NOTION_TOKEN` - API token (alternative to keyring storage)
- `NOTION_WORKSPACE` - Default workspace name for multi-workspace support

### Config File (Optional)

notion-cli supports a YAML configuration file at `~/.config/notion-cli/config.yaml` for persistent settings.

```yaml
# Example configuration
output: json
color: always
default_workspace: personal
```

**Note:** CLI flags always override config file settings. See [Configuration Documentation](docs/configuration.md) for details.

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

### Best Practices

- **Never commit tokens** - Keep API tokens out of version control
- Use `notion auth login` for personal use (browser-based OAuth)
- Use `notion auth add-token` for integration/bot access (prompted securely)
- Rotate API tokens regularly for security
- The CLI warns about tokens older than 90 days

### Configuration File Security

- Config directory created with `0700` permissions (owner read/write/execute only)
- Config file created with `0600` permissions (owner read/write only)
- Token information is NOT stored in the config file

## Commands

### Authentication

```bash
notion auth login              # Authenticate via browser (OAuth)
notion auth add-token          # Add integration token manually
notion auth status             # Check authentication status
notion auth logout             # Remove stored credentials
```

### Search

```bash
notion search                           # Search all pages and databases
notion search "project notes"           # Search with query
notion search "meeting" --filter page   # Search only pages
notion search "tasks" --filter database # Search only databases
notion search --page-size 10            # Limit results per page
```

### Users

```bash
notion user me              # Get current user
notion user list            # List all workspace users
notion user get <user-id>   # Get user by ID
```

### Pages

```bash
notion page get <page-id>                              # Get page
notion page create --parent <id> --properties <json>   # Create page
notion page update <page-id> --properties <json>       # Update page
notion page move <page-id> --parent <new-parent-id>    # Move page
notion page property <page-id> <property-id>           # Get property
```

### Databases

```bash
notion db get <database-id>                            # Get database
notion db query <database-id>                          # Query database
notion db create --parent <id> --properties <json>     # Create database
notion db update <database-id> --properties <json>     # Update database
```

**Query with filters and sorts:**

```bash
# Query with filter
notion db query <database-id> \
  --filter '{"property":"Status","select":{"equals":"Done"}}'

# Query with sorts
notion db query <database-id> \
  --sorts '[{"property":"Created","direction":"descending"}]'

# Query with pagination
notion db query <database-id> --page-size 10 --start-cursor abc123
```

### Blocks

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

### File Uploads

```bash
notion file upload <filepath>                      # Upload file
notion file get <upload-id>                        # Get upload status
notion file list                                   # List file uploads
```

**Upload and attach to page property:**

```bash
notion file upload ./receipt.pdf --page abc123 --property "Attachments"
```

### Data Sources

```bash
notion datasource templates                        # List available templates
notion datasource create --template <name>         # Create from template
notion datasource get <datasource-id>              # Get data source
notion datasource query <datasource-id>            # Query data source
notion datasource update <datasource-id> <json>    # Update data source
```

Alias: `notion ds` (shorthand for `notion datasource`)

## Output Formats

### Text (Default)

Human-readable output with formatting:

```bash
$ notion user me
NAME         EMAIL                  TYPE
John Doe     john@example.com       person
```

### JSON

Machine-readable output for scripting:

```bash
$ notion user me --output json
{
  "id": "user_123",
  "name": "John Doe",
  "avatar_url": "https://...",
  "type": "person",
  "person": {
    "email": "john@example.com"
  }
}
```

### Table

Structured table output:

```bash
$ notion user list --output table
+----------+----------------------+--------+
| NAME     | EMAIL                | TYPE   |
+----------+----------------------+--------+
| John Doe | john@example.com     | person |
| Bot      | bot@integration      | bot    |
+----------+----------------------+--------+
```

### YAML

YAML format for configuration-style output:

```bash
$ notion user me --output yaml
id: user_123
name: John Doe
type: person
person:
  email: john@example.com
```

Data goes to stdout, errors and progress to stderr for clean piping.

## Examples

### Create a page with content

```bash
# Create a page
notion page create \
  --parent <parent-id> \
  --properties '{"title":[{"text":{"content":"New Page"}}]}'

# Add content blocks
notion block append <page-id> \
  --children '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"text":{"content":"Hello world"}}]}}]'
```

### Query a database and filter results

```bash
# Get all completed tasks
notion db query <database-id> \
  --filter '{"property":"Status","select":{"equals":"Done"}}' \
  --output json | jq '.results[] | {title: .properties.Name.title[0].text.content}'
```

### Upload a file and attach to page

```bash
# Upload and attach in one command
notion file upload ./document.pdf --page <page-id> --property "Attachments"

# Or upload first, then attach manually
notion file upload ./image.png
# Use the returned upload ID to attach via page update
```

### Search and export results

```bash
# Search and save to JSON file
notion search "project" --output json > projects.json

# Search only databases
notion search --filter database --output table
```

### Add table of contents to page

```bash
# Add TOC at the beginning
notion block add-toc <page-id>

# Add colored TOC
notion block add-toc <page-id> --color blue_background
```

### Create multi-column layout

```bash
# Create 2-column layout
notion block add-columns <page-id> --columns 2

# Create 3-column layout
notion block add-columns <page-id> --columns 3
```

## Advanced Features

### Debug Mode

Enable verbose output for troubleshooting:

```bash
notion --debug user me
# Shows: HTTP request method, URL, headers
# Shows: HTTP response status, body
```

Debug output goes to stderr, keeping stdout clean for piping.

### Pagination

Handle large result sets with pagination:

```bash
# First page
notion search --page-size 10

# Next page (use start_cursor from previous response)
notion search --page-size 10 --start-cursor "abc123"

# Database query pagination
notion db query <database-id> --page-size 25 --start-cursor "xyz789"
```

### Agent-Friendly Flags

For automation and scripting with AI agents or shell scripts:

```bash
# Skip confirmation prompts (for destructive operations)
notion page delete <page-id> --yes
notion block delete <block-id> -y

# Limit results
notion db query <database-id> --limit 10
notion search "meeting notes" --limit 5

# Sort results
notion search "project" --sort-by created_time --desc
notion db query <database-id> --sort-by last_edited_time

# Filter JSON output with jq expressions
notion page get <page-id> --output json --query '.properties.Status'
notion db query <database-id> --output json --query '.results[].properties.Name'
```

The `--query` flag passes output through jq for filtering. Requires `--output json`.

## Global Flags

All commands support these flags:

- `--output <format>` - Output format: `text` (default), `json`, `table`, or `yaml`
- `--debug` - Enable debug output (shows HTTP requests/responses)
- `--workspace <name>` or `-w <name>` - Workspace to use (overrides NOTION_WORKSPACE env var)
- `--yes` or `-y` - Skip confirmation prompts (for destructive operations)
- `--limit <N>` - Limit number of results (for list/query commands)
- `--sort-by <field>` - Sort results by field (e.g., `created_time`, `last_edited_time`)
- `--desc` - Sort in descending order (use with `--sort-by`)
- `--query <expr>` - Filter JSON output with jq expression (requires `--output json`)
- `--help` or `-h` - Show help for any command
- `--version` or `-v` - Show version information

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (with Homebrew):
notion completion bash > $(brew --prefix)/etc/bash_completion.d/notion

# Linux:
notion completion bash > /etc/bash_completion.d/notion

# Or load for current session only:
source <(notion completion bash)
```

### Zsh

```zsh
# Save to completion directory:
notion completion zsh > "${fpath[1]}/_notion"

# Or add to .zshrc for auto-load:
echo 'source <(notion completion zsh)' >> ~/.zshrc

# Then restart your shell
```

### Fish

```fish
# Save to Fish completions directory:
notion completion fish > ~/.config/fish/completions/notion.fish

# Or load for current session only:
notion completion fish | source
```

### PowerShell

```powershell
# Load for current session:
notion completion powershell | Out-String | Invoke-Expression

# Or add to PowerShell profile for persistence:
notion completion powershell >> $PROFILE
```

After installing completions, restart your shell or source your shell configuration file.

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

### Available Make Targets

```bash
make build         # Build binary to bin/notion
make test          # Run tests
make lint          # Run linter
make fmt           # Format code
make fmt-check     # Check formatting (CI)
make clean         # Clean build artifacts
make ci            # Run all CI checks (fmt-check, lint, test)
```

## License

MIT

## Links

- [Notion API Documentation](https://developers.notion.com/reference)
- [GitHub Repository](https://github.com/salmonumbrella/notion-cli)
- [Configuration Guide](docs/configuration.md)
