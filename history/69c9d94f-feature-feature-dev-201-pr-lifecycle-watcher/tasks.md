# Tasks: Watcher 3 — PR Lifecycle Detection and Cleanup

**Complexity**: Simple (5 files, ~410 LOC)
**Traces to**: implementation-plan.md, technical-spec.md, spec.md

## Dependency Graph

```
T001 (config) ──┐
                ├──→ T003 (implementation) ──→ T004 (tests) ──→ T005 (wiring) ──→ T006 (verify)
T002 (config flag) ┘
```

T001 and T002 are parallelizable. T003-T006 are sequential.

## Tasks

### Config (Parallel)

- [ ] T001 [P] [FR-009] Add `ClosedPRTicketStatus string` field to Config struct in `internal/orchestrator/config.go`. Add parsing in `NewConfig()`: `ClosedPRTicketStatus: c.String("closed-pr-status")`. No validation needed.
  - **Acceptance**: Field exists, `NewConfig` parses it, `go build ./internal/orchestrator/` passes.

- [ ] T002 [P] [FR-009] Add `--closed-pr-status` CLI flag to `cmd/orchestrator/main.go` with default `"cancelled"` and env var `ORCH_CLOSED_PR_STATUS`.
  - **Acceptance**: Flag appears in `--help`, default is "cancelled", env var works, `go build ./cmd/orchestrator/` passes.

### Implementation (Sequential)

- [ ] T003 [FR-001,002,003,004,005,007,008] Create `internal/orchestrator/watcher_pr.go` with:
  - `NewPRWatcher()` method on `*Orchestrator` returning `shutdown.ServiceFunc` — ticker-based polling loop matching `NewLinearPoller` pattern
  - `pollPRLifecycle(ctx, logger)` — queries `ListJobsByStatus(state.StatusQueued)`, filters `PRNumber != nil`, calls `IsMerged` then `IsClosed` for each
  - `cleanupPR(ctx, job, ticketStatus, logger)` — best-effort cleanup: `DeleteWorktree`, `SetTicketStatus`, `SetJobStatus(StatusClosed)`, `ReleaseSlot`. Each step independent, errors logged.
  - Context check between job iterations for graceful shutdown
  - **Audit note**: Use `context.Background()` inside `cleanupPR` for the cleanup calls so mid-shutdown cleanup completes (FR-008)
  - **Acceptance**: `go build ./internal/orchestrator/` passes. Code follows `watcher_linear.go` patterns.

- [ ] T004 [FR-001 through FR-010] Create `internal/orchestrator/watcher_pr_test.go` with:
  - Test doubles: `stubWorktree` (implementing `worktree.Manager`), `stubLinear` (implementing `linear.Client`), `stubState` (implementing `state.StateStore`). Reuse `github.MockClient` from `internal/github/mock.go`.
  - Helper: `buildPRWatcherOrch()` assembling orchestrator with test doubles
  - Tests:
    - `TestPRWatcher_MergedPR` — full merge cleanup pipeline [FR-001,002,003]
    - `TestPRWatcher_ClosedPR` — close cleanup with configurable status [FR-001,002,004,009]
    - `TestPRWatcher_SkipNoPR` — jobs without PR number skipped [FR-007]
    - `TestPRWatcher_PartialFailure_Worktree` — DeleteWorktree fails, remaining steps proceed [FR-005]
    - `TestPRWatcher_PartialFailure_Linear` — SetTicketStatus fails, remaining steps proceed [FR-005]
    - `TestPRWatcher_PartialFailure_SetJobStatus` — SetJobStatus fails, ReleaseSlot still called [FR-005]
    - `TestPRWatcher_ContextCancelled` — exits cleanly on cancellation [FR-008]
    - `TestPRWatcher_MultiplePRs` — multiple PRs cleaned in one cycle [FR-001]
    - `TestPRWatcher_OpenPR` — open PR, no cleanup [FR-002]
    - `TestPRWatcher_Idempotent` — re-poll after cleanup, no side effects [FR-010]
    - `TestPRWatcher_MergedAndClosed` — IsMerged checked first, merge path taken [edge case]
  - **Acceptance**: `go test -count=1 -run TestPRWatcher ./internal/orchestrator/` passes (all 11 tests).

- [ ] T005 [All] Replace stub wiring in `internal/orchestrator/orchestrator.go`: change `prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)` to `prWatcher := o.NewPRWatcher()`.
  - **Acceptance**: `go build ./cmd/orchestrator/` passes. All existing tests pass.

- [ ] T006 [All] Final verification: `go test -count=1 ./internal/orchestrator/...` — all tests pass (existing + new). `go vet ./...` clean.
  - **Acceptance**: Zero test failures, zero vet warnings.

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T003, T004 |
| FR-002 | T003, T004 |
| FR-003 | T003, T004 |
| FR-004 | T003, T004 |
| FR-005 | T003, T004 |
| FR-006 | T003, T004 |
| FR-007 | T003, T004 |
| FR-008 | T003, T004 |
| FR-009 | T001, T002, T004 |
| FR-010 | T003, T004 |

## Execution Order

1. T001 + T002 (parallel)
2. T003
3. T004
4. T005
5. T006
