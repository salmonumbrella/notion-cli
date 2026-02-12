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
- **MCP Integration** - AI search, SQL queries, and markdown editing via Notion's MCP server

## MCP Integration (Notion MCP Server)

The `ntn mcp` command group connects to Notion's official MCP server at `https://mcp.notion.com/mcp`, providing capabilities not available through the REST API.

### Setup

```bash
# One-time OAuth login (opens browser)
ntn mcp login

# Verify connection
ntn mcp status
```

### Commands

| Command | Description |
|---------|-------------|
| `ntn mcp login` | OAuth login to Notion MCP server |
| `ntn mcp status` | Check MCP authentication status |
| `ntn mcp search <query>` | Search workspace (add `--ai` for AI-powered search) |
| `ntn mcp fetch <url-or-id>` | Fetch page/database as markdown |
| `ntn mcp create` | Create pages with markdown content |
| `ntn mcp edit <page-id>` | Edit page content or properties |
| `ntn mcp query '<sql>' <url>...` | Query databases using SQL |
| `ntn mcp query --view <url>` | Execute a database view |
| `ntn mcp move <id>... --parent <id>` | Move pages to new parent |
| `ntn mcp duplicate <page-id>` | Duplicate a page with content |
| `ntn mcp comment list <id>` | List comments |
| `ntn mcp comment add <id>` | Add a comment |
| `ntn mcp teams` | List workspace teamspaces |
| `ntn mcp users` | List workspace users |
| `ntn mcp db create` | Create a database |
| `ntn mcp db update` | Update database schema |
| `ntn mcp tools` | List all available MCP tools |

### Unique MCP Features

These capabilities are only available through the MCP backend:

- **AI Search** - Semantic search using Notion's AI (`ntn mcp search --ai "find action items from last week"`)
- **SQL Queries** - Query databases with SQL syntax instead of Notion filter JSON
- **Connected App Search** - Search across Slack, Google Drive, and other connected apps
- **Markdown-Native Editing** - Edit pages using markdown content directly
- **Page Moving & Duplication** - Move pages between parents and duplicate with full content

### Example: SQL Query Workflow

```bash
# Fetch a database to get its data source URL
ntn mcp fetch https://notion.so/workspace/Tasks-abc123

# Query it with SQL
ntn mcp query 'SELECT * FROM "collection://abc123" WHERE Status = ?' collection://abc123 --params '["In Progress"]'
```

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
notion s "project notes"

# Get current user
notion u me
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

- `--ro` (`--results-only`) - For list-like responses, output just the `.results` array (useful for piping to `jq`).
- `--limit`, `--sb` (`--sort-by`), `--desc`, `--latest`, `--recent` - Apply client-side sorting/limiting when possible.

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

Every command has a short alias for quick scripting:

| Command | Alias | Subcommands |
|---------|-------|-------------|
| `page` | `p` | `g`et, `c`reate, `u`pdate, `d`elete, `ls` list, `props`, `prop`, `mv`, `dup`, `ex`port, `cb` create-batch, `ub` update-batch |
| `block` | `b` | `g`et, `ls` children, `ap`pend, `u`pdate, `d`elete, `add`, `add-toc`, `add-breadcrumb`, `add-divider`, `add-columns` |
| `db` | | `g`et, `q`uery, `c`reate, `u`pdate, `ls` list |
| `datasource` | `ds` | `g`et, `q`uery, `c`reate, `u`pdate, `ls` list, `t`emplates |
| `comment` | `c` | `g`et, `ls` list, `a`dd |
| `user` | `u` | `g`et, `ls` list, `me` |
| `file` | `f` | `g`et, `up`load, `ls` list |
| `search` | `s` | |
| `resolve` | `r` | |
| `open` | `o` | |
| `skill` | `sk` | |
| `import` | `im` | |
| `webhook` | `wh` | |
| `mcp` | | `login`, `logout`, `status`, `search`, `fetch`, `create`, `edit`, `comment`, `move`, `duplicate`, `query`, `teams`, `users`, `db`, `tools` |

### Authentication

```bash
notion auth login                  # Authenticate via browser (OAuth)
notion auth add-token              # Add integration token manually
notion auth status                 # Check authentication status
notion auth logout                 # Remove stored credentials
```

### Search (`s`)

```bash
notion s                           # Search all pages and databases
notion s "project notes"           # Search with query
notion s --fi page                 # Search only pages
notion s --fi database             # Search only databases
notion s "project" --all --ro      # Fetch all results (array only)
```

### Resolve (`r`, `res`)

```bash
notion r "Meeting Notes"           # Return candidate IDs (skill aliases + search)
notion r "Projects" --type database
notion r standup                   # Skill alias
```

### Users (`u`)

```bash
notion u me                        # Get current user
notion u ls                        # List all workspace users
notion u g <user-id>               # Get user by ID
```

### Pages (`p`)

```bash
notion p g <page-id>                              # Get page
notion p c --pa <id> --props <json>               # Create page
notion p cb --pa <id> --pages <json>               # Create multiple pages
notion p ub --pages <json>                         # Update multiple pages
notion p dup <page-id>                             # Duplicate page
notion p ex <page-id> --format markdown            # Export page content
notion p u <page-id> --props <json>                # Update page
notion p mv <page-id> --pa <new-parent-id>         # Move page
notion p d <page-id>                               # Delete page
notion p prop <page-id> <property-id>              # Get property
notion p props <page-id>                           # List properties (optionally simplified)
```

### Databases (`db`)

```bash
notion db g <database-id>                            # Get database
notion db q <database-id>                            # Query database
notion db q <database-id> --ds <id>                   # Query a specific data source
notion db c --pa <id> --props <json>                 # Create database
notion db u <database-id> --props <json>             # Update database
```

Query with filters and sorts:

```bash
# Query with filter
notion db q <database-id> \
  --fi '{"property":"Status","select":{"equals":"Done"}}'

# Agent-friendly shorthand filters (server-side; combined with --filter using AND)
notion db q <database-id> --status Done
notion db q <database-id> --assignee me
notion db q <database-id> --priority High

# Query with filter from file/stdin (avoids shell escaping issues)
notion db q <database-id> --fi @filter.json
cat filter.json | notion db q <database-id> --fi -

# Query with sorts
notion db q <database-id> \
  --sorts '[{"property":"Created","direction":"descending"}]'

# Fetch all results as an array
notion db q <database-id> --all --ro
```

### Blocks (`b`)

```bash
notion b g <block-id>                        # Get block
notion b ls <block-id>                       # Get children
notion b ls <block-id> --plain               # Get children (simplified: id/type/text)
notion b ap <parent-id> --ch <json>          # Append blocks
notion b u <block-id> --content <json>       # Update block
notion b d <block-id>                        # Delete block
```

Quick block creation:

```bash
notion b add-toc <parent-id>                   # Add table of contents
notion b add-toc <parent-id> --color blue      # With color
notion b add-breadcrumb <parent-id>            # Add breadcrumb navigation
notion b add-divider <parent-id>               # Add horizontal divider
notion b add-columns <parent-id> --columns 3   # Add 3-column layout (2-5)
```

### Comments (`c`)

```bash
notion c ls <block-id>                       # List comments
notion c a <block-id> --text "Comment"       # Add comment

# Desire paths (agent-friendly)
notion c <page-id> "Looks great!"            # Add comment (positional)
notion c a <page-id> "Looks great!"          # Add comment (positional)
```

### File Uploads (`f`)

```bash
notion f up <filepath>                       # Upload file
notion f g <upload-id>                       # Get upload status
notion f ls                                  # List file uploads
```

Upload and attach to page property:

```bash
notion f up ./receipt.pdf --page <page-id> --prop "Attachments"
```

### Data Sources (`ds`)

```bash
notion ds t                                    # List available templates
notion ds c --template <name>                  # Create from template
notion ds g <datasource-id>                    # Get data source
notion ds q <datasource-id>                    # Query data source
notion ds u <datasource-id> <json>             # Update data source
```

Query with filters, sorts, and selection:

```bash
# Filter by property value
notion ds q <datasource-id> \
  --fi '{"property":"Status","status":{"equals":"Active"}}'

# Read filter from file (avoids shell escaping)
notion ds q <datasource-id> --ff filter.json

# Sort by timestamp (shorthand)
notion ds q <datasource-id> --sb last_edited_time --desc

# Sort with full Notion sorts JSON
notion ds q <datasource-id> \
  --sorts '[{"property":"Priority","direction":"ascending"}]'

# Client-side select/status filtering (exact, not-equals, or regex)
notion ds q <datasource-id> \
  --select-property "Category" --select-equals "Engineering"
notion ds q <datasource-id> \
  --select-property "Status" --select-not "Done"
notion ds q <datasource-id> \
  --select-property "Category" --select-match "(?i)eng"

# Fetch all results as a plain array
notion ds q <datasource-id> --limit 0 --ro

# Combine: filter + sort + limit
notion ds q <datasource-id> \
  --fi '{"property":"Status","status":{"equals":"Active"}}' \
  --sb created_time --desc --limit 20 --ro
```

### Fetch

```bash
notion fetch <notion-url>                              # Fetch page or database by URL
notion fetch <notion-url> --type page                  # Force page fetch
notion fetch <notion-url> --type database              # Force database fetch
```

### Webhooks (`wh`)

```bash
notion wh verify --secret <secret> --payload payload.json --signature <sig>
notion wh verify --secret <secret> --payload payload.json           # Compute signature
notion wh parse --payload payload.json
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
$ notion u me
NAME         EMAIL                  TYPE
John Doe     john@example.com       person
```

### JSON

Machine-readable output:

```bash
$ notion u me --output json
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
$ notion s "project" --output ndjson
{"object":"page", ...}
{"object":"page", ...}
```

## Examples

### Create a Page with Content

```bash
# Create a page
notion p c \
  --pa <parent-id> \
  --props '{"title":[{"text":{"content":"New Page"}}]}'

# Add content blocks
notion b ap <page-id> \
  --ch '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"text":{"content":"Hello world"}}]}}]'
```

### Query and Filter Database

```bash
# Get all completed tasks
notion db q <database-id> \
  --fi '{"property":"Status","select":{"equals":"Done"}}' \
  -o json | jq '.results[].properties.Name'
```

### Duplicate and Export Pages

```bash
# Duplicate a page with its content
notion p dup <page-id>

# Export page content to Markdown
notion p ex <page-id> --format markdown
```

### Batch Updates

```bash
# Update multiple pages in one command
notion p ub --pages '[{"id":"<page-id>","properties":{"Status":{"status":{"name":"Done"}}}}]'
```

### Automation

Use `--yes` (or `--no-input`) to skip confirmations, `--limit` to control result size, and `--sb` for ordering:

```bash
# Delete without confirmation
notion b d <block-id> -y

# Get recent items
notion s "project" --limit 5 --sb created_time --desc

# Pipeline example
notion db q <database-id> -o json | jq '.results[] | .id'

# Filter JSON output with jq expression
notion p g <page-id> -o json --jq '.properties.Status'

# Use a file for longer jq expressions
notion p g <page-id> -o json --qf ./query.jq

# Project fields without jq
notion db q <database-id> --ro --fields id,name,created_time

# JSONPath extraction
notion db q <database-id> --jsonpath '$.results[0].id'

# Latest shortcuts
notion s "project" --latest
notion s "project" --recent 5

# Fail if empty
notion s "project" --fe --limit 1
```

### JSON Input Shortcuts

Flags that accept JSON also support reading from a file or stdin:

```bash
# From a file
notion db q <database-id> --fi @filter.json

# From a file (properties)
notion p u <page-id> --props @props.json

# From a file (properties flag)
notion p u <page-id> --props-file props.json

# From stdin
cat filter.json | notion db q <database-id> --fi -

# From stdin (heredoc)
cat <<'JSON' | notion p u <page-id> --props -
{"Status":{"status":{"name":"Done"}}}
JSON
```

### Debug Mode

```bash
notion --debug u me
# Shows HTTP request/response details to stderr
```

## Global Flags

All commands support these flags:

- `--output <format>` - Output format: `text`, `json`, `ndjson`, `table`, or `yaml` (default: text)
- `--query <expr>` / `--jq <expr>` - JQ filter expression for JSON output
- `--query-file <path>` / `--qf` - Read JQ filter expression from file (use `-` for stdin)
- `--fields <paths>` / `--pick <paths>` / `--fds` - Project fields (comma-separated paths, `key=path` to rename)
- `--jsonpath <expr>` - Extract a value using JSONPath
- `--results-only` / `--ro` - Output just the results array (useful for piping to jq)
- `--latest` / `--recent <n>` - Shortcut for `--sb created_time --desc --limit N`
- `--fail-empty` / `--fe` - Exit with error when results are empty
- `--sort-by <field>` / `--sb` - Sort results by field (e.g., `created_time`, `last_edited_time`)
- `--desc` - Sort in descending order (use with `--sb`)
- `--workspace <name>`, `-w` - Workspace to use (overrides NOTION_WORKSPACE)
- `--debug` - Enable debug output (shows API requests/responses)
- `--yes`, `-y` / `--no-input` - Skip confirmation prompts (useful for scripts and automation)
- `--limit <N>` - Limit number of results
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
- `1` system/internal error
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
