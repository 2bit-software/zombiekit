---
status: complete
updated: 2026-03-29
---

# Research: Watcher 3 — PR Lifecycle Detection and Cleanup

## Executive Summary

Watcher 3 fills the final gap in the orchestrator's PR lifecycle: detecting when PRs are merged or closed and cleaning up associated resources. The codebase already provides all building blocks — `ListOpenPRs`, `IsMerged`, `IsClosed`, `DeleteWorktree`, `CleanBranch`, `ReleaseSlot`, and `SetTicketStatus`. The implementation follows established watcher patterns (ticker-based polling, context cancellation, log-and-continue error handling).

## Findings

### Codebase Context

**Existing Watcher Pattern**
All watchers follow the same structure (`watcher_linear.go`, `watcher_comment.go`):
- `New*()` returns `shutdown.ServiceFunc`
- Internal ticker-based loop with `select` on `ctx.Done()` and `ticker.C`
- Poll function called each tick, errors logged but don't halt the watcher
- Registered in `orchestrator.go` and passed to `shutdown.Manager.Run()`

**Watcher 3 Stub Already Exists**
- `internal/orchestrator/watchers.go:21-29` contains `NewWatcherStub(WatcherPRWatcher, ...)` — a no-op placeholder
- `WatcherPRWatcher` constant defined at `watchers.go:14`
- Already wired into `orchestrator.go` service list

**Available Interfaces**
| Interface | Method | Purpose for W3 |
|-----------|--------|----------------|
| `github.Client` | `ListOpenPRs(ctx, label)` | Find orchestrator-owned PRs |
| `github.Client` | `IsMerged(ctx, prNumber)` | Detect merged PRs |
| `github.Client` | `IsClosed(ctx, prNumber)` | Detect closed-without-merge PRs |
| `worktree.Manager` | `DeleteWorktree(ctx, path)` | Remove git worktree |
| `worktree.Manager` | `CleanBranch(ctx, branch)` | Delete local branch |
| `linear.Client` | `SetTicketStatus(ctx, id, status)` | Update Linear ticket |
| `state.StateStore` | `SetJobStatus(ctx, ticketID, status)` | Update job state |
| `state.StateStore` | `ReleaseSlot(ctx, projectID)` | Free concurrency slot |
| `state.StateStore` | `GetJobByPR(ctx, prNumber)` | Look up job from PR number |

**Job Status Flow (Verified from router.go)**
- `CreateJob` -> `StatusInProgress` (set by Watcher 1)
- `handleComplete` -> job stays `StatusInProgress`, `PRNumber` set via `SetPR`, slot NOT released
- `handleFailed` -> `StatusNeedsAttention`, slot released
- Watcher 3 transitions `StatusInProgress` (with PR) -> `StatusClosed` and releases slot

Critical finding: The callback router's `handleComplete` does NOT call `SetJobStatus(StatusComplete)` and does NOT call `ReleaseSlot`. The job remains `in-progress` with the PR number populated. The concurrency slot from the initial session stays held until Watcher 3 releases it.

**Coordination with Watcher 2**
The ticket notes: "Watcher 3 and Watcher 2 both poll the same PR list — coordinate via state: check job status in state store before processing; if job is already terminal, skip it."
- Watcher 2 already checks terminal states before processing comments (`watcher_comment.go:76-79`)
- Watcher 2 aborts on `IsMerged`/`IsClosed` during comment processing but does NOT perform cleanup — that's Watcher 3's job
- Both watchers may detect merge simultaneously — operations are idempotent (`SetJobStatus` and `ReleaseSlot` clamp safely)

**PR Lifecycle Gap**
`ListOpenPRs` only returns open PRs. Merged/closed PRs disappear. Watcher 3 must detect transitions via the StateStore: query `StatusInProgress` jobs with `PRNumber != nil` and check each PR's GitHub status.

**Branch Cleanup Limitation**
`DeleteWorktree` internally resolves the branch from the worktree and deletes both. If the worktree is already gone, `resolveBranch()` (unexported method) also fails. `CleanBranch(ctx, branch)` requires a branch name, but the `Job` struct does not store the branch name. When the worktree is manually deleted, the branch may be orphaned. A future improvement could store the branch name in the Job struct.

### Domain Knowledge

**Cleanup Ordering**
Resource cleanup should be resilient to partial failures. If worktree deletion fails (already deleted manually), the remaining steps (ticket status, slot release) should still proceed. This matches the existing rollback pattern in Watcher 1 (best-effort, log-and-continue).

**GitHub API Behavior**
- `IsMerged` and `IsClosed` work on any PR number regardless of current state
- A merged PR has both `IsMerged=true` and `IsClosed=true`
- Order of checks: `IsMerged` first, then `IsClosed` (merged implies closed)

**State-Based Detection vs Event-Based**
- Event-based (webhooks): Lower latency, but out of scope per context brief
- State-based (polling): Consistent with existing architecture, simpler to implement
- Recommended: Poll jobs with `in-progress` or `complete` status that have a PR number, then check each PR's merged/closed state

## Decision Points

- [x] **D1**: Detection strategy — Poll known jobs with PR numbers and check their GitHub status vs. compare `ListOpenPRs` against tracked jobs
  - **Decision**: Query `ListJobsByStatus(StatusInProgress)`, filter for `PRNumber != nil`, then check `IsMerged`/`IsClosed`. This avoids the race where a merged PR disappears from `ListOpenPRs`.

- [x] **D2**: Closed-without-merge ticket status — What Linear status should a closed-without-merge ticket get?
  - **Decision**: Configurable via `ClosedPRTicketStatus` config field. Default: "cancelled".

- [x] **D3**: Should Watcher 3 remove the tracking label from closed/merged PRs?
  - **Decision**: No. The label is inert on merged/closed PRs. `ListOpenPRs` only returns open PRs, so stale labels don't pollute results.

- [x] **D4**: Should cleanup include killing any active cmux session?
  - **Decision**: No. Watcher 3 only processes jobs where `PRNumber != nil` (PR already created). If a session is active during comment processing, Watcher 2 handles the session kill. Watcher 3 handles resource cleanup after the session ends.

## Recommendations

1. **Detection via StateStore query**: Add `ListJobsWithPR(ctx) ([]Job, error)` or reuse `ListJobsByStatus` with `StatusComplete`/`StatusInProgress` to find jobs that have PR numbers, then check each PR's GitHub status
2. **Best-effort cleanup**: Log and continue on individual step failures (worktree deletion, branch cleanup) — always attempt ticket status update and slot release
3. **Configurable closed-without-merge behavior**: Add `ClosedPRTicketStatus` config field (default: "cancelled")
4. **Skip jobs Watcher 2 is actively processing**: Check job status in state store — if `in-progress`, the session or comment watcher is still active. Only clean up jobs in `complete` status (router already set this) or detect the PR state change for jobs that somehow got stuck.

## Sources

- Linear ticket DEV-201: Watcher 3 scope and acceptance criteria
- Agent Context Brief: Architecture constraints and component boundaries
- Existing code: `watcher_linear.go`, `watcher_comment.go`, `router.go`, `watchers.go`
- Previous specs: DEV-200 (Watcher 1), DEV-203 (Watcher 2), DEV-202 (Router)
