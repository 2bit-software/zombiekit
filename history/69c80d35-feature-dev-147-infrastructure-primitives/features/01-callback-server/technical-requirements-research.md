---
status: complete
updated: 2026-03-28
---

# Technical Requirements: Agent Callback HTTP Server (DEV-184)

These are implementation hints and technical preferences extracted from the ticket and context document. They inform HOW to build, not WHAT to build.

## Package Location

`internal/callback/` -- matches the project layout from the Agent Context Brief.

## Router

stdlib `http.ServeMux` with Go 1.22+ path parameters. No chi/gorilla needed for 3 routes + health check.

## Event Delivery Mechanism

Buffered Go channel (size 64). Expose via `Events() <-chan Event`. Non-blocking send with 503 backpressure in handlers. Channel closed on server shutdown.

## Server Lifecycle

- Implement `Run(ctx context.Context) error` matching `shutdown.ServiceFunc` signature
- Follow `internal/server/server.go` pattern: ListenAndServe in goroutine, select on ctx.Done vs errCh
- Shutdown timeout: 5 seconds (matching existing patterns)
- Set `ReadHeaderTimeout: 5s`, `WriteTimeout: 10s`, `IdleTimeout: 30s`

## Port Configuration

Default port 8666, configurable via `CALLBACK_PORT` env var. The orchestrator injects `WORK_CALLBACK_URL=http://localhost:{port}/{ticket-id}` when spawning agent sessions.

## JSON Parsing

Generic `decodeJSON[T]` helper using:
- `http.MaxBytesReader` (64KB limit)
- `json.NewDecoder` with `DisallowUnknownFields()`
- Reject trailing content with `dec.More()`

## Logging

Use `logging.Logger()` singleton (this runs in the orchestrator process, not MCP).

## Error Types

Follow `internal/linear/errors.go` pattern with `ErrorKind` enum if needed. For the callback server, standard HTTP error responses may suffice without custom error types.

## Testing Strategy

- Integration tests with real HTTP server on ephemeral port
- Use `httptest.NewServer` or start server in goroutine with `freePort(t)` helper
- Test concurrent request handling with `sync.WaitGroup`
- Test graceful shutdown with context cancellation
- Test malformed JSON with 400 responses

## Dependencies

- Zero new external dependencies (stdlib only for HTTP)
- Internal: `internal/logging`, `internal/shutdown` (optional integration)
