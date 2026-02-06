# Notion CLI — Wiki in your Terminal

Command-line interface for the Notion API with secure authentication, file uploads, and data source management.

## Features

- **Authentication** - OAuth browser login or integration token, stored securely in OS keychain
- **Blocks** - Manage content blocks with quick helpers for TOC, breadcrumbs, dividers, and columns
- **Comments** - List and add comments to pages
- **Data Sources** - Create and query Notion data sources (API v2025-09-03)
- **Databases** - Create, query, and update databases with full property support
- **Export** - Export pages to Markdown or JSON block trees
- **File Uploads** - Upload files with automatic chunking for large files (multi-part upload)
- **Fetch by URL** - Fetch pages or databases by Notion URL
- **jq Integration** - Filter JSON output with jq expressions via `--query`
- **Pages** - Create, update, and move pages with full property support
- **Pages (Batch/Duplicate)** - Batch-create pages and duplicate pages with content
- **Search** - Full-text search across pages and databases with filtering and pagination
- **Webhooks** - Verify and parse webhook payloads

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
- `NOTION_OUTPUT` - Output format: `text` (default), `json`, `ndjson`, `table`, or `yaml`
- `NOTION_API_BASE_URL` - Override Notion API base URL (useful for proxies and tests)
- `NOTION_NO_UPDATE_CHECK` - Set to any value to disable update checks (the CLI also auto-disables update checks when stdout is not a TTY)
- `NO_COLOR` - Set to any value to disable colors (standard convention)

### Agent-Friendly Global Flags

- `--results-only` - For list-like responses, output just the `.results` array (useful for piping to `jq`).
- `--limit`, `--sort-by`, `--desc`, `--latest`, `--recent` - Apply client-side sorting/limiting when possible.

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
notion search "project" --all --results-only  # Fetch all results (array only)
```

### Resolve

```bash
notion resolve "Meeting Notes"     # Return candidate IDs (skill aliases + search)
notion resolve "Projects" --type database
notion resolve standup             # Skill alias
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
notion page create-batch --parent <id> --pages <json>  # Create multiple pages
notion page update-batch --pages <json>                # Update multiple pages
notion page duplicate <page-id>                        # Duplicate page
notion page export <page-id> --format markdown         # Export page content
notion page update <page-id> --properties <json>       # Update page
notion page move <page-id> --parent <new-parent-id>    # Move page
notion page property <page-id> <property-id>           # Get property
notion page properties <page-id>                       # List properties (optionally simplified)
```

### Databases

```bash
notion db get <database-id>                            # Get database
notion db query <database-id>                          # Query database
notion db query <database-id> --data-source <id>        # Query a specific data source
notion db create --parent <id> --properties <json>     # Create database
notion db update <database-id> --properties <json>     # Update database
```

Query with filters and sorts:

```bash
# Query with filter
notion db query <database-id> \
  --filter '{"property":"Status","select":{"equals":"Done"}}'

# Agent-friendly shorthand filters (server-side; combined with --filter using AND)
notion db query <database-id> --status Done
notion db query <database-id> --assignee me
notion db query <database-id> --priority High

# Query with filter from file/stdin (avoids shell escaping issues)
notion db query <database-id> --filter @filter.json
cat filter.json | notion db query <database-id> --filter -

# Query with sorts
notion db query <database-id> \
  --sorts '[{"property":"Created","direction":"descending"}]'

# Fetch all results as an array
notion db query <database-id> --all --results-only
```

### Blocks

```bash
notion block get <block-id>                        # Get block
notion block children <block-id>                   # Get children
notion block children <block-id> --plain           # Get children (simplified: id/type/text)
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

# Desire paths (agent-friendly)
notion comment <page-id> "Looks great!"            # Add comment (positional)
notion comment add <page-id> "Looks great!"        # Add comment (positional)
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

Query with filters, sorts, and selection:

```bash
# Filter by property value
notion ds query <datasource-id> \
  --filter '{"property":"Status","status":{"equals":"Active"}}'

# Read filter from file (avoids shell escaping)
notion ds query <datasource-id> --filter-file filter.json

# Sort by timestamp (shorthand)
notion ds query <datasource-id> --sort-by last_edited_time --desc

# Sort with full Notion sorts JSON
notion ds query <datasource-id> \
  --sorts '[{"property":"Priority","direction":"ascending"}]'

# Client-side select/status filtering (exact, not-equals, or regex)
notion ds query <datasource-id> \
  --select-property "Category" --select-equals "Engineering"
notion ds query <datasource-id> \
  --select-property "Status" --select-not "Done"
notion ds query <datasource-id> \
  --select-property "Category" --select-match "(?i)eng"

# Fetch all results as a plain array
notion ds query <datasource-id> --limit 0 --results-only

# Combine: filter + sort + limit
notion ds query <datasource-id> \
  --filter '{"property":"Status","status":{"equals":"Active"}}' \
  --sort-by created_time --desc --limit 20 --results-only
```

Alias: `notion ds` works as shorthand for `notion datasource`.

### Fetch

```bash
notion fetch <notion-url>                              # Fetch page or database by URL
notion fetch <notion-url> --type page                  # Force page fetch
notion fetch <notion-url> --type database              # Force database fetch
```

### Webhooks

```bash
notion webhook verify --secret <secret> --payload payload.json --signature <sig>
notion webhook verify --secret <secret> --payload payload.json           # Compute signature
notion webhook parse --payload payload.json
```

### API

Raw API access (useful for new endpoints and debugging):

```bash
notion api request GET /users
notion api request POST /search --body '{"query":"project"}'
notion api request POST /databases/<id>/query --body @query.json
notion api request GET /blocks/<id>/children --paginate
```

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

### NDJSON

Newline-delimited JSON (one JSON object per line):

```bash
$ notion search "project" --output ndjson
{"object":"page", ...}
{"object":"page", ...}
```

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

### Duplicate and Export Pages

```bash
# Duplicate a page with its content
notion page duplicate <page-id>

# Export page content to Markdown
notion page export <page-id> --format markdown
```

### Batch Updates

```bash
# Update multiple pages in one command
notion page update-batch --pages '[{"id":"<page-id>","properties":{"Status":{"status":{"name":"Done"}}}}]'
```

### Automation

Use `--yes` (or `--no-input`) to skip confirmations, `--limit` to control result size, and `--sort-by` for ordering:

```bash
# Delete without confirmation
notion block delete <block-id> --yes

# Get recent items
notion search "project" --limit 5 --sort-by created_time --desc

# Pipeline example
notion db query <database-id> --output json | jq '.results[] | .id'

# Filter JSON output with jq expression
notion page get <page-id> --output json --query '.properties.Status'

# Use a file for longer jq expressions
notion page get <page-id> --output json --query-file ./query.jq

# Project fields without jq
notion db query <database-id> --results-only --fields id,name,created_time

# JSONPath extraction
notion db query <database-id> --jsonpath '$.results[0].id'

# Latest shortcuts
notion search "project" --latest
notion search "project" --recent 5

# Fail if empty
notion search "project" --fail-empty --limit 1
```

### JSON Input Shortcuts

Flags that accept JSON also support reading from a file or stdin:

```bash
# From a file
notion db query <database-id> --filter @filter.json

# From a file (properties)
notion page update <page-id> --properties @props.json

# From a file (properties flag)
notion page update <page-id> --properties-file props.json

# From stdin
cat filter.json | notion db query <database-id> --filter -

# From stdin (heredoc)
cat <<'JSON' | notion page update <page-id> --properties -
{"Status":{"status":{"name":"Done"}}}
JSON
```

### Debug Mode

```bash
notion --debug user me
# Shows HTTP request/response details to stderr
```

## Global Flags

All commands support these flags:

- `--output <format>` - Output format: `text`, `json`, `ndjson`, `table`, or `yaml` (default: text)
- `--query <expr>` / `--jq <expr>` - JQ filter expression for JSON output
- `--fields <paths>` / `--pick <paths>` - Project fields (comma-separated paths, `key=path` to rename)
- `--jsonpath <expr>` - Extract a value using JSONPath
- `--latest` / `--recent <n>` - Shortcut for `--sort-by created_time --desc --limit N`
- `--fail-empty` - Exit with error when results are empty
- `--workspace <name>`, `-w` - Workspace to use (overrides NOTION_WORKSPACE)
- `--debug` - Enable debug output (shows API requests/responses)
- `--query <expr>` - JQ filter expression for JSON output
- `--query-file <path>` - Read JQ filter expression from file (use `-` for stdin)
- `--yes`, `-y` - Skip confirmation prompts (useful for scripts and automation)
- `--no-input` - Alias for `--yes` (non-interactive mode)
- `--limit <N>` - Limit number of results
- `--sort-by <field>` - Sort results by field (e.g., `created_time`, `last_edited_time`)
- `--desc` - Sort in descending order (use with `--sort-by`)
- `--error-format <mode>` - Error output format: `auto`, `text`, or `json`
- `--quiet` - Suppress non-essential output
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

## Exit Codes (Automation)

When a command fails, the process exit code is stable and intended for automation:

- `0` success
- `2` user/validation error
- `3` auth error
- `4` not found
- `5` rate limit
- `6` temporary failure (circuit breaker)
- `130` canceled (Ctrl+C / context canceled)

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
