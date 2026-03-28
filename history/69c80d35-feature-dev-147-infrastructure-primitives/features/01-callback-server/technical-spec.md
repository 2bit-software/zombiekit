---
status: complete
updated: 2026-03-28
---

# Technical Spec: Agent Callback HTTP Server (DEV-184)

## Package Layout

```
internal/callback/
  doc.go           # Package godoc (contract documentation)
  event.go         # EventKind, Event struct
  server.go        # CallbackServer, New, Run, Events
  decode.go        # decodeJSON[T] helper
  handlers.go      # Route handlers, writeJSON, writeError
  server_test.go   # Integration tests
```

## Type Definitions

### Event (event.go)

```go
// EventKind identifies the type of callback event.
type EventKind string

const (
    EventComplete        EventKind = "complete"
    EventCommentResolved EventKind = "comment-resolved"
    EventFailed          EventKind = "failed"
)

// Event represents a parsed callback from an agent session.
// It is a value type -- safe to pass across goroutine boundaries without copying.
//
// The Kind field determines which route-specific fields are populated:
//   - EventComplete: Branch is set
//   - EventCommentResolved: CommentID and Resolution are set
//   - EventFailed: Reason is set, CommentID may be set
//
// Unused fields for a given Kind are zero-valued.
type Event struct {
    Kind      EventKind
    TicketID  string
    Timestamp time.Time

    // Complete
    Branch string

    // CommentResolved / Failed
    CommentID string

    // CommentResolved
    Resolution string

    // Failed
    Reason string
}
```

### CallbackServer (server.go)

```go
// CallbackServer receives HTTP POST callbacks from agent sessions
// and delivers parsed events to a consumer via a buffered channel.
type CallbackServer struct {
    port       int
    events     chan Event
    httpServer *http.Server
    mux        *http.ServeMux
}
```

### Constructor

```go
// New creates a CallbackServer that will listen on the given port.
// The event channel is buffered at 64 entries. If the buffer fills,
// incoming requests receive 503 Service Unavailable.
func New(port int) *CallbackServer
```

### Public Methods

```go
// Run starts the HTTP server and blocks until ctx is cancelled.
// On cancellation, it drains in-flight requests (5s timeout) and
// closes the events channel before returning.
func (s *CallbackServer) Run(ctx context.Context) error

// Events returns a read-only channel of parsed callback events.
// The channel is closed when Run returns.
func (s *CallbackServer) Events() <-chan Event
```

## Request/Response Schemas

### Request Payloads (unexported structs in handlers.go)

```go
type completePayload struct {
    Status   string `json:"status"`
    TicketID string `json:"ticket_id"`
    Branch   string `json:"branch"`
}

type commentResolvedPayload struct {
    Status     string `json:"status"`
    TicketID   string `json:"ticket_id"`
    CommentID  string `json:"comment_id"`
    Resolution string `json:"resolution"`
}

type failedPayload struct {
    Status    string `json:"status"`
    TicketID  string `json:"ticket_id"`
    Reason    string `json:"reason"`
    CommentID string `json:"comment_id,omitempty"`
}
```

### Response Shapes

Success: `200 OK`, `Content-Type: application/json`
```json
{"ok": true}
```

Validation error: `400 Bad Request`, `Content-Type: application/json`
```json
{"error": "missing required field: branch"}
```

Backpressure: `503 Service Unavailable`, `Content-Type: application/json`
```json
{"error": "event queue full, retry later"}
```

Health check: `200 OK`, `Content-Type: text/plain`
```
ok
```

## Route Registration

```go
mux.HandleFunc("POST /{ticketID}/complete", s.handleComplete)
mux.HandleFunc("POST /{ticketID}/comment-resolved", s.handleCommentResolved)
mux.HandleFunc("POST /{ticketID}/failed", s.handleFailed)
mux.HandleFunc("GET /healthz", s.handleHealthz)
```

## Handler Flow (all three routes follow this pattern)

```
1. Extract ticketID := r.PathValue("ticketID")
2. Decode JSON body via decodeJSON[T](r, 64*1024)
   -> 400 on decode error
3. Validate required fields are non-empty
   -> 400 on missing field
4. Validate status field matches expected value for route
   -> 400 on mismatch
5. Log warning if payload.TicketID != ticketID (URL vs body)
6. Build Event{} with Kind, TicketID (from URL), Timestamp, route fields
7. Non-blocking channel send:
   select {
   case s.events <- event:
       writeJSON(w, 200, map[string]bool{"ok": true})
   default:
       writeError(w, 503, "event queue full, retry later")
   }
```

## Server Lifecycle

```
New(port) -> Run(ctx) blocks:
  1. Create http.Server with timeouts
  2. ListenAndServe in goroutine -> errCh
  3. select:
     - ctx.Done() -> Shutdown(5s timeout) -> close(events) -> return nil
     - errCh -> return err
```

## JSON Decode Helper

```go
func decodeJSON[T any](r *http.Request, maxBytes int64) (T, error) {
    var v T
    r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&v); err != nil {
        return v, err
    }
    if dec.More() {
        return v, errors.New("request body must contain a single JSON object")
    }
    return v, nil
}
```

## Configuration

| Parameter | Source | Default | Description |
|-----------|--------|---------|-------------|
| Port | `CALLBACK_PORT` env var / constructor arg | 8666 | HTTP listen port |
| Buffer size | Hardcoded constant | 64 | Event channel buffer |
| Max body size | Hardcoded constant | 65536 (64KB) | JSON body size limit |
| Shutdown timeout | Hardcoded constant | 5s | Graceful shutdown deadline |
| ReadHeaderTimeout | Hardcoded | 5s | Slow header protection |
| WriteTimeout | Hardcoded | 10s | Response write deadline |
| IdleTimeout | Hardcoded | 30s | Keep-alive idle limit |

## Dependencies

- `net/http` (stdlib)
- `encoding/json` (stdlib)
- `context` (stdlib)
- `time` (stdlib)
- `log/slog` (stdlib, via `internal/logging`)
- `fmt`, `errors`, `net` (stdlib)

No new external dependencies.

## FR Traceability

| FR | Implementation |
|----|----------------|
| FR-001 | Route registration in `server.go` |
| FR-002 | Payload structs + validation in `handlers.go` |
| FR-003 | `Event` struct + `Events()` channel in `event.go`/`server.go` |
| FR-004 | `writeJSON(w, 200, ...)` in handlers |
| FR-005 | `writeError(w, 400, ...)` in handlers |
| FR-006 | Non-blocking channel send with `default` case -> 503 |
| FR-007 | `New(port)` constructor, port from `CALLBACK_PORT` |
| FR-008 | `Run()` shutdown logic with 5s timeout, `close(events)` |
| FR-009 | `handleHealthz` in `server.go` |
| FR-010 | Channel-based delivery (inherently concurrent-safe) |
| FR-011 | `doc.go` package documentation |
