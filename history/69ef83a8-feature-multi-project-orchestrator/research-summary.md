# Research Summary: Multi-Project Orchestrator

## Codebase Architecture (Current State)

### Entry Point and Composition Root
- `cmd/orchestrator/main.go` → subcommands: `run`, `jobs`, `slots`
- `cmd/orchestrator/run.go:20-74` is the true composition root — constructs all dependencies and calls `orchestrator.New(...).Run()`
- Config parsed from CLI flags (urfave/cli) with `ORCH_*` env var fallbacks

### Config (`internal/orchestrator/config.go:19-47`)
All fields are single-valued. Key fields: `ProjectID`, `RepoDir`, `GitHubOwner`, `GitHubRepo`, `BaseBranch`, `ConcurrencyLimit`, `LinearAPIKey`, `GitHubToken`, `CallbackPort`, `DBPath`, `PollInterval`.

### Orchestrator Struct (`internal/orchestrator/orchestrator.go:20-27`)
Holds: `cfg *Config`, `store state.StateStore`, `linear linear.Client`, `github github.Client`, `worktrees worktree.Manager`, `sessions cmux.SessionManager`. All singletons.

### Watcher Architecture
All three watchers are methods on `*Orchestrator` returning `shutdown.ServiceFunc`:
- **LinearPoller** (`watcher_linear.go:27-45`): ticker loop, calls `PollReadyTickets(ctx, label, o.cfg.ProjectID)`
- **PRWatcher** (`watcher_pr.go:20-38`): ticker loop, lists jobs by status, checks merged/closed
- **CommentWatcher** (`watcher_comment.go:21-39`): ticker loop with `CommentDispatcher` — per-PR goroutine queues

Lifecycle: `shutdown.Manager` uses `golang.org/x/sync/errgroup`. All 5 services (callback, router, 3 watchers) are peers. If any returns, context cancels everything.

### State Store (`internal/state/store.go`)
SQLite with WAL mode, `MaxOpenConns(1)`.

**Jobs table** (001_initial_schema.sql): PK `ticket_id TEXT`. Migration 002 added `project_id TEXT NOT NULL DEFAULT ''` as a column but did NOT change the PK.

**Comment watermarks table**: PK `pr_number INTEGER`.

**Concurrency slots table**: PK `project_id TEXT` — already per-project in schema.

Key issue: `ListJobsByStatus`, `GetJob`, `GetJobByPR` do NOT filter by project. PRWatcher iterates all queued jobs regardless of project.

### GitHub Client (`internal/github/http_client.go:39-76`)
Constructor takes `(token, owner, repo)` — scoped to one repo. **Needs one per project.**

### Worktree Manager (`internal/worktree/types.go`)
Holds `repoDir`, `worktreesRoot`. **Needs one per project.**

### Linear Client (`internal/linear/http_client.go`)
Constructor takes `(apiKey)` only. `PollReadyTickets` takes `projectID` as parameter. The GraphQL query already filters by project. **Can be shared (singleton per API key).**

### Callback Server (`internal/callback/server.go`)
Routes: `POST /{ticketID}/complete`, `/{ticketID}/comment-resolved`, `/{ticketID}/failed`. No project concept in URLs or events. Router looks up job by ticket_id to implicitly resolve project.

### Concurrency Slots
`TryAcquireSlot(ctx, projectID, limit)` and `ReleaseSlot(ctx, projectID)` are already project-scoped in the DB. All callers hard-code `o.cfg.ProjectID`.

## Multi-Project Impact Matrix

| Component | Current | Multi-Project Change | Effort |
|---|---|---|---|
| Config | Single-valued CLI flags | TOML `[global]` + `[[project]]` | High |
| Orchestrator struct | Singleton | Per-project runners | High |
| Linear client | Singleton, project-agnostic | Keep singleton (shared API key) | None |
| GitHub client | Singleton, one owner/repo | One per project | Low |
| Worktree manager | Singleton, one repo | One per project | Low |
| Session manager | Singleton, project-agnostic | Keep singleton | None |
| Callback server | Single port, ticket_id routing | Add project_id to URL path | Medium |
| Event routing | Single channel to single router | EventDemuxer fan-out | Medium |
| State store | Shared, queries unscoped | Add project_id to all queries | Medium |
| DB schema | ticket_id PK, pr_number PK | Composite PKs with project_id | Medium |
| Shutdown manager | Flat errgroup | Two-tier: infra errgroup + per-project WaitGroup | Medium |
| Slot management | Already per-project in DB | Just wire per-project config | Low |

## Domain Research Findings

### TOML Config
- BurntSushi/toml already in go.mod — no new dependency
- `[[project]]` array-of-tables maps to `[]ProjectConfig` slice
- `time.Duration` fields decode from strings like `"30s"` natively

### CLI Flag Coexistence
- Config file owns project definitions; CLI flags override globals only
- Use `c.IsSet()` to check if flag was explicitly set (respects env vars)
- Secrets: global fallback pattern — project inherits from `[global]` if omitted

### Goroutine Lifecycle
- Two-tier architecture recommended:
  - Top-level errgroup for infrastructure (callback server) — if infra dies, everything stops
  - Per-project `sync.WaitGroup` + restart loop — one watcher failure doesn't kill siblings
  - `runWithRestart` pattern with exponential backoff for watcher recovery

### SQLite Composite PK Migration
- Requires table recreation (ALTER TABLE can't change PKs in SQLite)
- Standard pattern: create new table → copy data → drop old → rename
- PRAGMA foreign_keys must be handled outside transaction

### Callback Routing
- Recommended: URL path prefix `/project/{projectID}/{ticketID}/...`
- EventDemuxer pattern: O(1) routing to per-project channels
- Backward compatibility possible by supporting both old and new URL patterns

## Open Questions Identified

1. **Credential scoping**: Ticket says single credential set. But what if projects span different GitHub orgs? Should we support per-project tokens from the start?
2. **PR number collisions**: Comment watermarks keyed by pr_number alone. Different GitHub repos can have the same PR number. Composite key `(project_id, pr_number)` needed.
3. **Ticket ID uniqueness**: Callback routing relies on ticket_id being globally unique. True within a Linear workspace, but what if the orchestrator watches projects across workspaces?
4. **Migration of existing data**: Migration 002 set `project_id = ''` for existing rows. The composite PK migration needs a strategy for backfilling the correct project_id.
5. **Error isolation**: If a project's GitHub token is revoked, should just that project's watchers stop, or should it be flagged and retried?
