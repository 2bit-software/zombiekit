# Implementation Plan: `brains workspace` CLI

**Spec**: `spec.md`
**Research**: `research.md`
**Branch**: `69f3a818-feature-orchestrator-cli-extraction`

## Overview

Three new top-level CLI command groups on the `brains` binary:

- `brains worktree` — thin passthrough to `internal/worktree.Manager`
- `brains sandbox` — thin passthrough to `internal/sandbox`
- `brains workspace` — composition (worktree + `.ai/ticket.md` + sandbox + optional cmux spawn) with rollback

Plus one new shared package:

- `internal/workspace` — extracts `setupWorktree`, the rollback sequence, and the optional cmux spawn step from `internal/orchestrator/watcher_linear.go` so daemon and CLI share one code path.

## Phases

### Phase 0 — Extract `internal/workspace` package

**Goal**: Lift the orchestrator-coupled composition glue into a reusable package. No behavior change.

1. Create `internal/workspace/` with these files:
   - `workspace.go` — `Manager` struct + `Prep` + `Teardown` + `MarkerPath` + `ShortTitle`
   - `marker.go` — read/write `.ai/workspace.json`
   - `errors.go` — typed errors, mirrors `internal/worktree` style
   - `workspace_test.go` — table-driven tests with stub worktree.Manager + fake sandbox.

2. **Public API**:
   ```go
   // PrepInput captures everything needed to set up a workspace.
   type PrepInput struct {
       TicketID    string
       Title       string
       Description string  // written to .ai/ticket.md
       Sandbox     bool    // create sandbox after worktree
       Spawn       *SpawnInput  // nil = no cmux session
   }

   type SpawnInput struct {
       Prompt    string
       ExtraEnv  map[string]string
       SessionTitle string  // shown in cmux UI
   }

   type PrepResult struct {
       WorktreePath string
       Branch       string
       SandboxName  string  // empty if Sandbox=false
       SessionRef   string  // empty if Spawn=nil
   }

   // Manager holds shared deps and runs the prep/teardown sequences.
   type Manager struct {
       wt          worktree.Manager
       sbxConfig   sandbox.Config
       cmuxManager *cmux.CmuxManager  // nil disables --spawn
       logger      *slog.Logger
   }

   func NewManager(wt worktree.Manager, sbxCfg sandbox.Config, opts ...Option) *Manager
   func WithCmux(m *cmux.CmuxManager) Option
   func WithLogger(l *slog.Logger) Option

   // Prep does worktree create → write ticket.md → write workspace.json → sandbox create → spawn session.
   // On any failure, rolls back the steps that succeeded.
   func (m *Manager) Prep(ctx context.Context, in PrepInput) (PrepResult, error)

   // Teardown does the reverse: kill session → cleanup sandbox → delete worktree+branch.
   // Idempotent. Continues on individual failures and reports a multi-error.
   func (m *Manager) Teardown(ctx context.Context, ticketID string) error

   // ShortTitle is the title sanitizer (lifted from orchestrator.shortTitle).
   func ShortTitle(title string) string
   ```

3. **Marker file `.ai/workspace.json`** (per D3):
   ```json
   {
     "ticket_id": "DEV-123",
     "title": "feature title",
     "branch": "DEV-123/feature-title",
     "worktree_path": "/abs/path/to/worktrees/DEV-123",
     "sandbox_name": "zk-dev-123",
     "created_at": "2026-04-30T12:00:00Z"
   }
   ```
   Written by `Prep`, read by `Teardown` and `gc`. Required so `Teardown` can find a worktree path given only a ticket ID.

4. **Update `internal/orchestrator/watcher_linear.go`** to call `workspace.Manager.Prep` instead of `setupWorktree` + inline sandbox+cmux. Delete `setupWorktree` and `shortTitle`. Update tests in `watcher_linear_test.go` to stub the new `workspace.Manager` interface (or take the same `worktree.Manager` interface — TBD during implementation; the orchestrator tests already stub `worktree.Manager`, so passing a real `workspace.Manager` over a stubbed `worktree.Manager` is the path of least resistance).

### Phase 1 — `brains worktree` (thin passthrough)

**Goal**: Operator-driven worktree CRUD using the same conventions as the daemon.

1. Create `internal/cli/worktree.go` with `newWorktreeCommand() *cli.Command`. Subcommands:
   - `create <ticket-id> <title>` — wraps `worktree.Manager.CreateWorktree`. Prints worktree path to stdout.
   - `delete <path>` — wraps `worktree.Manager.DeleteWorktree`.
   - `push <path> <branch>` — wraps `worktree.Manager.PushBranch`.
   - `clean-branch <branch>` — wraps `worktree.Manager.CleanBranch`.
   - `list` — runs `git worktree list --porcelain` scoped to the configured root.

2. Each subcommand instantiates the manager via a small helper `loadWorktreeManager(c *cli.Context) (*worktree.GitManager, *cli.ProjectConfig, error)` that:
   - Reads `--config` flag (default: `./orchestrator.toml` if present, else error with hint).
   - Picks project via `--project` flag (default: first project; error if config has multiple and no `--project` given).
   - Calls `worktree.New(repoDir, WithWorktreesRoot(...), WithCopyFiles(...))`.

3. Register in `internal/cli/root.go` Commands slice: `newWorktreeCommand()`.

4. Tests: `internal/cli/worktree_test.go` — exercise each subcommand with a real temp repo (mirrors `internal/worktree/manager_test.go` style).

### Phase 2 — `brains sandbox` (thin passthrough)

**Goal**: Operator-driven sandbox CRUD using the same naming conventions as the daemon.

1. Create `internal/cli/sandbox.go` with `newSandboxCommand() *cli.Command`. Subcommands:
   - `create <ticket-id> <worktree-path>` — wraps `sandbox.Create`. Flags: `--mounts` (repeatable), `--memory`, `--template`. Defaults from `sandbox.DefaultConfig`.
   - `cleanup <ticket-id>` — wraps `sandbox.Cleanup` (idempotent).
   - `available` — exits 0 if `sandbox.Available()`, else 1 with diagnostic.
   - `name <ticket-id>` — prints `sandbox.Name(ticketID)` to stdout.
   - `list` — wraps `sbx ls --quiet`, filters to `zk-*`.

2. Register in `internal/cli/root.go`: `newSandboxCommand()`.

3. Tests: skip integration tests for `create`/`cleanup` (require Docker); unit-test `name` and the `--mounts` parsing helper.

### Phase 3 — `brains workspace` (composition)

**Goal**: One-shot prep/teardown using the new `internal/workspace` package.

1. Create `internal/cli/workspace.go` with `newWorkspaceCommand() *cli.Command`. Subcommands:

   - `prep <ticket-id>` — flags:
     - `--title <string>` (required)
     - `--description <string>` or `--description-file <path>` (one required)
     - `--no-sandbox` (default: sandbox if `sandbox.Available()`)
     - `--spawn` (default false; if set, requires `cmux` on PATH and uses `sandbox.NewCommandBuilder` when sandbox is on)
     - `--prompt <string>` (default: same as orchestrator: "Read .ai/ticket.md ...")
     - `--config <path>`, `--project <id>`
     Calls `workspace.Manager.Prep`. Prints `PrepResult` as either text (default) or JSON (`--format json`).

   - `teardown <ticket-id>` — flags: `--config`, `--project`, `--force` (proceed even if marker missing). Calls `workspace.Manager.Teardown`.

   - `gc` — flags: `--dry-run` (default true), `--config`, `--project`. Walks worktrees root + `sbx ls --quiet`, finds entries without a matching marker / unknown sandbox names, lists or removes.

2. `--spawn` integration:
   - Build `sandbox.Config` from defaults + flags.
   - Create `cmuxManager, err := cmux.New(cmux.WithCommandBuilder(sandbox.NewCommandBuilder(cfg)))` only if `--spawn` is set.
   - Pass to `workspace.NewManager(wt, sbxCfg, workspace.WithCmux(cmuxManager))`.
   - Pass `PrepInput.Spawn = &SpawnInput{Prompt: ..., ExtraEnv: {WORK_CALLBACK_URL: ...}}` only if `--spawn` is set.
   - **Note**: `WORK_CALLBACK_URL` makes no sense without the orchestrator daemon, but the spawned session expects it. Default to a no-op placeholder URL and let `--callback-url` flag override. Document this clearly in `--help`.

3. Register in `internal/cli/root.go`: `newWorkspaceCommand()`.

4. Tests:
   - `internal/workspace/workspace_test.go` — covers `Prep` happy path, sandbox-disabled path, spawn-disabled path, rollback on sandbox failure, `Teardown` idempotency, marker round-trip.
   - `internal/cli/workspace_test.go` — flag parsing + dispatching to a stub `Manager`.

### Phase 4 — Cleanup

1. Delete `cmd/sandbox-test/` once `brains sandbox` + `brains worktree` reach parity (verified by running the same commands `cmd/sandbox-test` documents in its `--help`).
2. Update `INFRASTRUCTURE.md` with the new command surface.
3. Update `README.md` if it advertises `cmd/sandbox-test`.

## Step Ordering / Dependencies

```
Phase 0 (internal/workspace pkg + orchestrator refactor)
  ↓
  ├── Phase 1 (brains worktree)   [independent of 0, can parallelize]
  ├── Phase 2 (brains sandbox)    [independent of 0, can parallelize]
  └── Phase 3 (brains workspace)  [requires Phase 0]
        ↓
        Phase 4 (cleanup)
```

Phases 1 and 2 do not depend on Phase 0 — they only call the existing `internal/worktree` and `internal/sandbox` packages directly. Recommended order: **Phase 1 → Phase 2 → Phase 0 → Phase 3 → Phase 4** so that the simplest CLI wiring lands first and proves the test harness before the bigger refactor.

## Reuse Notes (from `reuse-audit.md`)

- **`loadProjectConfig`** wraps the existing `orchestrator.LoadOrchestratorConfig` — does NOT reimplement TOML parsing. Only adds `--project` and cwd-match logic on top.
- **Marker read/write** uses standard `json.Marshal` + `os.WriteFile`/`os.ReadFile` — no new abstraction.
- **`--config` flag** copies the boilerplate from `cmd/orchestrator/main.go:41-46` (same usage, aliases, env vars).
- **`workspace.Spawner` interface** mirrors `cmux.CmuxManager.SpawnSession` exactly — thin wrapper for testability.
- **`workspace.Sandbox` interface** wraps `internal/sandbox` package-level functions — necessary because `internal/sandbox` exposes no interface today and we need to fake Docker in tests.
- **`shortTitle`** is confirmed package-private to `internal/orchestrator`; lifting to `workspace.ShortTitle` is safe.

## Open Uncertainties

1. **`WORK_CALLBACK_URL` for `--spawn`**: Without the daemon, the spawned Claude session has nowhere to send completion events. Options:
   (a) Document that `--spawn` is fire-and-forget — operator manually monitors via `cmux`.
   (b) Add a `--callback-url` flag for advanced users running their own callback receiver.
   Recommend (a) for v1, (b) deferred.

2. **Multi-project config disambiguation**: If `orchestrator.toml` has multiple projects, CLI commands need a `--project` flag. Should we infer from `cwd` matching `RepoDir`? Cheap auto-detect, error if ambiguous. Recommend yes.

3. **Marker format**: Should `.ai/workspace.json` also include the operator's `--prompt` and the `--spawn` decision (for auditability)? Cheap to add. Recommend yes.

## Test Strategy

| Layer | Approach |
|-------|----------|
| `internal/workspace` | Integration-first. Use real `worktree.GitManager` against a temp repo (mirroring `internal/worktree/manager_test.go`). Stub `sandbox.Create/Cleanup` (would need a thin interface — see Reuse note below) and `cmux.SpawnSession`. Cover happy path + each rollback failure point. |
| `internal/cli/{worktree,sandbox,workspace}` | Real temp repo + stubbed `workspace.Manager` for the workspace cmd. Verify flag parsing, error exit codes, help output. |
| `internal/orchestrator` | Existing tests must keep passing after Phase 0 refactor. Update test stubs to satisfy whatever new interface `workspace.Manager` introduces. |

## Risk Notes

- **`internal/sandbox` is not currently behind an interface** — `sandbox.Create/Cleanup` are package-level functions. To stub them in `workspace_test.go`, we'd need either (a) introduce a `Sandbox` interface in `internal/workspace` and have `workspace.Manager` depend on the interface, with a default impl that calls `sandbox.Create/Cleanup`, or (b) skip mocking and run real Docker in tests. **Recommend (a)** — it's a small interface (~3 methods) and matches how `worktree.Manager` is shaped already.
- **Orchestrator test refactor**: `internal/orchestrator/watcher_linear_test.go` uses `stubWorktree`. After Phase 0, the runner will compose `workspace.Manager` over `worktree.Manager`. Either keep stubbing at `worktree.Manager` level (preferred — fewest test changes) or introduce a `workspace.Manager` interface and stub that. Decision deferred to Phase 0 implementation.
