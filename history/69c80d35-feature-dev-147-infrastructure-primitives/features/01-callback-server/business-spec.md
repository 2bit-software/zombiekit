# Feature Specification: Agent Callback HTTP Server

**Linear Ticket**: [DEV-184](https://linear.app/heinsight/issue/DEV-184/define-and-implement-the-agent-callback-http-server)
**Parent Epic**: [DEV-147](https://linear.app/heinsight/issue/DEV-147/epic-build-the-infrastructure-primitives) -- Infrastructure Primitives
**Created**: 2026-03-28
**Status**: Draft

## Overview

The callback server is an internal HTTP endpoint that receives notifications from Claude Code agent sessions. When an agent completes work, resolves a PR comment, or encounters an unrecoverable failure, it POSTs to the callback server. The server parses the payload, validates it, and delivers a typed event to the orchestrator for processing.

This is one half of the orchestrator-agent communication boundary. The orchestrator tells agents what to work on (via environment variables when spawning sessions); agents tell the orchestrator what happened (via HTTP callbacks to this server).

**Primary deliverable**: The package-level contract documentation (godoc on exported types, event schemas, channel semantics) is the primary deliverable of this ticket. It unblocks DEV-149 (Agent Profiles), which can begin once the route shapes and payload schemas are defined in code.

## User Scenarios & Testing

### User Story 1 -- Agent completes work successfully (Priority: P1)

An agent session finishes implementing a ticket. It pushes its branch and writes `.ai/pr-description.md` to the worktree, then POSTs to `{WORK_CALLBACK_URL}/complete` with the ticket ID, status, and branch name. The orchestrator receives a `CompletionEvent` and proceeds to create a GitHub PR.

**Why this priority**: The happy path. Without completion callbacks, the orchestrator has no way to know an agent finished.

**Independent Test**: Start the callback server, POST a valid completion payload, verify the event is delivered on the events channel with correct fields.

**Acceptance Scenarios**:

1. **Given** the server is running on port 8666, **When** `POST /DEV-123/complete` is called with `{"status": "complete", "ticket_id": "DEV-123", "branch": "DEV-123/add-feature"}`, **Then** a `CompletionEvent` with matching fields is delivered to the events channel and 200 OK is returned.
2. **Given** the server is running, **When** `POST /DEV-123/complete` is called with a JSON body missing the required `branch` field, **Then** 400 Bad Request is returned with a descriptive error message and no event is delivered.

---

### User Story 2 -- Agent fails with an error (Priority: P1)

An agent session hits an unrecoverable error (test failures it can't fix, spec ambiguity, tooling breakdown). It POSTs to `{WORK_CALLBACK_URL}/failed` with the ticket ID, a reason string, and optionally a comment ID if the failure relates to a specific PR comment. The orchestrator receives a `FailureEvent` and moves the ticket to `needs-attention` status.

**Why this priority**: Equal to completion -- the orchestrator must know about failures to avoid orphaned sessions and stuck tickets.

**Independent Test**: POST a valid failure payload, verify the event is delivered. POST a failure payload with optional `comment_id` present, verify it's captured. POST without required `reason`, verify 400.

**Acceptance Scenarios**:

1. **Given** the server is running, **When** `POST /DEV-456/failed` is called with `{"status": "failed", "ticket_id": "DEV-456", "reason": "tests failing after 3 attempts"}`, **Then** a `FailureEvent` is delivered with `CommentID` empty and 200 OK is returned.
2. **Given** the server is running, **When** `POST /DEV-456/failed` is called with `{"status": "failed", "ticket_id": "DEV-456", "comment_id": "IC_abc123", "reason": "cannot resolve conflicting review feedback"}`, **Then** a `FailureEvent` is delivered with `CommentID` set to `"IC_abc123"` and 200 OK is returned.
3. **Given** the server is running, **When** `POST /DEV-456/failed` is called with `{"status": "failed", "ticket_id": "DEV-456"}` (missing `reason`), **Then** 400 Bad Request is returned.

---

### User Story 3 -- Agent resolves a PR review comment (Priority: P2)

After the orchestrator detects a new PR comment and spawns a fresh agent session to address it, the agent fixes the code, pushes, and POSTs to `{WORK_CALLBACK_URL}/comment-resolved` with the comment ID and a resolution summary. The orchestrator receives a `CommentResolvedEvent`.

**Why this priority**: Slightly lower than P1 because the initial PR creation flow (complete/failed) must work before comment-resolution cycles are relevant.

**Independent Test**: POST a valid comment-resolved payload, verify the event is delivered with correct comment ID and resolution text.

**Acceptance Scenarios**:

1. **Given** the server is running, **When** `POST /DEV-789/comment-resolved` is called with `{"status": "comment-resolved", "ticket_id": "DEV-789", "comment_id": "IC_def456", "resolution": "Added nil check as requested"}`, **Then** a `CommentResolvedEvent` is delivered and 200 OK is returned.
2. **Given** the server is running, **When** the payload is missing `comment_id`, **Then** 400 Bad Request is returned.

---

### User Story 4 -- Server handles concurrent callbacks (Priority: P2)

Two agent sessions working on different tickets POST callbacks at the same time. Both events are delivered without race conditions, data corruption, or dropped events.

**Why this priority**: Concurrency correctness is essential but is a property of the implementation, not a separate user flow.

**Independent Test**: Send N concurrent POST requests for different ticket IDs, verify all N events are delivered with correct ticket IDs.

**Acceptance Scenarios**:

1. **Given** the server is running, **When** 10 concurrent `POST /{ticket-id}/complete` requests arrive for different ticket IDs, **Then** all 10 `CompletionEvent`s are delivered without data races (verified under `-race` flag).

---

### User Story 5 -- Server starts on configurable port (Priority: P3)

The server defaults to port 8666 but can be configured via `CALLBACK_PORT` environment variable to avoid conflicts.

**Why this priority**: Port configurability is a nice-to-have; the default works for the standard deployment.

**Independent Test**: Start the server with `CALLBACK_PORT=9999`, POST to port 9999, verify the event is delivered.

**Acceptance Scenarios**:

1. **Given** `CALLBACK_PORT` is not set, **When** the server starts, **Then** it listens on port 8666.
2. **Given** `CALLBACK_PORT=9999`, **When** the server starts, **Then** it listens on port 9999.

---

### User Story 6 -- Server shuts down gracefully (Priority: P3)

When the orchestrator process exits (SIGINT/SIGTERM), the callback server drains any in-flight requests before stopping. No events are lost mid-delivery.

**Why this priority**: Graceful shutdown prevents data loss but is rarely exercised in practice.

**Independent Test**: Start the server, initiate shutdown via context cancellation while a request is in-flight, verify the request completes and the server exits cleanly.

**Acceptance Scenarios**:

1. **Given** the server is running and a request is in-flight, **When** the context is cancelled, **Then** the in-flight request completes (200 OK), the events channel is closed, and the server returns nil.

---

### Edge Cases

- What happens when the JSON body is empty? -> 400 Bad Request with `{"error": "empty request body"}`.
- What happens when the JSON body exceeds 64KB? -> 400 Bad Request (via `MaxBytesReader`).
- What happens when `ticket_id` in the URL path doesn't match `ticket_id` in the JSON body? -> The URL path ticket ID is authoritative. The body `ticket_id` is validated for presence but the path value is used for event construction. Mismatch is logged as a warning but not rejected.
- What happens when `status` field doesn't match the route? -> 400 Bad Request with `{"error": "status field must be 'complete' for this route"}` (or similar per route).
- What happens when the events channel buffer is full? -> 503 Service Unavailable with `{"error": "event queue full, retry later"}`. The agent can retry.
- What happens when an unknown route is hit (e.g., `POST /DEV-123/unknown`)? -> 404 Not Found (default ServeMux behavior).
- What happens when a GET request hits a POST-only route? -> 405 Method Not Allowed (Go 1.22+ ServeMux handles this automatically).
- What happens when the JSON body has unknown fields? -> 400 Bad Request (`DisallowUnknownFields`). Catches agent integration bugs early.
- What happens when the same ticket POSTs `/complete` twice? -> Both events are delivered. The callback server is stateless; idempotency is the orchestrator's responsibility.
- What happens when Content-Type header is missing or not `application/json`? -> Server attempts JSON parsing regardless. No Content-Type validation (agents are internal, not a public API).

## Requirements

### Functional Requirements

- **FR-001**: System MUST listen for HTTP POST requests on three routes: `POST /{ticketID}/complete`, `POST /{ticketID}/comment-resolved`, `POST /{ticketID}/failed`. The `{ticketID}` path parameter is a free-form string (typically `DEV-123` format but not validated).
- **FR-002**: System MUST parse JSON request bodies and validate required fields per route (see Payload Schemas below).
- **FR-003**: System MUST deliver parsed events to a consumer via `Events() <-chan Event`. The `Event` type is a tagged union struct with a `Kind` field discriminator. The consumer uses a switch on `Kind` and accesses route-specific fields. The server owns and creates the channel internally; the consumer receives a read-only channel.
- **FR-004**: System MUST return `200 OK` with body `{"ok": true}` when an event is successfully delivered to the channel.
- **FR-005**: System MUST return `400 Bad Request` with body `{"error": "<description>"}` when the request body is malformed or missing required fields. The `Content-Type` header MUST be `application/json` on all responses.
- **FR-006**: System MUST return `503 Service Unavailable` with body `{"error": "event queue full, retry later"}` when the event channel buffer is full.
- **FR-007**: System MUST default to port 8666 and allow override via `CALLBACK_PORT` environment variable.
- **FR-008**: System MUST shut down gracefully when its context is cancelled, draining in-flight requests within a 5-second timeout before returning. The events channel MUST be closed after shutdown.
- **FR-009**: System MUST expose a health check endpoint (`GET /healthz`) that returns `200 OK` with body `ok` (text/plain). Uses `/healthz` to match existing codebase convention.
- **FR-010**: System MUST handle concurrent requests without race conditions. All tests MUST pass with `-race` flag.
- **FR-011**: The package MUST include godoc documentation on all exported types describing the callback contract, event schemas, and channel semantics. This is the primary deliverable that unblocks DEV-149.

### Payload Schemas

All payloads are JSON objects. All string fields are non-empty when required.

**`POST /{ticketID}/complete`**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | yes | Must be `"complete"`. Validated to match the route. |
| `ticket_id` | string | yes | Ticket identifier. Validated for presence; URL path value is authoritative for event construction. |
| `branch` | string | yes | Git branch name the agent pushed to. |

**`POST /{ticketID}/comment-resolved`**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | yes | Must be `"comment-resolved"`. Validated to match the route. |
| `ticket_id` | string | yes | Ticket identifier. URL path value is authoritative. |
| `comment_id` | string | yes | GitHub comment ID the agent addressed. |
| `resolution` | string | yes | Free-form text describing how the comment was resolved. |

**`POST /{ticketID}/failed`**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | yes | Must be `"failed"`. Validated to match the route. |
| `ticket_id` | string | yes | Ticket identifier. URL path value is authoritative. |
| `reason` | string | yes | Free-form text describing the failure. |
| `comment_id` | string | no | GitHub comment ID if failure relates to a specific PR comment. Empty string or omitted if not applicable. |

**`status` field validation**: The `status` field value MUST match the route being called. `POST /{ticketID}/complete` requires `"status": "complete"`. A mismatch returns 400. This provides defense-in-depth against agents constructing the wrong URL.

**`ticket_id` field**: Present in the struct for validation and logging. The URL path `{ticketID}` is authoritative for event construction. If URL and body disagree, a warning is logged but the request is not rejected.

### Key Entities

- **Event**: A tagged union struct with a `Kind` field (`EventKind` string enum: `"complete"`, `"comment-resolved"`, `"failed"`), a `TicketID` string, a `Timestamp` time.Time, and route-specific fields (`Branch`, `CommentID`, `Resolution`, `Reason`). Unused fields for a given kind are zero-valued. This is a value type (not pointer) for concurrency safety.
- **CallbackServer**: The HTTP server. Constructed via `New(port int) *CallbackServer`. Exposes `Run(ctx context.Context) error` (blocking, compatible with `shutdown.ServiceFunc`) and `Events() <-chan Event` (read-only channel, buffered internally at 64 events).

## Success Criteria

### Measurable Outcomes

- **SC-001**: All three route handlers parse valid payloads and deliver events to the channel correctly.
- **SC-002**: All three route handlers reject invalid payloads with 400 and descriptive errors.
- **SC-003**: Concurrent requests (10+) for different ticket IDs all deliver events without races (tests pass with `-race`).
- **SC-004**: Server starts on default port and on a configured port.
- **SC-005**: Server shuts down within 5 seconds of context cancellation, draining in-flight requests.

## Testing Requirements

### Test Strategy

Integration tests using real HTTP requests against the callback server running on an ephemeral port. No mocks -- the server is a leaf node with no upstream dependencies. All tests run with `-race` flag. No build tags required (no external service dependencies).

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | POST to each route with valid payload, verify 200 and event delivery |
| FR-002 | Integration | POST with missing fields per route, verify 400 with error message |
| FR-003 | Integration | Verify events appear on `Events()` channel with correct types and fields |
| FR-004 | Integration | Verify 200 response after successful event delivery |
| FR-005 | Integration | Malformed JSON, empty body, oversized body -> 400 |
| FR-006 | Integration | Fill channel buffer, POST again, verify 503 |
| FR-007 | Integration | Start with port override, verify server binds correctly |
| FR-008 | Integration | Cancel context during request, verify drain and clean exit |
| FR-009 | Integration | GET /healthz returns 200 |
| FR-010 | Integration | 10 concurrent POSTs, verify all events delivered, run with -race |

### Edge Case Coverage

- Empty JSON body -> 400 with "empty request body" or similar
- JSON with unknown fields -> 400 (DisallowUnknownFields)
- Oversized body (>64KB) -> 400
- URL ticket ID vs body ticket ID mismatch -> warning logged, URL value used
- Channel full -> 503
- Wrong HTTP method -> 405 (automatic)
- Unknown route -> 404 (automatic)
