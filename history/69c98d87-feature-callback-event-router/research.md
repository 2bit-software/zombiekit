---
status: complete
updated: 2026-03-29
---

# Research: Callback Event Router

## Executive Summary

The callback server already emits structured events (CompletionEvent, CommentResolvedEvent, FailureEvent) onto a buffered channel. The orchestrator creates the server but nobody consumes the channel yet. All downstream clients (GitHubClient, LinearClient, StateStore) exist with the methods needed, except LinearClient lacks a `PostComment` method for the FailureEvent handler.

## Findings

### Codebase Context

**Callback Server** (`internal/callback/`)
- `server.go` exposes `Events() <-chan Event` -- a buffered (64) read-only channel
- `event.go` defines `Event` struct with `Kind`, `TicketID`, `Timestamp`, and kind-specific fields (`Branch`, `CommentID`, `Resolution`, `Reason`)
- Handlers parse JSON payloads, validate required fields, and push onto the channel. Returns 503 if buffer full.
- Channel is closed when `Run` returns (on context cancellation).

**Orchestrator** (`internal/orchestrator/`)
- `orchestrator.go` creates `callback.New(port)` and passes `callbackSrv.Run` to shutdown manager, but never reads from `callbackSrv.Events()`.
- `watchers.go` defines PR watcher and comment watcher as stubs (log start/stop, block on ctx).
- `watcher_linear.go` implements the ticket pickup pipeline -- the only real watcher today.
- The `Orchestrator` struct holds `store`, `linear`, `worktrees`, `sessions` but NOT `github.Client`.

**StateStore** (`internal/state/`)
- `SetPR(ctx, ticketID, prNumber)` -- records PR number against a job.
- `SetCommentWatermark(ctx, prNumber, commentID)` -- upserts last-processed comment ID.
- `ReleaseSlot(ctx, projectID)` -- decrements active_count, clamps at zero.
- `SetJobStatus(ctx, ticketID, status)` -- updates job status.
- `GetJob(ctx, ticketID)` -- returns `*Job` with `WorktreePath`, `PRNumber`, `Status`, etc.

**GitHubClient** (`internal/github/`)
- `CreatePR(ctx, CreatePRInput) (int, error)` -- returns PR number.
- `UpdatePRBody(ctx, prNumber, body)` -- updates PR description.
- `PostCommentReply(ctx, prNumber, kind, commentID, body) (int64, error)` -- replies to a comment.
- `ApplyLabel(ctx, prNumber, label)` -- idempotent label application.

**LinearClient** (`internal/linear/`)
- `SetTicketStatus(ctx, id, status)` -- two-step: resolves workflow state ID, then issues update mutation.
- `GetTicket(ctx, id)` -- fetches ticket details.
- **GAP: No `PostComment` or `CreateComment` method exists.** The FailureEvent handler needs to post the failure reason as a Linear comment. This method must be added to the interface and implemented.

**Archival & Friction** (`internal/archival/`, `internal/friction/`)
- These packages don't exist yet. The ticket says to wire stubs with correct signatures for Epic 5.

### Domain Knowledge

**Event-driven pattern**: The callback server already implements a producer-consumer pattern via a Go channel. The router is the consumer side -- range over the channel, switch on `Event.Kind`, dispatch to handler functions.

**Goroutine isolation**: The ticket specifies handling events in a goroutine so the callback server returns 200 OK immediately. This is already satisfied -- the callback handlers push onto a buffered channel and return immediately. The consumer goroutine (the router) is separate.

**Partial failure handling**: If post-session processing fails midway, the ticket requires moving the ticket to `needs-attention` with full context logging. This implies each handler should be wrapped in recovery logic.

## Decision Points

- [x] **D1**: Where does the event router live? → `internal/orchestrator/` as a new file (e.g., `router.go`), since it consumes the event channel and coordinates existing clients.
- [x] **D2**: How are archival and friction auditor stubs defined? → As Go interfaces in their own packages (`internal/archival/`, `internal/friction/`) with no-op implementations. The router accepts these interfaces via dependency injection.
- [x] **D3**: Does `LinearClient` need a `PostComment` method or should we use a different mechanism? → Yes, add `PostComment(ctx, issueID, body) error` to `linear.Client` interface. Implement via `commentCreate` GraphQL mutation.
- [x] **D4**: How does the router get the worktree path for a ticket? → Via `StateStore.GetJob(ticketID)` which returns `Job.WorktreePath`.
- [x] **D5**: What label does the orchestrator apply to PRs? → Configuration value (e.g., `ai-managed`). Pass via config.

## Recommendations

1. **Add `PostComment` to LinearClient interface** -- Required for FailureEvent handling. Signature: `PostComment(ctx context.Context, issueID string, body string) error`. Implement via the `commentCreate` GraphQL mutation.

2. **Add `github.Client` to the Orchestrator struct** -- Currently missing. The orchestrator needs it for CreatePR, UpdatePRBody, and PostCommentReply.

3. **Define archival and friction interfaces in their own packages** -- `internal/archival.Archiver` and `internal/friction.Auditor` with the correct method signatures. Provide no-op implementations that satisfy the interfaces.

4. **Router as a `shutdown.ServiceFunc`** -- Reads from the event channel in a loop, dispatches to typed handler methods. Fits naturally into the existing shutdown manager pattern.

## Sources

- Linear ticket DEV-202: Callback event router and post-session processing
- Agent Context Brief: Autonomous Dev Pipeline (Linear document)
- Codebase: `internal/callback/`, `internal/orchestrator/`, `internal/state/`, `internal/github/`, `internal/linear/`
