# Initiative: singleton-logger

**Type**: refactor
**Status**: complete
**Created**: 2026-01-19T15:38:39-08:00
**ID**: 696ec07f-refactor-singleton-logger

## Description

Introduce a singleton logger pattern across the ZombieKit application. Currently, logging is likely scattered using various approaches (console.log, direct logger instantiation, etc.). This refactor will:

1. Create a centralized singleton logger module with initialization and getter functions
2. Update all entrypoints to initialize the logger at startup
3. Replace all logging callsites throughout the application to use the singleton getter

The key constraint is that we will NOT use dependency injection. Instead, a getter function will ensure the singleton is instantiated before returning it (fail-fast if not initialized).

## Goals

1. **Centralized Logging**: All application logging flows through a single, consistently-configured logger instance
2. **Behavior Preservation**: Existing log output behavior must remain unchanged (levels, formats, destinations)
3. **Entrypoint Initialization**: Every application entrypoint explicitly initializes the logger before any other code runs
4. **Fail-Fast Getter**: The logger getter throws if called before initialization, catching integration errors early
5. **Staged Migration**: Implement in phases to allow incremental verification

## Completion

**Completed**: 2026-01-19
**Duration**: < 1 day

### Outcomes

| Task | Status | Description |
|------|--------|-------------|
| T001-T004 | Complete | Singleton infrastructure (InitLogger, Logger, ResetLogger, tests) |
| T005-T006 | Complete | Entrypoint migration (serve.go, gui.go) |
| T007-T008 | Complete | Config package migration (removed logger params) |
| T009-T010 | Complete | Config caller updates |
| T011-T016 | Complete | Web package migration and helper functions |

### Files Changed

- `internal/logging/logger.go` - Added singleton, InitLogger, Logger, ResetLogger
- `internal/logging/logger_test.go` - New file with singleton tests
- `internal/cli/serve.go` - Migrated to singleton pattern
- `internal/cli/gui.go` - Migrated to singleton pattern
- `internal/config/loader.go` - Removed logger parameters
- `internal/config/tools.go` - Removed logger parameter from WarnUnknownTools
- `internal/web/server.go` - Removed logger field from Server struct
- `internal/web/middleware.go` - Use logging.Logger()
- `internal/web/search.go` - Use logging.Logger()
- `internal/web/plugin.go` - Removed logger from PluginRegistry
- `internal/web/server_test.go` - Updated for singleton pattern
- `internal/web/search_test.go` - Updated for singleton pattern

### Verification

- `go build ./...` passes
- `go test ./internal/logging/...` passes (5/5 tests)
- `go test ./internal/web/...` passes (11/11 tests)
- `go test ./internal/config/...` passes

### Notes

All 16 tasks completed successfully. The singleton pattern is now in place with:
- `logging.InitLogger()` called at each entrypoint before any other code
- `logging.Logger()` providing fail-fast access to the singleton
- `logging.ResetLogger()` available for test isolation
- No dependency injection - all code uses the getter function
