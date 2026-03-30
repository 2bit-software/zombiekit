# Tasks: Orchestrator Admin CLI

**Complexity**: Medium (~12 files, 3 modules)
**Total tasks**: 14
**Critical path**: T001 → T002 → T004 → T007 → T010

## Dependency Graph

```
T001 (migration)
  ↓
T002 (Job.ProjectID + scan helper) ←── T003 [P] (ConcurrencySlot type + ListSlots)
  ↓
T004 (ListAllJobs + DeleteJob store methods)
  ↓                                    T005 [P] (ValidStatuses + store tests)
T006 (CreateJob signature change)
  ↓
T007 (admin service: ListJobs, GetJob, DeleteJob)
  ↓                                    T008 [P] (admin service: SetJobStatus, ListSlots, ResetSlots)
T009 (admin service tests)
  ↓
T010 (CLI: restructure main.go into subcommands)
  ↓
T011 (CLI: jobs subcommands + output formatting)
  ↓                                    T012 [P] (CLI: slots subcommands + output formatting)
T013 (CLI: openStore helper with exist-check)
  ↓
T014 (build + full test pass)
```

## Tasks

### Layer 1: Schema

- [ ] T001 [FR-013] Create migration `internal/state/migrations/002_add_project_id.sql` with `ALTER TABLE jobs ADD COLUMN project_id TEXT NOT NULL DEFAULT ''`

### Layer 2: Store types and methods

- [ ] T002 [FR-013] Add `ProjectID string` field to `Job` struct in `internal/state/store.go`. Extract a `scanJob(*sql.Rows) (Job, error)` helper that scans all columns including `project_id`. Update `GetJob`, `ListJobsByStatus`, `GetJobByPR` to use the helper. Update all SELECT queries to include `project_id` in the column list.

- [ ] T003 [P] [FR-006] Add `ConcurrencySlot` struct to `internal/state/store.go`. Add `ListSlots(ctx) ([]ConcurrencySlot, error)` to `StateStore` interface and SQLite implementation. SQL: `SELECT project_id, active_count, slot_limit FROM concurrency_slots`. Add no-op stubs to all test doubles implementing `StateStore` (`watcher_linear_test.go:stubState`, `watcher_pr_test.go:prStubState`, `orchestrator_test.go`).

- [ ] T004 [FR-001, FR-004] Add `ListAllJobs(ctx) ([]Job, error)` and `DeleteJob(ctx, ticketID string) error` to `StateStore` interface and SQLite implementation in `internal/state/store.go`. `ListAllJobs` uses `scanJob` helper, orders by `updated_at DESC`, returns `[]Job{}` for empty. `DeleteJob` does `DELETE FROM jobs WHERE ticket_id = ?`, returns `ErrJobNotFound` if `RowsAffected() == 0`. Add no-op stubs to all test doubles.

- [ ] T005 [P] [FR-005] Add `ValidStatuses` slice to `internal/state/store.go`: `var ValidStatuses = []string{StatusQueued, StatusInProgress, StatusNeedsAttention, StatusComplete, StatusClosed}`. Add store-level tests in `internal/state/store_test.go` for `ListAllJobs` (empty + populated + order), `DeleteJob` (exists + not-found), `ListSlots` (empty + populated).

### Layer 3: CreateJob signature change

- [ ] T006 [FR-013] Change `CreateJob` signature to `CreateJob(ctx, ticketID, worktreePath, cmuxSession, projectID string) error` in `internal/state/store.go` (interface + implementation). Update INSERT SQL to include `project_id`. Update call site in `internal/orchestrator/watcher_linear.go:144` to pass `o.cfg.ProjectID`. Update all `CreateJob` stubs in test files to accept the 4th parameter. Verify compile.

### Layer 4: Admin service

- [ ] T007 [FR-001, FR-003, FR-004, FR-011] Create `internal/admin/service.go` with `Service` struct, `New(store)` constructor, `JobFilter` type, `DeleteResult` type. Implement `ListJobs(ctx, filter)`, `GetJob(ctx, ticketID)` (wraps nil → `ErrJobNotFound`), and `DeleteJob(ctx, ticketID)` (get job → delete → release slot if `ProjectID != ""`).

- [ ] T008 [P] [FR-005, FR-006, FR-007, FR-011] In `internal/admin/service.go`, implement `SetJobStatus(ctx, ticketID, status)` with validation against `state.ValidStatuses`, `ListSlots(ctx)` pass-through, and `ResetSlots(ctx)` pass-through.

- [ ] T009 [FR-011] Create `internal/admin/service_test.go` with integration tests against real SQLite: `TestListJobs_All`, `TestListJobs_FilterByStatus`, `TestListJobs_FilterMultipleStatuses`, `TestListJobs_Empty`, `TestGetJob_Exists`, `TestGetJob_NotFound`, `TestDeleteJob_Success`, `TestDeleteJob_ReleasesSlot`, `TestDeleteJob_NoSlotRelease_EmptyProjectID`, `TestDeleteJob_NotFound`, `TestSetJobStatus_Valid`, `TestSetJobStatus_Invalid`, `TestSetJobStatus_NotFound`, `TestListSlots`, `TestResetSlots`, `TestResetSlots_AlreadyZero`.

### Layer 5: CLI refactor

- [ ] T010 [FR-012] Restructure `cmd/orchestrator/main.go`: move daemon logic into `runCommand()` subcommand with all 17 daemon-specific flags. Keep `--db-path` as global flag (NOT `Required`). Bare `orchestrator` shows help. Verify daemon still works via `orchestrator run`.

- [ ] T011 [FR-001, FR-002, FR-003, FR-004, FR-005, FR-009, FR-010] Create `cmd/orchestrator/admin.go` with `jobsCommand()` returning nested subcommands: `list` (with `--status` `StringSliceFlag`), `get`, `delete`, `set-status`. Implement handlers: parse args, call `admin.Service`, format output with `text/tabwriter` for lists and `Key: Value` for detail. Use `cli.Exit(msg, 1)` for errors. Confirmation messages per FR-010 format.

- [ ] T012 [P] [FR-006, FR-007, FR-009, FR-010] In `cmd/orchestrator/admin.go`, add `slotsCommand()` with nested `list` and `reset` subcommands. Implement handlers with tabwriter output for list and confirmation message for reset (including "No slots to reset" for zero count).

- [ ] T013 [FR-008] Implement `openStore(c *cli.Context, mustExist bool) (*state.SQLiteStore, error)` helper in `cmd/orchestrator/admin.go`. Returns error if `--db-path` is empty. When `mustExist=true`, checks `os.Stat` before opening. Calls `Migrate` after open. Admin handlers call `openStore(c, true)`, `runCommand` calls `openStore(c, false)`.

### Layer 6: Validation

- [ ] T014 Run `go build ./...` and `go test ./internal/state/... ./internal/admin/... ./internal/orchestrator/...`. Verify `orchestrator --help` shows subcommands. Verify `orchestrator run --help` shows daemon flags. Verify `orchestrator jobs list --help` works. Smoke test: `orchestrator jobs list --db-path .data/orchestrator.db`.

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T002, T004, T007, T011 |
| FR-002 | T011 |
| FR-003 | T007, T011 |
| FR-004 | T004, T007, T011 |
| FR-005 | T005, T008, T011 |
| FR-006 | T003, T008, T012 |
| FR-007 | T008, T012 |
| FR-008 | T013 |
| FR-009 | T011, T012 |
| FR-010 | T011, T012 |
| FR-011 | T007, T008, T009 |
| FR-012 | T010 |
| FR-013 | T001, T002, T006 |

## Parallel Opportunities

| Group | Parallel tasks | After |
|-------|---------------|-------|
| Store types | T003 ‖ T002 | T001 |
| Store methods | T005 ‖ T004 | T002 |
| Admin service | T008 ‖ T007 | T006 |
| CLI commands | T012 ‖ T011 | T010 |
