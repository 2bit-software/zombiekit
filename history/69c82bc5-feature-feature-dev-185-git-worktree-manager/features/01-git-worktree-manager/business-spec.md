# Business Specification: Git Worktree Manager

## Purpose

Provide a self-contained worktree lifecycle primitive that the orchestrator can call to create isolated working directories for agent sessions. Each worktree gets its own branch, enabling parallel ticket work without checkout conflicts.

## Actors

- **Orchestrator** — the sole caller; creates worktrees when spawning agents, deletes them on completion or failure
- **Agent** — works inside the worktree but never calls the worktree manager directly

## Interface Contract

```go
type Manager interface {
    CreateWorktree(ctx context.Context, ticketID, shortTitle string) (path string, err error)
    DeleteWorktree(ctx context.Context, path string) error
    CleanBranch(ctx context.Context, branch string) error
}
```

Constructor:

```go
func New(repoDir string, opts ...Option) (*GitManager, error)
```

Options:

```go
WithWorktreesRoot(path string) Option  // Override default worktrees root
```

The constructor eagerly validates that `repoDir` is a git repository and that `git` is available on PATH. Returns an error if either check fails.

## Capabilities

### C1: Create Worktree

Given a ticket ID and short title, create a new git worktree with a dedicated branch.

**Inputs:**
- `ctx` — context for cancellation/timeout
- `ticketID` (required) — e.g., "DEV-185"
- `shortTitle` (required) — human-readable label, e.g., "git worktree manager"

**Outputs:**
- Absolute path to the created worktree directory

**Behavior:**
- Worktree created at `{worktrees-root}/{ticket-id}` (flat, deterministic)
- Branch named `{ticket-id}/{sanitized-short-title}`
- Branch created from current HEAD of the main worktree
- If a worktree already exists at that path, return an error (no silent overwrite)
- If the branch name already exists (whether or not it has a worktree), return an error
- Creates the worktrees root directory if it does not exist

**Short title sanitization rules (applied in this order):**
1. Lowercase the entire string
2. Replace spaces with hyphens
3. Strip characters that are not alphanumeric, hyphens, or underscores (ASCII only — non-ASCII characters are stripped)
4. Collapse consecutive hyphens into one
5. Trim leading and trailing hyphens
6. Truncate to 40 characters maximum
7. Trim trailing hyphens again (in case truncation exposed one)
8. If result is empty, use `"untitled"` as fallback

### C2: Delete Worktree

Given a worktree path, remove the worktree directory and its associated branch.

**Inputs:**
- `ctx` — context for cancellation/timeout
- `path` (required) — absolute path to the worktree directory

**Outputs:**
- None (success) or error

**Behavior:**
- Resolves the associated branch by querying git for the worktree's branch reference
- Removes the worktree directory (force-removes dirty state — uncommitted work is the agent's responsibility to commit before signaling completion)
- Deletes the associated local branch (force-delete, since merge decisions are the orchestrator's responsibility)
- If the path does not exist or is not a worktree, return an error
- If the worktree is locked, return an error (caller must unlock explicitly if intended)

### C3: Clean Branch

Given a branch name, delete the local branch only. Used when the worktree has already been removed (e.g., manual cleanup) but the branch persists.

**Inputs:**
- `ctx` — context for cancellation/timeout
- `branch` (required) — branch name to delete

**Outputs:**
- None (success) or error

**Behavior:**
- Deletes the local branch (force-delete)
- If the branch has an active worktree, return an error
- If the branch does not exist, return an error

## Configuration

| Parameter | Source | Default | Description |
|-----------|--------|---------|-------------|
| Worktrees root | `WithWorktreesRoot` option | `../worktrees` resolved relative to the repository working tree root (e.g., repo at `/home/user/repo` -> `/home/user/worktrees`) | Directory where worktrees are created |
| Repository path | constructor arg `repoDir` | (required) | Path to the repository working tree root (the directory containing `.git`) |

## Error Design

Follow the `ErrorKind` enum pattern from `internal/linear/errors.go`:

```go
type ErrorKind int

const (
    ErrPathExists ErrorKind = iota + 1
    ErrBranchExists
    ErrNotAWorktree
    ErrWorktreeLocked
    ErrBranchInUse
    ErrBranchNotFound
    ErrGitUnavailable
    ErrNotARepository
    ErrGitCommand         // catch-all for unexpected git failures
)

type Error struct {
    Kind    ErrorKind
    Message string       // human-readable, includes context
    Err     error        // underlying error (git stderr output)
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }
```

Provide `Is<Kind>(err error) bool` helper functions for each kind.

**Error classification:** Parse git stderr output to map to the appropriate `ErrorKind`. Unrecognized git errors map to `ErrGitCommand`.

### Error Conditions

| Condition | Capability | ErrorKind |
|-----------|-----------|-----------|
| Worktree path already exists | C1 | `ErrPathExists` |
| Branch name already exists (any reason) | C1 | `ErrBranchExists` |
| Path is not a valid worktree | C2 | `ErrNotAWorktree` |
| Worktree is locked | C2 | `ErrWorktreeLocked` |
| Branch has active worktree | C3 | `ErrBranchInUse` |
| Branch does not exist | C3 | `ErrBranchNotFound` |
| Git not available on PATH | constructor | `ErrGitUnavailable` |
| Not inside a git repository | constructor | `ErrNotARepository` |
| Unexpected git failure | Any | `ErrGitCommand` |

## Invariants

1. One worktree per ticket ID — the path `{worktrees-root}/{ticket-id}` is unique
2. Worktree creation is idempotent-safe: calling twice with the same ticket ID fails predictably (not silently succeeds)
3. After successful `DeleteWorktree`, neither the directory nor the branch exist
4. No remote side effects — all operations are local-only
5. No worktree preservation decisions — the caller decides whether to delete on failure

## Verification

Tests must use real local git repositories (via `t.TempDir()` + `git init`), not mocked git commands. Skip tests when git is unavailable on PATH.

## Out of Scope

- Remote operations (push, fetch)
- GitHub interaction (that's `GitHubClient`)
- Worktree preservation decisions on failure (caller's responsibility)
- Merge detection or PR lifecycle
- Listing worktrees (orchestrator can call git directly if needed later)
- Pruning stale worktrees (file a separate ticket if needed)
- Concurrency safety (orchestrator serializes calls per ticket ID)
