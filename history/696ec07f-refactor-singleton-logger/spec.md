---
status: approved
---

# Refactoring Specification: Singleton Logger

## Scope

Introduce a singleton logger pattern to the ZombieKit application, replacing the current dependency injection approach. The logger will be initialized once at each entrypoint and accessed via a fail-fast getter throughout the codebase.

**Affected packages:**
- `internal/logging` - Core changes (singleton, getter, init)
- `internal/cli` - Entrypoint initialization (serve.go, gui.go only)
- `internal/web` - Remove DI, use getter
- `internal/config` - Use getter

**Out of scope:**
- `internal/mcp` - MCP tools cannot use this logger (stdout is the MCP protocol interface)
- `internal/cli/recall.go`, `internal/cli/db.go`, `internal/cli/memory.go` - No existing logging
- CLI user-facing output (`fmt.Printf` in commands) - these are NOT logs
- External dependencies

## Before State Summary

The codebase uses dependency injection for logging:

```go
// Entrypoints create logger
logger := logging.SetupLogger(level, false, nil)

// Passed through constructors
server := web.NewServer(..., logger)

// Stored in structs
type Server struct { logger *slog.Logger }

// Used via struct field
s.logger.Info("message", ...)
```

**Problems with current approach:**
1. Logger parameter threading through call chains
2. Constructor signatures include logger even when rarely used
3. Helper functions require explicit logger parameter
4. New code must know which struct has the logger

## After State Summary

The codebase will use a singleton pattern:

```go
// Entrypoints initialize singleton (once per process)
logging.InitLogger(level, false, nil)

// Any code can get logger via getter
logging.Logger().Info("message", ...)

// Helpers use singleton internally
logging.LogToolCall("tool", start, err)
```

**Benefits:**
1. No parameter threading
2. Simpler constructor signatures
3. Easy to add logging anywhere
4. Fail-fast catches missing initialization

## Behavior Preservation Criteria

| Behavior | Verification Method |
|----------|---------------------|
| Log output to stderr | Unit test for default writer |
| Log levels (debug/info/warn/error) | Unit test level filtering |
| Text format default | Unit test output format |
| JSON format when requested | Unit test JSON output |
| BRAINS_LOG_LEVEL env var | Integration test CLI flags |
| --log-level CLI flag | Integration test CLI flags |
| Structured key-value logging | Unit test log format |

## Migration Path

### Stage 1: Add Singleton Infrastructure

**File:** `internal/logging/logger.go`

Add:
```go
var singleton *slog.Logger

// InitLogger initializes the singleton logger.
// Panics if called more than once (prevents accidental re-initialization).
func InitLogger(level string, jsonOutput bool, w io.Writer) *slog.Logger {
    if singleton != nil {
        panic("logging: InitLogger called more than once")
    }
    singleton = SetupLogger(level, jsonOutput, w)
    return singleton
}

// Logger returns the singleton logger.
// Panics if InitLogger was not called.
func Logger() *slog.Logger {
    if singleton == nil {
        panic("logging: Logger() called before InitLogger()")
    }
    return singleton
}

// ResetLogger clears the singleton for testing.
// Should only be called from tests.
func ResetLogger() {
    singleton = nil
}
```

Keep `SetupLogger()` for backward compatibility during migration.

### Stage 2: Update Entrypoints

Update each entrypoint to call `InitLogger()`:

**File:** `internal/cli/serve.go`
```go
// Before
logger := logging.SetupLogger(logLevel, false, nil)

// After
logger := logging.InitLogger(logLevel, false, nil)
// Keep using local var for now - can remove in Stage 3
```

**File:** `internal/cli/gui.go`
```go
// Same pattern as serve.go
logger := logging.InitLogger(logLevel, false, nil)
```

### Stage 3: Migrate Callsites

Replace struct field access with getter calls:

**Pattern A: Struct field → getter**
```go
// Before
s.logger.Info("message")

// After
logging.Logger().Info("message")
```

**Pattern B: Function parameter → getter**
```go
// Before
func doThing(logger *slog.Logger) {
    logger.Info("message")
}

// After
func doThing() {
    logging.Logger().Info("message")
}
```

**Files to migrate:**

*Web package (remove DI, use getter):*
1. `internal/web/server.go` - Remove logger field from Server struct, use getter in methods
2. `internal/web/middleware.go` - Use getter instead of s.logger
3. `internal/web/plugin.go` - Remove logger from PluginRegistry, update NewPluginRegistry signature
4. `internal/web/search.go` - Use getter instead of s.logger

*Config package (remove logger parameter, use getter internally):*
5. `internal/config/loader.go` - Remove logger parameter from LoadConfig, LoadLocalConfig, etc.
6. `internal/config/tools.go` - Remove logger parameter from LoadToolsConfig

*CLI package (update calls to config functions):*
7. `internal/cli/serve.go` - Update config function calls (no longer pass logger)
8. `internal/cli/gui.go` - Update config function calls, update NewServer/NewPluginRegistry calls

### Stage 4: Update Helper Functions

Modify helpers to use singleton internally:

**LogToolCall:**
```go
// Before
func LogToolCall(logger *slog.Logger, toolName string, start time.Time, err error) {
    // ... uses logger parameter
}

// After
func LogToolCall(toolName string, start time.Time, err error) {
    // ... uses Logger() internally
}
```

**LogDBOperation:**
```go
// Before
func LogDBOperation(logger *slog.Logger, op string, start time.Time, err error) {
    // ... uses logger parameter
}

// After
func LogDBOperation(op string, start time.Time, err error) {
    // ... uses Logger() internally
}
```

### Stage 5: Cleanup

1. Remove logger parameters from constructors
2. Remove logger fields from structs
3. Update any tests that inject mock loggers
4. Consider deprecating `SetupLogger()` (or keep for library use cases)

## Verification Strategy

### Unit Tests

```go
func TestInitLogger_SetsSingleton(t *testing.T) {
    defer logging.ResetLogger()

    logger := logging.InitLogger("info", false, nil)

    if logger != logging.Logger() {
        t.Error("Logger() should return same instance")
    }
}

func TestLogger_PanicsBeforeInit(t *testing.T) {
    defer logging.ResetLogger()

    defer func() {
        if r := recover(); r == nil {
            t.Error("Expected panic")
        }
    }()

    logging.Logger() // Should panic
}

func TestInitLogger_PanicsOnDoubleInit(t *testing.T) {
    defer logging.ResetLogger()

    logging.InitLogger("info", false, nil)

    defer func() {
        if r := recover(); r == nil {
            t.Error("Expected panic")
        }
    }()

    logging.InitLogger("info", false, nil) // Should panic
}
```

### Integration Tests

1. Run `brains serve --log-level debug` - verify debug output
2. Run `brains gui --log-level error` - verify only errors shown
3. Set `BRAINS_LOG_LEVEL=warn` and verify level respected

### Manual Verification

1. Start GUI (`brains gui`), verify HTTP request logging works
2. Verify config loading logs appear at startup with `--log-level debug`

## Rollback Plan

If issues arise:

1. Revert to previous commit
2. OR keep singleton infrastructure but don't migrate callsites
3. Tests should continue passing with either approach

The migration is designed to be incremental:
- Stage 1-2 can be deployed without changing existing code
- Stage 3-4 can be done file-by-file
- Each stage can be reverted independently

## Implementation Notes

### Thread Safety

Go's `slog.Logger` is already thread-safe. The singleton pattern adds:
- Write once (`InitLogger`) → read many (`Logger()`)
- No mutex needed for read-only access after initialization
- `ResetLogger()` is test-only, single-threaded

**Critical constraint:** `InitLogger()` MUST be called before spawning any goroutines that might call `Logger()`. Both `serve.go` and `gui.go` follow this pattern - they initialize logging before starting HTTP servers or goroutines.

### Test Isolation

Tests using the singleton must:
1. Call `defer logging.ResetLogger()` at start
2. Call `logging.InitLogger()` with test configuration
3. Not run in parallel with other singleton tests (or use separate processes)

### Gradual Migration

Existing code using DI continues to work:
- `SetupLogger()` still exists
- Structs can keep logger fields during transition
- Migration can happen module-by-module

### Context-Based Logger Functions

The existing `WithLogger()` and `FromContext()` functions will be retained but are not used by any code in scope. They remain available for future use cases where context propagation is preferred over the singleton (e.g., request-scoped loggers with trace IDs).
