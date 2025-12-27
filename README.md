# Notion CLI — Wiki in your Terminal

Command-line interface for the Notion API with secure authentication, file uploads, and data source management.

## Features

- **Authentication** - OAuth browser login or integration token, stored securely in OS keychain
- **Blocks** - Manage content blocks with quick helpers for TOC, breadcrumbs, dividers, and columns
- **Bulk Operations** - Bulk update or archive database pages with `--where` filters
- **Comments** - List and add comments to pages
- **Data Sources** - Create and query Notion data sources (API v2025-09-03)
- **Databases** - Create, query, and update databases with full property support
- **Export** - Export pages to Markdown or JSON block trees
- **File Uploads** - Upload files with automatic chunking for large files (multi-part upload)
- **Fetch by URL** - Fetch pages or databases by Notion URL
- **Import** - Import Markdown or CSV into Notion
- **jq Integration** - Filter JSON output with jq expressions via `--query`
- **MCP Integration** - AI search, SQL queries, and markdown editing via Notion's MCP server
- **Pages** - Create, update, move, duplicate, and batch-operate on pages
- **Search** - Full-text search across pages and databases with filtering and pagination
- **Skill File** - Manage aliases for databases, users, and pages for agent-friendly workflows
- **Webhooks** - Verify and parse webhook payloads
- **Workers Passthrough** - Run Notion's official Workers CLI via `ntn workers ...` with pinned versioning

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
ntn auth login
```

**Integration token:**
```bash
ntn auth add-token
# You'll be prompted securely for the token
```

### 2. Test Authentication

```bash
ntn auth status
```

### 3. Start Using

```bash
# Search your workspace
ntn s "project notes"

# Get current user
ntn u me

# Open a page in the browser
ntn o <page-id>
```

## Canonical Subcommand Aliases

All subcommands automatically support these short aliases across every command group:

| Subcommand | Aliases |
|------------|---------|
| `list` | `ls` |
| `get` | `g`, `show` |
| `create` | `c`, `mk` |
| `update` | `u`, `up`, `edit` |
| `delete` | `d`, `rm` |
| `search` | `q`, `query`, `find` |

These work everywhere: `ntn p g` (page get), `ntn db q` (database query), `ntn c ls` (comment list), etc.

## Root-Level Desire Paths

Common operations are available directly at the root level without specifying a command group:

```bash
ntn login                    # Same as: ntn auth login
ntn logout                   # Same as: ntn auth logout
ntn whoami                   # Same as: ntn u me
ntn list                     # Same as: ntn p ls
ntn get <id>                 # Auto-detects entity type (page/db/block)
ntn create                   # Same as: ntn p c (with smart defaults)
ntn delete <id>              # Same as: ntn p d <id>
```

## Command Reference

Every command has a short alias for quick scripting:

| Command | Aliases | Subcommands |
|---------|---------|-------------|
| `page` | `p`, `pages` | `g`et, `c`reate, `u`pdate, `d`elete, `ls` list, `props`, `prop`, `mv`, `dup`, `ex`port, `cb` create-batch, `ub` update-batch, `sync`, `enrich` |
| `block` | `b`, `blocks` | `g`et, `ls` children, `ap`pend, `u`pdate, `d`elete, `add`, `add-toc`, `add-breadcrumb`, `add-divider`, `add-columns` |
| `db` | `database`, `databases` | `g`et, `q`uery, `c`reate, `u`pdate, `ls` list, `bak` backup |
| `datasource` | `ds` | `g`et, `q`uery, `c`reate, `u`pdate, `ls` list, `t`emplates |
| `comment` | `c`, `comments` | `g`et, `ls` list, `a`dd |
| `user` | `u`, `users` | `g`et, `ls` list, `me` |
| `file` | `f`, `files` | `g`et, `up`load, `ls` list |
| `search` | `s`, `find` | |
| `resolve` | `r`, `res` | |
| `open` | `o` | |
| `fetch` | | |
| `import` | `im` | `csv` |
| `bulk` | | `update`, `archive` |
| `skill` | `sk` | `init`, `sync`, `path`, `edit` |
| `config` | `cfg` | `ls` list, `g`et, `set`-default, `e`dit |
| `workspace` | `ws` | `i`nfo |
| `webhook` | `wh` | `verify`, `parse` |
| `api` | | `request`, `status` |
| `mcp` | | `login`, `logout`, `status`, `s`earch, `f`etch, `c`reate, `e`dit, `cm` comment, `mv` move, `dup`licate, `q`uery, `mn` meeting-notes, `tm` teams, `u`sers, `db`, `tools`, `call` |
| `workers` | `wk` | local `new` scaffold + `doctor`, plus passthrough to official Notion Workers CLI (`deploy`, `runs`, `exec`, etc.) |

---

### Authentication (`auth`)

```bash
ntn auth login                  # Authenticate via browser (OAuth)
ntn auth login --no-browser     # Print OAuth URL without auto-opening browser
ntn auth add-token              # Add integration token manually
ntn auth status                 # Check authentication status
ntn auth logout                 # Remove stored credentials
ntn auth switch                 # Switch workspace (--account, --workspace)
```

---

### Search (`s`, `find`)

```bash
ntn s                           # Search all pages and databases
ntn s "project notes"           # Search with query
ntn s --fi page                 # Search only pages
ntn s --fi database             # Search only databases
ntn s "project" --li            # Light search (id, object, title, url)
ntn s "project" --all --ro      # Fetch all results (array only)
ntn s "project" --sort recent   # Sort by most recent
```

---

### Resolve (`r`, `res`)

Resolves names to Notion IDs via skill aliases and search:

```bash
ntn r "Meeting Notes"           # Return candidate IDs (skill aliases + search)
ntn r "Projects" --type database
ntn r standup                   # Skill alias
ntn r "Tasks" --check           # Silent check (exit code only, no output)
```

---

### Open (`o`)

Opens a Notion page or database in the browser:

```bash
ntn o <page-id>                 # Open by ID
ntn o <notion-url>              # Open by URL
```

---

### Users (`u`)

```bash
ntn u me                        # Get current user
ntn u ls                        # List all workspace users
ntn u ls --li                   # Light list (id, name, email, type)
ntn u g <user-id>               # Get user by ID
```

---

### Pages (`p`)

#### Basic operations

```bash
ntn p g <page-id>                              # Get page
ntn p g <page-id> --li                         # Light page lookup (id, object, title, url; compact JSON by default)
ntn p g <page-id> --enrich                     # Get page with parent_title and child_count
ntn p g <page-id> --editable                   # Get page with only editable properties
ntn p g <page-id> --include-children           # Get page with child blocks
ntn p g <page-id> --include-children --children-depth 2  # Include nested children
ntn p c --pa <id> --props <json>               # Create page
ntn p u <page-id> --props <json>               # Update page
ntn p d <page-id>                              # Delete page (archive)
ntn p mv <page-id> --pa <new-parent-id>        # Move page
ntn p dup <page-id>                            # Duplicate page with content
ntn p ex <page-id> --format markdown           # Export page content
```

#### Properties

```bash
ntn p props <page-id>                          # List all properties
ntn p props <page-id> --simple                 # Simplified values
ntn p props <page-id> --types                  # Include property types
ntn p props <page-id> --only-set               # Only properties with values
ntn p props <page-id> --with-values            # Include resolved values
ntn p prop <page-id> <property-id>             # Get a single property
```

#### Property shorthand flags

Instead of raw JSON, use shorthand flags for common properties:

```bash
ntn p c --pa <id> --title "My Page"            # Set title
ntn p u <page-id> --status "Done"              # Set status property
ntn p u <page-id> --priority "High"            # Set priority property
ntn p u <page-id> --assignee me                # Set assignee (user ID or alias)
ntn p u <page-id> --mention @user-id           # Add mention in rich text
ntn p u <page-id> --rich-text "**bold** text"  # Markdown-aware rich text
ntn p u <page-id> --dry-run                    # Preview changes without applying
```

#### Batch operations

```bash
ntn p cb --pa <id> --pages <json>              # Create multiple pages
ntn p ub --pages <json>                        # Update multiple pages
```

#### Sync

```bash
ntn p sync --pa <parent-id>                    # Sync pages
ntn p sync --pa <parent-id> --dry-run          # Preview sync changes
```

---

### Databases (`db`)

#### Basic operations

```bash
ntn db g <database-id>                         # Get database
ntn db ls                                      # List databases
ntn db ls --li                                 # Light list (id, object, title, url)
ntn db ls --title-match "Tasks"                # Filter by title
ntn db c --pa <id> --props <json>              # Create database
ntn db u <database-id> --props <json>          # Update database
ntn db u <database-id> --dry-run               # Preview update
ntn db bak <database-id>                       # Backup database
```

#### Query

```bash
ntn db q <database-id>                         # Query all pages
ntn db q <database-id> --ds <id>               # Query specific data source
ntn db q <database-id> --all --ro              # Fetch all results as array
ntn db q <database-id> --page-size 50          # Set page size
ntn db q <database-id> --start-cursor <cursor> # Resume pagination
```

#### Filters

```bash
# JSON filter
ntn db q <database-id> \
  --fi '{"property":"Status","select":{"equals":"Done"}}'

# Filter from file (avoids shell escaping)
ntn db q <database-id> --fi @filter.json
ntn db q <database-id> --ff filter.json

# Filter from stdin
cat filter.json | ntn db q <database-id> --fi -

# Property shorthand filters (server-side; combined with --filter using AND)
ntn db q <database-id> --status Done
ntn db q <database-id> --assignee me
ntn db q <database-id> --priority High
```

#### Sorts

```bash
# JSON sorts
ntn db q <database-id> \
  --sorts '[{"property":"Created","direction":"descending"}]'

# Sorts from file
ntn db q <database-id> --sorts-file sorts.json
```

#### Client-side select filtering

```bash
ntn db q <database-id> \
  --select-property "Category" --select-equals "Engineering"
ntn db q <database-id> \
  --select-property "Status" --select-not "Done"
ntn db q <database-id> \
  --select-property "Category" --select-match "(?i)eng"
```

---

### Blocks (`b`)

#### Basic operations

```bash
ntn b g <block-id>                        # Get block
ntn b ls <block-id>                       # List children
ntn b ls <block-id> --plain               # Simplified output (id/type/text)
ntn b ls <block-id> --depth 3             # Recursively fetch nested children
ntn b ls <block-id> --all                 # Fetch all children (paginated)
ntn b ap <parent-id> --ch <json>          # Append blocks
ntn b ap <parent-id> --chf blocks.json    # Append blocks from file
ntn b ap <parent-id> --ch <json> --dr     # Dry-run append
ntn b u <block-id> --content <json>       # Update block
ntn b d <block-id>                        # Delete block
```

#### Quick block creation

```bash
ntn b add paragraph <parent-id> "Hello world"
ntn b add heading <parent-id> "Title" --level 1
ntn b add bullet <parent-id> "List item"
ntn b add code <parent-id> 'fmt.Println("hi")' --language go
ntn b add todo <parent-id> "Buy milk"
ntn b add callout <parent-id> "Note" --emoji "💡"
ntn b add-toc <parent-id>                   # Table of contents
ntn b add-breadcrumb <parent-id>            # Breadcrumb navigation
ntn b add-divider <parent-id>               # Horizontal divider
ntn b add-columns <parent-id> --columns 3   # Column layout (2-5)
```

#### File & image uploads to page body

```bash
ntn b add file <parent-id> --file ./report.pdf
ntn b add file <parent-id> --file ./data.txt --caption "Raw data"
ntn b add image <parent-id> --file ./photo.jpg --caption "Team offsite"
```

**Supported file extensions:**

| Category | Extensions |
|----------|-----------|
| Document | `.pdf` `.txt` `.json` `.doc` `.docx` `.dotx` `.xls` `.xlsx` `.xltx` `.ppt` `.pptx` `.potx` |
| Image | `.gif` `.heic` `.jpeg` `.jpg` `.png` `.svg` `.tif` `.tiff` `.webp` `.ico` |
| Audio | `.aac` `.mp3` `.m4a` `.m4b` `.mp4` `.ogg` `.wav` `.wma` |
| Video | `.mp4` `.mov` `.avi` `.mkv` `.webm` `.mpeg` `.flv` `.wmv` |

> **Note:** `.md` files are **not supported** by the Notion API. Rename to `.txt` before uploading:
> ```bash
> cp notes.md notes.txt && ntn b add file <parent-id> --file ./notes.txt
> ```

---

### Comments (`c`)

```bash
ntn c ls <page-id>                       # List comments
ntn c ls <page-id> --li                  # Light list (id, discussion_id, text, created_by)
ntn c ls <page-id> --all                 # List all comments (paginated)
ntn c g <comment-id>                     # Get comment
ntn c a <page-id> --text "Comment"       # Add comment

# Desire paths (positional arguments)
ntn c <page-id> "Looks great!"           # Add comment (auto-detects)
ntn c a <page-id> "Looks great!"         # Add comment (positional text)
```

---

### File Uploads (`f`)

```bash
ntn f up <filepath>                       # Upload file
ntn f up ./receipt.pdf --page <page-id> --prop "Attachments"  # Upload and attach
ntn f g <upload-id>                       # Get upload status
ntn f ls                                  # List file uploads
ntn f ls --li                             # Light list (id, file_name, status, size, timestamps)
```

---

### Data Sources (`ds`)

```bash
ntn ds t                                    # List available templates
ntn ds c --template <name>                  # Create from template
ntn ds g <datasource-id>                    # Get data source
ntn ds q <datasource-id>                    # Query data source
ntn ds u <datasource-id> <json>             # Update data source
ntn ds ls                                   # List data sources
ntn ds ls --li                              # Light list (id, object, title, url)
```

Query with filters, sorts, and selection (same flags as `db q`):

```bash
ntn ds q <datasource-id> \
  --fi '{"property":"Status","status":{"equals":"Active"}}' \
  --sb created_time --desc --limit 20 --ro
```

---

### Fetch

```bash
ntn fetch <notion-url>                    # Auto-detect page or database
ntn fetch <notion-url> --type page        # Force page fetch
ntn fetch <notion-url> --type database    # Force database fetch
```

---

### Import (`im`)

```bash
ntn im --file content.md                  # Import markdown as Notion blocks
ntn im --file content.md --dry-run        # Preview import
ntn im --file content.md --batch-size 50  # Control batch size
ntn im csv --file data.csv                # Import CSV to database
ntn im csv --file data.csv --mapping <json> --dry-run
```

---

### Bulk Operations (`bulk`)

```bash
ntn bulk update <database-id> \
  --where '{"property":"Status","select":{"equals":"Stale"}}' \
  --set '{"Status":{"select":{"name":"Archived"}}}' \
  --dry-run

ntn bulk archive <database-id> \
  --where '{"property":"Done","checkbox":{"equals":true}}' \
  --limit 100 --dry-run
```

---

### Skill File (`sk`)

Manage the skill file that stores aliases for databases, users, and pages:

```bash
ntn sk init                              # Initialize by scanning workspace
ntn sk sync                              # Update skill file
ntn sk sync --add-new                    # Add newly discovered items
ntn sk path                              # Print skill file path
ntn sk edit                              # Open skill file in editor
```

---

### Config (`cfg`)

```bash
ntn cfg ls                               # List workspaces
ntn cfg g                                # Get workspace config
ntn cfg set <workspace>                  # Set default workspace
ntn cfg e                                # Edit config file
```

---

### Workspace (`ws`)

```bash
ntn ws i                                 # Get current workspace info
```

---

### Webhooks (`wh`)

```bash
ntn wh verify --secret <secret> --payload payload.json --signature <sig>
ntn wh verify --secret <secret> --payload payload.json   # Compute signature
ntn wh parse --payload payload.json
```

---

### API

Raw API access and diagnostics:

```bash
ntn api request GET /users
ntn api request POST /search --body '{"query":"project"}'
ntn api request POST /databases/<id>/query --body @query.json
ntn api request GET /blocks/<id>/children --paginate
ntn api status                           # Show rate limit status
ntn api status --refresh                 # Refresh rate limit info
```

---

### MCP Integration (Notion MCP Server)

The `ntn mcp` command group connects to Notion's official MCP server at `https://mcp.notion.com/mcp`, providing capabilities not available through the REST API.

#### Setup

```bash
ntn mcp login                            # One-time OAuth login (opens browser)
ntn mcp status                           # Verify connection
ntn mcp logout                           # Remove MCP token
```

#### Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `ntn mcp search <query>` | `s` | Search workspace (add `-a` for AI search) |
| `ntn mcp fetch <url-or-id>` | `f` | Fetch page/database as markdown |
| `ntn mcp create` | `c` | Create pages with markdown content |
| `ntn mcp edit <page-id>` | `e` | Edit page content or properties |
| `ntn mcp query '<sql>' <url>...` | `q` | Query databases using SQL |
| `ntn mcp move <id>... -p <id>` | `mv` | Move pages to new parent |
| `ntn mcp duplicate <page-id>` | `dup` | Duplicate a page with content |
| `ntn mcp comment list <id>` | `cm ls` | List comments (use `--include-all-blocks` for block threads) |
| `ntn mcp comment add <id>` | `cm a` | Add a page/block-context comment |
| `ntn mcp teams` | `tm` | List workspace teamspaces |
| `ntn mcp users` | `u` | List workspace users |
| `ntn mcp db create` | `db c` | Create a database from SQL DDL schema |
| `ntn mcp db update` | `db u` | Update data source schema with SQL DDL statements |
| `ntn mcp meeting-notes` | `mn` | Query your meeting notes data source |
| `ntn mcp call <tool-name> [args-json]` | `invoke` | Invoke any MCP tool directly |
| `ntn mcp tools` | | List all available MCP tools (`--schema` includes JSON schemas) |

#### MCP Create Flags

```bash
ntn mcp c --parent <id> --title "Page Title" --content "Markdown body"
ntn mcp c --parent <id> --file content.md     # Content from file
ntn mcp c --parent <id> --data-source <id> --properties <json>
ntn mcp c --data-source <id> --template-id <template-id> --properties '{"Task":"Follow up"}'
```

#### MCP Edit Flags

```bash
ntn mcp e <page-id> --replace "new content"                              # Replace all content
ntn mcp e <page-id> --replace-range "start...end" --new "new section"    # Replace selected range
ntn mcp e <page-id> --insert-after "start...end" --new "appended text"   # Insert after selected content
ntn mcp e <page-id> --properties <json>                                  # Edit properties
ntn mcp e <page-id> --apply-template <template-id>                       # Apply template to page
ntn mcp e <page-id> --replace "..." --allow-deleting-content             # Permit deleting child content
```

#### MCP Query Flags

```bash
ntn mcp q 'SELECT * FROM "collection://abc"' collection://abc
ntn mcp q 'SELECT * FROM "collection://abc" WHERE Status = ?' collection://abc -P '["Done"]'
ntn mcp q -v "https://notion.so/workspace/Tasks-abc?v=def"   # Execute saved view
ntn mcp mn --filter '{"operator":"and","filters":[{"property":"title","filter":{"operator":"string_contains","value":{"type":"exact","value":"standup"}}}]}'
ntn mcp fetch collection://abc                                 # Fetch a specific data source
ntn mcp fetch https://workspace.notion.site/My-Site-abc123     # Fetch a Notion Site page
ntn mcp cm ls <page-id> --include-all-blocks                   # Include block discussions
ntn mcp call notion-fetch --args '{"id":"https://www.notion.so/..."}' # Generic MCP tool invocation
```

#### Unique MCP Features

These capabilities are only available through the MCP backend:

- **AI Search** - Semantic search using Notion's AI (`ntn mcp s -a "find action items from last week"`)
- **SQL Queries** - Query databases with SQL syntax instead of Notion filter JSON
- **Connected App Search** - Search across Slack, Google Drive, and other connected apps
- **Markdown-Native Editing** - Edit pages using markdown content directly
- **Page Moving & Duplication** - Move pages between parents and duplicate with full content

---

### Workers Integration (Official CLI Passthrough)

Use this command group when you want official Notion Workers behavior without replacing this CLI:

```bash
ntn workers new my-worker
ntn workers status my-worker
ntn workers status my-worker --no-compare
ntn workers upgrade my-worker --dry-run
ntn workers upgrade my-worker --plan
ntn workers upgrade my-worker --force
ntn workers upgrade my-worker --from-metadata --force
ntn workers deploy
ntn workers runs list
ntn workers exec sayHello --local -d '{"name":"World"}'
ntn workers doctor
```

By default, this proxies to:

```bash
npx --yes ntn@<workers_cli_version from internal/workers/compat.json> workers ...
```

Environment overrides:

- `NTN_WORKERS_CLI_VERSION` (default from `internal/workers/compat.json`)
- `NTN_WORKERS_NPX_BIN` (default `npx`)

Scaffold metadata:

- `ntn workers new` writes `.ntn-workers-template.json` in the project root with pinned template commit and CLI version.
- `ntn workers status [path]` reports pinned vs project template commit state (`in_sync`, `behind`, `ahead`, `diverged`, etc).
- `ntn workers status --no-compare` skips GitHub compare API calls and reports local pin mismatch only.
- `ntn workers upgrade [path]` targets the CLI's current pinned template commit by default (good for weekly compat pin updates).
- `ntn workers upgrade --plan` computes a file-level change preview (`add`, `modify`, and `extra` path counts) without writing files.
- `ntn workers upgrade --from-metadata` re-syncs to the commit currently stored in `.ntn-workers-template.json`.
- `ntn workers upgrade` requires `--force` to apply file overwrites; use `--dry-run` to preview.

---

## Global Flags

All commands support these flags:

### Output

| Flag | Short | Aliases | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `--out`, `--format` | Output format: `text`, `json`, `ndjson`, `jsonl`, `table`, `yaml` |
| `--json` | `-j` | | Shorthand for `--output json` |
| `--compact-json` | | `--cj` | Emit compact single-line JSON (enabled by default with `--light` unless overridden) |
| `--quiet` | | | Suppress non-essential output |
| `--error-format` | | | Error output format: `auto`, `text`, `json` |

### Filtering & Projection

| Flag | Short | Aliases | Description |
|------|-------|---------|-------------|
| `--query` | `-q` | `--jq`, `--qr` | JQ filter expression (supports path aliases) |
| `--query-file` | | `--qf` | Read JQ expression from file (use `-` for stdin) |
| `--fields` | | `--pick`, `--fds` | Project fields (comma-separated, `key=path` to rename) |
| `--jsonpath` | | | Extract value using JSONPath |
| `--items-only` | | `--results-only`, `--io`, `--ro`, `--i` | Output just the results array (jq should use `.[]`, not `.results[]`) |
| `--fail-empty` | | `--fe` | Exit with error when results are empty |

### Sorting & Limiting

| Flag | Short | Aliases | Description |
|------|-------|---------|-------------|
| `--sort-by` | | `--sb` | Sort by field (supports path aliases, e.g. `ct` for `created_time`) |
| `--desc` | | | Sort descending (use with `--sb`) |
| `--limit` | | | Limit number of results |
| `--latest` | | | Shortcut for `--sb created_time --desc --limit 1` |
| `--recent` | | | Shortcut for `--sb created_time --desc --limit N` |

### Session & Control

| Flag | Short | Aliases | Description |
|------|-------|---------|-------------|
| `--workspace` | `-w` | | Workspace to use (overrides `NOTION_WORKSPACE`) |
| `--debug` | | | Show API requests/responses on stderr |
| `--yes` | `-y` | `--no-input` | Skip confirmation prompts |
| `--help` | | | Show help for any command |
| `--version` | | | Show version information |

### Common Per-Command Flags

These flags appear on multiple commands:

| Flag | Aliases | Used by | Description |
|------|---------|---------|-------------|
| `--parent` | `--pa` | page create/move/dup, db create/update, mcp create | Parent page or database ID |
| `--properties` | `--props` | page create/update, db create/update | Properties as JSON |
| `--properties-file` | `--props-file` | page create/update | Read properties from file |
| `--datasource` | `--ds` | db query, page create/dup | Data source ID |
| `--filter` | `--fi` | db query, ds query, search | Filter expression (JSON or `@file`) |
| `--filter-file` | `--ff` | db query, ds query | Read filter from file |
| `--children` | `--ch` | block append/update | Block children as JSON |
| `--children-file` | `--chf` | block append | Read children from file |
| `--dry-run` | `--dr` | page update/sync, db update, block append/update, bulk, import | Preview without applying |
| `--light` | `--li` | user list, search, page list/get, db list, ds list, comment list, file list | Compact lookup payload (IDs + key fields) |
| `--all` | | db query, ds query, search, block children, comment list | Fetch all pages |
| `--page-size` | | db query, ds query, search, block children | Results per page |
| `--start-cursor` | | db query, ds query, search, block children | Resume pagination |

---

## Configuration

### Environment Variables

- `NOTION_TOKEN` - API token (alternative to keyring storage)
- `NOTION_CREDENTIALS_DIR` - Credential storage root for keyring fallback files (`<dir>/notion-cli/keyring`)
- `OPENCLAW_CREDENTIALS_DIR` - Shared credentials root used when `NOTION_CREDENTIALS_DIR` is not set
- `NOTION_KEYRING_PASSWORD` - File-keyring passphrase for non-interactive/headless environments
- `NOTION_NO_BROWSER` - Set truthy value (`1`, `true`, `yes`, `on`) to skip browser auto-open in `auth login`
- `NOTION_WORKSPACE` - Default workspace name for multi-workspace support
- `NOTION_OUTPUT` - Output format: `text` (default), `json`, `ndjson`, `table`, or `yaml`
- `NOTION_API_BASE_URL` - Override Notion API base URL (useful for proxies and tests)
- `NOTION_NO_UPDATE_CHECK` - Set to any value to disable update checks (also auto-disabled when stdout is not a TTY)
- `NO_COLOR` - Set to any value to disable colors (standard convention)

OpenClaw compatibility: when present, `~/.openclaw/.env` is auto-loaded at startup.
Values from that file are only applied for variables that are not already set in the process environment.

### Config File (Optional)

notion-cli supports a YAML configuration file at `~/.config/notion-cli/config.yaml`:

```yaml
output: json
color: always
default_workspace: personal
```

CLI flags always override config file settings.

---

## Output Formats

### Text (default)

Human-readable tables with colors:

```bash
$ ntn u me
NAME         EMAIL                  TYPE
Example User     user@example.test       person
```

### JSON

```bash
$ ntn u me -o json
# or: ntn u me -j
{
  "id": "user_123",
  "name": "Example User",
  "type": "person"
}
```

### NDJSON / JSONL

Newline-delimited JSON (one object per line):

```bash
$ ntn s "project" -o ndjson
{"object":"page", ...}
{"object":"page", ...}
```

### Table

Formatted ASCII table output:

```bash
$ ntn db q <database-id> -o table
```

### YAML

```bash
$ ntn u me -o yaml
```

Data always goes to stdout, errors and progress to stderr for clean piping.

---

## JSON Input Shortcuts

Flags that accept JSON also support reading from files or stdin:

```bash
# @file syntax
ntn db q <database-id> --fi @filter.json
ntn p u <page-id> --props @props.json

# Dedicated file flags
ntn p u <page-id> --props-file props.json
ntn db q <database-id> --ff filter.json

# Stdin with -
cat filter.json | ntn db q <database-id> --fi -

# Heredoc
cat <<'JSON' | ntn p u <page-id> --props -
{"Status":{"status":{"name":"Done"}}}
JSON
```

---

## Path Aliases

Path aliases shorten jq/field/jsonpath/sort expressions. Supported in `--query`/`--jq`, `--fields`/`--pick`, `--jsonpath`, and `--sort-by`.

Alias rewrite applies to lowercase dot-path segments.
Use quoted bracket keys to force a literal (e.g. `.properties["st"]`).
Queries loaded from `--query-file` are also normalized.

| Canonical key | Alias(es) |
|---|---|
| `properties` | `props`, `pr` |
| `rich_text` | `rt` |
| `plain_text` | `pt`, `p` |
| `results` | `rs` |
| `object` | `ob` |
| `parent` | `pa` |
| `children` | `ch` |
| `has_children` | `hc` |
| `created_time` | `ct` |
| `last_edited_time` | `lt` |
| `created_by` | `cb` |
| `last_edited_by` | `lb` |
| `archived` | `ar` |
| `in_trash` | `it` |
| `public_url` | `pu` |
| `data_sources` | `ds` |
| `data_source_id` | `dsi` |
| `database_id` | `dbi` |
| `page_id` | `pid` |
| `workspace_id` | `wid` |
| `discussion_id` | `did` |
| `comment_id` | `cid` |
| `parent_title` | `ptt` |
| `child_count` | `cc` |
| `next_cursor` | `nc` |
| `has_more` | `hm` |
| `start_cursor` | `sc` |
| `page_size` | `ps` |
| `sorts` | `so` |
| `filter` | `fi` |
| `query` | `qy` |
| `multi_select` | `ms` |
| `phone_number` | `ph` |
| `time_zone` | `tz` |
| `unique_id` | `uid` |
| `upload_url` | `uu` |
| `expiry_time` | `et` |
| `file_name` | `fn` |
| `mime_type` | `mt` |
| `is_inline` | `ii` |
| `initial_data_source` | `ids` |
| `verification_token` | `vt` |
| `_meta` | `meta` |
| `status` | `st` |
| `select` | `sl` |
| `relation` | `rl` |
| `people` | `pe` |
| `checkbox` | `cbx` |
| `number` | `nu` |
| `files` | `fl` |
| `content` | `co` |
| `text` | `tx` |
| `title` | `ti`, `t` |
| `name` | `nm` |
| `type` | `ty` |
| `url` | `ur` |
| `cover` | `cv` |
| `icon` | `ic` |

---

## Examples

### Pipeline & Extraction

```bash
# Filter JSON output with jq
ntn p g <page-id> -j --jq '.properties.Status'

# Same query using path aliases
ntn p g <page-id> -j --jq '.pr["Invoice Alert"].rt[0].p'

# Read jq from file
ntn p g <page-id> -j --qf ./query.jq

# Project specific fields
ntn db q <database-id> --i --fields id,name,created_time

# Rename fields during projection
ntn db q <database-id> --i --fields id,alert=pr["Invoice Alert"].rt.0.p

# JSONPath extraction
ntn db q <database-id> --jsonpath '$.results[0].id'

# JSONPath with aliases
ntn db q <database-id> --jsonpath '$.rs[0].pr["Invoice Alert"].rt[0].p'
```

### Sorting & Limiting

```bash
# Sort by alias
ntn s "project" --sb ct --desc --limit 1

# Latest / recent shortcuts
ntn s "project" --latest
ntn s "project" --recent 5

# Fail if empty (useful in scripts)
ntn s "project" --fe --limit 1
```

### Create a Page with Content

```bash
# Create with title shorthand
ntn p c --pa <parent-id> --title "New Page"

# Create with full properties JSON
ntn p c --pa <parent-id> --props '{"title":[{"text":{"content":"New Page"}}]}'

# Add content blocks
ntn b ap <page-id> \
  --ch '[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"text":{"content":"Hello world"}}]}}]'
```

### Query and Filter Database

```bash
ntn db q <database-id> \
  --fi '{"property":"Status","select":{"equals":"Done"}}' \
  -j | jq '.results[].properties.Name'
```

### Bulk Operations

```bash
# Mark all "Stale" items as "Archived" (preview first)
ntn bulk update <database-id> \
  --where '{"property":"Status","select":{"equals":"Stale"}}' \
  --set '{"Status":{"select":{"name":"Archived"}}}' \
  --dry-run

# Archive completed tasks
ntn bulk archive <database-id> \
  --where '{"property":"Done","checkbox":{"equals":true}}' \
  --limit 100
```

### Automation

```bash
# Delete without confirmation
ntn b d <block-id> -y

# Pipeline
ntn db q <database-id> -j | jq '.results[] | .id'

# Debug mode
ntn --debug u me
```

---

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

### Best Practices

- Use `ntn auth login` for personal use (browser-based OAuth)
- Use `ntn auth add-token` for integration/bot access
- Never commit tokens to version control
- Rotate API tokens regularly

---

## Shell Completions

### Bash

```bash
# macOS (Homebrew):
ntn completion bash > $(brew --prefix)/etc/bash_completion.d/ntn

# Linux:
ntn completion bash > /etc/bash_completion.d/ntn

# Or source directly:
source <(ntn completion bash)
```

### Zsh

```zsh
ntn completion zsh > "${fpath[1]}/_ntn"
```

### Fish

```fish
ntn completion fish > ~/.config/fish/completions/ntn.fish
```

### PowerShell

```powershell
ntn completion powershell | Out-String | Invoke-Expression
```

---

## Exit Codes (Automation)

Stable exit codes for scripting:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | System/internal error |
| `2` | User/validation error |
| `3` | Auth error |
| `4` | Not found |
| `5` | Rate limit |
| `6` | Temporary failure (circuit breaker) |
| `130` | Canceled (Ctrl+C / context canceled) |

---

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
