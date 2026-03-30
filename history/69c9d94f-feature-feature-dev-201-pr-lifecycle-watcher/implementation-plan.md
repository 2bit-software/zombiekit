# Implementation Plan: Watcher 3 — PR Lifecycle Detection and Cleanup

**Created**: 2026-03-29
**Traces to**: spec.md, technical-spec.md

## Implementation Order

Steps are ordered by dependency. Each step is independently testable.

### Step 1: Config Addition

**Files**: `internal/orchestrator/config.go`, `cmd/orchestrator/main.go`
**FRs**: FR-009
**Estimated LOC**: ~10

1. Add `ClosedPRTicketStatus string` to `Config` struct
2. Add parsing in `NewConfig()`: `ClosedPRTicketStatus: c.String("closed-pr-status")`
3. Add CLI flag in `main.go`:
   ```go
   &cli.StringFlag{
       Name:    "closed-pr-status",
       Usage:   "Linear ticket status for PRs closed without merge",
       Value:   "cancelled",
       EnvVars: []string{"ORCH_CLOSED_PR_STATUS"},
   },
   ```
4. No validation needed (any string is a valid Linear status)

**Verification**: `go build ./cmd/orchestrator/` compiles. Existing tests pass.

### Step 2: Watcher Implementation

**Files**: `internal/orchestrator/watcher_pr.go` (new)
**FRs**: FR-001 through FR-008, FR-010
**Estimated LOC**: ~100

1. Create `watcher_pr.go` with:
   - `NewPRWatcher()` method on `*Orchestrator` returning `shutdown.ServiceFunc`
   - Ticker-based polling loop (matching `NewLinearPoller` pattern)
   - `pollPRLifecycle(ctx, logger)` — queries `ListJobsByStatus(StatusQueued)`, filters `PRNumber != nil`, checks `IsMerged`/`IsClosed`
   - `cleanupPR(ctx, job, ticketStatus, logger)` — best-effort cleanup pipeline (DeleteWorktree, SetTicketStatus, SetJobStatus, ReleaseSlot)
2. Each cleanup step is independent with error logging
3. Context check between job iterations for graceful shutdown

**Verification**: Code compiles. No side effects until wired in Step 4.

### Step 3: Tests

**Files**: `internal/orchestrator/watcher_pr_test.go` (new)
**FRs**: All
**Estimated LOC**: ~300

1. Create test doubles: `stubWorktree`, `stubLinear`, `stubState`
   - Reuse `github.MockClient` from `internal/github/mock.go`
2. Create test helper: `buildPRWatcherOrch()` with configurable dependencies
3. Implement test cases (see technical-spec.md for full list):
   - Happy path: merged PR cleanup
   - Happy path: closed-without-merge PR cleanup
   - Skip: no PR number
   - Skip: terminal status (not returned by query)
   - Partial failures: worktree, Linear, SetJobStatus
   - Context cancellation
   - Multiple PRs in one cycle
   - Idempotency

**Verification**: `go test -run TestPRWatcher ./internal/orchestrator/` passes.

### Step 4: Wiring

**Files**: `internal/orchestrator/orchestrator.go`
**FRs**: All (integration)
**Estimated LOC**: ~2 (one line change)

1. Replace `prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)` with `prWatcher := o.NewPRWatcher()`
2. No other changes needed — dependencies are already available on `*Orchestrator`

**Verification**: `go build ./cmd/orchestrator/` compiles. All tests pass.

## Dependency Graph

```
Step 1 (config) ─┐
                  ├─→ Step 2 (implementation) ─→ Step 3 (tests) ─→ Step 4 (wiring)
                  │
(no other deps)───┘
```

Steps 1 and 2 can be done in either order. Step 3 depends on Step 2. Step 4 depends on Steps 1-3.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `DeleteWorktree` fails for already-deleted worktree | Medium | Low | Best-effort cleanup, log and continue |
| Race between Watcher 2 and Watcher 3 on same PR | Low | None | Both operations are idempotent |
| `ListJobsByStatus(StatusQueued)` returns many jobs | Low | Low | Filter is O(n) in memory, n bounded by historical job count |

## Out of Scope

- Storing branch name in Job struct (would enable orphaned branch cleanup)
- Removing tracking label from cleaned-up PRs
- Adding `StatusInProgress`/`StatusComplete` transitions to the job lifecycle
- Webhook-based detection (currently polling, per context brief)
