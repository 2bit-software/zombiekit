# Technical Requirements Research: Watcher 3 — PR Lifecycle Detection and Cleanup

## Implementation Hints from Ticket

### Detection Strategy

The ticket specifies polling `GitHubClient.ListOpenPRs` for orchestrator-owned PRs. However, merged/closed PRs won't appear in this list. The actual strategy should be:

1. Query StateStore for jobs with a PR number that are NOT in terminal state (`StatusComplete`, `StatusClosed`, `StatusNeedsAttention`)
2. For each such job, call `IsMerged(ctx, prNumber)` and `IsClosed(ctx, prNumber)`
3. Act on the detected transition

Alternative: Track "known open PRs" in memory and detect disappearances from `ListOpenPRs`. This is fragile (state lost on restart) and doesn't distinguish merge from close.

### Cleanup Pipeline

Per ticket, the cleanup sequence for both merge and close-without-merge:

```
1. DeleteWorktree(ctx, job.WorktreePath)
2. CleanBranch(ctx, branch)           // branch derived from worktree
3. SetTicketStatus(ctx, ticketID, status)
4. SetJobStatus(ctx, ticketID, terminalStatus)
5. ReleaseSlot(ctx, projectID)
```

Each step must be independent — failure of step 1 should not prevent steps 3-5.

### Coordination with Watcher 2

From ticket: "check job status in state store before processing; if job is already terminal, skip it."

- Watcher 2 already handles merge/close detection during active comment processing and kills the session
- Watcher 3's role is cleanup AFTER the session is done — it acts on jobs that are already in a terminal or near-terminal state
- Race condition: Both watchers poll the same PRs. Watcher 3 should only act when the job is NOT `in-progress` (active session running). If `in-progress`, the session/Watcher 2 will handle the transition.

### Config Additions

- `ClosedPRTicketStatus string` — Linear status for PRs closed without merge (default: "cancelled")
- Possibly: `MergedPRTicketStatus string` — though ticket says "done" which is likely standard

### Target File

Implementation goes in `internal/orchestrator/watcher_pr.go`, replacing the `NewWatcherStub(WatcherPRWatcher, ...)` call in `orchestrator.go`.

### Testing Pattern

Follow the same pattern as `watcher_linear_test.go` and `watcher_comment_test.go`:
- Stub implementations of `github.Client`, `worktree.Manager`, `linear.Client`, `state.StateStore`
- Test the polling + cleanup logic
- Test partial failure scenarios (worktree already deleted, etc.)
- Test graceful shutdown via context cancellation

### StateStore Query

May need a new query method: `ListJobsWithPR(ctx) ([]Job, error)` — returns all jobs that have a non-null PR number and are not in terminal state. This avoids loading ALL jobs. Alternatively, `ListJobsByStatus` already exists and can filter by `StatusInProgress` and `StatusComplete` — then filter in Go for jobs with PR numbers.

Recommendation: Use existing `ListJobsByStatus` for now. Add a dedicated method only if performance requires it.
