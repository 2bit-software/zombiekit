# Implementation Plan: Watcher 2 — PR Comment Queue

## Overview

4 phases, 12 steps. Each step is independently testable. Dependencies are strict — later steps must not start before their prerequisites are complete.

## Phase 0: Prerequisites (changes to existing code)

### Step 0.1: Add `GetJobByPR` to StateStore

**Why**: Comment watcher polls by PR number but all job operations require ticket ID. No lookup path exists.

**Files**:
- `internal/state/store.go` — Add `GetJobByPR(ctx context.Context, prNumber int64) (*Job, error)` to interface + SQLite impl
- `internal/state/store_test.go` — Test: create job, set PR, get by PR number

**Implementation**:
```sql
SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at
FROM jobs WHERE pr_number = ?
```

**Note**: Uses `int64` for prNumber to match existing `SetPR`/watermark signatures. Callers convert from `int` where needed. The spec calls for standardizing all PR number types to `int` — this is deferred to a separate cleanup PR (touches StateStore interface, SQLite impl, and all callers across watchers and router). Tracked as a follow-up.

**Verify**: `go test ./internal/state/...`

---

### Step 0.2: Add `BotUsername` to Config

**Why**: FR-006 requires filtering bot-authored comments. Need a config field for the bot's GitHub username.

**Files**:
- `internal/orchestrator/config.go` — Add `BotUsername string` field, add to `Validate()` (required)
- `cmd/brains/orchestrator.go` (or wherever CLI flags are defined) — Add `--bot-username` / `ORCH_BOT_USERNAME` flag

**Verify**: Config validation test passes with new field.

---

### Step 0.3: Add `ReleaseSlot` to `handleCommentResolved`

**Why**: The slot acquired by the comment watcher before `SpawnSession` must be released when the comment resolution succeeds. Currently only `handleFailed` releases slots.

**Files**:
- `internal/orchestrator/router.go` — Add `ReleaseSlot` call at the end of `handleCommentResolved`, after watermark is set

**Verify**: Existing router tests pass + new test verifies slot is released on comment-resolved.

---

## Phase 1: Core Types

### Step 1.1: Define `SessionResult` and `CommentDispatcher`

**Why**: The signaling mechanism between the callback router and per-PR goroutines. This is the architectural centerpiece.

**Files**:
- `internal/orchestrator/comment_dispatcher.go` — New file

**Types**:
```go
type SessionResultKind string
const (
    SessionResolved SessionResultKind = "resolved"
    SessionFailed   SessionResultKind = "failed"
)

type SessionResult struct {
    Kind     SessionResultKind
    TicketID string
    PRNumber int
}

type prQueue struct {
    comments chan github.PRComment
    cancel   context.CancelFunc
}

type CommentDispatcher struct {
    mu       sync.Mutex
    queues   map[int]*prQueue           // PR number -> queue
    sessions map[string]chan SessionResult // ticketID -> completion channel
    logger   *slog.Logger
}
```

**Methods**:
- `NewCommentDispatcher(logger) *CommentDispatcher`
- `RegisterSession(ticketID string, prNumber int) <-chan SessionResult` — creates a buffered(1) channel, stores in map, returns read end
- `NotifyResult(ticketID string, result SessionResult)` — writes to channel if registered, logs warning if not
- `GetQueue(prNumber int) *prQueue` — returns existing queue or nil
- `CreateQueue(prNumber int, cancel context.CancelFunc) *prQueue` — creates queue with buffered channel
- `RemoveQueue(prNumber int)` — cancels and removes
- `ActivePRs() []int` — returns list of PR numbers with active queues (for reaping)

**Verify**: Unit tests for register/notify round-trip, duplicate registration, notify-without-registration.

---

## Phase 2: Comment Watcher

### Step 2.1: Implement polling loop

**Why**: The outer loop that polls `ListOpenPRs` and dispatches new comments to per-PR queues.

**Files**:
- `internal/orchestrator/watcher_comment.go` — New file

**Pattern**: Follows `watcher_linear.go` — method on `Orchestrator` returning `shutdown.ServiceFunc`.

```go
func (o *Orchestrator) NewCommentWatcher(dispatcher *CommentDispatcher) shutdown.ServiceFunc
```

**Poll cycle logic**:
1. `ListOpenPRs(ctx, cfg.TrackingLabel)` — get tracked PRs
2. For each PR:
   a. `GetJobByPR(ctx, prNumber)` — look up job
   b. Skip if job is nil or status is terminal (complete, closed, needs-attention)
   c. `GetCommentWatermark(ctx, prNumber)` — get last processed ID
   d. `GetCommentsSince(ctx, prNumber, CommentKindReview, watermark)` — fetch new comments
   e. Filter out comments where `Author == cfg.BotUsername`
   f. For each remaining comment: dispatch to per-PR queue (create queue lazily if needed)
3. Reap stale queues: for each active queue, if its PR is no longer in the `ListOpenPRs` result, cancel and remove it

**Error handling**: Log and skip individual PRs on error. Never fail the entire poll cycle for one PR.

**Verify**: Integration test with mock `github.Client` and `StateStore`.

---

### Step 2.2: Implement per-PR goroutine

**Why**: Each PR gets a dedicated goroutine that processes comments serially, one session at a time.

**Files**:
- `internal/orchestrator/watcher_comment.go` — Same file, `runPRQueue` function

**Goroutine logic**:
```
for {
    select {
    case <-ctx.Done():
        return  // shutdown or PR reaped
    case comment, ok := <-queue.comments:
        if !ok:
            return  // queue closed (failure path)

        // Check PR state before dispatching
        if merged/closed:
            killSession if active, drain queue, return

        // Acquire concurrency slot (blocking with context check)
        for {
            acquired := TryAcquireSlot(ctx, projectID, limit)
            if acquired: break
            select {
            case <-ctx.Done(): return
            case <-time.After(5s): continue  // retry
            }
        }

        // Write comment payload
        writeCommentJSON(worktreePath, comment)

        // Register completion channel
        done := dispatcher.RegisterSession(ticketID, prNumber)

        // Spawn session
        SpawnSession(ctx, ticketID, title, worktreePath, env)

        // Block until result
        select {
        case <-ctx.Done():
            return
        case result := <-done:
            if result.Kind == SessionFailed:
                // Advance watermark to last enqueued comment
                // Drain remaining channel
                return
            // SessionResolved: loop to next comment
        }
    }
}
```

**Key details**:
- `writeCommentJSON` writes `{worktree}/.ai/comment.json` with id, author, body, path, diff_hunk
- Slot acquisition retries every 5s with context check
- On failure result: drain channel (tracking max ID from drained comments), advance watermark to highest ID across processed + drained comments, exit goroutine
- On merge detection: `KillSession`, drain channel, exit goroutine

**Verify**: Integration test with controlled session results.

---

## Phase 3: Integration Wiring

### Step 3.1: Wire `CommentDispatcher` into Router

**Why**: Router needs to call `NotifyResult` after handling comment-related events.

**Files**:
- `internal/orchestrator/router.go` — Add `dispatcher *CommentDispatcher` field to `Router`
- `internal/orchestrator/router.go` — Update `NewRouter` signature to accept `*CommentDispatcher`
- `internal/orchestrator/router.go` — Call `dispatcher.NotifyResult(...)` at end of `handleCommentResolved` and `handleFailed`

**`handleCommentResolved` additions** (after existing watermark/archive logic):
```go
r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
    Kind:     SessionResolved,
    TicketID: evt.TicketID,
    PRNumber: prNumber,
})
```

**`handleFailed` additions** (after existing slot release/status update):
```go
r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
    Kind:     SessionFailed,
    TicketID: evt.TicketID,
})
```

**Verify**: Existing router tests compile with updated signature. New tests verify NotifyResult is called.

---

### Step 3.2: Wire comment watcher into `Orchestrator.Run()`

**Why**: Replace the stub with the real implementation.

**Files**:
- `internal/orchestrator/orchestrator.go` — Create `CommentDispatcher`, pass to Router and comment watcher

**Changes**:
```go
dispatcher := NewCommentDispatcher(logger)

router := NewRouter(
    callbackSrv.Events(),
    o.store, o.github, o.linear,
    archival.NoopArchiver{}, friction.NoopAuditor{},
    dispatcher,  // NEW
    o.cfg, logger,
)

commentWatcher := o.NewCommentWatcher(dispatcher)  // replaces stub
```

**Verify**: Orchestrator starts cleanly, comment watcher logs startup message.

---

## Phase 4: Tests

### Step 4.1: Unit tests for CommentDispatcher

**File**: `internal/orchestrator/comment_dispatcher_test.go`

**Tests**:
- `TestRegisterAndNotify` — register session, notify result, verify channel receives
- `TestNotifyWithoutRegistration` — notify for unregistered ticketID, verify no panic
- `TestCreateAndRemoveQueue` — lifecycle of PR queues
- `TestActivePRs` — verify list reflects current state

---

### Step 4.2: Integration tests for comment watcher

**File**: `internal/orchestrator/watcher_comment_test.go`

**Tests** (using mock interfaces):
- `TestPollDetectsNewComments` — verify `GetCommentsSince` called with correct watermark
- `TestSerialProcessing` — two comments on same PR processed one at a time
- `TestBotCommentFiltered` — verify bot-authored comments skipped
- `TestTerminalJobSkipped` — verify needs-attention/complete/closed PRs skipped
- `TestMergeDetection` — verify session killed and queue drained on merge
- `TestFailureDrainsQueue` — verify queue cleared and watermark advanced on failure
- `TestGracefulShutdown` — verify context cancellation exits cleanly
- `TestPRReaping` — verify goroutine cancelled when PR leaves tracked set
- `TestSlotBlocking` — verify dispatch waits when no slots available
- `TestIndependentPRQueues` — verify two PRs process concurrently
- `TestFollowUpCommentAfterResolution` — resolve a comment, verify watermark advanced, post a follow-up (higher ID), verify it's detected on next poll

---

## Dependency Graph

```
0.1 (GetJobByPR) ──┐
0.2 (BotUsername) ──┼── 1.1 (Dispatcher types) ── 2.1 (Poll loop) ──┐
0.3 (ReleaseSlot) ──┘                             2.2 (PR goroutine)─┼── 3.1 (Wire Router)
                                                                      ├── 3.2 (Wire Orchestrator)
                                                                      └── 4.1, 4.2 (Tests)
```

Phase 0 steps are independent of each other. Phase 1 depends on Phase 0. Phase 2 depends on Phase 1. Phase 3 depends on Phase 2. Phase 4 can be written alongside Phase 2-3.

## Risk Register

| Risk | Mitigation |
|------|------------|
| Session callback never arrives | Per-PR goroutine blocks indefinitely; shutdown manager timeout handles it. Consider adding a 30-min safety timeout in a follow-up. |
| Race between poller reaping and goroutine dispatching | Mutex on dispatcher map prevents concurrent create/remove. Context cancellation propagates cleanly. |
| High comment volume on single PR overwhelms channel | Buffered channel (capacity 100). If full, log warning and skip — comments will be re-fetched on next poll (watermark hasn't advanced). |
