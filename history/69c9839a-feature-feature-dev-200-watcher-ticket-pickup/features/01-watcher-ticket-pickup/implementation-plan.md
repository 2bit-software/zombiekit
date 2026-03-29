# Implementation Plan: Watcher 1 — Ready Ticket Pickup

## Overview

Replace the `NewWatcherStub(WatcherLinearPoller, ...)` in `orchestrator.Run()` with a real polling watcher that picks up "ai-ready" tickets, creates worktrees, spawns agent sessions, and manages the full lifecycle with rollback on failure.

**No spikes needed**: All interfaces are implemented and tested. This is pure coordination logic.

## Implementation Steps

### Step 1: Config — Add `ProjectID` and `RepoDir`

**Files**: `internal/orchestrator/config.go`, `cmd/orchestrator/main.go`
**FRs**: FR-002 (projectID for slot acquisition), FR-003 (repoDir for worktree manager)

Add two new fields to `Config`:

```go
ProjectID string  // Linear project identifier for slot scoping
RepoDir   string  // Git repo root for worktree creation
```

Add validation:
- `ProjectID` not empty
- `RepoDir` not empty, must exist and contain a `.git` directory

Add CLI flags in `main.go`:
- `--project-id` / `ORCH_PROJECT_ID`
- `--repo-dir` / `ORCH_REPO_DIR`

**Verification**: Existing config tests pass + new validation tests for the two fields.

---

### Step 2: Orchestrator — Add dependencies to struct

**Files**: `internal/orchestrator/orchestrator.go`
**FRs**: All (dependencies are needed by the watcher)

Extend the `Orchestrator` struct:

```go
type Orchestrator struct {
    cfg      *Config
    store    state.StateStore
    linear   linear.Client
    worktrees worktree.Manager
    sessions cmux.SessionManager
}
```

Update `New()`:

```go
func New(cfg *Config, store state.StateStore, linear linear.Client, wt worktree.Manager, sess cmux.SessionManager) *Orchestrator
```

**Verification**: Existing tests compile (update constructor calls in test files).

---

### Step 3: Watcher — Implement `NewLinearPoller`

**Files**: New file `internal/orchestrator/watcher_linear.go`
**FRs**: FR-001 through FR-014 (this is the core)

Create a function that returns a `shutdown.ServiceFunc`:

```go
func (o *Orchestrator) NewLinearPoller() shutdown.ServiceFunc
```

**Poll loop structure**:
```
ticker := time.NewTicker(o.cfg.PollInterval)
for {
    select {
    case <-ctx.Done():
        return nil
    case <-ticker.C:
        o.pollAndProcess(ctx)
    }
}
```

**`pollAndProcess(ctx)` logic**:
1. Call `o.linear.PollReadyTickets(ctx, "ai-ready")`
2. On error: log, return (retry next tick)
3. For each ticket:
   - Check `ctx.Err()` — if cancelled, stop processing more tickets (FR-011)
   - Call `o.processTicket(ctx, ticket)`
   - On error: log, continue to next ticket

**`processTicket(ctx, ticket)` logic** — the pipeline:
1. Check for existing job: `o.store.GetJob(ctx, ticket.Identifier)` — skip if exists (FR-012)
2. Acquire slot: `o.store.TryAcquireSlot(ctx, o.cfg.ProjectID, o.cfg.ConcurrencyLimit)` — skip if false (FR-002)
3. Create worktree: `o.worktrees.CreateWorktree(ctx, ticket.Identifier, shortTitle)` (FR-003)
4. Write ticket file: `os.MkdirAll(worktreePath+"/.ai", 0o755)` + `os.WriteFile(worktreePath+"/.ai/ticket.md", ...)` (FR-004)
5. Build env map with `WORK_CALLBACK_URL` (FR-005)
6. Spawn session: `o.sessions.SpawnSession(ctx, ticket.Identifier, ticket.Title, worktreePath, env)` (FR-006)
7. Create job: `o.store.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef)` (FR-007)
8. Set ticket status: `o.linear.SetTicketStatus(ctx, ticket.ID, "In Progress")` — log on error, continue (FR-008, FR-014)
9. Remove label: `o.linear.RemoveLabel(ctx, ticket.ID, "ai-ready")` — log on error, continue (FR-009, FR-014)

**Rollback on failure** (steps 3-7):
- After step 7 fails: kill session + delete worktree + release slot
- After step 6 fails: delete worktree + release slot
- After step 5 fails: delete worktree + release slot
- After step 4 fails: delete worktree + release slot
- After step 3 fails: release slot
- After step 2 fails: nothing (normal deferral)

**After rollback**: Apply "needs-attention" label, remove "ai-ready" label (FR-013). Log if these Linear calls fail.

Use deferred cleanup pattern:
```go
slotAcquired := false
var worktreePath string
var sessionRef string
cleanup := func() { /* reverse-order cleanup */ }
defer func() {
    if err != nil { cleanup() }
}()
```

**Verification**: Tests in Step 4.

---

### Step 4: Tests — Integration tests with test doubles

**Files**: New file `internal/orchestrator/watcher_linear_test.go`
**FRs**: All

Create in-memory test doubles:

```go
type mockLinearClient struct {
    tickets      []linear.Ticket
    pollErr      error
    setStatusErr error
    removeLabelErr error
    applyLabelErr  error
    calls        []string  // record call order
}

type mockWorktreeManager struct {
    createErr error
    deleteErr error
    paths     map[string]string  // ticketID -> path
    calls     []string
}

type mockSessionManager struct {
    spawnErr error
    killErr  error
    calls    []string
}

// state.StateStore — use the real SQLiteStore with in-memory DB, or a mock
```

**Test cases** (mapped to FRs):

| Test | FR | Description |
|------|-----|-------------|
| TestLinearPoller_SingleTicket | FR-001,003-009 | Happy path: one ticket → full pipeline executes in order |
| TestLinearPoller_ConcurrencyLimit | FR-002 | limit=1, 2 tickets → only 1 processed |
| TestLinearPoller_TicketFileWritten | FR-004 | Verify `.ai/ticket.md` content matches ticket description |
| TestLinearPoller_CallbackURL | FR-005 | Verify env map has correct WORK_CALLBACK_URL |
| TestLinearPoller_RollbackOnSpawnFailure | FR-010 | SpawnSession errors → DeleteWorktree + ReleaseSlot called |
| TestLinearPoller_RollbackOnCreateJobFailure | FR-010 | CreateJob errors → KillSession + DeleteWorktree + ReleaseSlot |
| TestLinearPoller_GracefulShutdown | FR-011 | Cancel context → current ticket finishes, no new work |
| TestLinearPoller_SkipExistingJob | FR-012 | GetJob returns existing → ticket skipped |
| TestLinearPoller_NeedsAttentionOnFailure | FR-013 | Pipeline failure → "needs-attention" label applied |
| TestLinearPoller_LinearFailureAfterJob | FR-014 | SetTicketStatus errors → logged, job continues |
| TestLinearPoller_ConcurrencyMultiPoll | FR-002 | limit=1, 2 tickets: first poll processes 1, release slot, second poll processes the other |
| TestLinearPoller_RollbackOnWorktreeFailure | FR-010 | CreateWorktree errors → ReleaseSlot called, no DeleteWorktree |
| TestLinearPoller_ShutdownBetweenPolls | FR-011 | Cancel context between ticks → watcher exits without processing |
| TestLinearPoller_EmptyPoll | edge | No tickets → no downstream calls |
| TestLinearPoller_PollError | edge | PollReadyTickets errors → logged, retry next tick |
| TestLinearPoller_EmptyDescription | edge | Ticket with empty description → `.ai/ticket.md` created with empty content |
| TestLinearPoller_RemoveLabelFailureAfterJob | FR-014 | RemoveLabel errors after job created → logged, job continues |

**Verification**: All tests pass.

---

### Step 5: Wire — Connect in `main.go` and `Run()`

**Files**: `cmd/orchestrator/main.go`, `internal/orchestrator/orchestrator.go`
**FRs**: All (integration point)

In `main.go`, after store creation:
```go
linearClient, err := linear.NewClient(cfg.LinearAPIKey)
worktreeMgr, err := worktree.New(cfg.RepoDir, worktree.WithWorktreesRoot(cfg.WorktreesRoot))
sessionMgr, err := cmux.New()
orch := orchestrator.New(cfg, store, linearClient, worktreeMgr, sessionMgr)
```

In `orchestrator.Run()`, replace:
```go
linearPoller := NewWatcherStub(WatcherLinearPoller, o.cfg.PollInterval)
```
with:
```go
linearPoller := o.NewLinearPoller()
```

Keep the other two watcher stubs (PR watcher, comment watcher) — they're separate tickets.

**Verification**: `go build ./cmd/orchestrator` succeeds. Manual smoke test with a real Linear ticket if possible.

## Dependency Graph

```
Step 1 (Config)
    ↓
Step 2 (Orchestrator struct)
    ↓
Step 3 (Watcher implementation)
    ↓
Step 4 (Tests)  ←→  Step 5 (Wiring)
```

Steps 4 and 5 are independent of each other but both depend on Step 3.

## FR Traceability

| FR | Step |
|----|------|
| FR-001 | Step 3 (poll loop) |
| FR-002 | Step 1 (ProjectID config) + Step 3 (TryAcquireSlot call) |
| FR-003 | Step 1 (RepoDir config) + Step 3 (CreateWorktree call) + Step 5 (worktree.New wiring) |
| FR-004 | Step 3 (file write) |
| FR-005 | Step 3 (env map construction) |
| FR-006 | Step 3 (SpawnSession call) |
| FR-007 | Step 3 (CreateJob call) |
| FR-008 | Step 3 (SetTicketStatus call) |
| FR-009 | Step 3 (RemoveLabel call) |
| FR-010 | Step 3 (rollback logic) |
| FR-011 | Step 3 (context cancellation in loop) |
| FR-012 | Step 3 (GetJob check) |
| FR-013 | Step 3 (needs-attention label on failure) |
| FR-014 | Step 3 (log-and-continue on Linear failure) |

## Remaining Uncertainties

None. All interfaces verified, all decisions resolved.
