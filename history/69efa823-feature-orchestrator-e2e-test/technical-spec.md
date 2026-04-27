# Technical Spec: Orchestrator E2E Integration Test

## Architecture

Single test file `internal/orchestrator/e2e_test.go` in the `orchestrator` package (same package as production code — access to unexported methods and fields).

## Component Wiring

```
                    ┌─────────────────────┐
                    │    e2eFixture       │
                    │                     │
                    │  linear.MockClient  │──── PollReadyTickets, SetTicketStatus, GetTicket, etc.
                    │  github.MockClient  │──── CreatePR, ListOpenPRs, GetCommentsSince, etc.
                    │  worktree.GitManager│──── CreateWorktree, DeleteWorktree, PushBranch
                    │  state.SQLiteStore  │──── CreateJob, GetJob, SetPR, ListSlots, etc.
                    │  stubSession        │──── SpawnSession (returns ref, no-op)
                    │  mockArchiver       │──── Archive (records calls)
                    │  mockAuditor        │──── Audit (records calls)
                    │  eventsCh (chan)     │──── EventComplete, EventCommentResolved
                    │                     │
                    │  ProjectRunner      │──── pollAndProcess, eventRouter, pollComments, pollPRLifecycle
                    │   └─ dispatcher     │──── RegisterSession, NotifyResult (created internally)
                    └─────────────────────┘
```

## Fixture Design

```go
type e2eFixture struct {
    t          *testing.T
    ctx        context.Context
    cancel     context.CancelFunc
    store      state.StateStore
    linear     *linear.MockClient
    github     *github.MockClient
    sessions   *stubSession
    runner     *ProjectRunner
    archiver   *mockArchiver
    auditor    *mockAuditor
    logger     *slog.Logger
    eventsCh   chan callback.Event  // send side — test writes here
    repoDir    string
    bareDir    string              // bare remote for PushBranch

    // State captured across phases
    worktreePath string
    prNumber     int64
    commentID    int64
}
```

### Initialization Flow

1. Create temp dirs: repo, bare remote, worktrees root, SQLite DB
2. Initialize git repo with initial commit + bare remote
3. Create real `state.NewSQLiteStore` + `store.Migrate(ctx)`
4. Create real `worktree.New(repoDir)` (with worktrees root option if available)
5. Create mock clients (Linear, GitHub) with Phase 1 defaults
6. Create stub session manager
7. Create events channel: `make(chan callback.Event, 8)` (buffered)
8. Create ProjectRunner via `NewProjectRunner`
9. Inject `mockArchiver` and `mockAuditor` on runner
10. Start `eventRouter` goroutine (long-lived, cancelled by `t.Cleanup`)
11. Register cleanup: close events channel, cancel context, close store

### Event Router Lifecycle

The `eventRouter` runs for the entire test in a background goroutine. It blocks on the events channel. When the test completes, `t.Cleanup` closes the events channel and cancels the context, causing `eventRouter` to return.

```go
routerCtx, routerCancel := context.WithCancel(ctx)
routerDone := make(chan struct{})
go func() {
    defer close(routerDone)
    f.runner.eventRouter(routerCtx)
}()
t.Cleanup(func() {
    routerCancel()
    <-routerDone
})
```

Events channel must NOT be closed — instead, use context cancellation to stop the router. Closing the channel would prevent Phase 4 from sending events.

## Synchronization Strategy

The eventRouter processes events asynchronously. After sending an event, the test needs to wait for side effects to complete. Use a poll-with-timeout helper:

```go
func waitFor(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(5 * time.Millisecond)
    }
    t.Fatalf("timed out waiting for: %s", msg)
}
```

Usage: `waitFor(t, 2*time.Second, func() bool { return len(f.github.Calls) >= 2 }, "CreatePR and ApplyLabel")`

## Git Setup for PushBranch

The real `GitManager.PushBranch` calls `git push`, requiring a remote. The fixture creates a bare repo as a local remote:

```go
bareDir := t.TempDir()
runGit(t, bareDir, "init", "--bare")
repoDir := t.TempDir()
runGit(t, repoDir, "init")
runGit(t, repoDir, "config", "user.name", "Test")
runGit(t, repoDir, "config", "user.email", "test@test.com")
os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0o644)
runGit(t, repoDir, "add", "README.md")
runGit(t, repoDir, "commit", "-m", "initial")
runGit(t, repoDir, "remote", "add", "origin", bareDir)
runGit(t, repoDir, "push", "-u", "origin", "main")
```

## Mock Types (new)

### mockArchiver

```go
type mockArchiver struct {
    mu    sync.Mutex
    calls []mockArchiverCall
}
type mockArchiverCall struct {
    ticketID  string
    eventKind callback.EventKind
}
func (a *mockArchiver) Archive(_ context.Context, ticketID string, eventKind callback.EventKind) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.calls = append(a.calls, mockArchiverCall{ticketID, eventKind})
    return nil
}
```

### mockAuditor

Same structure as mockArchiver, implementing `Auditor.Audit(ctx, ticketID, eventKind) error`.

## Phase-Specific Mock Configuration

### Phase 1 (initial setup)
```go
linear.PollReadyTicketsFn = returns [{ID: "uuid-1", Identifier: "DEV-999", Title: "Test feature", Description: "Spec"}]
linear.SetTicketStatusFn = no-op (records)
linear.RemoveLabelFn = no-op (records)
```

### Phase 2 (before sending EventComplete)
```go
linear.GetTicketFn = returns {Identifier: "DEV-999", Title: "Test feature"}
github.CreatePRFn = returns prNumber=42
github.ApplyLabelFn = no-op (records)
```

### Phase 3 (before calling pollComments)
```go
github.ListOpenPRsFn = returns [{Number: 42, Labels: ["ai-managed"]}]
github.GetCommentsSinceFn = returns [{ID: 100, Body: "Fix typo", Author: "reviewer"}]
```

### Phase 4 (before sending EventCommentResolved)
```go
github.UpdatePRBodyFn = no-op (records)
github.PostCommentReplyFn = no-op (records)
```

### Phase 5 (before calling pollPRLifecycle)
```go
github.IsMergedFn = returns true
github.ListOpenPRsFn = returns [{Number: 42}]  // or via ListJobsByStatus
```

## Key Assertions Per Phase

### Phase 1
- `store.GetJob` returns status="queued"
- `store.ListSlots` shows ActiveCount=1
- Worktree dir exists, `.ai/ticket.md` file exists
- Linear calls: PollReadyTickets, SetTicketStatus("In Progress"), RemoveLabel("ai-ready")
- Session calls: SpawnSession

### Phase 2
- `store.GetJob` has PRNumber=42, status still "queued"
- `store.ListSlots` still ActiveCount=1
- GitHub calls: CreatePR("DEV-999: Test feature", ...), ApplyLabel("ai-managed")
- Archiver: called with ("DEV-999", EventComplete)
- Auditor: called with ("DEV-999", EventComplete)

### Phase 3
- Session calls: new SpawnSession entry (comment-resolution)
- Per-PR goroutine is running (dispatcher has active PR)

### Phase 4
- `store.GetCommentWatermark` == 100
- `store.ListSlots` ActiveCount decreased (ReleaseSlot)
- GitHub calls: UpdatePRBody, PostCommentReply
- Archiver: called with ("DEV-999", EventCommentResolved)

### Phase 5
- `store.GetJob` has status="closed"
- `store.ListSlots` ActiveCount=0
- Worktree dir no longer exists
- Linear calls: SetTicketStatus("done")
