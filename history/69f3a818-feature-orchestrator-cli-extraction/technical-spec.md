# Technical Spec: `brains workspace` CLI

## Package Layout

```
internal/
  workspace/
    workspace.go      # Manager + Prep + Teardown
    marker.go         # .ai/workspace.json read/write
    errors.go         # typed errors
    workspace_test.go
    doc.go
  cli/
    worktree.go       # newWorktreeCommand
    worktree_test.go
    sandbox.go        # newSandboxCommand
    sandbox_test.go
    workspace.go      # newWorkspaceCommand
    workspace_test.go
    config.go         # loadOrchestratorConfig + project selection helper (NEW)
```

## Interface Boundaries

### `internal/workspace`

```go
package workspace

import (
    "context"
    "log/slog"
    "github.com/2bit-software/zombiekit/internal/cmux"
    "github.com/2bit-software/zombiekit/internal/sandbox"
    "github.com/2bit-software/zombiekit/internal/worktree"
)

// Sandbox is the subset of internal/sandbox the manager depends on.
// Introduced so workspace_test.go can fake out Docker.
type Sandbox interface {
    Available() bool
    Create(ctx context.Context, name, worktreePath string, cfg sandbox.Config) error
    Cleanup(ctx context.Context, name string)
    Name(ticketID string) string
}

// Spawner is the subset of cmux.CmuxManager the manager depends on.
// nil disables --spawn.
type Spawner interface {
    SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string, prompt string) (string, error)
    KillSession(ctx context.Context, ticketID string) error
}

type Manager struct {
    wt        worktree.Manager
    sbx       Sandbox
    sbxCfg    sandbox.Config
    spawner   Spawner   // optional
    logger    *slog.Logger
}

type Option func(*Manager)
func WithSpawner(s Spawner) Option
func WithLogger(l *slog.Logger) Option
func WithSandbox(s Sandbox) Option  // defaults to internal/sandbox passthrough

func NewManager(wt worktree.Manager, sbxCfg sandbox.Config, opts ...Option) *Manager

type PrepInput struct {
    TicketID, Title, Description string
    Sandbox                      bool
    Spawn                        *SpawnInput  // nil = no spawn
}

type SpawnInput struct {
    Prompt       string
    ExtraEnv     map[string]string
    SessionTitle string
}

type PrepResult struct {
    WorktreePath, Branch, SandboxName, SessionRef string
}

func (m *Manager) Prep(ctx context.Context, in PrepInput) (PrepResult, error)
func (m *Manager) Teardown(ctx context.Context, ticketID string) error

// ShortTitle is the same sanitizer currently in internal/orchestrator.shortTitle.
func ShortTitle(title string) string

// MarkerPath returns the absolute path to .ai/workspace.json for a worktree.
func MarkerPath(worktreePath string) string

// ReadMarker returns the marker contents, or ErrNoMarker if absent.
func ReadMarker(worktreePath string) (Marker, error)
```

### Default `Sandbox` impl

A package-level `defaultSandbox{}` struct that calls through to `internal/sandbox`. `WithSandbox(...)` overrides for testing.

### Marker

```go
type Marker struct {
    TicketID     string    `json:"ticket_id"`
    Title        string    `json:"title"`
    Branch       string    `json:"branch"`
    WorktreePath string    `json:"worktree_path"`
    SandboxName  string    `json:"sandbox_name,omitempty"`
    Spawned      bool      `json:"spawned,omitempty"`
    Prompt       string    `json:"prompt,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
}
```

## `Prep` sequence (with rollback)

```
1. wt.CreateWorktree(ticketID, ShortTitle(title)) → worktreePath
2. mkdir worktreePath/.ai
3. write worktreePath/.ai/ticket.md (description)
4. write worktreePath/.ai/workspace.json (marker)
   ─ rollback to step 0 (delete worktree) if any of 2-4 fail
5. if in.Sandbox && sbx.Available(): sbx.Create(name, worktreePath, sbxCfg)
   ─ rollback (delete worktree) if create fails
6. if in.Spawn != nil: spawner.SpawnSession(...)
   ─ rollback (sbx cleanup, delete worktree) if spawn fails
7. return PrepResult
```

## `Teardown` sequence

```
1. read marker (best-effort; if missing, derive paths from ticketID + cfg)
2. spawner.KillSession(ticketID) if spawner != nil  (best-effort, log on error)
3. sbx.Cleanup(name)  (idempotent)
4. wt.DeleteWorktree(worktreePath)
5. return aggregated error (multierror) of any non-best-effort failures
```

## CLI flag schema

### `brains worktree`

```
brains worktree create <TICKET-ID> <TITLE>      [--config PATH] [--project ID]
brains worktree delete <PATH>                   [--config PATH] [--project ID]
brains worktree push <PATH> <BRANCH>            [--config PATH] [--project ID]
brains worktree clean-branch <BRANCH>           [--config PATH] [--project ID]
brains worktree list                            [--config PATH] [--project ID]
```

### `brains sandbox`

```
brains sandbox create <TICKET-ID> <WORKTREE>    [--mounts PATH ...] [--memory N] [--template T]
brains sandbox cleanup <TICKET-ID>
brains sandbox available
brains sandbox name <TICKET-ID>
brains sandbox list
```

### `brains workspace`

```
brains workspace prep <TICKET-ID> --title T [--description D | --description-file F]
                                  [--no-sandbox] [--spawn] [--prompt P]
                                  [--callback-url URL] [--format text|json]
                                  [--config PATH] [--project ID]
brains workspace teardown <TICKET-ID> [--force] [--config PATH] [--project ID]
brains workspace gc [--dry-run] [--config PATH] [--project ID]
```

Global: existing `--verbose`, `--log-level`, `--db-type`.

## Config loading helper

New file `internal/cli/config.go`:

```go
func loadProjectConfig(c *cli.Context) (*orchestrator.ProjectConfig, error)
```

Logic:
1. Read `--config` flag; if empty, look for `./orchestrator.toml`; if missing, return error with hint.
2. Call `orchestrator.LoadOrchestratorConfig(path)`.
3. If `--project` flag set: pick that project by `id`.
4. Else if config has 1 project: use it.
5. Else if `cwd` matches a `project.RepoDir` (or `cwd` is inside one): use that project.
6. Else: error listing all project IDs.

## Refactor of `internal/orchestrator/watcher_linear.go`

After Phase 0:

```go
// Before:
worktreePath, err := p.setupWorktree(ctx, ticket)
if err != nil { ... }
sbxName := sandbox.Name(ticket.Identifier)
if p.sandboxOn {
    if err := sandbox.Create(ctx, sbxName, worktreePath, p.sandboxConfig); err != nil { ... }
}
sessionRef, err := p.sessions.SpawnSession(ctx, ticket.Identifier, ...)
// ... (rollback inline)

// After:
result, err := p.workspace.Prep(ctx, workspace.PrepInput{
    TicketID:    ticket.Identifier,
    Title:       ticket.Title,
    Description: ticket.Description,
    Sandbox:     p.sandboxOn,
    Spawn:       &workspace.SpawnInput{Prompt: prompt, ExtraEnv: env, SessionTitle: ticket.Title},
})
if err != nil {
    p.markTicketNeedsAttention(ctx, ticket)
    return "", "", err
}
return result.SessionRef, result.WorktreePath, nil
```

`p.workspace` is built once in `NewProjectRunner`:

```go
p.workspace = workspace.NewManager(p.worktrees, p.sandboxConfig,
    workspace.WithSpawner(p.sessions),
    workspace.WithLogger(p.logger),
)
```

Rollback inside `runTicketPipeline` is removed — it now lives in `workspace.Prep` itself.

## Test plan

### `internal/workspace/workspace_test.go`

Use real `worktree.GitManager` against a `t.TempDir()` repo (initialized via `git init`). Stub `Sandbox` and `Spawner` with simple in-memory recorders.

Cases:
- `Prep_Success_AllSteps` — sandbox=true, spawn=set → all four steps run, marker written.
- `Prep_NoSandbox_NoSpawn` — sandbox=false, spawn=nil → only worktree+marker.
- `Prep_SandboxFails_RollsBackWorktree` — sandbox returns error → worktree deleted.
- `Prep_SpawnFails_RollsBackSandboxAndWorktree` — spawn returns error → both cleaned.
- `Prep_TicketMdWriteFails_RollsBackWorktree` — simulate via read-only worktree dir.
- `Teardown_Success` — happy path.
- `Teardown_NoMarker_FallsBackToConvention` — marker missing, ticket ID resolves to conventional path.
- `Teardown_SandboxMissing_NoOp` — `Cleanup` is idempotent.
- `Teardown_SessionMissing_NoOp` — `KillSession` returns "not found", treated as best-effort.
- `Marker_RoundTrip` — write + read returns same struct.

### `internal/cli/worktree_test.go`

Real `git init` repo, exercise each subcommand via `app.Run([]string{...})`. Assert worktree paths, branches.

### `internal/cli/sandbox_test.go`

Skip integration tests for `create`/`cleanup` (require Docker). Test `name` and `--mounts` parsing.

### `internal/cli/workspace_test.go`

Stub `workspace.Manager` (introduce a small interface in `internal/cli` if needed). Verify flag parsing, exit codes, JSON output shape.

## Migration of existing code

| File | Change |
|------|--------|
| `internal/orchestrator/watcher_linear.go` | Delete `setupWorktree`, `shortTitle`. Replace `runTicketPipeline` body with `workspace.Prep` call. |
| `internal/orchestrator/runner.go` | `NewProjectRunner` constructs `workspace.Manager` and stores on `ProjectRunner`. |
| `internal/orchestrator/router.go:100,145` | Replace direct `sandbox.Cleanup` calls with `p.workspace.Teardown` (which encapsulates cleanup) — or leave as-is since they're already cleanly decoupled. **Decision**: leave as-is for now; revisit if duplication becomes painful. |
| `internal/orchestrator/watcher_linear_test.go` | Tests stub `worktree.Manager`; keep that, but pass a real `workspace.Manager` over the stub. May need to introduce a `workspaceManager` interface on `ProjectRunner` for cleaner stubbing. |
| `cmd/orchestrator/run.go` | No change. |
| `cmd/sandbox-test/main.go` | DELETE in Phase 4. |
| `internal/cli/root.go` | Add three `Commands` entries. |

## Performance / concurrency

- `Prep` and `Teardown` are sequential per ticket. No locking needed since orchestrator already serializes per-ticket via concurrency slots, and CLI users running these commands are responsible for not racing the daemon (matches existing `orchestrator jobs delete` behavior).
- Marker write is a single small JSON file; no atomic-rename needed (acceptable to lose marker on hard crash mid-write — `Teardown --force` covers that case).
