# Implementation Plan: Orchestrator E2E Integration Test

## Overview

Single test file `internal/orchestrator/e2e_test.go` containing two test functions:
- `TestE2E_FullLifecycle` — 5 sequential subtests exercising the complete orchestrator loop
- `TestE2E_CrashRecovery` — orphan detection via state store primitives

All dependencies are in-process (mocks, real SQLite, real git worktrees). No build tags needed.

## Implementation Steps

### Step 1: Test helpers and mock types

Create shared test infrastructure at the top of `e2e_test.go`:

1. **`initTestRepo(t *testing.T) string`** — copy from `worktree/manager_test.go:14-33` (different package, can't import). Also copy `runGit(t, dir, args)` from same file :36-43.

2. **`e2eArchiver`** — enhanced version of existing `mockArchiver` (router_test.go:160). The existing mock discards `eventKind`; we need it for assertions. New type records both `ticketID` and `eventKind`:
   ```go
   type e2eArchiver struct {
       mu    sync.Mutex
       calls []archiverCall
   }
   type archiverCall struct {
       ticketID  string
       eventKind callback.EventKind
   }
   ```
   **Why not extend existing:** modifying `mockArchiver` in router_test.go would change its test contract for no benefit to those tests.

3. **`e2eAuditor`** — same pattern as e2eArchiver, implements `Auditor` interface

4. **`stubSession`** — REUSE existing from `watcher_linear_test.go:123-158` (same package, directly accessible)

5. **`e2eFixture`** — new fixture (no existing fixture covers full scope). Bundles all shared state:
   ```go
   type e2eFixture struct {
       t          *testing.T
       ctx        context.Context
       store      state.StateStore
       linear     *linear.MockClient
       github     *github.MockClient
       worktrees  worktree.Manager
       sessions   *stubSession
       runner     *ProjectRunner
       archiver   *e2eArchiver
       auditor    *e2eAuditor
       logger     *slog.Logger
       repoDir    string
       // Mutable state captured across phases
       worktreePath string
       prNumber     int64
       commentID    int64
   }
   ```

6. **`newE2EFixture(t *testing.T) *e2eFixture`** — wires everything together:
   - `initTestRepo(t)` for temp git repo
   - `state.NewSQLiteStore(ctx, tempPath)` with `t.Cleanup(store.Close)`
   - `worktree.New(repoDir)` for real worktree manager
   - `linear.MockClient{}` with initial PollReadyTickets configured
   - `github.MockClient{}` (methods configured per phase)
   - `stubSession{}` (from existing test pattern)
   - `NewProjectRunner(cfg, store, linear, github, wt, sessions, events, false, sandbox.Config{}, logger)`
   - Post-construction: `runner.archiver = archiver`, `runner.auditor = auditor`

**Dependency:** None. This is pure scaffolding.

### Step 2: Phase 1 subtest — Ticket Pickup

```go
t.Run("phase1_ticket_pickup", func(t *testing.T) { ... })
```

**Setup:**
- Configure `linear.PollReadyTicketsFn` to return one ticket:
  ```go
  linear.Ticket{
      ID: "uuid-test", Identifier: "DEV-999",
      Title: "Test feature", Description: "Spec content",
  }
  ```
- Configure `linear.SetTicketStatusFn`, `RemoveLabelFn` as no-ops (recording)

**Execute:** `f.runner.pollAndProcess(f.ctx)`

**Assert:**
- `f.linear.Calls` contains: PollReadyTickets("ai-ready", "test-linear-project"), SetTicketStatus("DEV-999", "In Progress"), RemoveLabel("DEV-999", "ai-ready")
- `f.store.GetJob(ctx, "test-project", "DEV-999")` returns job with status "queued"
- `f.store.ListSlots(ctx)` shows ActiveCount=1
- Worktree directory exists on disk
- `{worktreePath}/.ai/ticket.md` exists and contains ticket content
- `f.sessions.calls` contains "SpawnSession"

**Capture:** `f.worktreePath` from job.WorktreePath for subsequent phases

**Dependency:** Step 1 (fixture)

### Step 3: Phase 2 subtest — Session Completion

```go
t.Run("phase2_session_completion", func(t *testing.T) { ... })
```

**Setup:**
- Write `.ai/pr-description.md` to `f.worktreePath`:
  ```go
  os.MkdirAll(filepath.Join(f.worktreePath, ".ai"), 0o755)
  os.WriteFile(filepath.Join(f.worktreePath, ".ai", "pr-description.md"),
      []byte("## Test PR\nThis is a test."), 0o644)
  ```
- Configure `linear.GetTicketFn` to return ticket details (Identifier: "DEV-999", Title: "Test feature")
- Configure `github.CreatePRFn` to return PR number 42
- Configure `github.ApplyLabelFn` as no-op
- Reset `f.linear.Calls` and `f.github.Calls` slices to isolate phase assertions

**Execute:** Send event to the long-lived events channel (eventRouter is already running in background goroutine, started at fixture setup):
```go
f.eventsCh <- callback.Event{
    Kind:      callback.EventComplete,
    ProjectID: "test-project",
    TicketID:  "DEV-999",
    Branch:    "dev-999/test-feature",
}
// Wait for side effects via require.Eventually
require.Eventually(t, func() bool {
    return len(f.github.Calls) >= 2
}, 2*time.Second, 5*time.Millisecond, "expected CreatePR and ApplyLabel calls")
```

**Assert:**
- `f.github.Calls` contains: CreatePR with title "DEV-999: Test feature", ApplyLabel with "ai-managed"
- `f.store.GetJob(ctx, "test-project", "DEV-999")` has PRNumber=42, status still "queued"
- `f.store.ListSlots(ctx)` still ActiveCount=1 (slot NOT released)
- `f.archiver.calls` has one entry with ticketID="DEV-999", eventKind=EventComplete
- `f.auditor.calls` has one entry with ticketID="DEV-999", eventKind=EventComplete

**Capture:** `f.prNumber = 42`

**Dependency:** Step 2 (Phase 1 must have run first)

### Step 4: Phase 3 subtest — Comment Detection

```go
t.Run("phase3_comment_detection", func(t *testing.T) { ... })
```

**Setup:**
- Configure `github.ListOpenPRsFn` to return PR 42 with tracking label "ai-managed"
- Configure `github.GetCommentsSinceFn` to return one comment: `{ID: 100, Body: "Fix the typo", Author: "reviewer"}`
- Configure `github.IsMergedFn` to return `false, nil` (required: `handleQueuedComment` calls `prStillOpen` before processing)
- Configure `github.IsClosedFn` to return `false, nil` (required: `prStillOpen` checks both)
- Reset mock call slices

**Execute:** `f.runner.pollComments(f.ctx, f.runner.dispatcher, f.logger)`

**Note:** `pollComments` spawns a per-PR goroutine that calls `RegisterSession` then `SpawnSession` (stub), then blocks on the dispatcher's `SessionResult` channel. The `pollComments` call itself returns after dispatching — the goroutine continues running in the background. `processComment` calls `acquireSlotBlocking` which acquires a second concurrency slot (ActiveCount goes from 1 to 2).

**Assert:**
- `f.sessions.calls` contains a new "SpawnSession" entry (comment-resolution session)
- `f.runner.dispatcher.ActivePRs()` includes PR 42 (queue is live)
- `f.store.ListSlots(ctx)` shows ActiveCount=2 (Phase 1 slot + Phase 3 slot)

**Capture:** `f.commentID = 100`

**Dependency:** Step 3 (Phase 2 must have set PR number)

### Step 5: Phase 4 subtest — Comment Resolution

```go
t.Run("phase4_comment_resolution", func(t *testing.T) { ... })
```

**Setup:**
- Configure `github.UpdatePRBodyFn`, `github.PostCommentReplyFn` as no-ops
- Reset mock call slices, archiver/auditor call slices

**Execute:** Send event to the long-lived events channel (same runner, same dispatcher as Phase 3):
```go
f.eventsCh <- callback.Event{
    Kind:      callback.EventCommentResolved,
    ProjectID: "test-project",
    TicketID:  "DEV-999",
    CommentID: "100",
}
// Wait for side effects — handleCommentResolved also calls dispatcher.NotifyResult internally
require.Eventually(t, func() bool {
    return len(f.github.Calls) >= 2
}, 2*time.Second, 5*time.Millisecond, "expected UpdatePRBody and PostCommentReply")
```

**Assert:**
- `f.github.Calls` contains: UpdatePRBody, PostCommentReply(42, 100, ...)
- `f.store.GetCommentWatermark(ctx, "test-project", 42)` == 100
- `f.store.ListSlots(ctx)` ActiveCount=1 (was 2 after Phase 3, ReleaseSlot called)
- `f.archiver.calls` has entry with eventKind=EventCommentResolved
- `f.auditor.calls` has entry with eventKind=EventCommentResolved
- Per-PR goroutine from Phase 3 unblocked (dispatcher.NotifyResult was called internally by handleCommentResolved)

**Dependency:** Step 4 (Phase 3 goroutine must be running, waiting on SessionResult)

### Step 6: Phase 5 subtest — Merge Cleanup

```go
t.Run("phase5_merge_cleanup", func(t *testing.T) { ... })
```

**Setup:**
- Configure `github.IsMergedFn` to return true
- Configure `github.IsClosedFn` to return `false, nil` (safety net — `IsMerged` short-circuits, but configure for robustness)
- Configure `github.ListOpenPRsFn` to return PR 42 (still open from GitHub's perspective for the watcher to find, but IsMerged returns true)
- Reset mock call slices

**Execute:** `f.runner.pollPRLifecycle(f.ctx, f.logger)`

**Assert:**
- Worktree directory no longer exists on disk
- `f.linear.Calls` contains: SetTicketStatus("DEV-999", "done")
- `f.store.GetJob(ctx, "test-project", "DEV-999")` has status "closed"
- `f.store.ListSlots(ctx)` ActiveCount=0 (all slots freed)

**Dependency:** Step 5 (job must be in "queued" status with PR number set)

### Step 7: Crash recovery test

```go
func TestE2E_CrashRecovery(t *testing.T) { ... }
```

Separate function, independent of the lifecycle test.

**Execute:**
1. Create fresh SQLite store in temp dir
2. `store.TryAcquireSlot(ctx, "test-project", 2)` — acquire a slot
3. `store.CreateJob(ctx, "ORPHAN-1", "/tmp/fake-worktree", "session-orphan", "test-project")`
4. `store.SetJobStatus(ctx, "test-project", "ORPHAN-1", state.StatusInProgress)`

**Assert:**
1. `store.ListJobsByStatus(ctx, "test-project", state.StatusInProgress)` returns 1 job with TicketID "ORPHAN-1"
2. Job has inspectable WorktreePath and TicketID
3. `store.ResetAllSlots(ctx)` returns count > 0
4. `store.ListSlots(ctx)` shows ActiveCount=0

**Dependency:** None (standalone test)

## Fixture Architecture Decision

The biggest implementation challenge is managing the events channel across phases. Three options:

**Option A (recommended): Long-lived eventRouter goroutine**
- Create one events channel at fixture setup
- Start `eventRouter` in a goroutine at setup, cancel via context at cleanup
- Phases 2 and 4 send events to the channel, then wait for side effects
- Requires a synchronization mechanism (poll mock call counts or use `time.Sleep` with generous timeout)

**Option B: Per-phase eventRouter**
- Close and recreate the events channel per phase
- Requires rebuilding the ProjectRunner each time (since events is set in constructor)
- Loses the dispatcher state (new runner = new dispatcher), breaking Phase 3→4 flow

**Option C: Direct handler calls**
- Skip eventRouter entirely, call `handleEvent` directly
- Most deterministic, but bypasses the channel-based composition we're trying to test

**Decision: Option A.** It tests the real composition. For synchronization, use testify's `require.Eventually` (already a project dependency) instead of a custom helper:
```go
require.Eventually(t, func() bool {
    return len(f.github.Calls) >= 2
}, 2*time.Second, 5*time.Millisecond, "expected CreatePR and ApplyLabel calls")
```

## File Structure

```
internal/orchestrator/e2e_test.go
```

Single file, ~350-450 lines. No new packages or files beyond the test.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Timing flakiness from async eventRouter | Use `require.Eventually` with 2s timeout / 5ms poll |
| Phase 3 goroutine leak if Phase 4 doesn't unblock it | Cancel context in t.Cleanup — goroutine will exit |
| MockClient.Calls data race | `linear.MockClient` and `github.MockClient` use bare `[]Call` slices with NO mutex. The eventRouter goroutine writes Calls while the test goroutine reads them. **Fix:** wrap `require.Eventually` condition reads inside the mock's existing method calls (which are serialized), or accept the race and run without `-race` flag, or add a locking wrapper in the test. Best option: add `sync.Mutex` to both MockClient types (small production mock change, benefits all tests). |
| PushBranch fails because temp repo has no remote | Create a bare repo as local remote (see below) |

## Critical Implementation Detail: PushBranch

The real `worktree.GitManager.PushBranch` calls `git push`, which requires a remote. A temp repo has no remote. Two options:

1. **Use real worktree for Create/Delete, mock for PushBranch** — would need a composite or wrapper
2. **Use a stub worktree manager for everything** — loses the real-git-operations value
3. **Add a remote to the temp repo** — `git remote add origin {another-temp-dir}` + `git init --bare` on the remote

**Decision: Option 3.** Create a bare repo as the "remote", add it to the test repo. This keeps real git operations throughout. Pattern:
```go
bareDir := t.TempDir()
runGit(t, bareDir, "init", "--bare")
runGit(t, repoDir, "remote", "add", "origin", bareDir)
runGit(t, repoDir, "push", "-u", "origin", "main")
```
