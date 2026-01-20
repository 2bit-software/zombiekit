# Research: Singleton Logger Refactoring

## Executive Summary

ZombieKit is a Go application using the standard library `log/slog` for structured logging. The codebase already has a well-designed logger module (`internal/logging/logger.go`) with setup functions and context-based propagation. Currently, 9 files use structured logging via dependency injection. The user requests a singleton pattern with a fail-fast getter instead of DI.

## Findings

### Current Logger Implementation

**Location:** `/Users/morgan/Projects/personal/zombiekit/internal/logging/logger.go`

**Capabilities:**
- `SetupLogger(level, jsonOutput, writer)` - Creates configured `*slog.Logger`
- `SetLevel(level string)` - Runtime log level changes via `LevelVar`
- `WithLogger(ctx, logger)` - Context-based logger storage
- `FromContext(ctx)` - Retrieves logger from context (fallback to `slog.Default()`)
- `LogToolCall()` - Helper for MCP tool invocation logging
- `LogDBOperation()` - Helper for database operation logging

**Current Pattern:** Dependency injection - loggers created at entrypoints and passed as constructor/function parameters.

### Logging Callsites

| Category | Count | Location |
|----------|-------|----------|
| logger.Info() | 9 | web/, config/, cli/ |
| logger.Warn() | 10 | web/, config/, cli/ |
| logger.Error() | 8 | web/, config/, cli/ |
| logger.Debug() | 4 | config/, cli/ |
| **Total structured logging** | **31** | **9 files** |
| fmt.Printf/Println | ~68 | CLI commands (user-facing output, NOT logs) |

**Files currently using logging:**
1. `internal/cli/serve.go` - MCP server initialization
2. `internal/cli/gui.go` - Web GUI startup
3. `internal/config/loader.go` - Configuration loading
4. `internal/config/tools.go` - Tool configuration
5. `internal/web/server.go` - HTTP server
6. `internal/web/middleware.go` - Request logging, panic recovery
7. `internal/web/plugin.go` - Plugin registry
8. `internal/web/search.go` - Search endpoints
9. `internal/logging/logger.go` - Helper functions

### Entrypoints

**Primary CLI Entrypoint:** `cmd/brains/main.go`
- Bootstraps the entire application
- Registers embedded assets
- Creates urfave/cli app

**Service Entrypoints (require logger initialization):**
1. `internal/cli/serve.go` - `runServe()` - MCP server (stdio/sse/http)
2. `internal/cli/gui.go` - `runGUI()` - Web GUI server
3. `internal/cli/recall.go` - `recallWatchClaudeAction()` - Background importer

**Other CLI Commands (may need logger):**
- `internal/cli/memory.go` - Memory operations
- `internal/cli/import.go` - Database migration
- `internal/cli/db.go` - Database operations
- `internal/cli/profile.go` - Profile management
- `internal/cli/init.go` - Project initialization
- `internal/cli/version.go` - Version display

### Configuration

**Environment Variable:** `BRAINS_LOG_LEVEL`
- Valid values: `debug`, `info`, `warn`/`warning`, `error`
- Default: `info`

**CLI Flags:** `--log-level` on serve and gui commands

### Dependencies

**No external logging libraries** - uses Go standard library `log/slog` (Go 1.21+)
- Logrus appears as indirect dependency (via testcontainers) but not used directly

### Module Boundaries

**Current DI Pattern:**
```go
// Entry point creates logger
logger := logging.SetupLogger(level, jsonOutput, nil)

// Passed to constructors
server := web.NewServer(..., logger)

// Stored in struct
type Server struct {
    logger *slog.Logger
}
```

**Context Propagation (defined but not widely used):**
```go
ctx = logging.WithLogger(ctx, logger)
logger = logging.FromContext(ctx)
```

## Decision Points

1. **Singleton vs Context Propagation**: User explicitly wants singleton with getter, NOT context propagation
2. **Fail-Fast Behavior**: Getter should panic/error if called before initialization
3. **Backward Compatibility**: Existing DI code can gradually migrate to singleton

## Risk Assessment

**Low Risk:**
- Standard pattern, easy to understand
- Fail-fast catches integration errors early
- Existing code continues working during migration

**Medium Risk:**
- Tests may need modification if they relied on injecting mock loggers
- Parallel test execution could conflict if singleton state isn't managed

**Mitigation:**
- Provide `ResetLogger()` for tests
- Document initialization requirements clearly

## Recommendations

1. Add singleton state to `internal/logging/logger.go`:
   - `var singleton *slog.Logger`
   - `InitLogger(level, jsonOutput, writer)` - initializes singleton
   - `Logger()` - returns singleton or panics if not initialized

2. Update entrypoints in order:
   - `cli/serve.go` - calls `InitLogger()`
   - `cli/gui.go` - calls `InitLogger()`
   - `cli/recall.go` - calls `InitLogger()`

3. Migrate callsites gradually:
   - Replace `logger.Info(...)` with `logging.Logger().Info(...)`
   - Remove logger parameters from constructors

## BEFORE State

**Current Structure:**
```go
// internal/logging/logger.go
package logging

var logLevel = new(slog.LevelVar)  // Global level var

func SetupLogger(level, jsonOutput, w) *slog.Logger  // Returns new logger
func SetLevel(level)                                  // Changes global level
func WithLogger(ctx, logger) context.Context          // Context propagation
func FromContext(ctx) *slog.Logger                    // Context retrieval
func LogToolCall(logger, toolName, start, err)        // Takes logger param
func LogDBOperation(logger, op, start, err)           // Takes logger param
```

**Current Usage Pattern:**
```go
// Entrypoint (cli/serve.go)
logger := logging.SetupLogger(logLevel, false, nil)

// Constructor injection
server := web.NewServer(..., logger)

// Struct storage
type Server struct {
    logger *slog.Logger
}

// Method usage
s.logger.Info("request", ...)
```

**Behavior Contracts:**
1. Logger outputs to stderr by default
2. Log level configurable via `BRAINS_LOG_LEVEL` env var
3. Supports debug, info, warn, error levels
4. Text format by default, JSON optional
5. Structured key-value logging (not string formatting)

## AFTER State

**Target Structure:**
```go
// internal/logging/logger.go
package logging

var logLevel = new(slog.LevelVar)  // Unchanged
var singleton *slog.Logger          // NEW: singleton instance
var initialized bool                // NEW: initialization flag

func InitLogger(level, jsonOutput, w) *slog.Logger  // NEW: initializes singleton
func Logger() *slog.Logger                           // NEW: fail-fast getter
func SetupLogger(...)                                // KEEP for backward compat
func SetLevel(level)                                 // Unchanged
func WithLogger(ctx, logger)                         // KEEP (may deprecate later)
func FromContext(ctx)                                // KEEP (may deprecate later)
func LogToolCall(toolName, start, err)              // CHANGED: no logger param
func LogDBOperation(op, start, err)                 // CHANGED: no logger param
func ResetLogger()                                   // NEW: for testing
```

**Target Usage Pattern:**
```go
// Entrypoint (cli/serve.go)
logging.InitLogger(logLevel, false, nil)

// Anywhere in codebase
logging.Logger().Info("request", ...)

// Helpers use singleton internally
logging.LogToolCall("tool-name", start, err)
```

## INVARIANTS (Must Not Change)

1. **Log output destination**: Stderr by default
2. **Log format**: Text by default, JSON when requested
3. **Log levels**: Same 4 levels (debug, info, warn, error)
4. **Structured logging**: Key-value pairs, not string formatting
5. **Environment variable**: `BRAINS_LOG_LEVEL` continues to work
6. **CLI flag**: `--log-level` continues to work

## VARIANCE ALLOWED (Can Change)

1. How logger is obtained (DI → singleton getter)
2. Helper function signatures (can remove logger parameter)
3. Constructor signatures (can remove logger parameter)
4. Struct fields (can remove logger field)
5. Addition of `InitLogger()`, `Logger()`, `ResetLogger()` functions

## Sources

- `/Users/morgan/Projects/personal/zombiekit/internal/logging/logger.go`
- `/Users/morgan/Projects/personal/zombiekit/cmd/brains/main.go`
- `/Users/morgan/Projects/personal/zombiekit/internal/cli/*.go`
- `/Users/morgan/Projects/personal/zombiekit/internal/web/*.go`
- `/Users/morgan/Projects/personal/zombiekit/internal/config/*.go`
