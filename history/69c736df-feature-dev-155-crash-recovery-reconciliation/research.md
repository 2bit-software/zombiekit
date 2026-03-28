---
status: complete
updated: 2026-03-27
---

# Research: Crash-Recovery Reconciliation on Startup

## Executive Summary

Every major job orchestrator (Airflow, Nomad, olivere/jobqueue) converges on the same pattern: on startup, query for jobs in active states, classify as orphaned, transition to an attention-required state, and log a structured report. For a single-process orchestrator like ZombieKit, this simplifies dramatically -- any job in "in-progress" at startup is orphaned by definition. The recommended implementation is a pure function that takes a list of jobs and returns a reconciliation plan (functional core), with an imperative shell that queries the DB, applies the plan, and logs the report.

## Findings

### Codebase Context

- **StateStore interface** (`internal/state/store.go`): Has `CreateJob`, `GetJob`, `SetPR`, watermark, and slot methods. Missing `ListJobsByStatus` and `SetJobStatus` -- both needed for reconciliation.
- **Job struct**: Contains `TicketID`, `WorktreePath`, `CmuxSession`, `PRNumber`, `Status`, `CreatedAt`, `UpdatedAt`. Status is free-form `TEXT` with default "queued".
- **SQLite store**: WAL mode, single connection (`SetMaxOpenConns(1)`), 5s busy timeout, foreign keys enabled.
- **Error pattern**: Sentinel errors in `errors.go` (`ErrJobNotFound`, `ErrJobExists`, `ErrInvalidDBPath`).
- **Test pattern**: `setupTestStore(t)` helper using `t.TempDir()`, testify assertions, table-driven tests.
- **Startup pattern**: `internal/startup/service.go` defines `Service` interface with `Name()` and `Run(ctx)`. Shutdown manager coordinates services via errgroup.
- **No existing reconciliation or crash recovery logic anywhere in the codebase.**
- **DB() accessor** on SQLiteStore exposes raw `*sql.DB` for custom queries (noted in DEV-154).

### Domain Knowledge

- **Airflow**: `adopt_or_reset_orphaned_tasks` runs on startup before the scheduler loop. Queries QUEUED/RUNNING tasks, joins against scheduler table to detect orphans, resets or adopts them.
- **olivere/jobqueue**: Clean Go API with `StartupBehaviour` enum (`None`, `MarkAsFailed`). `Store.Start()` is the reconciliation hook called during manager startup.
- **Nomad**: `reconciler.Compute()` is a pure function that classifies allocations into buckets and produces a `reconcileResults` plan. Imperative code applies the plan.
- **Temporal**: Uses timeout-based detection (heartbeat), not startup reconciliation. Not applicable for single-process, but heartbeat concept is a future extension point.

## Decision Points

- [x] **D1**: Which statuses are "active" (non-terminal)? -> "in-progress" only for initial implementation. "queued" can be added later.
- [x] **D2**: Should concurrency slots be released for orphaned jobs? -> Yes, as part of the imperative shell. Reconciliation plan identifies jobs; slot release is a side effect of applying the plan.
- [x] **D3**: Where does the reconciliation function live? -> `internal/state/reconcile.go` alongside the store, since it operates on the same types.
- [x] **D4**: Should status values be defined as constants? -> Yes, a `const` block prevents typos and makes the valid set explicit.

## Recommendations

1. **Add `ListJobsByStatus` and `SetJobStatus` to StateStore interface** -- natural CRUD extensions, prerequisites for reconciliation.
2. **Implement `PlanReconciliation(jobs []Job, now time.Time) ReconciliationPlan`** as a pure function -- trivially testable with zero DB involvement.
3. **Implement `ApplyReconciliation`** as the imperative shell that queries, plans, applies, and logs.
4. **Call reconciliation synchronously at startup** before the event loop. If it errors, fail startup (fail fast).
5. **Log per-job details and summary at Info level**, including worktree path and stale duration. Log explicitly when no orphans found.
6. **Define status constants** (`StatusQueued`, `StatusInProgress`, `StatusNeedsAttention`, `StatusComplete`, `StatusClosed`).

## Sources

- [Airflow Scheduler Documentation](https://airflow.apache.org/docs/apache-airflow/stable/administration-and-deployment/scheduler.html)
- [Airflow adopt_or_reset_orphaned_tasks Discussion](https://github.com/apache/airflow/discussions/27983)
- [olivere/jobqueue Go Package](https://pkg.go.dev/github.com/olivere/jobqueue)
- [Nomad Scheduler Source](https://github.com/hashicorp/nomad/tree/main/scheduler)
- [Temporal Activity Recovery Discussion](https://community.temporal.io/t/activity-recovery-worker-behaviour-in-case-of-crash/8301)
