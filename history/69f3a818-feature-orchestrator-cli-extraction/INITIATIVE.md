# Initiative: orchestrator-cli-extraction

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-30
**ID**: 69f3a818-feature-orchestrator-cli-extraction

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | completed | 2026-04-30 12:35 |
| plan | plan | completed | 2026-04-30 12:50 |
| tasks | tasks | completed | 2026-04-30 13:00 |
| implement | implement | in_progress | 2026-04-30 13:00 |

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
