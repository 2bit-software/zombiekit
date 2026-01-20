---
status: complete
updated: 2026-01-19
---

# Research: Unified Startup Command (DEV-89)

## Executive Summary

ZombieKit currently requires users to start services in separate terminals or via shell scripts (`task up`). This research explores patterns for implementing a native `brains start` command that orchestrates multiple services with combined log output and graceful shutdown.

## Findings

### Codebase Context

**CLI Architecture**
- Framework: `github.com/urfave/cli/v2`
- Commands registered in `internal/cli/root.go`
- Each command in separate file (e.g., `gui.go`, `recall.go`)

**Existing Signal Handling** (`gui.go:136-147`)
```go
ctx, cancel := context.WithCancel(context.Background())
done := make(chan os.Signal, 1)
signal.Notify(done, os.Interrupt, syscall.SIGTERM)
go func() { <-done; cancel() }()
```

**Current `task up` Flow** (`Taskfile.yml:39-64`)
1. `recall:preflight` - validate dependencies
2. `build` - compile binary
3. `db:up` / `db:migrate` - database setup
4. Background: `./bin/brains recall watch claude &` with trap
5. Foreground: `./bin/brains gui --port ${WEBGUI_PORT:-9981}`

**Logging** (`internal/logging/logger.go`)
- Singleton pattern: `InitLogger()` / `Logger()`
- Uses `slog` for structured logging
- Levels: debug, info, warn, error
- Supports text/JSON output

**Configuration** (`internal/config/config.go`)
- `LoadStorageConfigFromEnv()` for runtime config
- Env vars: `BRAINS_GUI_PORT`, `BRAINS_BACKEND`, etc.

### Domain Knowledge

**Process Management Patterns in Go**
- `exec.Cmd` for child processes - requires stdout/stderr pipe management
- In-process goroutines - simpler, shared context, easier shutdown coordination
- `errgroup` from `golang.org/x/sync/errgroup` - manages goroutine lifecycle with error propagation

**Log Interleaving Best Practices**
- Prefix-based: `[service] message` - simple, works everywhere
- Color-coded: ANSI codes per service - more readable, terminal-dependent
- Structured logging with service field - machine-parseable

**Similar Tools**
- `docker-compose up` - prefix-based, color-coded service names
- `foreman` (Ruby) - color-coded, `.Procfile` config
- `hivemind` (Go) - simple prefix-based output

## Decision Points

- [x] **D1**: Configuration format → `.brains/config.yml` (local) → `~/.brains/config.yml` (global) → env override
- [x] **D2**: Log strategy → Prefix-based `[service]` with optional color via env toggle
- [x] **D3**: Process model → In-process goroutines (not child processes)
- [x] **D4**: Error handling → Log failures, continue other services (per spec)

## Recommendations

1. **In-process service orchestration**: Run gui and recall services as goroutines sharing a single context. This provides:
   - Simpler shutdown coordination via context cancellation
   - Shared logging infrastructure
   - No pipe management or zombie process concerns

2. **Configuration via YAML**: Use `.brains/config.yml` (local) with fallback to `~/.brains/config.yml` (global):
   ```yaml
   services:
     gui:
       enabled: true
       port: 9981
     recall:
       enabled: true
       source: claude
       interval: 30s
   ```

3. **Prefixed log output**: Each service prefixes its logs with `[gui]` or `[recall]`. Consider optional color via `BRAINS_LOG_COLOR=true`.

4. **Graceful shutdown sequence**: On SIGINT/SIGTERM, cancel context and wait for all services with a timeout (10 seconds) before force exit.

## Sources

- `internal/cli/gui.go` - existing web GUI implementation
- `internal/cli/recall.go` - existing recall/watch implementation
- `Taskfile.yml` - current `task up` orchestration
- `internal/logging/logger.go` - logging infrastructure
- Linear DEV-89 comments - approved business specification
