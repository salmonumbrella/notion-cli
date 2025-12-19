# Configuration

notion-cli supports a YAML configuration file for storing preferences and defaults.

## Configuration File Location

The configuration file is located at:
```
~/.config/notion-cli/config.yaml
```

To see the path on your system:
```bash
notion config path
```

## Managing Configuration

### View Current Configuration
```bash
notion config show
```

### Set Configuration Values
```bash
notion config set <key> <value>
```

### Examples
```bash
# Set default output format to JSON
notion config set output json

# Set color mode to always use colors
notion config set color always

# Set default workspace
notion config set default_workspace personal
```

## Configuration Options

### `output`
Default output format for all commands.

**Valid values:** `text`, `json`, `table`, `yaml`

**CLI override:** `--output <format>`

```bash
notion config set output json
```

### `color`
Default color mode for terminal output.

**Valid values:** `auto`, `always`, `never`

**CLI override:** `--color <mode>` (when implemented)

```bash
notion config set color always
```

### `default_workspace`
Default workspace name for multi-workspace support.

**CLI override:** Not yet implemented

```bash
notion config set default_workspace personal
```

### `workspaces`
Workspace-specific configurations. Currently for future use.

```yaml
workspaces:
  personal:
    token_source: keyring
    output: json
  work:
    token_source: env:NOTION_TOKEN_WORK
    output: table
```

## Priority Order

Configuration values are applied in this order (highest priority first):

1. **CLI flags** - Explicitly set flags (e.g., `--output json`)
2. **Config file** - Values from `~/.config/notion-cli/config.yaml`
3. **Defaults** - Built-in CLI defaults

For example:
```bash
# Config file has: output: json
notion page list              # Uses JSON output (from config)
notion page list --output text  # Uses text output (CLI flag overrides config)
```

## Security

- The configuration directory is created with `0700` permissions (owner read/write/execute only)
- The configuration file is created with `0600` permissions (owner read/write only)
- Token information is not stored in the config file; use the keyring or environment variables

## Example Configuration

See [config-example.yaml](./config-example.yaml) for a complete example with all available options.

## Optional Configuration

The configuration file is completely optional. notion-cli works perfectly without it by using:
- CLI flags for each command
- Built-in defaults
- Environment variables for tokens

The config file is simply a convenience for setting persistent defaults.
