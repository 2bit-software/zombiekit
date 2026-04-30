---
status: complete
updated: 2026-04-30
---

# Research: Sandbox + Worktree CLI in `brains`

## Executive Summary

The orchestrator's worktree and sandbox primitives are already cleanly bounded in `internal/worktree` and `internal/sandbox` — both packages have minimal orchestrator coupling and the `cmd/sandbox-test` binary is a working proof that they can be driven from a thin CLI today. The `brains` binary uses urfave/cli/v2 with modular command registration in `internal/cli/`, so adding `brains sandbox` and `brains worktree` is mechanical. The only meaningful refactor is lifting `setupWorktree`'s `.ai/ticket.md` writer (currently in `internal/orchestrator/watcher_linear.go`) into a shared package if we want a `brains workspace prep` composition command.

## Findings

### Codebase Context

#### `internal/worktree` — turnkey

- Exports `Manager` interface: `CreateWorktree`, `DeleteWorktree`, `CleanBranch`, `PushBranch`.
- Constructor: `New(repoDir, opts...)` with `WithWorktreesRoot`, `WithCopyFiles`.
- Typed errors via `Error.Kind` with `IsPathExists/IsBranchExists/IsNotAWorktree/IsWorktreeLocked/IsBranchInUse/IsBranchNotFound/IsNotARepository` helpers.
- Pure git CLI wrapper. Already used by both `internal/orchestrator/runner.go` and `cmd/orchestrator/admin.go` (`jobsDelete`), proving it works outside the watcher path.
- **No refactor needed.**

#### `internal/sandbox` — turnkey

- Exports: `Config`, `DefaultConfig`, `Name(ticketID)`, `Available()`, `Create(ctx, name, worktreePath, cfg)`, `Cleanup(ctx, name)`, `NewCommandBuilder(cfg)`, `RewriteCallbackHost`.
- Wraps the `sbx` CLI; `Cleanup` is idempotent and tolerates missing `sbx`.
- `cmd/sandbox-test/main.go` is a complete working CLI on top of these primitives — it's the de facto prototype for what we're building into `brains`.
- `NewCommandBuilder` returns a `cmux.CommandBuilder` callback. This is the only sandbox API with cmux coupling, and `brains sandbox` doesn't need it (operators run `sbx exec -it ...` themselves).
- **No refactor needed.**

#### `internal/orchestrator` — composition lives here

The orchestrator-only code that touches worktree+sandbox:

- `runTicketPipeline` (watcher_linear.go:75) — composes `setupWorktree` → `sandbox.Create` → `cmux.SpawnSession` with state-store and concurrency-slot bookkeeping. Inlined; not callable.
- `setupWorktree` (watcher_linear.go:151) — calls `worktrees.CreateWorktree`, then writes `.ai/ticket.md` from a `linear.Ticket` struct. Inlined; would need to move to a shared package for reuse.
- Failure rollback (watcher_linear.go:121-128) — kills session, calls `sandbox.Cleanup`, calls `worktrees.DeleteWorktree`. Inlined; would need to move.
- `handleComplete` / `handleFailed` cleanup paths (router.go:100, 145) — call `sandbox.Cleanup` directly with `sandbox.Name(ticketID)`. Already decoupled.
- `cleanupPR` (watcher_pr.go) — calls `worktrees.DeleteWorktree`. Already decoupled.

**Refactor needed only for FR-010, FR-011** (workspace prep / teardown): lift the `.ai/ticket.md` writer + rollback sequence into a new package, e.g., `internal/workspace`.

#### `cmd/brains` + `internal/cli` — well-suited for new subcommands

- Framework: **urfave/cli/v2**.
- Registration is modular in `internal/cli/root.go`: each command lives in its own file (`profile.go`, `skill.go`, `hook.go`, etc.) and exposes a `newXxxCommand() *cli.Command` factory.
- Existing global flags: `--verbose`, `--db-type`, `--log-level`. Subcommands can add their own.
- Existing pattern for nested subcommands (e.g., `brains profile compose`, `brains memory add`) — fits our `brains sandbox create` / `brains worktree delete` shape exactly.
- **Drop-in additions**: `internal/cli/sandbox.go`, `internal/cli/worktree.go`, optional `internal/cli/workspace.go`.

### Domain Knowledge

- The orchestrator's name-derivation conventions matter for compatibility: ticket `DEV-1` → sandbox `zk-dev-1` (lowercase), worktree `<root>/DEV-1` (preserved case), branch `DEV-1/<sanitized-title>`. The CLI must use the same `sandbox.Name` and `shortTitle` helpers (the latter is currently package-private in `internal/orchestrator`; either export it or duplicate the trivial logic).
- Mount conventions: default sandbox mounts `~/.claude`, `~/.brains`, plus user-configured paths. `sandbox.DefaultConfig` already encodes this.
- Stale-recovery is part of correctness: the orchestrator cleans up stale worktrees/sandboxes on each pickup (idempotent). The CLI inherits this for free from `worktree.CreateWorktree` and the `Cleanup`-before-`Create` pattern.

## Decision Points

- [ ] **D1**: Lift `setupWorktree` and its rollback block into a new `internal/workspace` package vs. keep them inline and duplicate in CLI? **Recommend lift** (FR-S004) — duplication of sandbox+worktree composition logic in two places is a ticking time bomb.
- [ ] **D2**: Read `orchestrator.toml` directly from `brains` vs. extract a shared config schema first? **Recommend direct reuse for v1**, extract later if the schema diverges.
- [ ] **D3**: Drop a `.ai/workspace.json` marker (ticket ID + worktree + sandbox name) so `teardown`/`gc` have a reliable record? **Recommend yes** — small file, makes GC robust against renamed worktrees.
- [ ] **D4**: Include `--spawn` (cmux session spawn) in v1? **Recommend defer** — operators can `sbx exec -it ... claude` themselves; cmux integration is the most coupled part of the orchestrator pipeline.

## Recommendations

1. **Phase 1 (P1, ~1 day)**: Add `brains worktree {create, delete, push, clean-branch, list}` and `brains sandbox {create, cleanup, available, name, list}` as thin passthrough wrappers. No refactoring of `internal/orchestrator` required. Delete `cmd/sandbox-test` once parity is reached.
2. **Phase 2 (P2, ~3 days)**: Extract `setupWorktree` (the `.ai/ticket.md` writer minus the `linear.Ticket` coupling — take title + description as plain args) and the rollback block into `internal/workspace`. Add `brains workspace {prep, teardown, gc}`. Update `internal/orchestrator/watcher_linear.go` to call the new package.
3. **Phase 3 (P3, ~½ day)**: Wire global `--config` to `orchestrator.LoadOrchestratorConfig` so worktree root, copy-files, and sandbox mounts are consistent across daemon and CLI.

## Lifecycle Coverage Matrix

Maps every meaningful orchestrator step to its CLI equivalent. Steps marked "n/a — daemon" are intentionally not exposed.

| # | Orchestrator Step | Location | Already Callable? | CLI Equivalent | Refactor? |
|---|---|---|---|---|---|
| 1 | Load config | `multi_config.go:LoadOrchestratorConfig` | yes | implicit via `--config` flag | no |
| 2 | Init logging | `internal/logging.InitLogger` | yes | reuse existing `--log-level` | no |
| 3 | Init state store | `state.NewSQLiteStore` | yes | n/a — daemon | — |
| 4 | Reconcile orphans | `state.ApplyReconciliation` | yes | n/a — daemon | — |
| 5 | Resolve sandbox mode | `cmd/orchestrator/run.go:resolveSandboxMode` | yes (private) | `brains sandbox available` | export or duplicate |
| 6 | Init cmux | `cmux.New` | yes | n/a — out of scope (`--spawn` deferred) | — |
| 7 | Callback server | `callback.New` | yes | n/a — daemon | — |
| 8 | Per-project runners | `orchestrator.NewProjectRunner` | yes | n/a — daemon | — |
| 9 | Linear poller | inlined in `RunSupervised` | no | n/a — daemon | — |
| 10 | Setup worktree | `watcher_linear.go:setupWorktree` | no (inlined) | `brains worktree create` + `brains workspace prep` | **lift to `internal/workspace`** |
| 11 | Create sandbox | `sandbox.Create` | yes | `brains sandbox create` | no |
| 12 | Spawn session | `cmux.SpawnSession` | yes | n/a — out of scope | — |
| 13 | PR watcher | inlined | no | n/a — daemon | — |
| 14 | Comment watcher | inlined | no | n/a — daemon | — |
| 15 | Per-PR queue | inlined | no | n/a — daemon | — |
| 16 | Event router | inlined | no | n/a — daemon | — |
| 17 | Handle complete (push, create PR) | `router.go:handleComplete` | no | n/a — daemon | — |
| 18 | Handle failure (rollback) | inlined in watcher_linear.go | no | `brains workspace teardown` | **lift to `internal/workspace`** |
| 19 | Handle comment resolved | `router.go:handleCommentResolved` | no | n/a — daemon | — |
| 20 | Cleanup sandbox | `sandbox.Cleanup` | yes | `brains sandbox cleanup` | no |
| 21 | Delete worktree | `worktree.DeleteWorktree` | yes | `brains worktree delete` | no |
| 22 | Push branch | `worktree.PushBranch` | yes | `brains worktree push` | no |
| 23 | Clean orphan branch | `worktree.CleanBranch` | yes | `brains worktree clean-branch` | no |
| 24 | Stale sandbox/worktree GC | implicit on pickup | partial | `brains workspace gc` | new code |
| 25 | Admin: jobs delete (`--worktree`/`--session`) | `cmd/orchestrator/admin.go:jobsDelete` | yes (CLI) | overlapping with `brains workspace teardown` | none — keep both, document overlap |

## Sources

- `cmd/orchestrator/{main,run,admin}.go`
- `internal/orchestrator/{orchestrator,config,multi_config,runner,router,comment_dispatcher,watcher_linear,watcher_pr,watcher_comment,watchers,interfaces}.go`
- `internal/worktree/{manager,types,errors,sanitize,doc}.go`
- `internal/sandbox/{sandbox,command,url}.go`
- `cmd/sandbox-test/main.go` (working CLI prototype)
- `cmd/brains/main.go` and `internal/cli/root.go` (CLI structure)
