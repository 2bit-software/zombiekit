# Business Spec: Orchestrator E2E Integration Test

## Problem

The orchestrator has solid unit test coverage (63+ tests with stubs for all external dependencies), but no test exercises the full workflow loop end-to-end. Each watcher and the callback router are tested in isolation — there's no verification that they compose correctly through the full lifecycle.

## Requirements

### R1: Full lifecycle test

A single `TestE2E_FullLifecycle` with sequential subtests sharing a real SQLite state store. Each subtest builds on cumulative state from prior phases. Watchers are driven manually via their single-tick methods (no timers).

#### Phase 1: Ticket Pickup

Call `pollAndProcess(ctx)`. Verify:
- `linear.MockClient.PollReadyTickets` called with label `"ai-ready"` (hardcoded in watcher) and the configured project ID
- `TryAcquireSlot` succeeds (returns true)
- Real worktree created in temp git repo via `worktree.New(repoDir)`
- `.ai/ticket.md` written to the worktree with ticket content
- Stub session spawned (returns `"session-{ticketID}"`)
- `linear.SetTicketStatus` called with "In Progress"
- `linear.RemoveLabel` called to remove "ai-ready"
- Job created in state store with status "queued"
- Concurrency slot count is 1

**Setup required:**
- Temp git repo via `t.TempDir()` + `git init` + initial commit
- `linear.MockClient.PollReadyTicketsFn` returns one test ticket
- `linear.MockClient.SetTicketStatusFn`, `RemoveLabelFn` configured (no-op, record calls)
- Stub session manager
- Real SQLite state store via `state.NewSQLiteStore(ctx, tempPath)`

#### Phase 2: Session Completion

Write `.ai/pr-description.md` to the worktree path (simulating agent output). Inject `callback.EventComplete` into the events channel. Start `eventRouter` in a goroutine, close channel after event is consumed. Verify:
- `worktrees.PushBranch` called (first step of `handleComplete`)
- `linear.GetTicket` called (for PR title construction)
- `github.CreatePR` called with title `"{ticket.Identifier}: {ticket.Title}"` and body from pr-description.md
- `store.SetPR` called — PR number stored on job
- `github.ApplyLabel` called with tracking label (after SetPR in execution order)
- Job status remains "queued" (completion does NOT change job status)
- Concurrency slot remains held (completion does NOT release slot)
- Archiver mock called with ticket ID and `callback.EventComplete`
- Auditor mock called with ticket ID and `callback.EventComplete`

**Setup required:**
- Write `.ai/pr-description.md` to `{job.WorktreePath}/.ai/pr-description.md`
- Configure `linear.MockClient.GetTicketFn` to return ticket details
- Configure `github.MockClient.CreatePRFn` to return a PR number (e.g., 42)
- Configure `github.MockClient.ApplyLabelFn`
- Event must include `Branch` field matching the worktree branch
- Mock archiver and auditor injected on runner post-construction

#### Phase 3: Comment Detection

Configure `github.MockClient.ListOpenPRsFn` to return the PR from Phase 2 (with tracking label). Configure `github.MockClient.GetCommentsSinceFn` to return one review comment. Call `pollComments(ctx, dispatcher, logger)`. Verify:
- Comment enqueued in per-PR queue
- Comment-resolution session spawn attempted via stub session manager
- Session blocks on `SessionResult` from dispatcher (serial processing)

**Setup required:**
- `ConcurrencyLimit >= 2` in ProjectConfig (slot from Phase 1 is still held)
- Create `CommentDispatcher` via `NewCommentDispatcher(logger)`
- Configure `github.ListOpenPRsFn` returning PR with tracking label
- Configure `github.GetCommentsSinceFn` returning one comment with ID > 0
- State store has job with PR number (set in Phase 2)
- Comment watermark at 0 (initial state)

**Note:** `pollComments` signature is `pollComments(ctx, dispatcher, logger)` — dispatcher and logger must be passed explicitly.

#### Phase 4: Comment Resolution

Inject `callback.EventCommentResolved` into a fresh events channel. Run `eventRouter` in a goroutine. The `handleCommentResolved` handler internally calls `dispatcher.NotifyResult(...)`, which unblocks the per-PR goroutine from Phase 3 — do NOT call `NotifyResult` manually in the test. Verify:
- `github.UpdatePRBody` called
- `github.PostCommentReply` called with the comment ID
- `store.SetCommentWatermark` advanced to the comment ID
- `store.ReleaseSlot` called (comment resolution releases a slot)
- Archiver mock called with ticket ID and `callback.EventCommentResolved`
- Auditor mock called with ticket ID and `callback.EventCommentResolved`

#### Phase 5: Merge Cleanup

Configure `github.MockClient.IsMergedFn` to return true. Call `pollPRLifecycle(ctx, logger)`. Verify:
- `worktrees.DeleteWorktree` called with the worktree path
- `linear.SetTicketStatus` called with "done"
- `store.SetJobStatus` called with "closed"
- `store.ReleaseSlot` called
- Job status in store is "closed"
- All concurrency slots released (count is 0)

**Note:** `pollPRLifecycle` only queries jobs with status "queued". The job must still be "queued" from Phase 2 for this to work. `CleanBranch` is NOT called by `cleanupPR` — do not assert it.

### R2: State observability

At each state transition, assert observable state via:
- **State store** (real SQLite): `GetJob`, `ListSlots`, `GetCommentWatermark` queries
- **Linear mock**: `Calls` slice — verify method names and argument values
- **GitHub mock**: `Calls` slice — verify method names and argument values
- **Archiver/Auditor mocks**: verify called with `(ticketID, eventKind)` — NOT "session context"

### R3: Crash recovery (orphan detection)

Separate from the lifecycle test. In a fresh test function:
1. Create a real SQLite state store
2. `CreateJob` with a test ticket ID
3. Manually `SetJobStatus` to "in-progress" (simulating crash after pickup, before completion)
4. Verify `ListJobsByStatus(ctx, projectID, "in-progress")` returns the orphaned job
5. Verify `ResetAllSlots(ctx)` resets orphaned slot counts
6. Verify the job's worktree path and ticket ID are inspectable for manual triage

**Note:** `ApplyReconciliation` in `internal/state/reconcile.go` exists as a higher-level function that scans for in-progress jobs, marks them needs-attention, and resets slots. This test exercises the low-level detection primitives directly. The "in-progress" status is synthetic — the orchestrator currently creates jobs as "queued" and never transitions them to "in-progress" — but we test it because it's the reconciler's contract for orphan detection.

### R4: Stub boundaries

| Dependency | Type | Rationale |
|------------|------|-----------|
| Sessions | Stub (`stubSession`) | No real Claude Code execution |
| Archival | Mock (records calls) | Verify called with correct args |
| Friction auditor | Mock (records calls) | Verify called with correct args |
| Worktrees | Real `worktree.New(tempRepo)` | Fast, tests real git operations |
| State store | Real `state.NewSQLiteStore` | Tests real SQL, cross-phase state |
| Linear | `linear.MockClient` | Call recording with args via `Calls` |
| GitHub | `github.MockClient` | Call recording with args via `Calls` |
| Comment dispatcher | Real `CommentDispatcher` | Tests real channel coordination |

Use `linear.MockClient` and `github.MockClient` (not the simpler `stub*` types from individual test files) because they record call arguments, which is needed for cross-phase assertions.

### R5: Isolation

- All mocks are in-process (no network, no credentials)
- Temp git repo via `t.TempDir()` + `git init`
- Temp SQLite DB via `t.TempDir()`
- No build tag required — runs as regular `go test`
- `t.Cleanup()` for all resources

### R6: Configuration

```go
ProjectConfig{
    ID:               "test-project",
    LinearProjectID:  "test-linear-project",
    GitHubOwner:      "test-owner",
    GitHubRepo:       "test-repo",
    BaseBranch:       "main",
    ConcurrencyLimit: 2,  // Must be >= 2: Phase 1 holds slot, Phase 3 acquires another
    TrackingLabel:    "ai-managed",
    BotUsername:      "test-bot",
    ClosedPRStatus:   "cancelled",
    PollInterval:     Duration{50 * time.Millisecond},
    CallbackPort:     9999,
    RepoDir:          tempRepoDir,
    WorktreesRoot:    tempWorktreesDir,
}
```

## Acceptance Criteria

- [ ] Full lifecycle test: 5 sequential subtests (pickup, completion, comment detection, comment resolution, merge cleanup) all pass with assertions at each transition
- [ ] Crash recovery test: orphaned job detected via `ListJobsByStatus`, slots reset via `ResetAllSlots`
- [ ] Archiver and auditor mocks called with `(ticketID, eventKind)` at completion and comment-resolution events
- [ ] No external dependencies — runs via `go test ./internal/orchestrator/...`
- [ ] No leaked resources — `t.TempDir()` and `t.Cleanup()` handle all teardown
- [ ] Comment dispatcher serial processing verified: second comment waits for first `SessionResult`
- [ ] All mock `Calls` slices inspected for correct method names and argument values

## Out of Scope

- Real ZombieKit agent/session execution
- Real archival or friction auditing logic
- Real Linear or GitHub API calls
- `EventFailed` path (meaningful but separate — can be a follow-up subtest)
- Testing `ApplyReconciliation` end-to-end (R3 tests the primitives it depends on)
- Multi-project orchestration
- CI pipeline integration
- `CleanBranch` during merge cleanup (not called by `cleanupPR`)
