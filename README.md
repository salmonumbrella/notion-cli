# Notion CLI — Wiki in your Terminal

Command-line interface for the Notion API with secure authentication, file uploads, and data source management.

## Features

- **Authentication** - OAuth browser login or integration token, stored securely in OS keychain
- **Blocks** - Manage content blocks with quick helpers for TOC, breadcrumbs, dividers, and columns
- **Comments** - List and add comments to pages
- **Data Sources** - Create and query Notion data sources (API v2025-09-03)
- **Databases** - Create, query, and update databases with full property support
- **File Uploads** - Upload files with automatic chunking for large files (multi-part upload)
- **jq Integration** - Filter JSON output with jq expressions via `--query`
- **Pages** - Create, update, and move pages with full property support
- **Search** - Full-text search across pages and databases with filtering and pagination

## Installation

### Homebrew

```bash
brew install salmonumbrella/tap/notion-cli
```

### Go Install

```bash
go install github.com/salmonumbrella/notion-cli/cmd/notion@latest
```

## Quick Start

### 1. Authenticate

Choose one of two methods:

**Browser (recommended):**
```bash
notion auth login
```

**Integration token:**
```bash
notion auth add-token
# You'll be prompted securely for the token
```

### 2. Test Authentication

```bash
notion auth status
```

### 3. Start Using

```bash
# Search your workspace
notion search "project notes"

# Get current user
notion user me
```

## Configuration

### Environment Variables

- `NOTION_TOKEN` - API token (alternative to keyring storage)
- `NOTION_WORKSPACE` - Default workspace name for multi-workspace support
- `NOTION_OUTPUT` - Output format: `text` (default), `json`, `table`, or `yaml`
- `NO_COLOR` - Set to any value to disable colors (standard convention)

### Config File (Optional)

notion-cli supports a YAML configuration file at `~/.config/notion-cli/config.yaml`:

```yaml
output: json
color: always
default_workspace: personal
```

CLI flags always override config file settings.

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

### Best Practices

- Use `notion auth login` for personal use (browser-based OAuth)
- Use `notion auth add-token` for integration/bot access
- Never commit tokens to version control
- Rotate API tokens regularly

## Commands

### Authentication

```bash
notion auth login                  # Authenticate via browser (OAuth)
notion auth add-token              # Add integration token manually
notion auth status                 # Check authentication status
notion auth logout                 # Remove stored credentials
```

### Search

```bash
notion search                      # Search all pages and databases
notion search "project notes"      # Search with query
notion search --filter page        # Search only pages
notion search --filter database    # Search only databases
```

### Users

```bash
notion user me                     # Get current user
notion user list                   # List all workspace users
notion user get <user-id>          # Get user by ID
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

Query with filters and sorts:

```bash
# Query with filter
notion db query <database-id> \
  --filter '{"property":"Status","select":{"equals":"Done"}}'

# Query with sorts
notion db query <database-id> \
  --sorts '[{"property":"Created","direction":"descending"}]'
```

### Blocks

```bash
notion block get <block-id>                        # Get block
notion block children <block-id>                   # Get children
notion block append <parent-id> --children <json>  # Append blocks
notion block update <block-id> --content <json>    # Update block
notion block delete <block-id>                     # Delete block
```

Quick block creation:

```bash
notion block add-toc <parent-id>                   # Add table of contents
notion block add-toc <parent-id> --color blue      # With color
notion block add-breadcrumb <parent-id>            # Add breadcrumb navigation
notion block add-divider <parent-id>               # Add horizontal divider
notion block add-columns <parent-id> --columns 3   # Add 3-column layout (2-5)
```

### Comments

```bash
notion comment list <block-id>                     # List comments
notion comment add <block-id> --text "Comment"     # Add comment
```

### File Uploads

```bash
notion file upload <filepath>                      # Upload file
notion file get <upload-id>                        # Get upload status
notion file list                                   # List file uploads
```

Upload and attach to page property:

```bash
notion file upload ./receipt.pdf --page <page-id> --property "Attachments"
```

### Data Sources

```bash
notion datasource templates                        # List available templates
notion datasource create --template <name>         # Create from template
notion datasource get <datasource-id>              # Get data source
notion datasource query <datasource-id>            # Query data source
notion datasource update <datasource-id> <json>    # Update data source
```

Alias: `notion ds` works as shorthand for `notion datasource`.

## Output Formats

### Text

Human-readable tables with colors and formatting:

```bash
$ notion user me
NAME         EMAIL                  TYPE
John Doe     john@example.com       person
```

### JSON

Machine-readable output:

```bash
$ notion user me --output json
{
  "id": "user_123",
  "name": "John Doe",
  "type": "person"
}
```

Data goes to stdout, errors and progress to stderr for clean piping.

## Examples

### Create a Page with Content

```bash
# Create a page
notion page create \
  --parent <parent-id> \
  --properties '{"title":[{"text":{"content":"New Page"}}]}'

# Add content blocks
notion block append <page-id> \
  --children '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"text":{"content":"Hello world"}}]}}]'
```

### Query and Filter Database

```bash
# Get all completed tasks
notion db query <database-id> \
  --filter '{"property":"Status","select":{"equals":"Done"}}' \
  --output json | jq '.results[].properties.Name'
```

### Automation

Use `--yes` to skip confirmations, `--limit` to control result size, and `--sort-by` for ordering:

```bash
# Delete without confirmation
notion block delete <block-id> --yes

# Get recent items
notion search "project" --limit 5 --sort-by created_time --desc

# Pipeline example
notion db query <database-id> --output json | jq '.results[] | .id'

# Filter JSON output with jq expression
notion page get <page-id> --output json --query '.properties.Status'
```

### Debug Mode

```bash
notion --debug user me
# Shows HTTP request/response details to stderr
```

## Global Flags

All commands support these flags:

- `--output <format>` - Output format: `text`, `json`, `table`, or `yaml` (default: text)
- `--workspace <name>`, `-w` - Workspace to use (overrides NOTION_WORKSPACE)
- `--debug` - Enable debug output (shows API requests/responses)
- `--query <expr>` - JQ filter expression for JSON output
- `--yes`, `-y` - Skip confirmation prompts (useful for scripts and automation)
- `--limit <N>` - Limit number of results
- `--sort-by <field>` - Sort results by field (e.g., `created_time`, `last_edited_time`)
- `--desc` - Sort in descending order (use with `--sort-by`)
- `--help` - Show help for any command
- `--version` - Show version information

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (Homebrew):
notion completion bash > $(brew --prefix)/etc/bash_completion.d/notion

# Linux:
notion completion bash > /etc/bash_completion.d/notion

# Or source directly:
source <(notion completion bash)
```

### Zsh

```zsh
notion completion zsh > "${fpath[1]}/_notion"
```

### Fish

```fish
notion completion fish > ~/.config/fish/completions/notion.fish
```

### PowerShell

```powershell
notion completion powershell | Out-String | Invoke-Expression
```

## Development

After cloning, install git hooks:

```bash
make setup
```

This installs [lefthook](https://github.com/evilmartians/lefthook) pre-commit and pre-push hooks for linting and testing.

## License

MIT

## Links

- [Notion API Documentation](https://developers.notion.com/reference)
