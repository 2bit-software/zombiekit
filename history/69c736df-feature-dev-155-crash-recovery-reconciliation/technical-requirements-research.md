# Technical Requirements Research: Crash-Recovery Reconciliation

## Technical Preferences (from Linear ticket)

- No contacting Linear or GitHub during reconciliation (Epic 3 concern)
- No automatic retry of interrupted jobs
- No worktree deletion on recovery (preserve worktree on failure)
- Keep it simple: startup scan that logs and marks
- Reconciliation must complete before any watcher goroutine starts polling

## Implementation Hints

### Architecture

- **Functional core**: `PlanReconciliation(jobs []Job, now time.Time) ReconciliationPlan` -- pure function, no I/O
- **Imperative shell**: `ApplyReconciliation(ctx, store, logger)` -- queries DB, calls pure function, applies transitions, logs report
- This follows the Nomad reconciler pattern and the project's functional core / imperative shell convention

### New StateStore Methods Needed

- `ListJobsByStatus(ctx, statuses ...string) ([]Job, error)` -- query for non-terminal jobs
- `SetJobStatus(ctx, ticketID, status string) error` -- update status and `updated_at`

### Status Constants

Define in `internal/state/store.go` or a separate `status.go`:
```
StatusQueued         = "queued"
StatusInProgress     = "in-progress"
StatusNeedsAttention = "needs-attention"
StatusComplete       = "complete"
StatusClosed         = "closed"
```

### File Placement

- `internal/state/reconcile.go` -- pure function + types
- `internal/state/reconcile_test.go` -- unit tests for pure function
- Store method additions in `internal/state/store.go`
- Store method tests in `internal/state/store_test.go`

### Logging Structure

Per-job log:
```
logger.Info("reconciliation: orphaned job detected",
    slog.String("ticket_id", job.TicketID),
    slog.String("previous_status", job.Status),
    slog.String("new_status", "needs-attention"),
    slog.String("worktree_path", job.WorktreePath),
    slog.Duration("stale_duration", staleDuration),
)
```

Summary log:
```
logger.Info("reconciliation complete",
    slog.Int("orphaned_count", len(plan.Orphaned)),
    slog.Duration("elapsed", elapsed),
)
```

No-op log:
```
logger.Info("reconciliation complete: no orphaned jobs found")
```
