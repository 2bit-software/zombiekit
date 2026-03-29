# Feature Specification: Watcher 2 — PR Comment Queue and Comment-Resolution Sessions

**Feature Branch**: `morganhein/dev-203-watcher-2-pr-comment-queue-and-comment-resolution-sessions`
**Created**: 2026-03-29
**Status**: Draft (Audit R1 revisions applied)
**Input**: Linear ticket DEV-203

## Architecture Notes

### Inter-Component Signaling

The comment watcher and callback router (DEV-202) must communicate session outcomes. The mechanism:

1. `CommentDispatcher` exposes `RegisterSession(ticketID string, prNumber int) <-chan SessionResult`
2. Per-PR goroutine calls `RegisterSession` before `SpawnSession`, then blocks on the returned channel
3. When the callback router handles `CommentResolvedEvent` or `FailureEvent`, it calls `CommentDispatcher.NotifyResult(ticketID string, result SessionResult)`
4. `SessionResult` is `{Kind: "resolved"|"failed", TicketID, PRNumber}`

The callback router must be given a reference to the `CommentDispatcher` at wiring time.

### PR-to-Ticket Lookup

`ListOpenPRs` returns `PRSummary{Number}` but job operations require `ticketID`. A new `GetJobByPR(ctx, prNumber int) (*Job, error)` method must be added to `StateStore`. This queries the `jobs` table by `pr_number`.

### Comment Types

**Scope: review comments only (`CommentKindReview`).**

Review comments are inline code feedback on diffs — the primary use case for "resolve this." Issue comments (top-level PR conversation) are out of scope for the initial implementation. They can be added by polling with `CommentKindIssue` in a follow-up.

### Slot Lifecycle for Comment Sessions

- **Acquire**: Per-PR goroutine calls `TryAcquireSlot` before `SpawnSession`
- **Release on success**: Callback router's `handleCommentResolved` must be updated to call `ReleaseSlot`
- **Release on failure**: Callback router's `handleFailed` already calls `ReleaseSlot`

### Watermark Lifecycle

- **Advance on success**: Callback router's `handleCommentResolved` calls `SetCommentWatermark` (already implemented in DEV-202)
- **Advance on failure/drain**: Watcher 2 advances watermark to the last enqueued comment ID before clearing the queue. This prevents retry loops if needs-attention is later cleared manually.

### Session Payload Delivery

Before calling `SpawnSession`, the per-PR goroutine writes the comment to `{worktreePath}/.ai/comment.json`:
```json
{
  "id": 12345,
  "author": "reviewer",
  "body": "This function should handle the nil case",
  "path": "pkg/handler.go",
  "diff_hunk": "@@ -10,5 +10,8 @@..."
}
```

The session's prompt/spec already exists at `{worktreePath}/.ai/ticket.md` from Watcher 1.

### Prerequisites

- **Type mismatch cleanup**: `StateStore` uses `int64` for `prNumber` in watermark methods, but `GetCommentsSince` and `PRSummary.Number` use `int`. Standardize to `int` across all PR number parameters before implementation.
- **Add `ReleaseSlot` call to `handleCommentResolved`** in the callback router.
- **Add `GetJobByPR` to `StateStore` interface and SQLite implementation.**

### Target File

Implementation goes in `internal/orchestrator/watcher_comment.go`, replacing the `NewWatcherStub(WatcherCommentWatcher, ...)` call in `orchestrator.go`.

## User Scenarios & Testing

### User Story 1 - New PR review comments are detected and queued (Priority: P1)

When a reviewer posts inline review comments on an orchestrator-managed PR, the comment watcher detects them on its next poll cycle and enqueues them for serial processing.

**Why this priority**: Without comment detection, there is no feature. This is the sensing loop.

**Independent Test**: Post a review comment on a tracked PR, wait one poll interval, verify the comment appears in the per-PR queue.

**Acceptance Scenarios**:

1. **Given** a PR with 3 new review comments past the watermark, **When** Watcher 2 polls, **Then** all 3 are enqueued and processed one at a time in order
2. **Given** a PR with no new comments, **When** Watcher 2 polls, **Then** no comments are enqueued and no sessions are spawned
3. **Given** a comment authored by the configured bot username, **When** Watcher 2 polls, **Then** that comment is skipped (not enqueued)
4. **Given** a comment was resolved and watermark advanced, **When** a reviewer posts a follow-up reply (higher ID), **Then** the follow-up is detected on the next poll and enqueued as a new comment

---

### User Story 2 - Comments are processed serially per-PR via AI sessions (Priority: P1)

For each queued comment, the watcher spawns a fresh AI coding session to resolve it, respecting the global concurrency limit. Only one comment per PR is in-flight at a time. The per-PR goroutine blocks on a `SessionResult` channel until the callback router signals completion or failure.

**Why this priority**: Serial processing prevents conflicting edits and ensures each comment gets the AI's full attention on the correct branch state.

**Independent Test**: Enqueue 2 comments for the same PR, verify session 2 does not start until session 1's `SessionResult` is received.

**Acceptance Scenarios**:

1. **Given** comment session 1 is in-flight for PR #42, **When** comment 2 is queued for PR #42, **Then** session 2 does not start until session 1's `SessionResult` is received
2. **Given** two PRs (#42 and #43) each have new comments simultaneously, **When** both are detected, **Then** each PR's queue is processed independently — different PRs don't block each other
3. **Given** the concurrency limit is reached (no slots available), **When** a comment is ready to dispatch, **Then** the per-PR goroutine waits (with context check) until a slot becomes available
4. **Given** a comment is dispatched, **When** `SpawnSession` is called, **Then** the comment payload has been written to `{worktree}/.ai/comment.json` beforehand

---

### User Story 3 - PR merge during active session triggers abort (Priority: P2)

If a PR is merged or closed while a comment-resolution session is active, the session is killed, the remaining comment queue for that PR is drained, and Watcher 3 handles the cleanup.

**Why this priority**: Prevents wasted compute on stale work and avoids conflicting with merged code.

**Independent Test**: Start a comment session, merge the PR externally, verify the session is killed and remaining queue is drained.

**Acceptance Scenarios**:

1. **Given** a PR merges while a comment session is active, **When** IsMerged is detected (checked before each dispatch and periodically), **Then** the active session is killed via `KillSession`, the remaining queue is drained without spawning more sessions, and the per-PR goroutine exits
2. **Given** a PR is closed without merging while a comment session is active, **When** IsClosed is detected, **Then** the same abort behavior applies

---

### User Story 4 - Session failure clears the comment queue (Priority: P2)

When a comment-resolution session fires a FailureEvent, no further comment sessions are spawned for that PR. The queue is cleared, the watermark is advanced, and the job moves to needs-attention.

**Why this priority**: Prevents infinite retry loops on comments the AI cannot resolve.

**Independent Test**: Spawn a session that fails (FailureEvent callback), verify remaining queue is drained, watermark is advanced, and job status is needs-attention.

**Acceptance Scenarios**:

1. **Given** a comment session fires a FailureEvent, **When** the `SessionResult{Kind: "failed"}` is received by the per-PR goroutine, **Then** the remaining comment queue for that PR is cleared, the watermark is advanced to the last enqueued comment ID, and no further comment sessions are spawned
2. **Given** the queue is cleared after failure, **When** new comments arrive on subsequent polls, **Then** they are NOT enqueued because the job is in needs-attention state (FR-007 applies)
3. **Given** needs-attention is manually cleared and the watermark was advanced past the failed comments, **When** the next poll runs, **Then** only new comments posted after the failure are processed

---

### User Story 5 - PR removed from tracked set triggers goroutine reaping (Priority: P3)

When a PR is no longer returned by `ListOpenPRs` (label removed, PR deleted), the per-PR goroutine for that PR is reaped.

**Why this priority**: Prevents goroutine leaks for orphaned PRs.

**Independent Test**: Create a per-PR goroutine, remove the tracking label from the PR, verify the goroutine is cancelled on the next poll.

**Acceptance Scenarios**:

1. **Given** a per-PR goroutine exists for PR #42, **When** PR #42 is no longer in the `ListOpenPRs` result, **Then** the goroutine's context is cancelled and it exits cleanly

---

### User Story 6 - Graceful shutdown drains cleanly (Priority: P3)

When the orchestrator receives SIGTERM, the comment watcher stops polling, waits for any in-flight session to complete (within the shutdown manager's timeout), and exits cleanly.

**Why this priority**: Required for clean deploys but not a primary behavior.

**Independent Test**: Send SIGTERM during an active comment session, verify the watcher exits without crashing.

**Acceptance Scenarios**:

1. **Given** the context is cancelled (shutdown signal), **When** the watcher is mid-poll, **Then** it exits cleanly
2. **Given** the context is cancelled with in-flight sessions, **When** the shutdown timeout expires, **Then** the shutdown manager handles forced exit (no separate timeout in the comment watcher)

---

### Edge Cases

- Rate limit during comment poll -> Poll is skipped for this PR, retried on next tick (existing rate limit transport handles backoff)
- State store unreachable when reading watermark -> Log error, skip this PR for this cycle, retry next tick
- Edited comment (same ID, updated body) -> Not re-processed; watermark is ID-based and the ID doesn't change on edit
- Callback server event buffer full -> Callback server returns 503; session's caller retries. Not a comment watcher concern.
- Session spawns but callback never arrives -> Per-PR goroutine blocks on `SessionResult` channel indefinitely; shutdown manager's timeout handles this case. Consider: add a per-session timeout in the per-PR goroutine (e.g., 30 minutes) as a safety net.

## Requirements

### Functional Requirements

- **FR-001**: System MUST poll for orchestrator-owned PRs (identified by tracking label from config) on the configured poll interval
- **FR-002**: System MUST fetch new review comments (`CommentKindReview`) past the stored watermark for each tracked PR using `GetCommentsSince`
- **FR-003**: System MUST enqueue new comments per-PR and process them serially (one session at a time per PR) using a goroutine-per-active-PR with a buffered channel
- **FR-004**: System MUST write the comment payload to `{worktree}/.ai/comment.json` and spawn a fresh AI session via `SpawnSession` for each queued comment
- **FR-005**: System MUST acquire a concurrency slot via `TryAcquireSlot` before spawning each session — comment sessions count against the global limit
- **FR-006**: System MUST skip comments authored by the configured bot username (new config field: `BotUsername`)
- **FR-007**: System MUST skip PRs whose job status is terminal (complete, closed, needs-attention) by looking up the job via `GetJobByPR`
- **FR-008**: System MUST abort the active session (`KillSession`) and drain the queue when a PR is merged (`IsMerged`) or closed (`IsClosed`), and reap the per-PR goroutine
- **FR-009**: System MUST clear the remaining comment queue and advance the watermark to the last enqueued comment ID when a `SessionResult{Kind: "failed"}` is received
- **FR-010**: System MUST reap per-PR goroutines when the PR is no longer in the `ListOpenPRs` result set (label removed, PR deleted)
- **FR-011**: System MUST exit cleanly when the context is cancelled (graceful shutdown), relying on the shutdown manager's timeout
- **FR-012**: System MUST register each dispatched session with `CommentDispatcher.RegisterSession` and block on the returned channel for the `SessionResult` before dispatching the next comment

### Key Entities

- **CommentDispatcher**: Owns the map of active per-PR goroutines (`map[int]*prQueue`). Provides `RegisterSession`/`NotifyResult` for signaling between the callback router and per-PR goroutines. Protected by mutex during map operations only.
- **prQueue**: Per-PR goroutine with a buffered channel of `PRComment`. Lazily created on first comment, reaped on PR merge/close/failure/removal from tracked set.
- **SessionResult**: `{Kind: "resolved"|"failed", TicketID, PRNumber}` — written by callback router, read by per-PR goroutine.
- **Comment Watermark**: Last successfully processed comment ID per PR, stored in StateStore. Advanced by callback router on success, by Watcher 2 on failure/drain.

## Success Criteria

### Measurable Outcomes

- **SC-001**: New review comments on tracked PRs are detected within one poll interval
- **SC-002**: Comments for the same PR are processed serially — no concurrent sessions per PR
- **SC-003**: PR merge/close during an active session results in session kill + queue drain within one poll interval
- **SC-004**: Session failure results in queue clear + watermark advance + needs-attention status with no further automated processing
- **SC-005**: Orchestrator can track and process comments across multiple PRs independently
- **SC-006**: No goroutine leaks — per-PR goroutines are reaped when PRs leave the tracked set

## Testing Requirements

### Test Strategy

Integration tests exercising the comment watcher against mock implementations of `github.Client`, `SessionManager`, `StateStore`, and a real `CommentDispatcher`:

- Test the polling loop, dispatching, queue draining, and failure handling
- Test the signaling contract between callback router and comment dispatcher
- Test graceful shutdown via context cancellation

The synthetic E2E test (DEV-204) covers the full vertical slice against real infrastructure.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify polling loop calls `ListOpenPRs` with tracking label on each tick |
| FR-002 | Integration | Verify `GetCommentsSince` is called with `CommentKindReview` and correct watermark per PR |
| FR-003 | Integration | Verify two comments for the same PR are dispatched serially (second waits for SessionResult) |
| FR-004 | Integration | Verify `comment.json` is written to worktree and `SpawnSession` is called with correct args |
| FR-005 | Integration | Verify `TryAcquireSlot` is called before `SpawnSession`; dispatch blocks when no slots available |
| FR-006 | Integration | Verify comments from configured bot username are filtered before enqueueing |
| FR-007 | Integration | Verify PRs with terminal job status are skipped |
| FR-008 | Integration | Verify `KillSession` is called and queue is drained on `IsMerged` or `IsClosed` detection |
| FR-009 | Integration | Verify queue is cleared, watermark advanced, no further sessions after FailureEvent |
| FR-010 | Integration | Verify per-PR goroutine is cancelled when PR leaves `ListOpenPRs` result |
| FR-011 | Integration | Verify watcher exits cleanly on context cancellation |
| FR-012 | Integration | Verify `RegisterSession` returns channel, `NotifyResult` unblocks the goroutine |

### Edge Case Coverage

- Rate limit during poll -> Verify poll is skipped, watcher continues on next tick
- State store unreachable -> Verify error is logged, PR is skipped for this cycle
- Empty comment list -> Verify no sessions spawned, no state changes
- PR removed from tracked set between polls -> Verify per-PR goroutine is reaped
- Session callback never arrives -> Verify per-PR goroutine blocks; shutdown manager timeout handles cleanup
