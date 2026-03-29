# Tasks: Watcher 1 — Ready Ticket Pickup

**Complexity**: Medium (7 files, ~350 LOC)
**Critical Path**: T001 → T003 → T005 → T007/T008/T009 → T010

## Dependency Graph

```
T001 (Config)
  ├── T002 (CLI flags) ─────────────────────┐
  └── T003 (Orchestrator struct)             │
        ├── T004 (Fix existing tests)        │
        ├── T005 (Watcher implementation)    │
        │     ├── T007 (Happy path tests) ──┤
        │     ├── T008 (Rollback tests) ────┤
        │     └── T009 (Edge case tests) ───┤
        └── T006 (Test doubles)             │
              ├── T007                       │
              ├── T008                       │
              └── T009                       │
                                             │
T010 (Wiring) ←── T002, T003, T005 ─────────┘
```

**Parallel groups:**
- After T001: T002 ‖ T003
- After T003: T004 ‖ T005 ‖ T006
- After T005+T006: T007 ‖ T008 ‖ T009
- After T002+T005: T010

## Tasks

- [ ] **T001** — Add `ProjectID` and `RepoDir` to Config with validation

  **File**: `internal/orchestrator/config.go`
  **Plan Step**: 1
  **FRs**: FR-002, FR-003

  Add two fields to the `Config` struct:
  ```go
  ProjectID string  // Linear project identifier for slot scoping
  RepoDir   string  // Git repo root directory (must contain .git)
  ```
  Add validation in `Validate()`:
  - `ProjectID` must not be empty
  - `RepoDir` must not be empty
  - `RepoDir` must exist and contain a `.git` directory

  Add corresponding tests in `internal/orchestrator/config_test.go`.

  **Acceptance**: `go test ./internal/orchestrator/ -run TestConfig` passes with new validation cases.

---

- [ ] **T002** [P:T003] — Add CLI flags for `ProjectID` and `RepoDir`

  **File**: `cmd/orchestrator/main.go`
  **Plan Step**: 1
  **Depends on**: T001

  Add two new CLI flags following the existing pattern:
  - `--project-id` / env `ORCH_PROJECT_ID` (string, required)
  - `--repo-dir` / env `ORCH_REPO_DIR` (string, required)

  Wire them into the `Config` struct construction (around line 84).

  **Acceptance**: `go build ./cmd/orchestrator` succeeds. `./orchestrator --help` shows the new flags.

---

- [ ] **T003** [P:T002] — Extend Orchestrator struct with dependencies

  **File**: `internal/orchestrator/orchestrator.go`
  **Plan Step**: 2
  **Depends on**: T001
  **FRs**: All (enables dependency injection for watcher)

  Add three new fields to `Orchestrator`:
  ```go
  linear    linear.Client
  worktrees worktree.Manager
  sessions  cmux.SessionManager
  ```

  Update `New()` constructor to accept the three new dependencies:
  ```go
  func New(cfg *Config, store state.StateStore, lc linear.Client, wt worktree.Manager, sm cmux.SessionManager) *Orchestrator
  ```

  **Acceptance**: Package compiles. Do NOT update tests yet (T004 handles that).

---

- [ ] **T004** — Update existing orchestrator tests for new constructor

  **File**: `internal/orchestrator/orchestrator_test.go`, `internal/orchestrator/watchers_test.go`
  **Plan Step**: 2
  **Depends on**: T003

  Update all existing `orchestrator.New(...)` calls in test files to pass the three new dependencies. Use `nil` for all three since existing tests don't exercise the watcher.

  **Acceptance**: `go test ./internal/orchestrator/` passes (all existing tests still green).

---

- [ ] **T005** — Implement `NewLinearPoller` watcher

  **File**: NEW `internal/orchestrator/watcher_linear.go`
  **Plan Step**: 3
  **Depends on**: T003
  **FRs**: FR-001 through FR-014

  Create the file with these functions:

  1. `(o *Orchestrator) NewLinearPoller() shutdown.ServiceFunc` — poll loop with `time.NewTicker`, select on `ctx.Done()` and ticker.
  2. `(o *Orchestrator) pollAndProcess(ctx context.Context)` — calls `PollReadyTickets`, iterates tickets, checks context between each.
  3. `(o *Orchestrator) processTicket(ctx context.Context, ticket linear.Ticket) error` — the pipeline:
     - GetJob check (skip if exists)
     - TryAcquireSlot (skip if false)
     - CreateWorktree
     - Write `.ai/ticket.md` (create `.ai/` dir)
     - SpawnSession with env map containing `WORK_CALLBACK_URL`
     - CreateJob
     - SetTicketStatus (log on error, continue)
     - RemoveLabel (log on error, continue)
     - Rollback on failure: reverse-order cleanup + apply "needs-attention" label
  4. `shortTitle(title string) string` — derive filesystem-safe name from ticket title.

  Use `ticket.Identifier` for worktree/session/job keys. Use `ticket.ID` for Linear API calls.
  Use `o.cfg.ProjectID` for `TryAcquireSlot`. Use `o.cfg.CallbackPort` for callback URL.

  Refer to technical-spec.md for the complete data flow and error handling table.

  **Acceptance**: Package compiles. Logic is correct per the pipeline order in the business spec.

---

- [ ] **T006** [P:T005] — Create test doubles for integration tests

  **File**: NEW `internal/orchestrator/watcher_linear_test.go` (top section)
  **Plan Step**: 4
  **Depends on**: T003

  Create method-recording mock implementations:

  1. `mockLinearClient` — implements `linear.Client`. Records calls. Configurable return values for each method (`pollTickets`, `pollErr`, `setStatusErr`, `removeLabelErr`, `applyLabelErr`).
  2. `mockWorktreeManager` — implements `worktree.Manager`. Records calls. Returns configurable paths (use temp dirs) and errors.
  3. `mockSessionManager` — implements `cmux.SessionManager`. Records calls. Returns configurable session refs and errors.
  4. `mockStateStore` — implements `state.StateStore`. Records calls. In-memory job storage. Configurable `createJobErr`, `acquireSlotResult`.

  Each mock should have a `calls []string` field recording method names in order, for verifying call sequences.

  **Acceptance**: All mocks implement their interfaces (`var _ linear.Client = &mockLinearClient{}`).

---

- [ ] **T007** [P:T008,T009] — Write happy path and concurrency tests

  **File**: `internal/orchestrator/watcher_linear_test.go`
  **Plan Step**: 4
  **Depends on**: T005, T006
  **FRs**: FR-001 through FR-009, FR-012

  Tests:
  - `TestLinearPoller_SingleTicket` — one ticket, full pipeline in correct order, verify call sequence
  - `TestLinearPoller_TicketFileWritten` — verify `.ai/ticket.md` content matches description
  - `TestLinearPoller_CallbackURL` — verify env map has `http://localhost:{port}/{identifier}`
  - `TestLinearPoller_ConcurrencyLimit` — limit=1, 2 tickets, only 1 processed
  - `TestLinearPoller_ConcurrencyMultiPoll` — limit=1, 2 tickets: first poll gets 1, release slot, second poll gets the other
  - `TestLinearPoller_SkipExistingJob` — GetJob finds existing job, ticket skipped

  **Acceptance**: All tests pass.

---

- [ ] **T008** [P:T007,T009] — Write rollback and failure tests

  **File**: `internal/orchestrator/watcher_linear_test.go`
  **Plan Step**: 4
  **Depends on**: T005, T006
  **FRs**: FR-010, FR-013, FR-014

  Tests:
  - `TestLinearPoller_RollbackOnSpawnFailure` — SpawnSession error → DeleteWorktree + ReleaseSlot, no SetTicketStatus
  - `TestLinearPoller_RollbackOnCreateJobFailure` — CreateJob error → KillSession + DeleteWorktree + ReleaseSlot
  - `TestLinearPoller_RollbackOnWorktreeFailure` — CreateWorktree error → ReleaseSlot only (no DeleteWorktree)
  - `TestLinearPoller_NeedsAttentionOnFailure` — pipeline failure → "needs-attention" applied, "ai-ready" removed
  - `TestLinearPoller_LinearFailureAfterJob` — SetTicketStatus error → logged, job continues
  - `TestLinearPoller_RemoveLabelFailureAfterJob` — RemoveLabel error → logged, job continues

  **Acceptance**: All tests pass.

---

- [ ] **T009** [P:T007,T008] — Write edge case and shutdown tests

  **File**: `internal/orchestrator/watcher_linear_test.go`
  **Plan Step**: 4
  **Depends on**: T005, T006
  **FRs**: FR-011, edge cases

  Tests:
  - `TestLinearPoller_GracefulShutdown` — cancel context mid-poll, current ticket finishes, no new work
  - `TestLinearPoller_ShutdownBetweenPolls` — cancel context between ticks, watcher exits cleanly
  - `TestLinearPoller_EmptyPoll` — no tickets, no downstream calls
  - `TestLinearPoller_PollError` — PollReadyTickets error → logged, no crash
  - `TestLinearPoller_EmptyDescription` — empty description → `.ai/ticket.md` created with empty content

  **Acceptance**: All tests pass.

---

- [ ] **T010** — Wire real clients in main.go and replace stub in Run()

  **Files**: `cmd/orchestrator/main.go`, `internal/orchestrator/orchestrator.go`
  **Plan Step**: 5
  **Depends on**: T002, T003, T005

  In `cmd/orchestrator/main.go`, after store creation:
  ```go
  linearClient, err := linear.NewClient(cfg.LinearAPIKey)
  worktreeMgr, err := worktree.New(cfg.RepoDir, worktree.WithWorktreesRoot(cfg.WorktreesRoot))
  sessionMgr, err := cmux.New()
  ```
  Pass all three to `orchestrator.New(cfg, store, linearClient, worktreeMgr, sessionMgr)`.

  In `internal/orchestrator/orchestrator.go` `Run()`, replace:
  ```go
  linearPoller := NewWatcherStub(WatcherLinearPoller, o.cfg.PollInterval)
  ```
  with:
  ```go
  linearPoller := o.NewLinearPoller()
  ```
  Keep PR watcher and comment watcher as stubs.

  **Acceptance**: `go build ./cmd/orchestrator` succeeds.

## Validation

### FR → Task Mapping

| FR | Tasks |
|----|-------|
| FR-001 | T005 (poll loop), T007 (SingleTicket test) |
| FR-002 | T001 (ProjectID config), T005 (TryAcquireSlot), T007 (ConcurrencyLimit test) |
| FR-003 | T001 (RepoDir config), T005 (CreateWorktree), T010 (worktree.New wiring) |
| FR-004 | T005 (file write), T007 (TicketFileWritten test) |
| FR-005 | T005 (env map), T007 (CallbackURL test) |
| FR-006 | T005 (SpawnSession), T007 (SingleTicket test) |
| FR-007 | T005 (CreateJob), T007 (SingleTicket test) |
| FR-008 | T005 (SetTicketStatus), T007 (SingleTicket test) |
| FR-009 | T005 (RemoveLabel), T007 (SingleTicket test) |
| FR-010 | T005 (rollback), T008 (rollback tests) |
| FR-011 | T005 (context cancel), T009 (shutdown tests) |
| FR-012 | T005 (GetJob check), T007 (SkipExistingJob test) |
| FR-013 | T005 (needs-attention), T008 (NeedsAttention test) |
| FR-014 | T005 (log-and-continue), T008 (LinearFailure tests) |

All 14 FRs covered. No orphan tasks.

## Summary

- **Total tasks**: 10
- **Parallel opportunities**: 3 groups (T002‖T003, T004‖T005‖T006, T007‖T008‖T009)
- **Critical path**: 5 steps (T001 → T003 → T005 → T007 → T010)
- **Execution order**: T001, T002‖T003, T004‖T005‖T006, T007‖T008‖T009, T010
