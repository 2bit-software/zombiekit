# Technical Spec: Watcher 3 — PR Lifecycle Detection and Cleanup

**Created**: 2026-03-29
**Traces to**: spec.md (business specification)

## Overview

Watcher 3 is a ticker-based polling loop that detects merged/closed PRs and performs resource cleanup. It follows the established watcher pattern from `watcher_linear.go` and `watcher_comment.go`.

## File Layout

| File | Purpose |
|------|---------|
| `internal/orchestrator/watcher_pr.go` | Watcher implementation |
| `internal/orchestrator/watcher_pr_test.go` | Integration tests |
| `internal/orchestrator/config.go` | Add `ClosedPRTicketStatus` field |
| `cmd/orchestrator/main.go` | Add `--closed-pr-status` CLI flag |
| `internal/orchestrator/orchestrator.go` | Replace stub with real constructor |

## Implementation Design

### Constructor

```go
// NewPRWatcher returns a ServiceFunc that polls for merged/closed PRs
// and performs cleanup (worktree deletion, ticket status, slot release).
func (o *Orchestrator) NewPRWatcher() shutdown.ServiceFunc {
    return func(ctx context.Context) error {
        logger := logging.Logger().With(slog.String("watcher", WatcherPRWatcher))
        logger.Info("pr watcher started", slog.Duration("poll_interval", o.cfg.PollInterval))

        ticker := time.NewTicker(o.cfg.PollInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                logger.Info("pr watcher stopping")
                return nil
            case <-ticker.C:
                o.pollPRLifecycle(ctx, logger)
            }
        }
    }
}
```

### Poll Function

```go
func (o *Orchestrator) pollPRLifecycle(ctx context.Context, logger *slog.Logger) {
    // FR-001: Query queued jobs with PR numbers
    jobs, err := o.store.ListJobsByStatus(ctx, state.StatusQueued)
    if err != nil {
        logger.Error("failed to list jobs", slog.String("err", err.Error()))
        return
    }

    for _, job := range jobs {
        if ctx.Err() != nil {
            return
        }

        // FR-007: Skip jobs without a PR (agent still working)
        if job.PRNumber == nil {
            continue
        }

        prNumber := int(*job.PRNumber)

        // FR-002: Check merge status first, then closed
        merged, err := o.github.IsMerged(ctx, prNumber)
        if err != nil {
            logger.Error("failed to check merge status",
                slog.String("ticket", job.TicketID),
                slog.Int("pr", prNumber),
                slog.String("err", err.Error()))
            continue
        }
        if merged {
            o.cleanupPR(ctx, job, "done", logger)
            continue
        }

        closed, err := o.github.IsClosed(ctx, prNumber)
        if err != nil {
            logger.Error("failed to check closed status",
                slog.String("ticket", job.TicketID),
                slog.Int("pr", prNumber),
                slog.String("err", err.Error()))
            continue
        }
        if closed {
            o.cleanupPR(ctx, job, o.cfg.ClosedPRTicketStatus, logger)
            continue
        }

        // PR still open — no action
    }
}
```

### Cleanup Pipeline

```go
// cleanupPR performs best-effort cleanup for a merged or closed PR.
// Each step is independent — failure at any step does not prevent
// subsequent steps from executing. (FR-003, FR-004, FR-005)
func (o *Orchestrator) cleanupPR(ctx context.Context, job state.Job, ticketStatus string, logger *slog.Logger) {
    logger = logger.With(
        slog.String("ticket", job.TicketID),
        slog.Int64("pr", *job.PRNumber),
        slog.String("ticket_status", ticketStatus),
    )
    logger.Info("cleaning up PR")

    // Step 1: Delete worktree (also deletes branch internally)
    if err := o.worktrees.DeleteWorktree(ctx, job.WorktreePath); err != nil {
        logger.Error("failed to delete worktree",
            slog.String("path", job.WorktreePath),
            slog.String("err", err.Error()))
    }

    // Step 2: Update Linear ticket status
    if err := o.linear.SetTicketStatus(ctx, job.TicketID, ticketStatus); err != nil {
        logger.Error("failed to set ticket status", slog.String("err", err.Error()))
    }

    // Step 3: Mark job as closed
    if err := o.store.SetJobStatus(ctx, job.TicketID, state.StatusClosed); err != nil {
        logger.Error("failed to set job status", slog.String("err", err.Error()))
    }

    // Step 4: Release concurrency slot
    if err := o.store.ReleaseSlot(ctx, o.cfg.ProjectID); err != nil {
        logger.Error("failed to release slot", slog.String("err", err.Error()))
    }

    logger.Info("PR cleanup complete")
}
```

### Config Changes

```go
// In config.go — add to Config struct:
ClosedPRTicketStatus string

// In NewConfig() — add parsing:
ClosedPRTicketStatus: c.String("closed-pr-status"),

// In cmd/orchestrator/main.go — add flag:
&cli.StringFlag{
    Name:    "closed-pr-status",
    Usage:   "Linear ticket status for PRs closed without merge",
    Value:   "cancelled",
    EnvVars: []string{"ORCH_CLOSED_PR_STATUS"},
},
```

### Wiring Change

```go
// In orchestrator.go Run() — replace:
prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)

// With:
prWatcher := o.NewPRWatcher()
```

## Test Design

### Test Doubles

```go
type stubWorktree struct {
    deleteErr error
    calls     []string
}

func (s *stubWorktree) CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error) {
    return "", nil // unused by Watcher 3
}

func (s *stubWorktree) DeleteWorktree(ctx context.Context, path string) error {
    s.calls = append(s.calls, "delete:"+path)
    return s.deleteErr
}

func (s *stubWorktree) CleanBranch(ctx context.Context, branch string) error {
    return nil // unused by Watcher 3
}
```

Other stubs follow the same pattern as `watcher_linear_test.go` and `watcher_comment_test.go`.

### Test Cases

| Test | FR | What it validates |
|------|-----|-------------------|
| `TestPRWatcher_MergedPR` | FR-001,002,003 | Full merge cleanup pipeline |
| `TestPRWatcher_ClosedPR` | FR-001,002,004 | Full close cleanup pipeline with configurable status |
| `TestPRWatcher_SkipNoPR` | FR-007 | Jobs without PR number are skipped |
| `TestPRWatcher_SkipClosed` | FR-006 | Jobs in StatusClosed are skipped (not returned by query) |
| `TestPRWatcher_PartialFailure_Worktree` | FR-005 | DeleteWorktree fails, remaining steps proceed |
| `TestPRWatcher_PartialFailure_Linear` | FR-005 | SetTicketStatus fails, remaining steps proceed |
| `TestPRWatcher_PartialFailure_SetStatus` | FR-005 | SetJobStatus fails, ReleaseSlot still called |
| `TestPRWatcher_ContextCancelled` | FR-008 | Exits cleanly on context cancellation |
| `TestPRWatcher_MultiplePRs` | FR-001 | Multiple PRs cleaned up in one cycle |
| `TestPRWatcher_OpenPR` | FR-002 | PR still open, no cleanup |
| `TestPRWatcher_Idempotent` | FR-010 | Second poll on cleaned job produces no side effects |

### Test Helper

```go
func buildPRWatcherOrch(t *testing.T, opts ...func(*testDeps)) *Orchestrator {
    t.Helper()
    // Creates orchestrator with configurable test doubles
    // Returns orchestrator with pollPRLifecycle accessible
}
```

## Dependencies

All dependencies are existing interfaces — no new interfaces or methods needed.

| Dependency | Interface | Methods Used |
|-----------|-----------|--------------|
| State store | `state.StateStore` | `ListJobsByStatus`, `SetJobStatus`, `ReleaseSlot` |
| GitHub | `github.Client` | `IsMerged`, `IsClosed` |
| Linear | `linear.Client` | `SetTicketStatus` |
| Worktree | `worktree.Manager` | `DeleteWorktree` |
| Config | `*Config` | `PollInterval`, `ProjectID`, `ClosedPRTicketStatus` |

## Key Assumptions

1. `CreateJob` sets initial status to `StatusQueued` — jobs stay in this status through the entire lifecycle until Watcher 3 or failure transitions them
2. The callback router's `handleComplete` sets `PRNumber` via `SetPR` but does NOT change job status or release the slot
3. `ListJobsByStatus(StatusQueued)` returns both new jobs (no PR yet) and PR-ready jobs — the `PRNumber != nil` filter distinguishes them
4. `ReleaseSlot` clamps to 0, making redundant calls safe
5. `SetJobStatus` is idempotent for the same status value
