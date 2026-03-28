---
status: complete
updated: 2026-03-28
---

# Implementation Plan: Agent Callback HTTP Server (DEV-184)

## Overview

Four implementation phases, ordered by dependency. Each phase produces testable, compilable code. The package lives at `internal/callback/`.

## Phase 1: Types and Package Foundation

**Goal**: Define all exported types and the package contract. This is the primary deliverable -- DEV-149 can begin after this phase.

**Files**:
- `internal/callback/doc.go` -- Package-level godoc describing the callback contract, event schemas, channel semantics, and usage example
- `internal/callback/event.go` -- `EventKind` constants, `Event` struct with all fields, constructor helpers

**Work**:
1. Create `doc.go` with package documentation covering:
   - What the callback server does (one paragraph)
   - The three routes and their payload schemas
   - How to consume events via `Events()` channel
   - How to start/stop the server
2. Create `event.go` with:
   - `EventKind` type (string) and constants: `EventComplete`, `EventCommentResolved`, `EventFailed`
   - `Event` struct: `Kind EventKind`, `TicketID string`, `Timestamp time.Time`, `Branch string`, `CommentID string`, `Resolution string`, `Reason string`
   - No constructor needed -- handlers build `Event` literals directly

**Traces to**: FR-003 (typed events), FR-011 (contract documentation)

**Validation**: Package compiles. Godoc renders correctly.

---

## Phase 2: Server Skeleton

**Goal**: Runnable HTTP server with health check, event channel, and graceful shutdown. No route handlers yet.

**Files**:
- `internal/callback/server.go` -- `CallbackServer` struct, `New()`, `Run()`, `Events()`, route registration, health check handler

**Work**:
1. Define `CallbackServer` struct:
   - `port int`
   - `events chan Event` (buffered, size 64)
   - `httpServer *http.Server`
   - `mux *http.ServeMux`
2. `New(port int) *CallbackServer` -- creates channel, mux, registers routes (stubs for now), health check
3. `Run(ctx context.Context) error` -- creates `http.Server`, starts `ListenAndServe` in goroutine, selects on `ctx.Done()` vs `errCh`, calls `Shutdown` with 5s timeout, closes events channel
4. `Events() <-chan Event` -- returns read-only channel
5. `handleHealthz(w, r)` -- returns `200 OK`, `text/plain`, body `ok`

**Traces to**: FR-007 (port config), FR-008 (graceful shutdown), FR-009 (health check)

**Validation**: Server starts, `/healthz` returns 200, context cancellation shuts down cleanly.

---

## Phase 3: JSON Parsing and Route Handlers

**Goal**: All three route handlers parse, validate, and deliver events.

**Files**:
- `internal/callback/decode.go` -- Generic `decodeJSON[T]` helper
- `internal/callback/handlers.go` -- `handleComplete`, `handleCommentResolved`, `handleFailed`, `writeJSON`, `writeError`

**Work**:
1. Create `decodeJSON[T](r *http.Request, maxBytes int64) (T, error)`:
   - `http.MaxBytesReader` with 64KB limit
   - `json.NewDecoder` with `DisallowUnknownFields()`
   - Reject trailing content via `dec.More()`
2. Define request payload structs (unexported, handler-internal):
   - `completePayload` -- `Status`, `TicketID`, `Branch` (json tags with snake_case)
   - `commentResolvedPayload` -- `Status`, `TicketID`, `CommentID`, `Resolution`
   - `failedPayload` -- `Status`, `TicketID`, `Reason`, `CommentID` (omitempty)
3. Create `writeJSON(w, status, data)` and `writeError(w, status, msg)` helpers
4. Implement `handleComplete(w, r)`:
   - Extract `ticketID` from `r.PathValue("ticketID")`
   - Decode body into `completePayload`
   - Validate required fields (non-empty `Status`, `TicketID`, `Branch`)
   - Validate `Status == "complete"`
   - Log warning if `payload.TicketID != ticketID` (URL path)
   - Build `Event{Kind: EventComplete, TicketID: ticketID, Branch: payload.Branch, Timestamp: time.Now()}`
   - Non-blocking send to `s.events` channel; 503 if full
   - Return `{"ok": true}` with 200
5. Implement `handleCommentResolved(w, r)` -- same pattern, validates `CommentID` and `Resolution` required
6. Implement `handleFailed(w, r)` -- same pattern, validates `Reason` required, `CommentID` optional

**Traces to**: FR-001 (routes), FR-002 (validation), FR-004 (200 OK), FR-005 (400 errors), FR-006 (503 backpressure), FR-010 (concurrent safety via channel)

**Validation**: All three routes accept valid payloads and reject invalid ones. Events appear on channel.

---

## Phase 4: Tests

**Goal**: Integration test suite covering all FRs and edge cases.

**Files**:
- `internal/callback/server_test.go` -- All tests

**Work**:
1. Test helper: `startTestServer(t, bufferSize ...int) (*CallbackServer, string)` -- starts server on ephemeral port, returns server and base URL, registers `t.Cleanup` to cancel context
2. **Happy path tests** (one per route):
   - POST valid payload -> 200, event on channel with correct fields
   - Assert `Content-Type: application/json` header on all JSON responses (FR-005 MUST requirement)
   - **Failed route with optional `comment_id`**: POST with `comment_id` present, verify `Event.CommentID` is populated correctly
3. **Validation tests** (table-driven per route):
   - Missing required fields -> 400 with error message, `Content-Type: application/json`
   - Status field mismatch -> 400
   - Empty body -> 400
   - Malformed JSON -> 400
   - Unknown fields -> 400
   - Oversized body (>64KB) -> 400
4. **Backpressure test**:
   - Create server with buffer size 2
   - POST 2 valid requests (don't drain channel)
   - POST 3rd -> 503, verify `Content-Type: application/json`
5. **Concurrency test**:
   - 10 goroutines POST concurrently for different ticket IDs
   - Drain channel, verify all 10 events received with correct IDs
   - Run with `-race`
6. **Shutdown test**:
   - Start server, cancel context, verify `Run` returns nil
   - Verify events channel is closed
7. **Health check test**:
   - GET `/healthz` -> 200, body `ok`, `Content-Type: text/plain`
8. **Method not allowed test**:
   - GET on a POST route -> 405
9. **Ticket ID mismatch test**:
   - POST with URL ticket ID != body ticket_id -> 200 (accepted), event uses URL value
10. **Duplicate callback test**:
    - POST `/complete` twice for same ticket ID -> both events delivered (stateless behavior)

**Traces to**: All FRs, all edge cases in spec

**Validation**: `go test -race ./internal/callback/...` passes.

---

## Dependency Order

```
Phase 1 (types) -> Phase 2 (server skeleton) -> Phase 3 (handlers) -> Phase 4 (tests)
```

Phase 1 can be reviewed/shipped independently to unblock DEV-149.

## Out of Scope (Deferred to Orchestrator Integration)

- **`CALLBACK_PORT` env var reading**: The `New(port int)` constructor takes a port directly. Reading `CALLBACK_PORT` and defaulting to 8666 belongs to `cmd/orchestrator/main.go` (or equivalent CLI entrypoint), which is part of Epic 3 (orchestrator core). This ticket delivers the package; wiring into the orchestrator is a separate concern.
- **`shutdown.Manager` registration**: The callback server's `Run(ctx) error` signature is compatible with `shutdown.ServiceFunc`, but registering it with the shutdown manager is orchestrator integration work.

## No Spikes Needed

- stdlib `http.ServeMux` path parameters are well-documented (Go 1.22+)
- Channel-based event delivery is standard Go
- No external APIs, no database, no third-party libraries
