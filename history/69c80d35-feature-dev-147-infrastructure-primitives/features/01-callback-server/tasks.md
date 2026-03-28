---
status: ready
updated: 2026-03-28
complexity: simple
files: 6
---

# Tasks: Agent Callback HTTP Server (DEV-184)

## Dependency Graph

```
T001 (doc.go + event.go) ──┐
                            ├── T003 (handlers.go + decode.go) ── T004 (tests)
T002 (server.go)  ──────────┘
```

T001 and T002 are parallelizable. T003 depends on both. T004 depends on T003.

## Tasks

- [ ] T001 [P] [FR-003,FR-011] Create package documentation and event types
  - File: `internal/callback/doc.go`
    - Package godoc: what the callback server does, three routes with payload schemas, how to consume via `Events()`, how to start/stop
  - File: `internal/callback/event.go`
    - `EventKind` type (string) with constants: `EventComplete = "complete"`, `EventCommentResolved = "comment-resolved"`, `EventFailed = "failed"`
    - `Event` struct: `Kind EventKind`, `TicketID string`, `Timestamp time.Time`, `Branch string`, `CommentID string`, `Resolution string`, `Reason string`
  - **Acceptance**: Package compiles. `go vet ./internal/callback/...` passes.

- [ ] T002 [P] [FR-007,FR-008,FR-009] Create server skeleton with lifecycle and health check
  - File: `internal/callback/server.go`
    - `CallbackServer` struct: `port int`, `events chan Event` (buffered 64), `httpServer *http.Server`, `mux *http.ServeMux`
    - `New(port int) *CallbackServer`: creates channel, mux, registers routes (stub handlers that return 501 for now), registers `/healthz`
    - `Run(ctx context.Context) error`: creates `http.Server` with timeouts (`ReadHeaderTimeout: 5s`, `WriteTimeout: 10s`, `IdleTimeout: 30s`), starts `ListenAndServe` in goroutine, selects on `ctx.Done()` vs `errCh`, calls `Shutdown` with 5s timeout, closes events channel
    - `Events() <-chan Event`: returns read-only channel
    - `handleHealthz(w, r)`: returns 200, `Content-Type: text/plain`, body `ok`
  - **Acceptance**: Server starts on specified port, `/healthz` returns 200, context cancellation shuts down cleanly, events channel is closed after shutdown.

- [ ] T003 [FR-001,FR-002,FR-004,FR-005,FR-006,FR-010] Implement JSON decode helper and route handlers
  - File: `internal/callback/decode.go`
    - `decodeJSON[T any](r *http.Request, maxBytes int64) (T, error)`: `MaxBytesReader` 64KB, `json.NewDecoder`, `DisallowUnknownFields()`, reject trailing content via `dec.More()`
  - File: `internal/callback/handlers.go`
    - Unexported payload structs: `completePayload{Status, TicketID, Branch}`, `commentResolvedPayload{Status, TicketID, CommentID, Resolution}`, `failedPayload{Status, TicketID, Reason, CommentID (omitempty)}`
    - `writeJSON(w http.ResponseWriter, status int, data any)`: sets `Content-Type: application/json`, encodes data
    - `writeError(w http.ResponseWriter, status int, msg string)`: calls `writeJSON` with `map[string]string{"error": msg}`
    - `handleComplete(w, r)`: extract `ticketID` from `r.PathValue("ticketID")`, decode `completePayload`, validate non-empty required fields, validate `Status == "complete"`, log warning on ticket ID mismatch, build `Event`, non-blocking channel send (503 if full), return `{"ok": true}` with 200
    - `handleCommentResolved(w, r)`: same pattern, validates `CommentID` and `Resolution` required, `Status == "comment-resolved"`
    - `handleFailed(w, r)`: same pattern, validates `Reason` required, `CommentID` optional, `Status == "failed"`
  - Update `server.go`: replace stub route registrations with real handlers
  - **Acceptance**: All three routes accept valid payloads (200 + event on channel) and reject invalid ones (400 + JSON error). Backpressure returns 503. `Content-Type: application/json` on all responses.

- [ ] T004 [All FRs] Write integration test suite
  - File: `internal/callback/server_test.go`
    - Test helper `startTestServer(t *testing.T, bufferSize ...int) (*CallbackServer, string)`: ephemeral port, context with cancel in `t.Cleanup`
    - Happy path tests (3): one per route, valid payload -> 200 + `Content-Type: application/json` + event with correct fields. Failed route tested with AND without optional `comment_id`.
    - Validation tests (table-driven per route): missing required fields, status mismatch, empty body, malformed JSON, unknown fields, oversized body -> all 400
    - Backpressure test: buffer size 2, fill channel, 3rd POST -> 503 with `Content-Type: application/json`
    - Concurrency test: 10 goroutines, different ticket IDs, all events delivered, `-race` safe
    - Shutdown test: cancel context -> `Run` returns nil, events channel closed
    - Health check test: `GET /healthz` -> 200, `Content-Type: text/plain`, body `ok`
    - Method not allowed test: `GET` on POST route -> 405
    - Ticket ID mismatch test: URL != body -> 200, event uses URL value
    - Duplicate callback test: same ticket posts `/complete` twice -> both events delivered
  - **Acceptance**: `go test -race -v ./internal/callback/...` passes with all tests green.

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T003 |
| FR-002 | T003 |
| FR-003 | T001, T002 |
| FR-004 | T003 |
| FR-005 | T003 |
| FR-006 | T003 |
| FR-007 | T002 |
| FR-008 | T002 |
| FR-009 | T002 |
| FR-010 | T003, T004 |
| FR-011 | T001 |

## Execution Summary

- **Total tasks**: 4
- **Parallel opportunities**: T001 + T002 can run concurrently
- **Critical path**: T001/T002 -> T003 -> T004
- **Next command**: `/brains.next` (implement phase)
