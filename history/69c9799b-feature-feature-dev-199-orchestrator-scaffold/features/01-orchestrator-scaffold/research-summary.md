# Research Summary: Orchestrator Scaffold

## Existing Infrastructure to Reuse

### 1. Shutdown Manager (`internal/shutdown/manager.go`)
- Already implements SIGINT/SIGTERM handling with errgroup
- Two-phase shutdown: graceful drain with timeout, then force exit on second signal or timeout
- Accepts `ServiceFunc = func(ctx context.Context) error`
- Tested with injectable signal channel

### 2. Service Interface (`internal/startup/service.go`)
- `Service` interface: `Name() string` + `Run(ctx context.Context) error`
- `ServiceLogger(name)` returns a grouped slog logger
- Used by the `start` command for multi-service orchestration

### 3. StateStore (`internal/state/store.go`)
- Full interface already defined: jobs, watermarks, concurrency slots
- SQLite implementation exists with WAL mode, migrations, auto-directory creation
- Constructor: `NewSQLiteStore(ctx, dbPath)` — handles everything including migrations
- Already handles the "DB doesn't exist yet" acceptance criterion

### 4. Logger (`internal/logging/logger.go`)
- Singleton pattern with `InitLogger(level, jsonOutput, writer)`
- Context-aware: `WithLogger(ctx, logger)` / `FromContext(ctx)`
- Text or JSON output modes

### 5. CLI Pattern (`cmd/zk-server/main.go`)
- urfave/cli with env var fallbacks (`EnvVars` field)
- `ZK_` prefix for env vars
- Pattern: define flags → `run` action → init logger → build config → create server → signal context → run

### 6. Existing Interfaces
- `linear.Client` — polls tickets, manages statuses/labels
- `github.Client` — creates PRs, manages comments/labels
- Both fully defined with HTTP implementations

## What Needs to Be Created

| Component | Package | Notes |
|-----------|---------|-------|
| `cmd/orchestrator/main.go` | cmd/orchestrator | Entry point, CLI flags, config loading |
| Orchestrator Config struct | internal/orchestrator | Config fields per ticket |
| Orchestrator startup/run | internal/orchestrator | Sequential init, watcher launch, shutdown |
| Reconciliation stub | internal/orchestrator | Called at startup, no-op initially |
| Watcher stubs | internal/orchestrator | Three goroutines that block until context cancelled |
| Callback server stub | internal/callback | HTTP server on fixed port, no routes yet |

## Env Var Naming Convention

Existing convention: `ZK_` prefix (zk-server), `BRAINS_` prefix (brains CLI).
The orchestrator is a new daemon — it should get its own prefix. Options:
- `ORCH_` — short, distinct
- `ZK_ORCH_` — namespaced under ZK
- Ticket says "config file" so env vars may be secondary

## Config Loading Strategy

The ticket says "env vars and/or config file". Existing patterns:
- urfave/cli handles env var → flag → default precedence already
- TOML config files exist (`internal/config/loader.go`) for the brains CLI
- Simplest: urfave/cli flags with `EnvVars` (matches zk-server exactly)
- TOML file support can be deferred or added as a `--config` flag

## Startup Sequence (from ticket)

```
1. init config (fail fast if invalid)
2. init logger
3. open state store (create + migrate if needed)
4. run reconciliation (must complete before watchers)
5. start callback server (bind port 8666)
6. launch three watcher goroutines
7. block until shutdown signal
```

Steps 1-4 are sequential with fail-fast. Steps 5-6 run as services under the shutdown manager.

## Shutdown Sequence

```
1. Catch SIGINT/SIGTERM
2. Cancel context (signals all watchers and callback server)
3. Watchers drain in-flight work
4. State store closed
5. Process exits
```

The existing `shutdown.Manager` handles steps 1-3 + force exit. Step 4 needs a deferred close after the manager returns.

## Key Design Decision: errgroup vs shutdown.Manager

The shutdown manager already wraps errgroup. The orchestrator needs:
- Sequential startup (steps 1-4) — before any services start
- Concurrent services (callback server + 3 watchers) — under shutdown manager

This maps cleanly to: sequential init in the `run` function, then pass 4 ServiceFuncs to `shutdown.Manager.Run()`.
