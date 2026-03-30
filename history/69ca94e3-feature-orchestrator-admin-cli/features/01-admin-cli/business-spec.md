# Feature Specification: Orchestrator Admin CLI

**Feature Branch**: `69ca94e3-feature-orchestrator-admin-cli`
**Created**: 2026-03-30
**Status**: Draft
**Input**: User description: "CLI commands for querying/updating the orchestrator. These commands will then be built as HTTP endpoints, and eventually a webgui, but for now focus on the commands"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List Jobs (Priority: P1)

As an operator, I need to see what jobs the orchestrator is tracking so I can understand its current state and identify problems.

**Why this priority**: Without visibility into job state, the operator is flying blind. This was the exact problem encountered — a stale "queued" job was blocking ticket pickup with no way to discover it short of raw SQL.

**Independent Test**: Run `orchestrator jobs list` against a database with known jobs and verify the output matches expected state.

**Acceptance Scenarios**:

1. **Given** the orchestrator has 3 jobs (1 queued, 1 in-progress, 1 closed), **When** I run `orchestrator jobs list`, **Then** I see all 3 jobs with their ticket ID, status, project ID, worktree path, session, PR number, and updated_at.
2. **Given** the orchestrator has jobs in mixed states, **When** I run `orchestrator jobs list --status queued`, **Then** I see only the queued jobs.
3. **Given** I pass multiple statuses `orchestrator jobs list --status queued --status in-progress`, **Then** I see only jobs matching either status.
4. **Given** the orchestrator has no jobs, **When** I run `orchestrator jobs list`, **Then** I see an empty result with no error.

---

### User Story 2 - Delete a Stale Job (Priority: P1)

As an operator, I need to remove a stale or broken job record so the orchestrator can re-pick-up the ticket or free resources.

**Why this priority**: This was the direct blocker discovered during debugging. A stale "queued" job from a previous run prevented DEV-226 from being picked up. The only workaround was raw `sqlite3` commands.

**Independent Test**: Create a job via the store, run `orchestrator jobs delete DEV-226`, verify the job is gone and the ticket can be picked up again.

**Acceptance Scenarios**:

1. **Given** a stale job exists for DEV-226, **When** I run `orchestrator jobs delete DEV-226`, **Then** the job record is removed from the database and the concurrency slot for the job's project is released.
2. **Given** no job exists for DEV-999, **When** I run `orchestrator jobs delete DEV-999`, **Then** I get a clear "job not found" error with exit code 1.

---

### User Story 3 - Show Concurrency Slots (Priority: P2)

As an operator, I need to see concurrency slot usage so I can diagnose why new tickets aren't being picked up.

**Why this priority**: Stale slots were the second hidden blocker — even after understanding the stale job, the slot was also held. Slot visibility is necessary for diagnosing capacity issues.

**Independent Test**: Run `orchestrator slots list` against a database with known slot state and verify the output.

**Acceptance Scenarios**:

1. **Given** the project has 1 of 1 slots in use, **When** I run `orchestrator slots list`, **Then** I see project ID, active count, and slot limit.
2. **Given** multiple projects exist, **When** I run `orchestrator slots list`, **Then** I see all projects' slot state.

---

### User Story 4 - Reset Concurrency Slots (Priority: P2)

As an operator, I need to force-reset slot counts when they're stuck due to a crash or bug.

**Why this priority**: Complements job deletion. Slots can become permanently held if the orchestrator crashes between slot acquisition and job completion.

**Independent Test**: Set up a database with stuck slots, run `orchestrator slots reset`, verify all active counts are 0.

**Acceptance Scenarios**:

1. **Given** a project has 1 of 1 slots in use with no active jobs, **When** I run `orchestrator slots reset`, **Then** all active counts are set to 0.
2. **Given** slots are already at 0, **When** I run `orchestrator slots reset`, **Then** the command succeeds with a "nothing to reset" message.

---

### User Story 5 - Show Job Detail (Priority: P3)

As an operator, I need to inspect a single job's full details including timestamps and associated PR.

**Why this priority**: Useful for investigating a specific ticket but not as critical as listing and deletion.

**Independent Test**: Run `orchestrator jobs get DEV-226` against a database with a known job and verify all fields are displayed.

**Acceptance Scenarios**:

1. **Given** a job exists for DEV-226 with a PR, **When** I run `orchestrator jobs get DEV-226`, **Then** I see all fields: ticket ID, status, project ID, worktree path, session, PR number, created_at, updated_at.
2. **Given** no job exists, **When** I run `orchestrator jobs get DEV-999`, **Then** I get "job not found" with exit code 1.

---

### User Story 6 - Update Job Status (Priority: P3)

As an operator, I need to manually change a job's status to recover from edge cases (e.g., mark a stuck job as needs-attention, or re-queue a job).

**Why this priority**: Provides manual override capability for recovery scenarios that automated reconciliation doesn't handle.

**Independent Test**: Create a job, run `orchestrator jobs set-status DEV-226 needs-attention`, verify the status changed.

**Acceptance Scenarios**:

1. **Given** a queued job for DEV-226, **When** I run `orchestrator jobs set-status DEV-226 needs-attention`, **Then** the job status is updated and updated_at is refreshed.
2. **Given** an invalid status "banana", **When** I run `orchestrator jobs set-status DEV-226 banana`, **Then** I get a validation error listing valid statuses (queued, in-progress, needs-attention, complete, closed).

---

### Edge Cases

- What happens when the database file doesn't exist at the specified path? -> Clear error: "database not found at [path]"
- What happens when the database is locked by a running orchestrator? -> SQLite WAL mode allows concurrent readers, so reads work. Writes may fail with busy timeout — display a clear error suggesting the user stop the daemon first.
- What happens when deleting a job that has an active session running? -> The delete command removes the database record and releases the slot. It does NOT kill the session or clean up the worktree. The operator handles those separately (or they'll be reconciled on next startup).
- What happens when `--db-path` is not provided and `ORCH_DB_PATH` is unset? -> Required flag — error with usage hint.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `jobs list` subcommand that displays all jobs with ticket ID, status, project ID, worktree path, session, PR number, and updated_at.
- **FR-002**: System MUST support `--status` filter on `jobs list` using repeated flags (e.g., `--status queued --status in-progress`), implemented as a `StringSliceFlag`.
- **FR-003**: System MUST provide a `jobs get <ticket-id>` subcommand that displays full details of a single job.
- **FR-004**: System MUST provide a `jobs delete <ticket-id>` subcommand that removes a job record and releases its concurrency slot. The slot release uses the `project_id` stored on the job record.
- **FR-005**: System MUST provide a `jobs set-status <ticket-id> <status>` subcommand that updates a job's status, validating against known status constants: `queued`, `in-progress`, `needs-attention`, `complete`, `closed`. No transition rules — any valid status can be set from any other status (this is a manual recovery tool).
- **FR-006**: System MUST provide a `slots list` subcommand that displays all project slot states (project ID, active count, limit).
- **FR-007**: System MUST provide a `slots reset` subcommand that sets all active counts to 0. This is a global operation across all projects.
- **FR-008**: All subcommands MUST share a global `--db-path` flag (with `ORCH_DB_PATH` env fallback). This flag is required — error if missing.
- **FR-009**: All subcommands MUST exit with code 0 on success and code 1 on failure.
- **FR-010**: Write operations (`delete`, `set-status`, `reset`) MUST print a confirmation message. Format: `<verb> <entity> <id> [details]`. Examples: `Deleted job DEV-226 (was: queued, project: 707aafa8)`, `Reset 1 project slot(s) to 0`, `Updated DEV-226 status: queued -> needs-attention`.
- **FR-011**: The business logic for each operation MUST live in an `internal/admin` service package, callable without CLI argument parsing, so it can be reused by future HTTP endpoints.
- **FR-012**: The daemon MUST be started via `orchestrator run` subcommand. Bare `orchestrator` with no subcommand MUST display help text listing available commands.
- **FR-013**: A database migration MUST add a `project_id TEXT` column to the `jobs` table. Existing jobs get an empty string for `project_id`. New jobs created by the orchestrator daemon MUST populate `project_id` from the daemon's `--project-id` config.

### Key Entities

- **Job**: An autonomous development task. Key attributes: ticket ID (PK), project ID, status, worktree path, session ref, PR number (nullable), timestamps.
- **ConcurrencySlot**: Per-project capacity tracking. Key attributes: project ID (PK), active count, slot limit.

## Architecture *(mandatory)*

### Admin Service Layer

A new `internal/admin` package provides the business logic for all admin operations. This is the reuse boundary — CLI calls it now, HTTP endpoints call it later.

```
internal/admin/
  service.go       # Service struct + constructor
  service_test.go  # Integration tests against real SQLite
```

**Service struct:**

```go
type Service struct {
    store state.StateStore
}

func New(store state.StateStore) *Service
```

**Methods (all take `context.Context`, return typed results):**

| Method | Signature | Notes |
|--------|-----------|-------|
| `ListJobs` | `(ctx, filter JobFilter) ([]state.Job, error)` | `JobFilter.Statuses []string` — empty means all |
| `GetJob` | `(ctx, ticketID string) (*state.Job, error)` | Returns `state.ErrJobNotFound` if missing |
| `DeleteJob` | `(ctx, ticketID string) (*DeleteResult, error)` | Looks up job, deletes it, releases slot using job's `project_id`. Returns `DeleteResult{Job, SlotReleased bool}` |
| `SetJobStatus` | `(ctx, ticketID, status string) error` | Validates status against known constants before calling store |
| `ListSlots` | `(ctx) ([]state.ConcurrencySlot, error)` | Pass-through to store |
| `ResetSlots` | `(ctx) (int, error)` | Pass-through to store, returns count of reset projects |

### New StateStore Methods

The following methods must be added to the `StateStore` interface and SQLite implementation:

| Method | Signature | SQL |
|--------|-----------|-----|
| `ListAllJobs` | `(ctx) ([]Job, error)` | `SELECT * FROM jobs ORDER BY updated_at DESC` |
| `DeleteJob` | `(ctx, ticketID string) error` | `DELETE FROM jobs WHERE ticket_id = ?`, returns `ErrJobNotFound` if no rows affected |
| `ListSlots` | `(ctx) ([]ConcurrencySlot, error)` | `SELECT * FROM concurrency_slots` |

### New Types

```go
// state/store.go
type ConcurrencySlot struct {
    ProjectID   string
    ActiveCount int
    SlotLimit   int
}

// admin/service.go
type JobFilter struct {
    Statuses []string  // empty = all jobs
}

type DeleteResult struct {
    Job          state.Job
    SlotReleased bool  // false if project_id was empty (legacy job)
}
```

### CLI Subcommand Structure

```
orchestrator run                                    # start daemon (moved from root)
orchestrator jobs list [--status <s>...]            # list jobs
orchestrator jobs get <ticket-id>                   # single job detail
orchestrator jobs delete <ticket-id>                # remove job + release slot
orchestrator jobs set-status <ticket-id> <status>   # update status
orchestrator slots list                             # show slot state
orchestrator slots reset                            # reset all slots
```

**Flag scoping:**
- Global flags: `--db-path` (shared by all subcommands)
- `run` subcommand flags: all daemon flags (`--linear-api-key`, `--github-token`, `--poll-interval`, etc.)
- Admin subcommand flags: `--status` on `jobs list` only

### Output Format

**List views** — tab-aligned columns via `text/tabwriter`:

```
TICKET      STATUS           PROJECT     PR    UPDATED
DEV-226     queued           707aafa8    -     2026-03-30 10:15:00
DEV-227     in-progress      707aafa8    42    2026-03-30 09:30:00
```

**Detail views** — `Key: Value` pairs, one per line:

```
Ticket:     DEV-226
Status:     queued
Project:    707aafa8-0cfc-4e49-9ec2-112a0328dee6
Worktree:   /Users/morgan/.claude/worktrees/DEV-226
Session:    workspace:23
PR:         -
Created:    2026-03-29 21:16:44
Updated:    2026-03-29 21:16:44
```

**Timestamps**: RFC 3339 truncated to seconds, local timezone.

**Confirmation messages**: Single line, format specified in FR-010.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operator can diagnose the "stale job blocking pickup" scenario entirely through CLI commands without touching sqlite3.
- **SC-002**: Operator can recover from a stale job + stuck slot situation in under 30 seconds using `jobs delete`.
- **SC-003**: All subcommands produce consistent output following the formats defined in the Architecture section.
- **SC-004**: All admin operations are testable by calling the `admin.Service` methods directly, without invoking CLI parsing.

## Testing Requirements *(mandatory)*

### Test Strategy

Integration tests against a real SQLite database in the `internal/admin` package. Each test creates a temporary database, runs migrations, populates it with known state, calls the `admin.Service` method, and asserts the outcome.

The CLI layer (`cmd/orchestrator/`) is thin wiring — no dedicated tests. urfave/cli handles flag parsing.

The new `StateStore` methods (`ListAllJobs`, `DeleteJob`, `ListSlots`) get their own unit tests in `internal/state/`.

### FR to Test Mapping

| FR | Test Type | Package | Description |
|----|-----------|---------|-------------|
| FR-001 | Integration | admin | List jobs from a populated database, verify all fields present |
| FR-002 | Integration | admin | List with status filter, verify only matching jobs returned; empty filter returns all |
| FR-003 | Integration | admin | Get existing job returns full details; get non-existent returns ErrJobNotFound |
| FR-004 | Integration | admin | Delete job: verify removed + slot released; delete job with empty project_id: verify slot not released; delete non-existent: ErrJobNotFound |
| FR-005 | Integration | admin | Set valid status: verify updated_at refreshed; set invalid status: validation error |
| FR-006 | Integration | admin | List slots from populated database, verify fields |
| FR-007 | Integration | admin | Reset slots: verify all active counts are 0; reset when already 0: returns 0 |
| FR-010 | N/A | — | Tested implicitly via CLI output inspection during manual testing |
| FR-011 | Integration | admin | All tests call Service methods directly, proving reusability |
| FR-013 | Unit | state | Migration adds project_id column; CreateJob populates project_id |

### Store Method Tests

| Method | Package | Description |
|--------|---------|-------------|
| `ListAllJobs` | state | Returns all jobs ordered by updated_at DESC; empty DB returns empty slice |
| `DeleteJob` | state | Deletes existing job; returns ErrJobNotFound for missing job |
| `ListSlots` | state | Returns all slots; empty DB returns empty slice |

### Edge Case Coverage

- Delete non-existent job -> `state.ErrJobNotFound`
- Set-status with invalid status -> validation error listing valid values
- Delete job with empty `project_id` (legacy row) -> job deleted, slot release skipped, `SlotReleased: false`
- List jobs with no jobs in database -> empty slice, no error
- Database file missing -> clear error message at store open time
- Concurrent access (daemon running) -> reads succeed, writes may fail with SQLite busy timeout
