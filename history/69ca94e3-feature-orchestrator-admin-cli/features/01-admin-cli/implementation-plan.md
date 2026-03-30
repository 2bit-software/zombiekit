# Implementation Plan: Orchestrator Admin CLI

## Overview

Add admin subcommands to the orchestrator binary for inspecting and managing jobs and concurrency slots. Introduces a reusable `internal/admin` service layer and migrates the daemon to a `run` subcommand.

## Dependency Graph

```
Step 1: Migration (project_id column)
  ↓
Step 2: New StateStore methods (DeleteJob, ListAllJobs, ListSlots, ConcurrencySlot type)
  ↓
Step 3: Update CreateJob to accept project_id
  ↓
Step 4: Admin service layer (internal/admin)
  ↓
Step 5: CLI subcommand refactor (main.go → subcommands)
  ↓
Step 6: Taskfile entry for orchestrator build
```

Steps 1-3 are schema + store changes. Step 4 is the service layer. Step 5 is CLI wiring. Step 6 is cleanup.

## Step 1: Add `project_id` column to jobs table

**FR**: FR-013
**Files**:
- Create `internal/state/migrations/002_add_project_id.sql`

**SQL**:
```sql
ALTER TABLE jobs ADD COLUMN project_id TEXT NOT NULL DEFAULT '';
```

SQLite `ALTER TABLE ADD COLUMN` supports defaults. Existing rows get empty string. No data migration needed — legacy jobs just have empty `project_id`, which the admin service handles gracefully (skips slot release).

**Tests**: Migration runner already tests idempotent application. The new migration is covered by the `Migrate()` call in existing store tests. Add one test verifying the column exists after migration.

---

## Step 2: New StateStore methods and types

**FR**: FR-001, FR-004, FR-006
**Files**:
- `internal/state/store.go` — add interface methods + `ConcurrencySlot` type + SQLite implementations
- `internal/state/store_test.go` — add tests for new methods

**Interface additions**:

```go
type ConcurrencySlot struct {
    ProjectID   string
    ActiveCount int
    SlotLimit   int
}

// Add to StateStore interface:
ListAllJobs(ctx context.Context) ([]Job, error)
DeleteJob(ctx context.Context, ticketID string) error
ListSlots(ctx context.Context) ([]ConcurrencySlot, error)
```

**SQL**:
- `ListAllJobs`: `SELECT ticket_id, worktree_path, cmux_session, pr_number, status, project_id, created_at, updated_at FROM jobs ORDER BY updated_at DESC`
- `DeleteJob`: `DELETE FROM jobs WHERE ticket_id = ?` — returns `ErrJobNotFound` if `RowsAffected() == 0`
- `ListSlots`: `SELECT project_id, active_count, slot_limit FROM concurrency_slots`

**Job struct update**: Add `ProjectID string` field. Update the `scanJob` helper (if one exists) or update all SELECT queries to include `project_id`.

**Mock/stub updates**: All test doubles implementing `StateStore` need the three new methods. Files affected:
- `internal/linear/mock.go` — no, this is Linear mock
- `internal/orchestrator/watcher_linear_test.go` — `stubState`
- `internal/orchestrator/watcher_pr_test.go` — `prStubState`
- `internal/orchestrator/orchestrator_test.go` — if it has its own stub
- Any other test files with `StateStore` stubs

**Tests**:
- `ListAllJobs`: empty DB returns `[]Job{}`; populated DB returns all jobs ordered by `updated_at DESC`
- `DeleteJob`: delete existing job, verify gone; delete non-existent returns `ErrJobNotFound`
- `ListSlots`: empty DB returns `[]ConcurrencySlot{}`; populated DB returns all slots

---

## Step 3: Update CreateJob to accept project_id

**FR**: FR-013
**Files**:
- `internal/state/store.go` — change `CreateJob` signature and SQL
- `internal/orchestrator/watcher_linear.go` — pass `o.cfg.ProjectID`
- All test stubs that implement `CreateJob`

**Interface change**:
```go
// Before:
CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
// After:
CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession, projectID string) error
```

**SQL update**:
```sql
INSERT INTO jobs (ticket_id, worktree_path, cmux_session, project_id, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
```

**Call site update** (`watcher_linear.go:144`):
```go
// Before:
o.store.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef)
// After:
o.store.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef, o.cfg.ProjectID)
```

**Tests**: Update all `CreateJob` stubs to accept 4th parameter. Update test assertions that check `CreateJob` call args.

---

## Step 4: Admin service layer

**FR**: FR-001 through FR-007, FR-011
**Files**:
- Create `internal/admin/service.go`
- Create `internal/admin/service_test.go`

**Service struct**:
```go
type Service struct {
    store state.StateStore
}

func New(store state.StateStore) *Service { return &Service{store: store} }
```

**Methods**:

| Method | Logic |
|--------|-------|
| `ListJobs(ctx, filter)` | If filter.Statuses empty → `store.ListAllJobs(ctx)`. Otherwise → `store.ListJobsByStatus(ctx, filter.Statuses...)` |
| `GetJob(ctx, ticketID)` | `store.GetJob(ctx, ticketID)`. **NOTE: `store.GetJob` returns `(nil, nil)` for missing jobs, NOT `ErrJobNotFound`.** The admin service must check for nil and return `state.ErrJobNotFound`. |
| `DeleteJob(ctx, ticketID)` | Get job (for project_id and confirmation output). Delete job. If job.ProjectID != "" → `store.ReleaseSlot(ctx, job.ProjectID)`. Return `DeleteResult{Job, SlotReleased}`. |
| `SetJobStatus(ctx, ticketID, status)` | Validate status against `state.ValidStatuses` (new constant slice). If invalid → return validation error listing valid values. Otherwise → `store.SetJobStatus(ctx, ticketID, status)`. |
| `ListSlots(ctx)` | Pass-through to `store.ListSlots(ctx)` |
| `ResetSlots(ctx)` | Pass-through to `store.ResetAllSlots(ctx)` |

**Types**:
```go
type JobFilter struct {
    Statuses []string
}

type DeleteResult struct {
    Job          state.Job
    SlotReleased bool
}
```

**Validation**: Add `ValidStatuses` to `state` package:
```go
var ValidStatuses = []string{StatusQueued, StatusInProgress, StatusNeedsAttention, StatusComplete, StatusClosed}
```

**Tests** (integration, real SQLite):
- `TestListJobs_All` — populate 3 jobs, list all, verify count and order
- `TestListJobs_FilterByStatus` — populate mixed statuses, filter by one, verify only matching returned
- `TestListJobs_FilterMultipleStatuses` — filter by two statuses
- `TestListJobs_Empty` — empty DB returns empty slice
- `TestGetJob_Exists` — verify all fields
- `TestGetJob_NotFound` — returns ErrJobNotFound
- `TestDeleteJob_Success` — delete job, verify gone
- `TestDeleteJob_ReleasesSlot` — create job with project_id, acquire slot, delete job, verify slot released
- `TestDeleteJob_NoSlotRelease_EmptyProjectID` — legacy job with empty project_id, verify slot not released
- `TestDeleteJob_NotFound` — returns ErrJobNotFound
- `TestSetJobStatus_Valid` — set to each valid status, verify updated
- `TestSetJobStatus_Invalid` — "banana" returns validation error
- `TestSetJobStatus_NotFound` — returns ErrJobNotFound
- `TestListSlots` — populate slots, verify fields
- `TestResetSlots` — populate stuck slots, reset, verify all 0
- `TestResetSlots_AlreadyZero` — all slots at 0, verify returns 0 (CLI prints "No slots to reset")

---

## Step 5: CLI subcommand refactor

**FR**: FR-008, FR-009, FR-010, FR-012
**Files**:
- `cmd/orchestrator/main.go` — restructure into subcommands
- Create `cmd/orchestrator/admin.go` — admin subcommand handlers

**App restructure**:

```go
app := &cli.App{
    Name: "orchestrator",
    Flags: []cli.Flag{
        // GLOBAL: --db-path is NOT Required here to avoid blocking --help/--version.
        // Validated in openStore() helper instead.
        &cli.StringFlag{Name: "db-path", EnvVars: []string{"ORCH_DB_PATH"}},
    },
    Commands: []*cli.Command{
        runCommand(),    // daemon — gets all 17 remaining flags
        jobsCommand(),   // jobs list/get/delete/set-status
        slotsCommand(),  // slots list/reset
    },
}
```

**runCommand()**: Move existing `run()` function body here. All 17 daemon-specific flags move to this subcommand's `Flags` field.

**jobsCommand()**: Nested subcommands:
```go
func jobsCommand() *cli.Command {
    return &cli.Command{
        Name: "jobs",
        Subcommands: []*cli.Command{
            {Name: "list", Flags: []cli.Flag{statusSliceFlag}, Action: jobsList},
            {Name: "get", Action: jobsGet},       // arg: <ticket-id>
            {Name: "delete", Action: jobsDelete},  // arg: <ticket-id>
            {Name: "set-status", Action: jobsSetStatus}, // args: <ticket-id> <status>
        },
    }
}
```

**slotsCommand()**: Nested subcommands:
```go
func slotsCommand() *cli.Command {
    return &cli.Command{
        Name: "slots",
        Subcommands: []*cli.Command{
            {Name: "list", Action: slotsList},
            {Name: "reset", Action: slotsReset},
        },
    }
}
```

**Handler pattern** (each handler in `admin.go`):
```go
func jobsList(c *cli.Context) error {
    store := openStore(c)  // shared helper: reads --db-path, opens SQLite, runs migrations
    defer store.Close()
    svc := admin.New(store)

    filter := admin.JobFilter{Statuses: c.StringSlice("status")}
    jobs, err := svc.ListJobs(c.Context, filter)
    if err != nil { return cli.Exit(err.Error(), 1) }

    // format with tabwriter
    return nil
}
```

**`openStore` helper**: Shared function that reads `--db-path` from global flags. Returns clear error if flag is empty ("--db-path is required"). For admin subcommands, checks that the database file exists before opening (returns "database not found at [path]" if missing — admin commands should never silently create an empty database). For the `run` subcommand, allows create-on-open (existing behavior). Both paths call `Migrate` after opening. Two variants: `openStoreReadOnly` (admin) and `openStoreOrCreate` (daemon), or a single `openStore(c, mustExist bool)` helper.

**Output formatting**: Use `text/tabwriter` for lists, `fmt.Fprintf` for detail views. Timestamps formatted as RFC 3339 to seconds in local timezone.

**Exit codes**: `cli.Exit(msg, 1)` for errors. Normal return for success (exit 0).

---

## Step 6: Taskfile update

**Files**:
- `Taskfile.dev.yml` — update `build:orchestrator` if the binary name or build args change

The existing `build:orchestrator` task runs `go build -ldflags "{{.LDFLAGS}}" -o bin/orchestrator ./cmd/orchestrator`. This shouldn't need changes unless the build target moves.

---

## Implementation Order (for PR)

All steps go in a single PR since they're tightly coupled. But implementation order matters for incremental compilation checks:

1. Migration file (step 1)
2. `ConcurrencySlot` type + `Job.ProjectID` field + store methods + store tests (step 2)
3. `CreateJob` signature change + caller updates + stub updates (step 3)
4. Admin service + tests (step 4)
5. CLI refactor (step 5)
6. Taskfile cleanup (step 6)
7. Build + run existing tests to verify no regressions
