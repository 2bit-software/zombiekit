# Technical Spec: Watcher 1 — Ready Ticket Pickup

## Architecture

The watcher is a **coordination layer** — it contains no domain logic, only orchestration of existing interfaces. It follows the existing `shutdown.ServiceFunc` pattern (`func(ctx context.Context) error`).

```
┌──────────────────────────────────────────────────┐
│                Orchestrator.Run()                 │
│                                                   │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐   │
│  │ Callback │  │  Linear  │  │  PR Watcher   │   │
│  │  Server  │  │  Poller  │  │  (stub)       │   │
│  └──────────┘  └────┬─────┘  └───────────────┘   │
│                     │                              │
│         ┌───────────┼───────────┐                  │
│         ▼           ▼           ▼                  │
│   linear.Client  worktree.Mgr  cmux.SessionMgr    │
│         │           │           │                  │
│         ▼           ▼           ▼                  │
│   state.StateStore (shared)                        │
└──────────────────────────────────────────────────┘
```

## Data Flow

### Happy Path

```
[Timer tick]
    │
    ▼
PollReadyTickets("ai-ready")
    │
    ▼ (for each ticket)
GetJob(ticket.Identifier) ──exists──▶ skip
    │ not found
    ▼
TryAcquireSlot(projectID, limit) ──false──▶ skip
    │ true
    ▼
CreateWorktree(ticket.Identifier, shortTitle)
    │
    ▼
Write .ai/ticket.md
    │
    ▼
SpawnSession(ticket.Identifier, title, path, env)
    │
    ▼
CreateJob(ticket.Identifier, path, sessionRef)
    │
    ▼
SetTicketStatus(ticket.ID, "In Progress")  ←── log on error, continue
    │
    ▼
RemoveLabel(ticket.ID, "ai-ready")  ←── log on error, continue
    │
    ▼
[done, next ticket]
```

### Failure Path

```
[Any step 3-7 fails]
    │
    ▼
Reverse-order cleanup:
  - KillSession (if spawned)
  - DeleteWorktree (if created)
  - ReleaseSlot (if acquired)
    │
    ▼
ApplyLabel(ticket.ID, "needs-attention")  ←── log on error
RemoveLabel(ticket.ID, "ai-ready")        ←── log on error
    │
    ▼
Log error, continue to next ticket
```

## Code Structure

### New File: `internal/orchestrator/watcher_linear.go`

```go
package orchestrator

// NewLinearPoller returns a ServiceFunc that polls Linear for
// ai-ready tickets and processes them through the pickup pipeline.
func (o *Orchestrator) NewLinearPoller() shutdown.ServiceFunc

// pollAndProcess runs one poll cycle: fetch tickets, process each.
func (o *Orchestrator) pollAndProcess(ctx context.Context)

// processTicket runs the pickup pipeline for a single ticket.
// Returns an error if the pipeline fails (after rollback).
func (o *Orchestrator) processTicket(ctx context.Context, ticket linear.Ticket) error

// shortTitle derives a filesystem-safe short title from a ticket title.
// e.g., "Watcher 1 — ready ticket pickup" → "ready-ticket-pickup"
func shortTitle(title string) string
```

### Modified: `internal/orchestrator/config.go`

```go
type Config struct {
    // ... existing fields ...
    ProjectID string        // Linear project ID for slot scoping
    RepoDir   string        // Git repo root directory
}
```

### Modified: `internal/orchestrator/orchestrator.go`

```go
type Orchestrator struct {
    cfg       *Config
    store     state.StateStore
    linear    linear.Client
    worktrees worktree.Manager
    sessions  cmux.SessionManager
}

func New(cfg *Config, store state.StateStore, lc linear.Client, wt worktree.Manager, sm cmux.SessionManager) *Orchestrator
```

### Modified: `cmd/orchestrator/main.go`

New client construction between store creation and orchestrator instantiation.

## Environment Map

The session is spawned with this env map:

```go
env := map[string]string{
    "WORK_CALLBACK_URL": fmt.Sprintf("http://localhost:%d/%s", o.cfg.CallbackPort, ticket.Identifier),
}
```

Future watchers may add more env vars. The map is constructed fresh per ticket.

## Ticket File Format

Written to `{worktreePath}/.ai/ticket.md`:

```markdown
{ticket.Description}
```

Raw description content, no wrapping. The agent knows to look for this file by convention.

## Error Handling Strategy

| Error Source | Behavior | Rationale |
|-------------|----------|-----------|
| `PollReadyTickets` fails | Log, skip this cycle | Transient; retry next tick |
| `GetJob` fails | Log, skip ticket | Can't determine if duplicate |
| `TryAcquireSlot` returns false | Skip ticket silently | Normal deferral, not an error |
| `TryAcquireSlot` errors | Log, skip ticket | Can't safely proceed |
| Steps 3-7 fail | Rollback + needs-attention | Resource cleanup required |
| Steps 8-9 fail | Log, continue | Job is running, state store is truth |
| Rollback fails | Log each failure | Best-effort cleanup |
| Needs-attention labeling fails | Log | Nothing more we can do |

## Concurrency Model

- **Single goroutine**: The watcher runs in one goroutine. Tickets within a poll batch are processed sequentially.
- **No goroutine-per-ticket**: Unnecessary complexity. Poll batches are small (bounded by Linear rate limits and concurrency slots).
- **Context cancellation**: Checked between tickets, not mid-pipeline. A ticket pipeline runs to completion or failure once started.

## Testing Approach

Test doubles are method-recording mocks:

```go
type mockLinearClient struct {
    mu     sync.Mutex
    calls  []call       // ordered list of (method, args)
    // configurable return values per method
    pollTickets    []linear.Ticket
    pollErr        error
    setStatusErr   error
    removeLabelErr error
    applyLabelErr  error
}

type call struct {
    Method string
    Args   []interface{}
}
```

Tests verify:
1. **Call order** — correct pipeline sequence
2. **Call args** — correct field usage (Identifier vs ID)
3. **Rollback calls** — cleanup happens in reverse order
4. **No unexpected calls** — e.g., SetTicketStatus NOT called after rollback
