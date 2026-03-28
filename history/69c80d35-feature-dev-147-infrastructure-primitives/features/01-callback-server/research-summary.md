---
status: complete
updated: 2026-03-28
---

# Research: Agent Callback HTTP Server (DEV-184)

## Executive Summary

The callback server receives HTTP POST notifications from Claude Code agent sessions when they complete work, resolve PR comments, or fail. It parses JSON payloads, validates them, and delivers typed events to the orchestrator via a Go channel. The codebase already has all necessary infrastructure patterns: stdlib `http.ServeMux` with Go 1.22+ path parameters, `shutdown.Manager` for graceful lifecycle, and channel-based event delivery.

## Findings

### Codebase Context

- **Two HTTP server patterns exist**: `internal/server/server.go` uses stdlib `http.ServeMux` (ConnectRPC), `internal/web/server.go` uses chi (GUI). The callback server matches the simpler stdlib pattern.
- **`shutdown.Manager`** (`internal/shutdown/manager.go`) coordinates multiple `ServiceFunc` via errgroup. The callback server should be just another `ServiceFunc`.
- **`StateStore`** (`internal/state/store.go`) already has `SetJobStatus` with status constants that map directly to callback events (`StatusComplete`, `StatusNeedsAttention`). The callback server should NOT import the state store -- it emits events, the orchestrator handles state transitions.
- **Error patterns**: `internal/linear/errors.go` uses `ErrorKind` enum + `Error` struct with `Unwrap()`. Follow this for callback-specific errors.
- **Logging**: Singleton `logging.Logger()` via `internal/logging/`. Since this server runs in the orchestrator process (not MCP), the singleton is safe to use.
- **Go 1.24.1** confirmed in `go.mod` -- full Go 1.22+ ServeMux path parameter support available.

### Domain Knowledge

- **Go 1.22+ ServeMux** supports `POST /{ticketID}/complete` syntax natively. `r.PathValue("ticketID")` extracts the parameter. Method-scoped patterns return 405 automatically for wrong methods.
- **Channel-based event delivery** is preferred over handler registration for a single-consumer pattern. Buffered channel (size 64) with non-blocking send and 503 backpressure is idiomatic.
- **JSON validation**: Use `json.NewDecoder` with `MaxBytesReader` (64KB) and `DisallowUnknownFields()` to catch integration bugs early.
- **Graceful shutdown**: `http.Server.Shutdown` drains in-flight requests. Use 5s timeout matching existing shutdown manager pattern.

## Decision Points

- [x] **D1**: Router choice -- stdlib `http.ServeMux` (not chi). Callback server has 3 routes + health check; no middleware needs.
- [x] **D2**: Event delivery -- buffered channel, not handler callbacks. Single consumer (orchestrator loop), natural `select` integration.
- [x] **D3**: No persistence -- events are fire-and-deliver. The orchestrator handles state persistence via `StateStore`.
- [x] **D4**: No authentication -- same-machine, internal only. Matches ticket scope-out.
- [x] **D5**: Event uses a single tagged-union struct with a `Kind` discriminator and route-specific fields. Parsed structs, not `json.RawMessage`. The coupling is acceptable -- the callback package defines the contract, and the contract IS the payload schemas.

## Recommendations

1. Use stdlib `http.ServeMux` with Go 1.22+ path parameters
2. Implement as `ServiceFunc` compatible with `shutdown.Manager`
3. Deliver events via `Events() <-chan Event` with buffered channel
4. Use value-type event structs (not pointers) for concurrency safety
5. Package at `internal/callback/` -- independent of state store
6. Create generic `decodeJSON[T]` helper for request body parsing

## Sources

- `internal/server/server.go` -- stdlib ServeMux + graceful shutdown pattern
- `internal/shutdown/manager.go` -- service coordination
- `internal/state/store.go` -- job status interface
- `internal/linear/errors.go` -- error type patterns
- Go 1.22 release notes -- ServeMux path parameters
