# Tasks: DEV-155 Crash-Recovery Reconciliation

## Dependency Graph

```
T001 ──┐
T002 ──┤
T003 ──┼──> T005 ──> T006 ──> T007 ──> T008 ──> T009
T004 ──┘
```

T001-T004 are parallelizable (all modify `internal/state/` but independent concerns).
T005+ are sequential.

## Tasks

### Phase 1: Status Constants and Store Method Extensions

- [ ] T001 [P] Define status constants in `internal/state/store.go`. Add `StatusQueued`, `StatusInProgress`, `StatusNeedsAttention`, `StatusComplete`, `StatusClosed` as package-level constants. Refactor `CreateJob` to use `StatusQueued` instead of string literal `"queued"`. Update existing test assertions in `internal/state/store_test.go` to use `StatusQueued` constant.
  - **Acceptance**: All existing tests pass. `StatusQueued` constant used in `CreateJob` and tests. No behavior change.
  - **FR**: Prerequisite (status constants)
  - **Files**: `internal/state/store.go`, `internal/state/store_test.go`

- [ ] T002 [P] Add `ListJobsByStatus(ctx context.Context, statuses ...string) ([]Job, error)` to the `StateStore` interface and implement on `SQLiteStore` in `internal/state/store.go`. SQL: `SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at FROM jobs WHERE status IN (?)` with dynamic placeholder expansion for variadic args. Return empty slice (not nil) when no matches. Add tests in `internal/state/store_test.go`: (1) `TestListJobsByStatus_FiltersCorrectly` -- mixed statuses, query one; (2) `TestListJobsByStatus_MultipleStatuses` -- query two statuses; (3) `TestListJobsByStatus_NoMatches` -- returns empty slice; (4) `TestListJobsByStatus_EmptyStore` -- returns empty slice.
  - **Acceptance**: All 4 tests pass. Interface updated. Method returns `[]Job` matching the requested statuses.
  - **FR**: FR-001
  - **Files**: `internal/state/store.go`, `internal/state/store_test.go`

- [ ] T003 [P] Add `SetJobStatus(ctx context.Context, ticketID string, status string) error` to the `StateStore` interface and implement on `SQLiteStore` in `internal/state/store.go`. SQL: `UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE ticket_id = ?`. Return `ErrJobNotFound` if no row affected. Add tests in `internal/state/store_test.go`: (1) `TestSetJobStatus_UpdatesStatusAndTimestamp` -- create job, set status, verify changed and `updated_at` advanced; (2) `TestSetJobStatus_NonExistent_ReturnsErrJobNotFound`.
  - **Acceptance**: Both tests pass. Interface updated. Status and `updated_at` are updated atomically.
  - **FR**: FR-002
  - **Files**: `internal/state/store.go`, `internal/state/store_test.go`

- [ ] T004 [P] Add `ResetAllSlots(ctx context.Context) (int, error)` to the `StateStore` interface and implement on `SQLiteStore` in `internal/state/store.go`. SQL: `UPDATE concurrency_slots SET active_count = 0 WHERE active_count > 0`. Return count of rows reset. Add tests in `internal/state/store_test.go`: (1) `TestResetAllSlots_ResetsActiveCounts` -- acquire slots for multiple projects, reset, verify all counts are 0 and return value matches; (2) `TestResetAllSlots_NoActiveSlots` -- returns 0.
  - **Acceptance**: Both tests pass. Interface updated. All active slot counts set to 0.
  - **FR**: FR-011
  - **Files**: `internal/state/store.go`, `internal/state/store_test.go`

### Phase 2: Reconciliation Pure Function

- [ ] T005 Define `OrphanedJob`, `ReconciliationPlan` types and implement `PlanReconciliation(jobs []Job, now time.Time) ReconciliationPlan` in `internal/state/reconcile.go`. Add `HasFindings() bool` method on `ReconciliationPlan`. The function iterates jobs, classifies any with `Status == StatusInProgress` as orphaned, computes `StaleDuration` as `now.Sub(job.UpdatedAt)`. Pure function -- no I/O, no logging. Add table-driven tests in `internal/state/reconcile_test.go`: (1) empty input -> empty plan; (2) all terminal (complete, closed) -> empty plan; (3) single in-progress -> one orphaned with correct fields; (4) multiple in-progress -> all detected; (5) mixed statuses -> only in-progress flagged; (6) nil PRNumber -> still detected; (7) non-nil PRNumber -> carried through; (8) stale duration = now - UpdatedAt.
  - **Acceptance**: All 8 test cases pass. Function is pure (no imports of I/O packages).
  - **FR**: FR-001, FR-009
  - **Files**: `internal/state/reconcile.go`, `internal/state/reconcile_test.go`

### Phase 3: Reconciliation Imperative Shell

- [ ] T006 Implement `ApplyReconciliation(ctx context.Context, store StateStore, logger *slog.Logger) error` in `internal/state/reconcile.go`. Sequence: (1) `store.ListJobsByStatus(ctx, StatusInProgress)`; (2) `PlanReconciliation(jobs, time.Now())`; (3) if no findings, log `"reconciliation complete: no orphaned jobs found"` and return nil; (4) for each orphaned job, call `store.SetJobStatus` and log per-job details (ticket_id, previous_status, new_status, worktree_path, stale_duration); (5) call `store.ResetAllSlots(ctx)` and log slots_reset count; (6) log summary (orphaned_count, slots_reset, elapsed). Return first error encountered (fail fast).
  - **Acceptance**: Function compiles and is callable with a real StateStore and logger.
  - **FR**: FR-002, FR-003, FR-004, FR-005, FR-007, FR-011
  - **Files**: `internal/state/reconcile.go`

- [ ] T007 Add integration tests for `ApplyReconciliation` in `internal/state/reconcile_test.go`: (1) clean state -- empty store, verify nil error; (2) single orphaned job -- create in-progress job, run, verify status is `needs-attention` in DB; (3) multiple orphaned jobs -- create several, verify all transitioned; (4) mixed statuses -- only in-progress affected; (5) slot reset -- acquire slots, create in-progress job, run, verify slots reset to 0; (6) DB error on query -- close store before reconciliation, verify error returned; (7) DB error on status update -- seed in-progress job, induce failure on SetJobStatus path, verify error returned.
  - **Acceptance**: All 7 integration tests pass.
  - **FR**: FR-001 through FR-011 (integration coverage)
  - **Files**: `internal/state/reconcile_test.go`

### Phase 4: Startup Integration

- [ ] T008 In `internal/cli/start.go`, add state store initialization and reconciliation call in `runStart()`, after config validation and before `mgr.Run()`. Use `BRAINS_DATA_DIR` / `~/.brains/` pattern for state store path (same as GUI service). Create `state.NewSQLiteStore`, defer `Close()`, call `state.ApplyReconciliation` with `logging.Logger()`. If either errors, return wrapped error (prevents startup).
  - **Acceptance**: `start` command initializes state store and runs reconciliation before launching services. Compilation succeeds. Error in reconciliation prevents service startup.
  - **FR**: FR-006, FR-007
  - **Files**: `internal/cli/start.go`

### Final Verification

- [ ] T009 Run full test suite (`go test ./internal/state/...`). Verify all existing tests still pass alongside new tests. Verify no compilation errors across the project (`go build ./...`).
  - **Acceptance**: Zero test failures. Zero build errors.
  - **Files**: All
