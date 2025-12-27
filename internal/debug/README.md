# Debug Package

The `debug` package provides HTTP request/response logging capabilities for the Notion CLI.

## Features

- **Context-based debug mode**: Debug mode is stored in context and can be checked anywhere in the application
- **HTTP transport wrapper**: Wraps `http.RoundTripper` to intercept and log all HTTP traffic
- **Token redaction**: Automatically redacts sensitive authorization tokens, showing only the last 4 characters
- **Body truncation**: Truncates long request/response bodies (500 chars for requests, 1000 chars for responses)
- **Detailed timing**: Shows response time for each request
- **Stderr output**: All debug output goes to stderr, keeping stdout clean for actual command output

## Usage

### Command-line

Enable debug mode by adding the `--debug` flag to any command:

```bash
notion --debug user me
notion --debug search "my query"
notion --debug page get <page-id>
```

### Programmatic

#### Context Management

```go
import "github.com/salmonumbrella/notion-cli/internal/debug"

// Enable debug mode in context
ctx := debug.WithDebug(context.Background(), true)

// Check if debug mode is enabled
if debug.IsDebug(ctx) {
    // Debug mode is on
}
```

#### HTTP Transport

```go
import (
    "net/http"
    "github.com/salmonumbrella/notion-cli/internal/debug"
)

// Create a debug transport
transport := debug.NewDebugTransport(http.DefaultTransport, os.Stderr)

// Use it in an HTTP client
client := &http.Client{
    Transport: transport,
}
```

#### Notion Client Integration

The Notion client has built-in support for debug mode:

```go
import "github.com/salmonumbrella/notion-cli/internal/notion"

// Create client with debug mode
client := notion.NewClient(token).WithDebug()
```

Or use the helper function from the cmd package:

```go
import "github.com/salmonumbrella/notion-cli/internal/cmd"

// Automatically enables debug if --debug flag was set
client := cmd.NewNotionClient(ctx, token)
```

## Output Format

### Request Logging

```
--> GET https://api.notion.com/v1/users/me
    Authorization: Bearer ...a8VX
    Notion-Version: 2025-09-03
    Content-Type: application/json
```

For POST/PATCH requests with a body:

```
--> POST https://api.notion.com/v1/search
    Authorization: Bearer ...a8VX
    Notion-Version: 2025-09-03
    Content-Type: application/json
    Body: {"query":"test"}
```

### Response Logging

```
<-- 200 OK (245ms)
    Content-Type: application/json
    X-Notion-Request-Id: abc-123-def
    Body: {"id":"...","name":"..."}
```

### Error Logging

```
<-- ERROR: connection refused (100ms)
```

## Security

- **Token Redaction**: Authorization headers are automatically redacted to show only the last 4 characters
- **Sensitive Data**: Be careful when using debug mode with sensitive data, as request/response bodies are logged
- **Stderr Output**: Debug output goes to stderr, not stdout, so it won't interfere with command output or pipelines

## Implementation Details

### DebugTransport

The `DebugTransport` implements `http.RoundTripper` and wraps another transport (typically `http.DefaultTransport`). It:

1. Logs request details before sending
2. Reads and logs the request body (while restoring it for the actual request)
3. Executes the underlying transport
4. Logs response details including timing
5. Reads and logs the response body (while restoring it for the caller)

### Body Reading Strategy

Both request and response bodies are read completely and then restored using `io.NopCloser(bytes.NewReader(bodyBytes))`. This ensures:

- The debug logger can access the full body
- The original request/response consumer can still read the body
- No race conditions or partial reads

### Truncation Thresholds

- Request bodies: 500 characters
- Response bodies: 1000 characters
- Bodies exceeding these limits are truncated with a `[truncated]` marker

## Testing

Run the debug package tests:

```bash
go test ./internal/debug/... -v
```

The test suite covers:
- Context management (`WithDebug`, `IsDebug`)
- Request/response logging
- Token redaction
- Body truncation
- Error handling
- Transport defaults
