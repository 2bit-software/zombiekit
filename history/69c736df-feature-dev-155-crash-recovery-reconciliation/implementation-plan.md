# Implementation Plan: DEV-155 Crash-Recovery Reconciliation

## Overview

Four phases, ordered by dependency. Each phase is independently testable.

## Phase 1: Status Constants and Store Method Extensions

**Files**: `internal/state/store.go`, `internal/state/store_test.go`, `internal/state/errors.go`

### 1.1 Define status constants

Add to `store.go`:
```go
const (
    StatusQueued         = "queued"
    StatusInProgress     = "in-progress"
    StatusNeedsAttention = "needs-attention"
    StatusComplete       = "complete"
    StatusClosed         = "closed"
)
```

Refactor `CreateJob` to use `StatusQueued` instead of the string literal `"queued"`.

### 1.2 Add `ListJobsByStatus` to StateStore interface

```go
ListJobsByStatus(ctx context.Context, statuses ...string) ([]Job, error)
```

SQLiteStore implementation: `SELECT * FROM jobs WHERE status IN (?)` with dynamic placeholder expansion. Returns empty slice (not nil) when no matches.

Add sentinel error: none needed (empty result is valid).

### 1.3 Add `SetJobStatus` to StateStore interface

```go
SetJobStatus(ctx context.Context, ticketID string, status string) error
```

SQLiteStore implementation: `UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE ticket_id = ?`. Returns `ErrJobNotFound` if no row affected.

### 1.4 Add `ResetAllSlots` to StateStore interface

```go
ResetAllSlots(ctx context.Context) (int, error)
```

SQLiteStore implementation: `UPDATE concurrency_slots SET active_count = 0 WHERE active_count > 0`. Returns count of rows reset.

Rationale: In a single-process orchestrator, a crash means ALL slots are orphaned. Per-job slot release is impossible because `Job` doesn't track `project_id`. Blanket reset is correct and simple.

### 1.5 Tests

- `TestListJobsByStatus_FiltersCorrectly` -- create jobs with mixed statuses, query for one status, verify only matching jobs returned
- `TestListJobsByStatus_MultipleStatuses` -- query for two statuses, verify both match
- `TestListJobsByStatus_NoMatches` -- query for status with no jobs, verify empty slice returned
- `TestListJobsByStatus_EmptyStore` -- query on empty store, verify empty slice
- `TestSetJobStatus_UpdatesStatusAndTimestamp` -- create job, set status, verify status changed and `updated_at` advanced
- `TestSetJobStatus_NonExistent_ReturnsErrJobNotFound` -- set status on missing ticket, verify error
- `TestResetAllSlots_ResetsActiveCounts` -- acquire slots for multiple projects, reset, verify all counts are 0
- `TestResetAllSlots_NoActiveSlots` -- reset with no active slots, verify returns 0

**Deliverable**: All new store methods pass tests. Existing tests still pass.

---

## Phase 2: Reconciliation Pure Function

**Files**: `internal/state/reconcile.go`, `internal/state/reconcile_test.go`

### 2.1 Define types

```go
type OrphanedJob struct {
    TicketID       string
    PreviousStatus string
    WorktreePath   string
    PRNumber       *int64
    StaleDuration  time.Duration
}

type ReconciliationPlan struct {
    Orphaned []OrphanedJob
}

func (p ReconciliationPlan) HasFindings() bool {
    return len(p.Orphaned) > 0
}
```

### 2.2 Implement `PlanReconciliation`

```go
func PlanReconciliation(jobs []Job, now time.Time) ReconciliationPlan
```

- Iterates over `jobs`
- Any job with `Status == StatusInProgress` is classified as orphaned
- Computes `StaleDuration` as `now.Sub(job.UpdatedAt)`
- Returns a `ReconciliationPlan` with the orphaned jobs list
- Pure function: no I/O, no logging, no side effects

### 2.3 Tests (table-driven)

- Empty input -> empty plan
- All terminal jobs (complete, closed) -> empty plan
- Single in-progress job -> one orphaned entry with correct fields
- Multiple in-progress jobs -> all independently detected
- Mixed statuses (queued, in-progress, complete) -> only in-progress flagged
- Job with nil PRNumber -> still detected, PRNumber is nil in OrphanedJob
- Job with PRNumber -> PRNumber carried through to OrphanedJob
- Stale duration calculation -> verify `now - UpdatedAt` is correct

**Deliverable**: Pure function passes all table-driven tests with zero DB involvement.

---

## Phase 3: Reconciliation Imperative Shell

**Files**: `internal/state/reconcile.go` (add to same file), `internal/state/reconcile_test.go`

### 3.1 Implement `ApplyReconciliation`

```go
func ApplyReconciliation(ctx context.Context, store StateStore, logger *slog.Logger) error
```

Sequence:
1. `store.ListJobsByStatus(ctx, StatusInProgress)` -- query orphaned jobs
2. `PlanReconciliation(jobs, time.Now())` -- compute plan
3. If no findings: log `"reconciliation complete: no orphaned jobs found"`, return nil
4. For each orphaned job:
   a. `store.SetJobStatus(ctx, job.TicketID, StatusNeedsAttention)` -- transition status
   b. Log per-job details (ticket_id, previous_status, new_status, worktree_path, stale_duration, has_pr)
5. `store.ResetAllSlots(ctx)` -- release all concurrency slots
6. Log summary (orphaned_count, slots_reset, elapsed)
7. Return nil (or first error encountered)

Error handling: If any store operation fails, return the error immediately (fail fast). Partial reconciliation is acceptable -- the next startup will pick up remaining orphaned jobs.

### 3.2 Tests

- **Integration: clean state** -- empty store, verify no-op log and nil error
- **Integration: single orphaned job** -- create in-progress job, run `ApplyReconciliation`, verify status changed to `needs-attention` in DB
- **Integration: multiple orphaned jobs** -- create several in-progress jobs, verify all transitioned
- **Integration: mixed statuses** -- create jobs with various statuses, verify only in-progress jobs affected
- **Integration: slot reset** -- acquire slots, create in-progress job, run reconciliation, verify slots reset to 0
- **Integration: DB error on query** -- close store before reconciliation, verify error returned
- **Integration: DB error on status update** -- seed an in-progress job, then induce a failure on `SetJobStatus` (e.g., close store after query succeeds), verify error returned

**Deliverable**: Full integration tests pass through the store.

---

## Phase 4: Startup Integration

**Files**: `internal/cli/start.go`

### 4.1 Add state store initialization

In `runStart()`, after config validation and before `mgr.Run()`:

```go
// Initialize state store
statePath := filepath.Join(dataDir, "state.db")
stateStore, err := state.NewSQLiteStore(ctx, statePath)
if err != nil {
    return fmt.Errorf("initialize state store: %w", err)
}
defer stateStore.Close()
```

Use the same `dataDir` pattern as the GUI service (`BRAINS_DATA_DIR` env var or `~/.brains/`).

### 4.2 Call reconciliation

```go
// Run crash-recovery reconciliation before launching services
if err := state.ApplyReconciliation(ctx, stateStore, logging.Logger()); err != nil {
    return fmt.Errorf("startup reconciliation: %w", err)
}
```

This runs synchronously, blocking service startup until complete (FR-006). If it errors, startup fails (FR-007).

### 4.3 No new tests for this phase

The integration point is thin glue code -- three lines of initialization, one function call, one error check. Verified by inspection and by the Phase 3 integration tests.

---

## Dependency Graph

```
Phase 1 (store methods)
  └─> Phase 2 (pure function) -- uses Job type and status constants
       └─> Phase 3 (imperative shell) -- uses PlanReconciliation + store methods
            └─> Phase 4 (startup integration) -- calls ApplyReconciliation
```

## FR Traceability

| FR | Phase | Implementation |
|----|-------|----------------|
| FR-001 | 1.2 + 2.2 | ListJobsByStatus queries in-progress; PlanReconciliation classifies |
| FR-002 | 3.1 | ApplyReconciliation calls SetJobStatus with StatusNeedsAttention |
| FR-003 | 3.1 | Per-job structured log in ApplyReconciliation |
| FR-004 | 3.1 | Summary log at end of ApplyReconciliation |
| FR-005 | 3.1 | No-op log when plan has no findings |
| FR-006 | 4.2 | Sequential call before mgr.Run() |
| FR-007 | 4.2 | Error return prevents mgr.Run() |
| FR-008 | N/A | No HTTP/API code in reconciliation |
| FR-009 | 2.2 | PlanReconciliation only produces "mark" actions |
| FR-010 | N/A | No filesystem operations |
| FR-011 | 3.1 | ResetAllSlots called after job transitions |

## Technical Decisions

1. **Reconciliation lives in `internal/state/`** -- it operates on state types and the state store. A separate package would add unnecessary indirection.
2. **Blanket slot reset instead of per-job release** -- Job doesn't track project_id, and in a single-process system all slots are orphaned on crash. `ResetAllSlots` is correct and simple.
3. **No config extension needed** -- state store path uses the existing `BRAINS_DATA_DIR` / `~/.brains/` pattern, same as memory storage. No new env vars.
4. **Partial reconciliation on error is acceptable** -- if marking the 3rd of 5 jobs fails, the remaining 2 will be caught on the next startup. No transaction wrapping needed.
