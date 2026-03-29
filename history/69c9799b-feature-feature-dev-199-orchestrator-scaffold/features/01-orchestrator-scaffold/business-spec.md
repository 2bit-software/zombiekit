# Business Specification: Orchestrator Config, Startup Scaffold, and Graceful Shutdown

**Source**: [DEV-199](https://linear.app/heinsight/issue/DEV-199/orchestrator-config-startup-scaffold-and-graceful-shutdown)
**Parent**: DEV-150 (Autonomous Dev Pipeline)

## Overview

The orchestrator is a long-running Go daemon that autonomously manages the lifecycle of development tickets — from Linear queue through to merged GitHub PR. This ticket creates the daemon's entry point, configuration, startup sequence, and graceful shutdown. No business logic; just the skeleton that wires existing components together.

## Scope

### In Scope

1. **Configuration** — A struct holding all orchestrator settings, loadable from CLI flags with env var fallbacks via urfave/cli
2. **Startup sequence** — Sequential initialization: config -> logger -> state store -> reconciliation -> concurrent services
3. **Graceful shutdown** — Via existing `shutdown.Manager` (handles SIGINT/SIGTERM, context cancellation, drain timeout, force exit)
4. **Structured logger** — slog-based logging initialized before any services start
5. **Wiring** — Connecting existing components (`callback.Server`, `state.ApplyReconciliation`, `state.NewSQLiteStore`) into the daemon lifecycle

### Out of Scope

- Watcher logic (this ticket creates goroutine launch points with stub implementations only)
- Callback server implementation (already exists at `internal/callback/` — this ticket only wires it)
- Reconciliation logic (already exists at `internal/state/reconcile.go` — this ticket only calls it)
- Any business logic whatsoever

## Configuration

### Required Fields

Use `urfave/cli` flags with `EnvVars` for automatic precedence: CLI flag > env var > default. Use `cli.DurationFlag` for duration types.

The `ORCH_` prefix distinguishes orchestrator config from the `ZK_`-prefixed zk-server config.

| Field | Type | CLI Flag | Env Var | Default | Description |
|-------|------|----------|---------|---------|-------------|
| Linear API Key | string | `--linear-api-key` | `ORCH_LINEAR_API_KEY` | (none) | Linear API authentication |
| GitHub Token | string | `--github-token` | `ORCH_GITHUB_TOKEN` | (none) | GitHub API authentication |
| Callback Port | int | `--callback-port` | `ORCH_CALLBACK_PORT` | 8666 | HTTP callback server port |
| Worktrees Root | string | `--worktrees-root` | `ORCH_WORKTREES_ROOT` | (none) | Directory for git worktrees |
| DB Path | string | `--db-path` | `ORCH_DB_PATH` | (none) | SQLite database file path |
| Concurrency Limit | int | `--concurrency-limit` | `ORCH_CONCURRENCY_LIMIT` | 1 | Max concurrent jobs per project |
| Poll Interval | duration | `--poll-interval` | `ORCH_POLL_INTERVAL` | 30s | Watcher polling interval |
| Log Level | string | `--log-level` | `ORCH_LOG_LEVEL` | info | Logging verbosity (debug, info, warn, error) |
| Log JSON | bool | `--log-json` | `ORCH_LOG_JSON` | false | JSON log output format |
| Shutdown Timeout | duration | `--shutdown-timeout` | `ORCH_SHUTDOWN_TIMEOUT` | 30s | Max time to drain on shutdown |

### Validation Rules

Validation collects ALL errors before returning, so the operator sees everything that's wrong at once.

- Linear API Key: required, non-empty
- GitHub Token: required, non-empty
- Callback Port: required, 1-65535
- Worktrees Root: required, non-empty; directory created via `os.MkdirAll(path, 0o755)` during validation
- DB Path: required, non-empty (file + parent dirs created by `state.NewSQLiteStore`)
- Concurrency Limit: must be >= 1
- Poll Interval: must be > 0
- Shutdown Timeout: must be > 0
- Log Level: must be one of debug, info, warn, error

### Fail-Fast Behavior

If any required config value is missing or invalid, the process must exit immediately with a descriptive error listing ALL validation failures. Format: `"config validation failed: --linear-api-key/ORCH_LINEAR_API_KEY is required; --db-path/ORCH_DB_PATH is required"`. No partial startup. No services started. Exit code 1.

## Startup Sequence

The startup sequence is **strictly sequential**. Each step must complete successfully before the next begins.

```
Step 1: Load and validate configuration
Step 2: Initialize structured logger
Step 3: Open state store (creates DB + runs migrations if needed)
Step 4: Run reconciliation (existing implementation)
Step 5: Start callback server + three watcher goroutines (concurrent, under shutdown manager)
```

### Step Details

**Step 1 -- Config**: Parse CLI flags/env vars into config struct via `NewConfig(c *cli.Context)`. Validate all fields. Fail fast on any error.

**Step 2 -- Logger**: Call `logging.InitLogger(level, jsonOutput, nil)` with configured level and format. `InitLogger` must only be called once per process — it panics on a second call. All subsequent log output uses this logger. In tests, call `defer logging.ResetLogger()` before `InitLogger`.

**Step 3 -- State Store**: Call `state.NewSQLiteStore(ctx, dbPath)`. This handles directory creation, DB creation, pragma setup, and migrations. If the DB file doesn't exist, it is created. If migrations are pending, they run. Fail fast on any error. Defer `store.Close()` immediately after successful open.

**Step 4 -- Reconciliation**: Call `state.ApplyReconciliation(ctx, store, logger)`. This is the existing implementation in `internal/state/reconcile.go` which scans for orphaned in-progress jobs and marks them as needs-attention. It must complete before any watchers start.

**Step 5 -- Services**: Create `shutdown.Manager` with the configured timeout. Pass the callback server's `Run` method and three watcher stubs to `manager.Run()`. The shutdown manager handles SIGINT/SIGTERM internally — do NOT set up a separate `signal.NotifyContext` in the orchestrator. All four services run concurrently. Each receives a context (created by the manager) that is cancelled on shutdown.

### Context Lifecycle

The `run()` action function in `main.go` creates a `context.Background()` for use in Steps 3-4 (state store init and reconciliation). This context is NOT passed into `shutdown.Manager.Run()` — the manager creates its own internal context and manages its own signal handling. This avoids competing signal handlers.

## Graceful Shutdown

### Trigger

SIGINT or SIGTERM received by the process. Handled internally by `shutdown.Manager`.

### Sequence

1. Shutdown manager cancels its internal context (created by the manager, not the caller)
2. Callback server stops accepting new connections and drains in-flight requests (5s internal timeout)
3. Watcher goroutines observe context cancellation and return nil
4. Shutdown manager waits for all services to return (up to configured shutdown timeout)
5. `shutdown.Manager.Run()` returns
6. State store is closed (via deferred `store.Close()` in the `run` function)
7. Process exits with code 0

### Force Exit

- Second SIGINT/SIGTERM: immediate `os.Exit(1)` (handled by shutdown.Manager)
- Shutdown timeout exceeded: immediate `os.Exit(1)` (handled by shutdown.Manager)

### Return Values

`shutdown.Manager.Run()` returns nil on clean shutdown (all services return nil after context cancellation). It returns a non-nil error if any service fails before or during shutdown. The `run()` function returns this error directly — urfave/cli translates non-nil to exit code 1.

## Watcher Stubs

Three watcher goroutines are launched. For this ticket, each is a stub that matches the `shutdown.ServiceFunc` signature (`func(ctx context.Context) error`):

1. **Linear Poller** — logs "started" with `slog.String("watcher", "linear-poller")`, blocks on `<-ctx.Done()`, logs "stopped", returns `nil`
2. **PR Watcher** — logs "started" with `slog.String("watcher", "pr-watcher")`, blocks on `<-ctx.Done()`, logs "stopped", returns `nil`
3. **Comment Watcher** — logs "started" with `slog.String("watcher", "comment-watcher")`, blocks on `<-ctx.Done()`, logs "stopped", returns `nil`

Each watcher receives the poll interval from config (unused by stubs, but threaded through for future use). Watcher stubs MUST return `nil` after context cancellation, NOT `ctx.Err()`. Returning `ctx.Err()` (`context.Canceled`) would cause errgroup to treat it as a service failure.

## Callback Server (Existing)

The callback server already exists at `internal/callback/`. Do NOT create a new one. Wire the existing implementation:

```
srv := callback.New(cfg.CallbackPort)
```

Pass `srv.Run` as one of the services to `shutdown.Manager.Run()`. The existing server:
- Binds to `:{callback_port}` (all interfaces)
- Has routes: `POST /{ticketID}/complete`, `POST /{ticketID}/comment-resolved`, `POST /{ticketID}/failed`, `GET /healthz`
- Produces events on a buffered channel (64 entries). Event consumption is deferred to the watcher/dispatcher ticket — during scaffold operation, no events are expected.
- Supports graceful shutdown via `http.Server.Shutdown()` with 5s internal timeout
- Logs the bound address at startup
- Uses `logging.Logger()` singleton — logger MUST be initialized (Step 2) before the callback server handles any request

## Entry Point

`cmd/orchestrator/main.go` — similar pattern to `cmd/zk-server/main.go`:
- urfave/cli app with flags and `Version: version.Get().Short()`
- `run` action function that performs the startup sequence
- Minimal `main()` — just app setup and error handling
- Do NOT use `signal.NotifyContext` — the shutdown manager handles signals

## Acceptance Criteria

1. **AC-1**: Given a valid config, when the orchestrator starts, then the state store opens, reconciliation runs, callback server binds to port 8666, and three watcher goroutines start — all before accepting any work.

2. **AC-2**: Given SIGTERM is received, when the signal handler fires, then all watchers receive a stop signal and the process exits cleanly within the configured shutdown timeout.

3. **AC-3**: Given a missing required config value (e.g., no Linear API key), when startup runs, then the process fails fast with a descriptive error listing all validation failures before doing anything else.

4. **AC-4**: Given the DB file doesn't exist yet, when startup runs, then it is created and migrated before watchers start.

5. **AC-5**: Given valid config, when the orchestrator starts, the startup sequence is strictly ordered: config -> logger -> state store -> reconciliation -> services. No service starts before reconciliation completes.

6. **AC-6**: Given SIGINT is sent twice, the process force-exits immediately on the second signal.

7. **AC-7**: Given the callback port is already in use, when the callback server attempts to bind, the error propagates through the shutdown manager and the process exits with a descriptive error.

8. **AC-8**: Given a valid config with `--log-json`, all log output is in JSON format.

9. **AC-9**: Each watcher stub logs its name on start and stop, confirming lifecycle events are observable.

10. **AC-10**: The state store is closed after all services have stopped (via deferred close in `run`, after `shutdown.Manager.Run()` returns).

## Constructor Signatures

```go
// internal/orchestrator/config.go
func NewConfig(c *cli.Context) (*Config, error)
// Parses urfave/cli context into Config struct. Validates all fields.
// Returns multi-error listing all validation failures.

// internal/orchestrator/orchestrator.go
type Orchestrator struct {
    cfg   *Config
    store state.StateStore
}

func New(cfg *Config, store state.StateStore) *Orchestrator
// Wires dependencies. No side effects.

func (o *Orchestrator) Run() error
// Owns the lifecycle: reconciliation -> build services -> shutdown manager.
// 1. state.ApplyReconciliation(ctx, o.store, logging.Logger())
// 2. Build callback server: callback.New(o.cfg.CallbackPort)
// 3. Build watcher stubs (three ServiceFuncs)
// 4. shutdown.New(o.cfg.ShutdownTimeout).Run(services...)
// Returns nil on clean shutdown, non-nil on service failure.

// cmd/orchestrator/main.go
func run(c *cli.Context) error
// Owns resource acquisition (thin glue):
//   1. cfg, err := orchestrator.NewConfig(c)
//   2. logging.InitLogger(cfg.LogLevel, cfg.LogJSON, nil)
//   3. store, err := state.NewSQLiteStore(ctx, cfg.DBPath)
//      defer store.Close()
//   4. return orchestrator.New(cfg, store).Run()
```

### Design Rationale

`main.go` owns resource acquisition (CLI parsing, logger init, store open/close) — untestable glue, ~10 lines. `Orchestrator` owns lifecycle (reconciliation -> services -> shutdown) — unit-testable with a mock `state.StateStore`. Tests verify: reconciliation runs before services, reconciliation failure prevents service launch, all four services are assembled correctly.

## Package Layout

```
cmd/orchestrator/
  main.go                — CLI app, flags, run action (resource acquisition)
internal/orchestrator/
  config.go              — Config struct, NewConfig, validation
  orchestrator.go        — Orchestrator struct, New, Run (lifecycle ownership)
  watchers.go            — watcher stub functions (three ServiceFunc factories)
```

Existing packages used as-is (NOT modified by this ticket):
- `internal/callback/` — callback HTTP server
- `internal/state/` — StateStore interface, SQLite implementation, reconciliation
- `internal/shutdown/` — graceful shutdown manager
- `internal/logging/` — structured logger

## Dependencies

All dependencies already exist in go.mod:
- `github.com/urfave/cli/v2` — CLI framework
- `golang.org/x/sync` — errgroup (via shutdown.Manager)
- `log/slog` — structured logging (stdlib)
- `internal/state` — StateStore + SQLite implementation + reconciliation
- `internal/callback` — HTTP callback server
- `internal/shutdown` — graceful shutdown manager
- `internal/logging` — logger initialization
- `internal/version` — version string
