# Implementation Plan: Singleton Logger

**Branch**: `696ec07f-refactor-singleton-logger` | **Date**: 2026-01-19 | **Spec**: [spec.md](./spec.md)

## Summary

Refactor the logging system from dependency injection to a singleton pattern with fail-fast getter. Changes span 8 files across 4 packages, with no external dependencies added.

## Technical Context

**Language/Version**: Go 1.24.1
**Primary Dependencies**: `log/slog` (stdlib)
**Storage**: N/A (in-memory singleton)
**Testing**: Go standard testing + existing test patterns
**Target Platform**: darwin/linux
**Constraints**: MCP tools excluded (stdout reserved for protocol)

## Constitution Check

No constitution defined for this project. General best practices apply:

| Principle | Status | Notes |
|-----------|--------|-------|
| Functional core, imperative shell | PASS | Logger is infrastructure (shell), not business logic |
| Explicit over hidden | PASS | Fail-fast panics make initialization errors visible |
| Testability | PASS | `ResetLogger()` enables test isolation |

**Gate Status**: PASS

## Project Structure

No new files created. Changes to existing structure:

```
internal/
├── logging/
│   └── logger.go          # Add singleton, getter, reset
├── cli/
│   ├── serve.go           # Update to InitLogger + remove DI
│   └── gui.go             # Update to InitLogger + remove DI
├── config/
│   ├── loader.go          # Remove logger param
│   └── tools.go           # Remove logger param
└── web/
    ├── server.go          # Remove logger field + DI
    ├── middleware.go      # Use getter
    ├── plugin.go          # Remove logger from registry
    └── search.go          # Use getter
```

## Implementation Phases

### Phase 1: Singleton Infrastructure
**Goal**: Add singleton without breaking existing code

**Changes**:
- Add `var singleton *slog.Logger` to `internal/logging/logger.go`
- Add `InitLogger()` function (calls `SetupLogger`, stores in singleton)
- Add `Logger()` getter (panics if not initialized)
- Add `ResetLogger()` for test cleanup

**Verification**:
- Existing tests pass (no behavior change yet)
- New unit tests for `InitLogger`, `Logger`, `ResetLogger`

**Files**: `internal/logging/logger.go`

---

### Phase 2: Entrypoint Migration
**Goal**: Switch entrypoints from `SetupLogger` to `InitLogger`

**Changes**:
- `cli/serve.go`: Replace `SetupLogger` with `InitLogger`
- `cli/gui.go`: Replace `SetupLogger` with `InitLogger`

**Verification**:
- `brains serve` starts correctly
- `brains gui` starts correctly
- Log output unchanged

**Files**: `internal/cli/serve.go`, `internal/cli/gui.go`

---

### Phase 3: Config Package Migration
**Goal**: Remove logger parameter from config functions

**Changes**:
- `config/loader.go`: Remove `logger` param from `LoadConfig`, `LoadLocalConfig`, `LoadGlobalConfig`, `LoadStorageConfig`. Use `logging.Logger()` internally.
- `config/tools.go`: Remove `logger` param from `LoadToolsConfig`. Use `logging.Logger()` internally.
- Update callers in `cli/serve.go` and `cli/gui.go`

**Verification**:
- Config loading logs appear at correct level
- No panics (logger is initialized before config loading)

**Files**: `internal/config/loader.go`, `internal/config/tools.go`, `internal/cli/serve.go`, `internal/cli/gui.go`

---

### Phase 4: Web Package Migration
**Goal**: Remove DI from web server and plugins

**Changes**:
- `web/server.go`: Remove `logger` field from `Server` struct, remove from `NewServer` signature
- `web/middleware.go`: Replace `s.logger` with `logging.Logger()`
- `web/plugin.go`: Remove `logger` from `PluginRegistry`, update `NewPluginRegistry` signature
- `web/search.go`: Replace `s.logger` with `logging.Logger()`
- Update callers in `cli/gui.go`

**Verification**:
- HTTP request logging works
- Panic recovery logs work
- Plugin registration logs work

**Files**: `internal/web/server.go`, `internal/web/middleware.go`, `internal/web/plugin.go`, `internal/web/search.go`, `internal/cli/gui.go`

---

### Phase 5: Helper Functions
**Goal**: Simplify helper function signatures

**Changes**:
- `logging/logger.go`: Update `LogToolCall` signature (remove logger param)
- `logging/logger.go`: Update `LogDBOperation` signature (remove logger param)
- Update any callers (currently none in scope)

**Verification**:
- Helper functions work with singleton
- Existing behavior preserved

**Files**: `internal/logging/logger.go`

---

### Phase 6: Test Updates
**Goal**: Ensure test isolation with singleton

**Changes**:
- Add `defer logging.ResetLogger()` to tests that use logging
- Add `logging.InitLogger()` in test setup where needed
- Ensure no `t.Parallel()` in logging-dependent tests

**Verification**:
- All tests pass
- No race conditions in test output

**Files**: `internal/web/*_test.go`, `internal/logging/logger_test.go`

## Artifacts Generated

| Artifact | Location | Status |
|----------|----------|--------|
| plan.md | history/696ec07f.../plan.md | complete |
| spec.md | history/696ec07f.../spec.md | approved |
| research.md | history/696ec07f.../research.md | complete |

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Tests fail due to singleton state | `ResetLogger()` + `defer` pattern |
| Panic if logger not initialized | Entrypoints initialize before any logging code runs |
| Race condition during init | Init happens before goroutines spawn |

## Next Step

Call `/brains.tasks` to generate the detailed task breakdown.
