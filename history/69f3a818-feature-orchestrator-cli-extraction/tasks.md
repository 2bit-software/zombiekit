# Tasks: `brains workspace` CLI

**Plan**: `implementation-plan.md`
**Spec**: `spec.md`

## Complexity

- **Files affected**: ~14 (3 new CLI command files, 4 new workspace package files, 5 new test files, 2 orchestrator file edits)
- **Lines of change**: ~1,200 (mostly net new; ~200 deletions in orchestrator)
- **Cross-module deps**: 4 (`internal/cli` → `internal/workspace` → `internal/{worktree,sandbox,cmux}`; `internal/orchestrator` → `internal/workspace`)
- **Classification**: **Medium**

## Task List

### Phase 1 — `brains worktree`

- [ ] T001 Add `internal/cli/config.go` with `loadProjectConfig(c *cli.Context) (*orchestrator.ProjectConfig, error)`. Wraps `orchestrator.LoadOrchestratorConfig`; resolves `--config` (default `./orchestrator.toml`); resolves `--project` flag with single-project default and cwd-match disambiguation; errors with all project IDs listed if ambiguous.
- [ ] T002 [P] [US1] Add `internal/cli/worktree.go` with `newWorktreeCommand()` and subcommands `create`, `delete`, `push`, `clean-branch`, `list`. Each constructs `worktree.GitManager` via `worktree.New(repoDir, WithWorktreesRoot, WithCopyFiles)` from `loadProjectConfig`. Surface typed errors from `internal/worktree.Error`.
- [ ] T003 Register `newWorktreeCommand()` in `internal/cli/root.go` Commands slice.
- [ ] T004 [P] [US1] Add `internal/cli/worktree_test.go` integration tests: real `git init` temp repo, exercise each subcommand via `app.Run([]string{...})`, assert worktree paths/branches.

### Phase 2 — `brains sandbox`

- [ ] T005 [P] [US2] Add `internal/cli/sandbox.go` with `newSandboxCommand()` and subcommands `create`, `cleanup`, `available`, `name`, `list`. Use `sandbox.DefaultConfig` overridable by `--mounts` (repeatable), `--memory`, `--template`. `list` wraps `sbx ls --quiet` filtered to `zk-*` prefix.
- [ ] T006 Register `newSandboxCommand()` in `internal/cli/root.go` Commands slice.
- [ ] T007 [P] [US2] Add `internal/cli/sandbox_test.go`: unit-test `name` subcommand and `--mounts` flag parsing helper. Skip Docker-dependent integration tests.

### Phase 0 — `internal/workspace` package

- [ ] T008 [P] Create `internal/workspace/errors.go` with typed errors: `ErrNoMarker`, `ErrPrepFailed`, plus `Error` struct with `Kind` and helpers (mirrors `internal/worktree/errors.go` style).
- [ ] T009 [P] Create `internal/workspace/marker.go` with `Marker` struct (TicketID, Title, Branch, WorktreePath, SandboxName, Spawned, Prompt, CreatedAt), `MarkerPath(worktreePath) string`, `ReadMarker(worktreePath) (Marker, error)`, `writeMarker(...)` (private). Uses `json.Marshal` + `os.WriteFile`.
- [ ] T010 Create `internal/workspace/workspace.go` with `Sandbox` and `Spawner` interfaces, `Manager` struct, `NewManager`, `WithSpawner`, `WithSandbox`, `WithLogger` options, `Prep(ctx, PrepInput) (PrepResult, error)`, `Teardown(ctx, ticketID) error`, `ShortTitle(title) string`. Implements rollback chain per technical-spec.md `Prep` sequence.
- [ ] T011 [P] Create `internal/workspace/doc.go` with package overview comment.
- [ ] T012 Add `internal/workspace/workspace_test.go`: real `worktree.GitManager` against `t.TempDir()` repo; stub `Sandbox` and `Spawner` recorders. Cover Prep happy path, no-sandbox, no-spawn, sandbox-fails-rollback, spawn-fails-rollback, ticket-md-write-fails-rollback; Teardown happy path, no-marker, sandbox-missing, session-missing; marker round-trip.

### Refactor orchestrator to use `internal/workspace`

- [ ] T013 Refactor `internal/orchestrator/runner.go`: `NewProjectRunner` constructs `workspace.Manager` via `workspace.NewManager(p.worktrees, p.sandboxConfig, WithSpawner(p.sessions), WithLogger(p.logger))`. Store on `ProjectRunner.workspace`.
- [ ] T014 Refactor `internal/orchestrator/watcher_linear.go`: replace `runTicketPipeline` body with `p.workspace.Prep(...)` call; delete `setupWorktree` and `shortTitle` helpers; remove inline rollback block (now inside `workspace.Prep`).
- [ ] T015 Update `internal/orchestrator/watcher_linear_test.go`: keep existing `stubWorktree`; pass real `workspace.Manager` over the stub. Add `stubSpawner` and `stubSandbox` if needed for the new code path. Confirm all existing assertions still pass.

### Phase 3 — `brains workspace`

- [ ] T016 Add `internal/cli/workspace.go` with `newWorkspaceCommand()` and subcommands:
  - `prep <ticket-id>` flags `--title`, `--description`/`--description-file`, `--no-sandbox`, `--spawn`, `--prompt`, `--callback-url`, `--format text|json`. Builds `workspace.Manager` (with `cmux.New(WithCommandBuilder(sandbox.NewCommandBuilder(cfg)))` only when `--spawn` set), calls `Prep`. Default prompt matches orchestrator's "Read .ai/ticket.md ..." string.
  - `teardown <ticket-id>` flags `--force`. Calls `workspace.Manager.Teardown`.
  - `gc` flags `--dry-run` (default true). Walks worktrees root + `sbx ls --quiet`; reports/removes orphans without markers and stale `zk-*` sandboxes.
- [ ] T017 Register `newWorkspaceCommand()` in `internal/cli/root.go` Commands slice.
- [ ] T018 [P] Add `internal/cli/workspace_test.go`: stub `workspace.Manager` (introduce small interface in `internal/cli` if needed), verify flag parsing, `--format json` output shape, exit codes.

### Phase 4 — Cleanup + verification

- [ ] T019 Delete `cmd/sandbox-test/` directory (parity reached via `brains sandbox` + `brains worktree`).
- [ ] T020 Update `INFRASTRUCTURE.md` with the new `brains worktree`, `brains sandbox`, `brains workspace` commands. Run `task test` (or equivalent) to verify the full suite is green; fix any regressions.

## Dependency Graph

```
T001 ────────┬─→ T002 ─→ T003
             │     ↓
             │   T004
             ├─→ T005 ─→ T006
             │     ↓
             │   T007
             └────────────────┐
                              │
T008 ┐                        │
T009 ┼─→ T010 ─→ T012         │
T011 ┘     │                  │
           ├─→ T013 ─→ T014 ─→ T015
           │
           └─→ T016 ─→ T017 ─→ T018
                              ↓
                            T019 ─→ T020
```

## Parallel Opportunities

| Wave | Tasks | Notes |
|------|-------|-------|
| 1 | T001, T008, T009, T011 | All independent foundation pieces |
| 2 | T002, T005, T010 | T002+T005 require T001; T010 requires T008+T009 |
| 3 | T003, T004, T006, T007, T012, T016 | Each unblocked by its phase 2 predecessor |
| 4 | T013, T017, T018 | T013 requires T012 green; T017+T018 require T016 |
| 5 | T014 | Sequential after T013 (same file area) |
| 6 | T015, T019 | T015 after T014; T019 once 1-3 pass |
| 7 | T020 | Final |

Realistic execution: most tasks land sequentially due to small repo + shared `internal/cli/root.go`. Treat the parallelism markers as "safe to parallelize" rather than mandatory.

## Critical Path

T001 → T010 → T013 → T014 → T015 → T016 → T020. Estimated ~3-4 days end-to-end with focused work.

## Spec Traceability

| FR | Task(s) |
|----|---------|
| FR-001 (worktree create) | T002 |
| FR-002 (worktree delete) | T002 |
| FR-003 (worktree push) | T002 |
| FR-004 (clean-branch) | T002 |
| FR-005 (worktree list) | T002 |
| FR-006 (sandbox create) | T005 |
| FR-007 (sandbox cleanup) | T005 |
| FR-008 (sandbox available) | T005 |
| FR-009 (sandbox name) | T005 |
| FR-010 (workspace prep) | T010, T016 |
| FR-011 (workspace teardown) | T010, T016 |
| FR-012 (workspace gc) | T016 |
| FR-013 (`--config` honored everywhere) | T001 |
| FR-014 (typed error surfacing) | T002, T008, T010 |
| SC-001 (`cmd/sandbox-test` deletable) | T019 |
| SC-002 (operator recovery) | T010, T016 |
| SC-003 (integration tests) | T004, T012, T015 |
| SC-004 (no duplication) | T013, T014 |
