# Technical Spec: Orchestrator Admin CLI

## New Files

| File | Purpose |
|------|---------|
| `internal/state/migrations/002_add_project_id.sql` | Schema migration |
| `internal/admin/service.go` | Admin service layer |
| `internal/admin/service_test.go` | Integration tests |
| `cmd/orchestrator/admin.go` | CLI subcommand handlers + output formatting |

## Modified Files

| File | Change |
|------|--------|
| `internal/state/store.go` | Add `ConcurrencySlot` type, `Job.ProjectID` field, `ValidStatuses`, 3 new interface methods + SQLite implementations |
| `internal/state/store_test.go` | Tests for new store methods |
| `cmd/orchestrator/main.go` | Restructure into subcommands: `run`, `jobs`, `slots` |
| `internal/orchestrator/watcher_linear.go` | Pass `o.cfg.ProjectID` to `CreateJob` |
| `internal/orchestrator/watcher_linear_test.go` | Update `stubState.CreateJob` signature |
| `internal/orchestrator/watcher_pr_test.go` | Update `prStubState.CreateJob` signature |
| `internal/orchestrator/orchestrator_test.go` | Update stub if present |

## Type Definitions

### state package additions

```go
// store.go

var ValidStatuses = []string{
    StatusQueued, StatusInProgress, StatusNeedsAttention, StatusComplete, StatusClosed,
}

type ConcurrencySlot struct {
    ProjectID   string
    ActiveCount int
    SlotLimit   int
}

// Job struct — add field:
type Job struct {
    TicketID     string
    WorktreePath string
    CmuxSession  string
    PRNumber     *int64
    Status       string
    ProjectID    string    // NEW
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### StateStore interface additions

```go
ListAllJobs(ctx context.Context) ([]Job, error)
DeleteJob(ctx context.Context, ticketID string) error
ListSlots(ctx context.Context) ([]ConcurrencySlot, error)
```

### CreateJob signature change

```go
// Before:
CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
// After:
CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession, projectID string) error
```

### admin package

```go
// service.go

type Service struct {
    store state.StateStore
}

func New(store state.StateStore) *Service

type JobFilter struct {
    Statuses []string
}

type DeleteResult struct {
    Job          state.Job
    SlotReleased bool
}

func (s *Service) ListJobs(ctx context.Context, filter JobFilter) ([]state.Job, error)
func (s *Service) GetJob(ctx context.Context, ticketID string) (*state.Job, error)
func (s *Service) DeleteJob(ctx context.Context, ticketID string) (*DeleteResult, error)
func (s *Service) SetJobStatus(ctx context.Context, ticketID, status string) error
func (s *Service) ListSlots(ctx context.Context) ([]state.ConcurrencySlot, error)
func (s *Service) ResetSlots(ctx context.Context) (int, error)
```

## Migration SQL

```sql
-- 002_add_project_id.sql
ALTER TABLE jobs ADD COLUMN project_id TEXT NOT NULL DEFAULT '';
```

## CLI Subcommand Tree

```
orchestrator
├── run          [all 17 daemon flags]
├── jobs
│   ├── list     [--status <s>...]
│   ├── get      <ticket-id>
│   ├── delete   <ticket-id>
│   └── set-status <ticket-id> <status>
└── slots
    ├── list
    └── reset
```

**Global flag**: `--db-path` / `ORCH_DB_PATH` (not marked `Required` on the flag to avoid blocking `--help`; validated in `openStore` helper). Admin subcommands require the DB file to exist; the `run` subcommand creates it if missing.

## Output Examples

### `orchestrator jobs list`

```
TICKET      STATUS           PROJECT     PR    UPDATED
DEV-226     queued           707aafa8    -     2026-03-30T10:15:00-07:00
DEV-227     in-progress      707aafa8    42    2026-03-30T09:30:00-07:00
```

Project ID truncated to first 8 chars for readability. PR shown as `-` when nil.

### `orchestrator jobs get DEV-226`

```
Ticket:     DEV-226
Status:     queued
Project:    707aafa8-0cfc-4e49-9ec2-112a0328dee6
Worktree:   /Users/morgan/.claude/worktrees/DEV-226
Session:    workspace:23
PR:         -
Created:    2026-03-29T21:16:44-07:00
Updated:    2026-03-29T21:16:44-07:00
```

Full project ID in detail view.

### `orchestrator jobs delete DEV-226`

```
Deleted job DEV-226 (was: queued, project: 707aafa8)
```

If slot released: `Deleted job DEV-226 (was: queued, project: 707aafa8, slot released)`

### `orchestrator jobs set-status DEV-226 needs-attention`

```
Updated DEV-226 status: queued -> needs-attention
```

### `orchestrator slots list`

```
PROJECT                                   ACTIVE    LIMIT
707aafa8-0cfc-4e49-9ec2-112a0328dee6     1         1
```

### `orchestrator slots reset`

```
Reset 1 project slot(s) to 0
```

Or if nothing to reset: `No slots to reset`

### Error output

```
Error: job not found: DEV-999
```

```
Error: invalid status "banana" (valid: queued, in-progress, needs-attention, complete, closed)
```

Exit code 1 for all errors.

## FR Traceability

| FR | Implementation |
|----|----------------|
| FR-001 | `admin.Service.ListJobs` + `jobsList` handler |
| FR-002 | `JobFilter.Statuses` + `StringSliceFlag` in CLI |
| FR-003 | `admin.Service.GetJob` + `jobsGet` handler |
| FR-004 | `admin.Service.DeleteJob` (compound: get + delete + release slot) |
| FR-005 | `admin.Service.SetJobStatus` with `ValidStatuses` check |
| FR-006 | `admin.Service.ListSlots` + `slotsList` handler |
| FR-007 | `admin.Service.ResetSlots` + `slotsReset` handler |
| FR-008 | Global `--db-path` flag on `cli.App` |
| FR-009 | `cli.Exit(msg, 1)` for errors, normal return for success |
| FR-010 | Confirmation messages in handler functions |
| FR-011 | `internal/admin` package — no CLI dependency |
| FR-012 | `runCommand()` subcommand, bare `orchestrator` shows help |
| FR-013 | `002_add_project_id.sql` + `CreateJob` signature change |
