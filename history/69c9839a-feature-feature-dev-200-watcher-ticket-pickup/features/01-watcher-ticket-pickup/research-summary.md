# Research Summary: Watcher 1 — Ticket Pickup

## All Dependencies Exist

Every interface and implementation needed for this watcher is already built:

| Component | Interface | Implementation | File |
|-----------|-----------|----------------|------|
| Linear client | `linear.Client` | `linear.HTTPClient` | `internal/linear/client.go`, `http_client.go` |
| Worktree manager | `worktree.Manager` | `worktree.GitManager` | `internal/worktree/types.go`, `manager.go` |
| Session manager | `cmux.SessionManager` | `cmux.CmuxManager` | `internal/cmux/types.go`, `manager.go` |
| State store | `state.StateStore` | `state.SQLiteStore` | `internal/state/store.go` |
| Callback server | `callback.CallbackServer` | (concrete) | `internal/callback/server.go` |

## Key Interface Signatures

### linear.Client
```go
PollReadyTickets(ctx context.Context, label string) ([]Ticket, error)
SetTicketStatus(ctx context.Context, id string, status string) error
RemoveLabel(ctx context.Context, id string, label string) error
```

### worktree.Manager
```go
CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error)
DeleteWorktree(ctx context.Context, path string) error
```

### cmux.SessionManager
```go
SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string) (workspaceRef string, err error)
KillSession(ctx context.Context, ticketID string) error
```

### state.StateStore (relevant methods)
```go
CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
ReleaseSlot(ctx context.Context, projectID string) error
SetJobStatus(ctx context.Context, ticketID string, status string) error
```

## Ticket Type
```go
type Ticket struct {
    ID, Identifier, Title, Description, Status, URL, TeamID string
    Labels []string
    Priority int
}
```

## What Needs to Change

### 1. Orchestrator Struct — New Dependencies
Currently: `cfg *Config`, `store state.StateStore`
Needs: `linear linear.Client`, `worktrees worktree.Manager`, `sessions cmux.SessionManager`

### 2. Orchestrator.Run() — Replace Stub
Currently calls `NewWatcherStub(WatcherLinearPoller, ...)`.
Needs to call a real watcher constructor that accepts all dependencies.

### 3. cmd/orchestrator/main.go — Create Real Clients
Currently only creates `state.NewSQLiteStore`. Needs to also create:
- `linear.NewClient(cfg.LinearAPIKey)` (or equivalent)
- `worktree.New(repoDir, worktree.WithWorktreesRoot(cfg.WorktreesRoot))`
- `cmux.New()`

### 4. Config — Possibly New Fields
May need: repo root directory (for worktree.New). Currently has `WorktreesRoot` but not the repo root itself. Need to check if `worktree.New` needs the repo dir.

## Construction Patterns

### linear.NewClient
```go
client, err := linear.NewClient(apiKey, linear.WithMaxRetries(3))
```

### worktree.New
```go
mgr, err := worktree.New(repoDir, worktree.WithWorktreesRoot(root))
```
Takes the repo root directory (where `.git` lives) and optionally a custom worktrees root.

### cmux.New
```go
mgr, err := cmux.New()
```
Checks that cmux binary exists and is running. No config needed.

## Error Rollback Pattern

The ticket specifies: if any step fails after worktree creation, delete worktree and release slot. This is a compensating transaction pattern:

```
acquire slot → create worktree → write ticket file → spawn session → create job → update linear
                 ↓ (failure at any point after worktree creation)
              delete worktree + release slot
```
