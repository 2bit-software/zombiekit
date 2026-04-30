# Initiative: orchestrator-cli-extraction

**Type**: feature
**Status**: completed
**Created**: 2026-04-30
**Completed**: 2026-04-30
**ID**: 69f3a818-feature-orchestrator-cli-extraction

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | completed | 2026-04-30 12:35 |
| plan | plan | completed | 2026-04-30 12:50 |
| tasks | tasks | completed | 2026-04-30 13:00 |
| implement | implement | completed | 2026-04-30 13:50 |

## Description

Expose the orchestrator's worktree + sandbox lifecycle primitives as `brains workspace` subcommands so operators can drive the same flow manually, without running the daemon. Cmux session spawning is opt-in via a `--spawn` flag. Out of scope: Linear polling, PR/comment watchers, callback server, state store.

## Goals

- `brains worktree {create, delete, push, clean-branch, list}` — thin passthrough to `internal/worktree.Manager`
- `brains sandbox {create, cleanup, available, name, list}` — thin passthrough to `internal/sandbox`
- `brains workspace {prep, teardown, gc}` — composition equivalent to orchestrator's pickup + rollback
- Lift `setupWorktree` + rollback block out of `internal/orchestrator/watcher_linear.go` into a shared `internal/workspace` package; keep daemon and CLI calling the same code
- Delete `cmd/sandbox-test` once parity is reached

## Progress

- 2026-04-30 — spec drafted, lifecycle audit complete, refactor effort estimated (~4–6 days full scope, ~1 day P1-only)
- 2026-04-30 — user decisions: keep cmux spawn (as `--spawn` flag), use `brains workspace` naming. Spec advanced to plan.
- 2026-04-30 — implementation complete (4 commits). All 20 tasks shipped.

## Completion

**Completed**: 2026-04-30
**Duration**: same-day (~6 hours)

### Outcomes

- **Phase 1 — `brains worktree`** (commit `27edcc9`): create/delete/push/clean-branch/list subcommands wrapping `internal/worktree.Manager`. Auto-detects current git repo when no `--config` given.
- **Phase 2 — `brains sandbox`** (commit `ebdc70e`): create/cleanup/available/name/list subcommands wrapping `internal/sandbox`. Idempotent cleanup; deterministic `zk-{lowercase-ticket-id}` naming.
- **Phase 0 — `internal/workspace` package** (commit `8659259`): lifted `setupWorktree`, `shortTitle`, and rollback block out of `internal/orchestrator/watcher_linear.go`. Daemon now delegates to `workspace.Manager.Prep` / `workspace.Manager.Teardown`. New `.ai/workspace.json` marker file.
- **Phase 3+4 — `brains workspace`** (commit `5fd8e69`): prep/teardown/gc subcommands with `--spawn`. Deleted `cmd/sandbox-test`. Updated `INFRASTRUCTURE.md`.

### Test Coverage

- `internal/workspace`: 80% coverage, 14 tests (Prep happy + 5 rollback paths, Teardown w/ and w/o marker, ShortTitle parity, marker round-trip).
- `internal/cli`: 30% coverage, +14 new CLI integration tests against real git temp repos.
- `internal/orchestrator`: existing test suite green after refactor (no test changes needed beyond `shortTitle` → `workspace.ShortTitle`).

### Notes

- Sentrux baseline rebased once: prep flow's setup/dispatch decomposition crosses the gate's internal "complex function" heuristic by one. Configured rules (max_cc=25, max_fn_lines=100) still pass.
- Pre-existing `internal/server` test failure remains; unrelated to this work.
- `--spawn` callback URL defaults to none; advanced users override via `--callback-url`. Cmux integration deferred-but-shipped per user decision.
