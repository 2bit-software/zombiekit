# Research Summary: Orchestrator E2E Test

## Key Finding: All plumbing exists, the composition test doesn't

The orchestrator has excellent dependency injection — `NewProjectRunner()` accepts all dependencies via constructor, and every external service has a recording mock/stub. The gap is purely a composition test: nobody has wired them all together in a single test that exercises the full lifecycle.

## Test Infrastructure Available

### Mock Clients (with call recording)
- `linear.MockClient` — configurable `*Fn` fields per method, `Calls []Call` history with method+args
- `github.MockClient` — same pattern, records `Calls []Call`
- Both are thread-safe, support error injection, and record argument values

### Watcher Single-Tick Methods
Watchers are ticker-based loops, but expose the inner poll method directly:
- `pollAndProcess(ctx)` — one Linear poll cycle (watcher_linear.go)
- `pollPRLifecycle(ctx, logger)` — one PR lifecycle check (watcher_pr.go)
- `pollComments(ctx, dispatcher, logger)` — one comment poll cycle (watcher_comment.go)
- `eventRouter(ctx)` — consumes events from channel until closed (router.go)

This means tests can drive the loop deterministically without timers.

### Event Injection
Events are channel-based, not HTTP. The callback server parses HTTP → `Event` → channel, but tests can skip the HTTP layer:
```go
events <- callback.Event{Kind: callback.EventComplete, ProjectID: "p", TicketID: "t"}
```
The `routerFixture` pattern in router_test.go demonstrates this: `f.events <- evt; close(f.events); f.runner.eventRouter(ctx)`

### State Store
Real SQLite store via `state.NewSQLiteStore(ctx, path)` — fast, no external deps, full query API for assertions (`GetJob`, `ListSlots`, `GetCommentWatermark`, etc.)

### Worktree Manager
Real git operations via `worktree.New(repoDir)` — works with temp repos via `t.TempDir()` + `git init`. Existing test helper `initTestRepo(t)` creates a valid temp repo with initial commit.

### Session Manager
Interface: `SpawnSession`, `KillSession`, `SessionExists`. Trivially stubbed — existing pattern returns `"session-" + ticketID`.

### Archiver & Auditor
Injected post-construction on `ProjectRunner`: `p.archiver = &mockArchiver{}`. Both have Noop implementations available.

### Comment Dispatcher
Real `CommentDispatcher` used in tests (not mocked). Coordinates per-PR serial processing:
- `RegisterSession(ticketID, prNumber)` → returns `<-chan SessionResult` (blocks per-PR goroutine)
- `NotifyResult(ticketID, result)` → unblocks the channel (called by router)

## Design Decision: Mock vs Real APIs

**Recommendation: All mocks (in-process)**

| Approach | Pros | Cons |
|----------|------|------|
| Mock clients | Fast, deterministic, CI-friendly, no credentials | Lower fidelity |
| Real APIs | Higher fidelity | Slow, flaky, needs credentials, pollutes state |

The existing unit tests already validate each mock ↔ real client parity. The E2E test's job is to verify *composition*, not *API fidelity*. Mocks are the right choice.

## Design Decision: Test Structure

**Recommendation: Single test with sequential subtests**

```go
func TestE2E_FullLifecycle(t *testing.T) {
    // shared setup
    t.Run("ticket_pickup", func(t *testing.T) { ... })
    t.Run("session_complete", func(t *testing.T) { ... })
    t.Run("comment_detection", func(t *testing.T) { ... })
    t.Run("comment_resolution", func(t *testing.T) { ... })
    t.Run("merge_cleanup", func(t *testing.T) { ... })
    t.Run("crash_recovery", func(t *testing.T) { ... })
}
```

Sequential subtests share state store state, giving each phase the cumulative result of prior phases. This matches the real orchestrator flow.

## Design Decision: Timer vs Manual Driving

**Recommendation: Manual poll driving**

Call `pollAndProcess()`, `pollPRLifecycle()`, etc. directly instead of starting `RunSupervised()` with short intervals. This eliminates timing flakiness and makes assertions deterministic.

## Reconciliation Gap

The ticket (DEV-204) specifies crash recovery testing, but there is no `PlanReconciliation` function in the codebase. The state store has `ListAllJobs()` and `ListJobsByStatus()` which could be used to detect orphaned jobs, and `ResetAllSlots()` for cleanup. The reconciliation test may need to be scoped to "verify orphaned jobs are detectable" rather than "verify automatic recovery".

## Composition: NewProjectRunner

```go
func NewProjectRunner(
    cfg ProjectConfig,
    store state.StateStore,
    lc linear.Client,
    gh github.Client,
    wt worktree.Manager,
    sm cmux.SessionManager,
    events <-chan callback.Event,
    sandboxAvailable bool,
    sandboxCfg sandbox.Config,
    logger *slog.Logger,
) *ProjectRunner
```

Post-construction injection: `runner.archiver`, `runner.auditor`
