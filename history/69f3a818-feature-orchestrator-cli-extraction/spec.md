# Feature Specification: Sandbox + Worktree CLI in `brains`

**Feature Branch**: `69f3a818-feature-orchestrator-cli-extraction`
**Created**: 2026-04-30
**Status**: Draft

## Summary

Expose the orchestrator's worktree and sandbox lifecycle primitives as first-class subcommands on the `brains` binary so an operator can drive the same flow manually — without running the daemon, polling Linear, or going through the watcher loops.

## User Scenarios & Testing

### User Story 1 — Manually create + tear down a worktree (P1)

As an operator, I want to create a git worktree for a ticket ID and tear it down later, using the same conventions the orchestrator uses (path layout, branch naming, configured copy-files), without writing a daemon config or starting the watcher.

**Why this priority**: This is the most common ad-hoc operation. Today operators either run the daemon or fall back to raw `git worktree` commands, losing the conventions baked into `internal/worktree`.

**Independent test**: Run `brains worktree create DEV-123 "feature title"`; verify a worktree exists at the expected path with the expected branch and the configured files copied. Then `brains worktree delete <path>` removes the worktree and branch.

**Acceptance scenarios**:
1. **Given** a clean repo, **when** the user runs `brains worktree create DEV-1 "short title"`, **then** the worktree is created at `<root>/DEV-1`, branch `DEV-1/short-title` exists, and configured copy-files are present.
2. **Given** an existing worktree, **when** the user runs `brains worktree delete <path>`, **then** the worktree dir and branch are removed.
3. **Given** a stale worktree from a prior crash, **when** the user runs `brains worktree create` for the same ticket, **then** the stale worktree is cleaned up before the new one is created (matching orchestrator behavior).

### User Story 2 — Manually create + use a sandbox (P1)

As an operator, I want to create a Docker Sandbox tied to a worktree (mounting `~/.claude`, `~/.brains`, plus configured paths) and clean it up later, with the same name-derivation rules the orchestrator uses.

**Why this priority**: Equally common. The existing `cmd/sandbox-test` binary proves the primitives are CLI-ready; this story formalizes them in `brains` so they survive past the prototype.

**Independent test**: `brains sandbox create DEV-123 <worktree>` produces a sandbox named `zk-dev-123` mounting the expected paths; `brains sandbox cleanup DEV-123` removes it; both are idempotent.

**Acceptance scenarios**:
1. **Given** `sbx` is on PATH, **when** the user runs `brains sandbox create DEV-1 /path/to/worktree`, **then** sandbox `zk-dev-1` exists with default mounts.
2. **Given** an existing sandbox, **when** the user runs `brains sandbox cleanup DEV-1`, **then** the sandbox is removed; rerunning the command is a no-op.
3. **Given** `sbx` is missing, **when** the user runs `brains sandbox available`, **then** the command exits non-zero with a clear message.

### User Story 3 — One-shot workspace prep (P2)

As an operator, I want a single `brains workspace prep DEV-123 --title "..."` command that does the full orchestrator pickup sequence (worktree → copy files → write `.ai/ticket.md` → sandbox), so I can hand a fully-prepared workspace to a session I run myself.

**Why this priority**: Useful, but compositional — Stories 1 and 2 cover the underlying primitives. P2 because it requires lifting `setupWorktree` (which currently writes `.ai/ticket.md` from a `linear.Ticket` struct) out of `internal/orchestrator/watcher_linear.go`.

**Independent test**: After running `brains workspace prep`, verify the worktree exists, the sandbox exists, `.ai/ticket.md` contains the supplied description, and the operator can attach via `sbx exec -it zk-... bash`.

### User Story 4 — Workspace teardown / GC (P2)

As an operator, I want `brains workspace teardown DEV-123` to do the orchestrator's rollback sequence (kill session if any, cleanup sandbox, delete worktree+branch) and `brains workspace gc` to scan for stale `zk-*` sandboxes and orphan worktrees and offer cleanup.

**Why this priority**: Operators today resort to raw `sbx`, `git worktree remove`, and `cmux` commands when the daemon crashes. A unified teardown/GC closes the gap.

### User Story 5 — Inspection (P3)

`brains worktree list` and `brains sandbox list` print current state in the same format an operator already gets from `git worktree list` / `sbx ls`, but filtered to project-relevant entries. Cheap addition, low value alone.

### Edge Cases

- `sbx` CLI missing → `sandbox` subcommands exit non-zero with actionable message; `workspace prep` falls back to worktree-only mode with `--no-sandbox`.
- Worktree already exists for ticket ID → reuse if branch matches, otherwise error with `--force` to overwrite.
- Branch already in use elsewhere → surface `worktree.IsBranchInUse` error verbatim.
- Stale sandbox with same name → idempotent cleanup before create (matches orchestrator).
- User runs `brains` commands while orchestrator daemon is also running → no locking; document that operators are responsible for not racing the daemon (matches existing `orchestrator jobs delete` behavior).

## Requirements

### Functional Requirements

- **FR-001**: `brains worktree create <ticket-id> <title>` MUST produce the same worktree layout as the orchestrator (path under configured root, branch `<ticket-id>/<sanitized-title>`, configured copy-files).
- **FR-002**: `brains worktree delete <path>` MUST remove the worktree directory and its branch.
- **FR-003**: `brains worktree push <path> <branch>` MUST push the branch to origin (parity with `worktree.Manager.PushBranch`).
- **FR-004**: `brains worktree clean-branch <branch>` MUST delete an orphan branch (parity with `worktree.Manager.CleanBranch`).
- **FR-005**: `brains worktree list` MUST list worktrees in the configured root.
- **FR-006**: `brains sandbox create <ticket-id> <worktree-path>` MUST create a sandbox named via `sandbox.Name(ticketID)` with `sandbox.DefaultConfig` mounts (overridable by flags).
- **FR-007**: `brains sandbox cleanup <ticket-id>` MUST be idempotent and tolerate missing `sbx` CLI.
- **FR-008**: `brains sandbox available` MUST return zero exit when `sbx` is on PATH, non-zero otherwise.
- **FR-009**: `brains sandbox name <ticket-id>` MUST print the deterministic sandbox name (so other tools can discover it).
- **FR-010**: `brains workspace prep <ticket-id>` MUST compose worktree create → write `.ai/ticket.md` (from `--title` and `--description` or `--description-file`) → sandbox create, with rollback if any step fails.
- **FR-011**: `brains workspace teardown <ticket-id>` MUST kill any cmux session for the ticket, cleanup the sandbox, and delete the worktree+branch — matching the orchestrator failure-rollback sequence.
- **FR-012**: `brains workspace gc` MUST list stale `zk-*` sandboxes and orphan worktrees and remove them (with `--dry-run` default).
- **FR-013**: All commands MUST honor a `--config` flag pointing at the same TOML the orchestrator uses (so worktree root and copy-files are consistent), with sensible CLI-flag overrides.
- **FR-014**: All commands MUST exit non-zero on failure with a clear message and surface typed errors from `internal/worktree` (e.g. `IsPathExists`, `IsBranchInUse`) as distinct exit codes or messages.

### Key Entities

- **Worktree**: A git worktree at `<worktrees-root>/<ticket-id>` on branch `<ticket-id>/<sanitized-title>`. Created/torn down by `internal/worktree.Manager`.
- **Sandbox**: A Docker Sandbox named `zk-<lowercase-ticket-id>` (via `sandbox.Name`) that mounts the worktree plus configured host paths. Created/torn down by `internal/sandbox`.
- **Workspace**: The composition of one Worktree + one Sandbox + a `.ai/ticket.md` file, identified by ticket ID.

## Success Criteria

- **SC-001**: An operator can fully replace `cmd/sandbox-test` with `brains sandbox` + `brains worktree` for the same workflows; `cmd/sandbox-test` is deletable.
- **SC-002**: An operator can recover a workspace after an orchestrator crash using only `brains workspace teardown` + `brains workspace prep`, without manual `git worktree` or `sbx` invocations.
- **SC-003**: All new commands have integration tests that exercise real `git` and (where available) real `sbx`, mirroring the existing `internal/worktree/manager_test.go` style.
- **SC-004**: No code is duplicated between `internal/orchestrator/watcher_linear.go` and the new CLI commands — the `setupWorktree` ticket-md write helper is lifted into a shared package and called from both sites.

## Out of Scope

- Polling Linear, watching PRs, comment dispatch, callback HTTP server — these are daemon-only concerns and stay in `cmd/orchestrator`.
- Spawning sessions / `cmux` integration — out of scope for the CLI surface (operators run `sbx exec -it ... claude ...` themselves). Optional `--spawn` flag deferred to a future iteration.
- State store (`internal/state`) integration — workspace prep/teardown does NOT write job records. The state store is daemon-private.
- Concurrency slot management — irrelevant outside the daemon.

## Refactoring Effort Estimate

| Scope | Effort | Notes |
|-------|--------|-------|
| `brains worktree *` (Stories 1, 5) | **S** (~½ day) | Pure passthrough to `internal/worktree.Manager`. No refactor needed; package is already cleanly bounded. |
| `brains sandbox *` (Story 2, 5) | **S** (~½ day) | Pure passthrough to `internal/sandbox`. `cmd/sandbox-test/main.go` is a working reference. |
| `brains workspace prep` (Story 3) | **M** (~1–2 days) | Lift `setupWorktree`'s `.ai/ticket.md` writer out of `internal/orchestrator/watcher_linear.go` into a shared package (e.g., `internal/workspace`). Compose the steps with rollback. |
| `brains workspace teardown` (Story 4) | **M** (~1 day) | Lift orchestrator's rollback block (sandbox cleanup + worktree delete) into the same shared package. |
| `brains workspace gc` (Story 4) | **S–M** (~1 day) | New code: scan `sbx ls` for `zk-*` and worktree root for orphans. |
| Wire global `--config` (FR-013) | **S** (~½ day) | Reuse `orchestrator.LoadOrchestratorConfig` from `internal/orchestrator/multi_config.go`. |

**Total**: ~4–6 days for full scope (P1 + P2 + P3). P1-only is ~1 day.

## Open Questions

1. **Config sharing**: Should `brains` read `orchestrator.toml` directly, or do we extract worktree-root/copy-files/sandbox-mounts into a shared config schema first? Direct reuse is fastest; shared schema is cleaner long-term.
2. **Workspace state directory**: The orchestrator persists job state in SQLite. The CLI explicitly does not. Should `brains workspace prep` drop a `.ai/workspace.json` marker file (ticket ID, worktree path, sandbox name) so `teardown`/`gc` have something to read besides naming conventions? Recommend yes — small file, big payoff for GC reliability.
3. **`--spawn` flag**: Out of scope per above, but worth confirming the user agrees. The `cmux` + sandbox `CommandBuilder` integration is non-trivial to expose cleanly.
4. **Naming**: `brains workspace` vs `brains pickup` vs `brains prep`. Workspace is descriptive; pickup matches orchestrator vocabulary. User preference?
