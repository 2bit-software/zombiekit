# Feature Specification: Crash-Recovery Reconciliation on Startup

**Feature Branch**: `69c736df-feature-dev-155-crash-recovery-reconciliation`
**Created**: 2026-03-27
**Status**: Draft
**Input**: Linear ticket DEV-155

## User Scenarios & Testing

### User Story 1 - Detect Interrupted Jobs on Startup (Priority: P1)

When the orchestrator process starts, it scans the state store for jobs that were in-progress when the previous process terminated. These jobs are marked as "needs-attention" so the operator knows they require human triage. The reconciliation report is logged before any watchers begin polling.

**Why this priority**: Without this, interrupted jobs sit in "in-progress" forever with no visibility. This is the core value of the feature.

**Independent Test**: Create a state store with an in-progress job, run reconciliation, verify the job status changes to "needs-attention" and a structured log entry is produced.

**Acceptance Scenarios**:

1. **Given** the state store contains a job in `in-progress` status, **When** reconciliation runs on startup, **Then** that job is identified in the report and its status is set to `needs-attention`
2. **Given** the state store contains a job in `in-progress` status with a recorded worktree path, **When** reconciliation runs, **Then** the report includes the worktree path for manual inspection

---

### User Story 2 - Clean Startup with No Interrupted Jobs (Priority: P1)

When there are no non-terminal jobs in the state store, the orchestrator starts normally. Reconciliation still runs (to confirm it wasn't skipped) but produces no warnings.

**Why this priority**: Equal to P1 because the no-op path must work correctly -- a false positive that blocks startup would be worse than no reconciliation at all.

**Independent Test**: Create a state store with only terminal-status jobs (complete, closed), run reconciliation, verify no jobs are flagged and a "no orphaned jobs" log entry is produced.

**Acceptance Scenarios**:

1. **Given** zero non-terminal jobs in the state store, **When** reconciliation runs, **Then** the process starts normally with no warnings
2. **Given** the state store contains jobs in `complete` or `closed` status only, **When** reconciliation runs, **Then** those jobs are not flagged

---

### User Story 3 - Structured Reconciliation Report (Priority: P2)

The reconciliation produces a structured report that includes per-job details (ticket ID, previous status, worktree path, how long the job was stale) and a summary (total orphaned count, elapsed time). This gives the operator enough information to triage without needing to query the database manually.

**Why this priority**: The report enriches the core detection (P1) with actionable context. Without it, the operator knows *something* was interrupted but not *what* or *where*.

**Independent Test**: Create a state store with multiple in-progress jobs with different worktree paths and ages, run reconciliation, verify the report contains per-job details and accurate summary counts.

**Acceptance Scenarios**:

1. **Given** reconciliation detects orphaned jobs, **When** the report is logged, **Then** each entry includes ticket ID, previous status, new status, worktree path, and stale duration
2. **Given** reconciliation completes (with or without findings), **When** the summary is logged, **Then** it includes the total orphaned count and elapsed time

---

### Edge Cases

- What happens when the state store is empty (no jobs at all)? Reconciliation completes as no-op.
- What happens when a job has status "queued" (never started)? Not flagged in initial implementation -- only "in-progress" jobs are considered orphaned.
- What happens when reconciliation itself fails (e.g., database error)? Startup fails. The orchestrator should not proceed with an unknown state.
- What happens when multiple jobs are in-progress? All are independently detected and marked.

## Requirements

### Functional Requirements

- **FR-001**: System MUST scan all jobs in the state store on startup and identify those with `in-progress` status as orphaned
- **FR-002**: System MUST mark jobs in `in-progress` status as `needs-attention`
- **FR-003**: System MUST log a structured report of each orphaned job including: ticket ID, previous status, new status, worktree path, and stale duration
- **FR-004**: System MUST log a summary after reconciliation completes, including total orphaned count and elapsed time
- **FR-005**: System MUST log an explicit confirmation when no orphaned jobs are found (distinguishes "clean" from "skipped")
- **FR-006**: System MUST complete reconciliation before any watcher goroutine starts polling
- **FR-007**: System MUST fail startup if reconciliation encounters an error (fail fast)
- **FR-008**: System MUST NOT contact Linear or GitHub during reconciliation
- **FR-009**: System MUST NOT automatically retry interrupted jobs
- **FR-010**: System MUST NOT delete worktrees of interrupted jobs
- **FR-011**: System MUST reset all concurrency slot counts to zero during reconciliation (in a single-process orchestrator, a crash means all slots are orphaned; per-job release is not possible because Job does not track project_id)

### Prerequisites

- Two new methods must be added to the `StateStore` interface: `ListJobsByStatus(ctx context.Context, statuses ...string) ([]Job, error)` and `SetJobStatus(ctx context.Context, ticketID string, status string) error`. These do not currently exist.
- `SetJobStatus` updates both the `status` column and `updated_at` to the current time (consistent with existing `SetPR` convention). Stale duration is captured in the reconciliation log before the status transition occurs.
- `SetJobStatus` is a general-purpose method with no transition validation. Callers are responsible for correctness.
- No schema migration is required -- the `status` column is `TEXT` with no constraint. `needs-attention` is a valid value.
- The startup path does not currently instantiate a `StateStore`. The reconciliation caller must create one (using a configured DB path) and call `ApplyReconciliation` before launching watcher goroutines.
- Define status constants (`StatusQueued`, `StatusInProgress`, `StatusNeedsAttention`, `StatusComplete`, `StatusClosed`) and refactor existing `CreateJob` literal to use `StatusQueued`.

### Known Limitations

- This feature detects and flags orphaned jobs but does not provide a mechanism for the operator to resolve them. A future ticket should add a CLI command or API to transition jobs out of `needs-attention`.
- End-to-end testing requires seeding the database with `in-progress` jobs directly, since no existing code transitions jobs from `queued` to `in-progress`. This transition is expected in a future ticket.
- `CmuxSession` is intentionally excluded from the reconciliation report -- it's relevant to cleanup but not to triage.

### Key Entities

- **Job**: Existing entity in the state store. Key attributes: ticket_id, worktree_path, status, updated_at. Reconciliation reads and transitions status.
- **ReconciliationPlan**: Output of the reconciliation logic. Contains a list of orphaned jobs with their details. Pure data -- no side effects.
- **OrphanedJob**: A job identified as needing attention. Includes ticket ID, previous status, worktree path, PR number (if any), and stale duration (calculated as `now - job.UpdatedAt`).

## Success Criteria

### Measurable Outcomes

- **SC-001**: An in-progress job left by a crashed process is detected and marked within the first second of startup
- **SC-002**: The operator can identify all interrupted jobs and their worktree paths from the startup log alone (no manual DB queries needed)
- **SC-003**: A clean startup (no orphans) produces no warnings and adds negligible latency
- **SC-004**: Reconciliation logic is testable with zero database involvement (pure function)

## Testing Requirements

### Test Strategy

- **Unit tests** for the pure reconciliation function (`PlanReconciliation`) -- table-driven tests with constructed Job values, no database
- **Integration tests** for the store methods (`ListJobsByStatus`, `SetJobStatus`) -- using `setupTestStore` pattern with in-memory SQLite
- **Integration test** for the full reconciliation flow (`ApplyReconciliation`) -- end-to-end through the store, verifying both state transitions and log output

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Unit | PlanReconciliation identifies in-progress jobs from a mixed-status list |
| FR-002 | Integration | ApplyReconciliation transitions in-progress jobs to needs-attention in the DB |
| FR-003 | Unit | OrphanedJob contains all required fields (ticket_id, worktree_path, stale_duration) |
| FR-004 | Integration | Summary log emitted after reconciliation with correct counts |
| FR-005 | Unit + Integration | No-op case returns empty plan; integration test verifies log output |
| FR-006 | N/A | Ordering guarantee -- verified by code structure (sequential call before event loop) |
| FR-007 | Integration | DB error during reconciliation propagates and prevents startup |
| FR-008 | N/A | Verified by absence -- reconciliation has no HTTP/API dependencies |
| FR-009 | Unit | PlanReconciliation never produces "retry" actions, only "mark" |
| FR-010 | N/A | Verified by absence -- no filesystem operations in reconciliation |
| FR-011 | Integration | All slot counts reset to zero during reconciliation |

### Edge Case Coverage

- Empty state store (no jobs) -> Unit test: empty input returns empty plan
- All jobs terminal (complete/closed) -> Unit test: no orphans detected
- Multiple in-progress jobs -> Unit test: all independently detected
- Job with no PR number -> Unit test: PRNumber is nil, still detected
- Job with PR number -> Unit test: PRNumber included in OrphanedJob
- DB error on query -> Integration test: error propagated
- DB error on status update -> Integration test: error propagated
