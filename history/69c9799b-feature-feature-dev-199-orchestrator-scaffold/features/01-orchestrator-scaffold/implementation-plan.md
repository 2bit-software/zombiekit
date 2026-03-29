# Implementation Plan: Orchestrator Scaffold

## Dependency Graph

```
Wave 1: [T1: config.go]
Wave 2: [T2: watchers.go]  (depends on Config type from T1)
Wave 3: [T3: orchestrator.go]  (depends on T1 + T2)
Wave 4: [T4: main.go]  (depends on T1 + T3)
Wave 5: [T5: config_test.go] [T6: orchestrator_test.go] [T7: watchers_test.go]  (parallel, depend on T1-T4)
```

## Tasks

### T1: Config struct and validation (`internal/orchestrator/config.go`)

**Creates**: `internal/orchestrator/config.go`
**Depends on**: nothing
**AC coverage**: AC-3 (fail-fast on invalid config)

- Define `Config` struct with all 10 fields from the spec
- Implement `NewConfig(c *cli.Context) (*Config, error)` that reads urfave/cli context
- Implement `Validate() error` that collects ALL validation errors into a multi-error
  - Required string fields: non-empty
  - CallbackPort: 1-65535
  - ConcurrencyLimit: >= 1
  - PollInterval, ShutdownTimeout: > 0
  - LogLevel: one of debug/info/warn/error
  - WorktreesRoot: `os.MkdirAll` on valid path
- Error format: `"config validation failed: --flag/ENV is required; --flag/ENV must be ..."`

### T2: Watcher stub functions (`internal/orchestrator/watchers.go`)

**Creates**: `internal/orchestrator/watchers.go`
**Depends on**: T1 (uses `time.Duration` from config, matches `shutdown.ServiceFunc`)
**AC coverage**: AC-9 (watcher lifecycle logging)

- Define `NewWatcherStub(name string, pollInterval time.Duration) shutdown.ServiceFunc`
  - Returns a closure: `func(ctx context.Context) error`
  - Logs `"watcher started"` with `slog.String("watcher", name)`
  - Blocks on `<-ctx.Done()`
  - Logs `"watcher stopped"` with `slog.String("watcher", name)`
  - Returns `nil` (NOT `ctx.Err()`)
- Three constants for watcher names: `WatcherLinearPoller`, `WatcherPRWatcher`, `WatcherCommentWatcher`

### T3: Orchestrator struct and Run (`internal/orchestrator/orchestrator.go`)

**Creates**: `internal/orchestrator/orchestrator.go`
**Depends on**: T1, T2
**AC coverage**: AC-1 (startup sequence), AC-2 (shutdown), AC-5 (ordering), AC-7 (port error propagation), AC-10 (store close ordering)

- Define `Orchestrator` struct: `cfg *Config`, `store state.StateStore`
- Implement `New(cfg *Config, store state.StateStore) *Orchestrator`
- Implement `Run() error`:
  1. `state.ApplyReconciliation(context.Background(), o.store, logging.Logger())`
     - If error: return immediately (services never start)
  2. Create callback server: `callback.New(o.cfg.CallbackPort)`
  3. Build three watcher stubs via `NewWatcherStub`
  4. Create shutdown manager: `shutdown.New(o.cfg.ShutdownTimeout)`
  5. `return mgr.Run(srv.Run, linearPoller, prWatcher, commentWatcher)`

### T4: CLI entry point (`cmd/orchestrator/main.go`)

**Creates**: `cmd/orchestrator/main.go`
**Depends on**: T1, T3
**AC coverage**: AC-3 (fail-fast), AC-4 (DB creation), AC-8 (JSON logging)

- `main()`: create `cli.App` with name "orchestrator", version, flags, action
- Define 10 CLI flags matching the config table (with `EnvVars` and defaults)
- `run(c *cli.Context) error`:
  1. `cfg, err := orchestrator.NewConfig(c)` — fail fast
  2. `logging.InitLogger(cfg.LogLevel, cfg.LogJSON, nil)`
  3. Log startup with version
  4. `store, err := state.NewSQLiteStore(ctx, cfg.DBPath)` — fail fast
  5. `defer store.Close()`
  6. `return orchestrator.New(cfg, store).Run()`
- Error handling: `slog.Error` + `os.Exit(1)` in main

### T5: Config validation tests (`internal/orchestrator/config_test.go`)

**Creates**: `internal/orchestrator/config_test.go`
**Depends on**: T1
**AC coverage**: AC-3

Test cases:
- Valid config passes validation
- Missing Linear API key fails with descriptive error
- Missing GitHub token fails with descriptive error
- Invalid callback port (0, 65536) fails
- Missing worktrees root fails
- Missing DB path fails
- Invalid concurrency limit (0, -1) fails
- Invalid poll interval (0) fails
- Invalid log level ("banana") fails
- Multiple missing fields: all errors collected in single return

### T6: Orchestrator lifecycle tests (`internal/orchestrator/orchestrator_test.go`)

**Creates**: `internal/orchestrator/orchestrator_test.go`
**Depends on**: T3
**AC coverage**: AC-1, AC-5, AC-7

Test with mock `state.StateStore` (implement the interface with call recording):
- `TestRun_ReconciliationRunsBeforeServices` — mock store records calls, verify `ListJobsByStatus` (from reconciliation) is called, then services start
- `TestRun_ReconciliationFailure_PreventsServices` — mock store returns error from `ListJobsByStatus`, verify Run returns that error, verify no service goroutines launched
- `TestRun_ServicesAssembled` — verify four services are passed to shutdown manager (harder to test directly — may need to observe through watcher log output or callback server port binding)

Note: Tests that call `logging.InitLogger` must use `defer logging.ResetLogger()`. Do not use `t.Parallel()` for these tests.

### T7: Watcher stub tests (`internal/orchestrator/watchers_test.go`)

**Creates**: `internal/orchestrator/watchers_test.go`
**Depends on**: T2
**AC coverage**: AC-9

- `TestWatcherStub_ReturnsNilOnCancel` — create stub, cancel context, verify returns nil
- `TestWatcherStub_BlocksUntilCancel` — create stub, verify it blocks (use goroutine + timer), then cancel and verify return

## File Summary

| File | Action | Wave |
|------|--------|------|
| `internal/orchestrator/config.go` | create | 1 |
| `internal/orchestrator/watchers.go` | create | 2 |
| `internal/orchestrator/orchestrator.go` | create | 3 |
| `cmd/orchestrator/main.go` | create | 4 |
| `internal/orchestrator/config_test.go` | create | 5 |
| `internal/orchestrator/orchestrator_test.go` | create | 5 |
| `internal/orchestrator/watchers_test.go` | create | 5 |

**No existing files are modified.**
