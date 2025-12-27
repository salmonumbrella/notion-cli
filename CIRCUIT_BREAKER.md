# Circuit Breaker Implementation

## Overview

The Notion API client now includes a circuit breaker pattern to prevent hammering a failing API. The circuit breaker automatically opens after consecutive 5xx errors and recovers after a timeout period.

## Features

- **Automatic failure detection**: Tracks consecutive 5xx HTTP errors
- **Configurable threshold**: Opens circuit after 5 consecutive failures (default)
- **Auto-recovery**: Automatically closes circuit after 30 seconds (default)
- **Success reset**: Any successful response resets the failure counter
- **Disabled by default**: Must be explicitly enabled to maintain backward compatibility
- **Thread-safe**: Uses mutex for safe concurrent access

## Usage

### Enable with Default Settings

```go
client := notion.NewClient("your-token").EnableCircuitBreaker()
```

Default settings:
- Threshold: 5 consecutive failures
- Recovery timeout: 30 seconds

### Enable with Custom Settings

```go
client := notion.NewClient("your-token").
    WithCircuitBreaker(10, 60*time.Second)
```

### Error Handling

When the circuit is open, requests fail immediately with `notion.ErrCircuitOpen`:

```go
resp, err := client.GetPage(ctx, "page-id")
if err == notion.ErrCircuitOpen {
    log.Println("Circuit breaker is open - API is experiencing issues")
    // Handle gracefully - maybe use cached data or queue for later
    return
}
```

## Behavior

### States

1. **Closed** (Normal operation)
   - All requests go through
   - Failures increment counter
   - Successes reset counter to 0

2. **Open** (Failing)
   - All requests fail immediately with `ErrCircuitOpen`
   - No requests reach the API
   - State persists for recovery timeout period

3. **Half-open** (Recovery attempt)
   - After recovery timeout, circuit becomes half-open
   - Next request is allowed through to test if API recovered
   - If successful → circuit closes
   - If fails → circuit opens again

### Failure Tracking

- Only 5xx errors count toward the failure threshold
- 4xx errors (client errors) do not trigger circuit breaker
- Each `doRequest` call counts as one attempt, even with retries
- The circuit breaker evaluates after all retries are exhausted

### Logging

The circuit breaker logs state changes:

```
[Circuit Breaker] Circuit opened after 5 consecutive failures
[Circuit Breaker] Circuit half-open - attempting recovery
[Circuit Breaker] Circuit closed - API recovered
```

## Integration Notes

### Compatibility

- **No breaking changes**: Circuit breaker is disabled by default
- **Works with existing retry logic**: Retries happen first, then circuit breaker evaluates
- **Thread-safe**: Safe to use from multiple goroutines
- **Applies to all request types**: GET, POST, PATCH, DELETE, and multipart uploads

### Testing

The implementation includes comprehensive tests:
- `TestCircuitBreaker_OpenAfterFailures`: Verifies circuit opens after threshold
- `TestCircuitBreaker_ResetOnSuccess`: Confirms success resets counter
- `TestCircuitBreaker_AutoRecovery`: Tests recovery timeout mechanism
- `TestCircuitBreaker_DisabledByDefault`: Ensures backward compatibility
- `TestCircuitBreaker_Only5xxErrors`: Validates only 5xx errors trigger circuit

## Example: Production Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/salmonumbrella/notion-cli/internal/notion"
)

func main() {
    // Create client with circuit breaker
    client := notion.NewClient(os.Getenv("NOTION_TOKEN")).
        WithCircuitBreaker(5, 30*time.Second).
        EnableCircuitBreaker()

    ctx := context.Background()

    // Make requests - circuit breaker protects against cascading failures
    for i := 0; i < 100; i++ {
        page, err := client.GetPage(ctx, "page-id")
        if err == notion.ErrCircuitOpen {
            log.Printf("Circuit open - backing off for 30s")
            time.Sleep(30 * time.Second)
            continue
        }
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }

        log.Printf("Successfully fetched page: %s", page.ID)
    }
}
```

## Design Decisions

1. **Disabled by default**: Preserves existing behavior, requires opt-in
2. **Per-client instance**: Each client has its own circuit breaker state
3. **Simple threshold**: Consecutive failures only (no time window)
4. **Auto-recovery**: No manual reset required
5. **5xx errors only**: Client errors (4xx) don't trigger circuit
6. **Post-retry evaluation**: Circuit breaker checks final result after retries

## Future Enhancements

Possible improvements (not currently implemented):
- Metrics/monitoring hooks for circuit state changes
- Exponential backoff for recovery timeout
- Sliding window failure tracking
- Half-open request limiting (currently allows one request)
- Per-endpoint circuit breakers
