# Feature Specification: Watcher 3 — PR Lifecycle Detection and Cleanup

**Feature Branch**: `morganhein/dev-201-watcher-3-pr-lifecycle-detection-and-cleanup`
**Created**: 2026-03-29
**Status**: Draft (Audit R3 — corrected job status to StatusQueued)
**Input**: Linear ticket DEV-201

## Architecture Notes

### Detection Strategy

The ticket describes "polling `GitHubClient.ListOpenPRs` for orchestrator-owned PRs." This won't work directly because merged/closed PRs disappear from the open PR list. Instead, the detection is state-based:

1. Query `StateStore.ListJobsByStatus(ctx, StatusQueued)` to find active jobs
2. Filter in-memory for jobs where `PRNumber != nil` — this means the callback router has already processed `EventComplete` and created the PR via `SetPR`
3. For each such job, call `IsMerged(ctx, int(*job.PRNumber))` then `IsClosed(ctx, int(*job.PRNumber))`
4. If merged: run the merge cleanup pipeline
5. If closed without merge: run the close cleanup pipeline
6. If neither: skip (PR is still open, awaiting review)

Jobs where `PRNumber == nil` are still in the initial agent session (no PR created yet) and are skipped.

This approach survives restarts (state is in SQLite), correctly distinguishes merge from close, and avoids the race where a merged PR leaves the `ListOpenPRs` result set. `GetJobByPR` is NOT used — `ListJobsByStatus` + in-memory filter is the chosen query method.

### Cleanup Pipeline

Both merge and close-without-merge follow the same cleanup sequence, differing only in the Linear ticket status:

1. `DeleteWorktree(ctx, job.WorktreePath)` — removes the git worktree AND deletes the branch internally (the `DeleteWorktree` implementation resolves the branch from the worktree and runs `git branch -D`). If this fails (e.g., worktree already removed manually), the error is logged and cleanup continues. The branch may be orphaned in this case — see Known Limitations.
2. `LinearClient.SetTicketStatus(ctx, job.TicketID, status)` — "done" for merge, `ClosedPRTicketStatus` for close
3. `StateStore.SetJobStatus(ctx, job.TicketID, StatusClosed)` — marks the job as terminal
4. `StateStore.ReleaseSlot(ctx, projectID)` — releases the concurrency slot held since the initial session

Each step is independent. Failure at any step does not prevent subsequent steps from executing. All failures are logged.

### Job Status Lifecycle

The callback router's `handleComplete` does NOT set `StatusComplete` and does NOT release the concurrency slot. After `handleComplete` runs, the job remains `StatusQueued` with the `PRNumber` field populated. The slot from the initial session stays held throughout the PR lifecycle.

Watcher 3 is responsible for the final state transition:
- `StatusQueued` (with `PRNumber != nil`) -> Watcher 3 detects merged/closed -> `StatusClosed` (set by Watcher 3) + slot released
- `StatusClosed` -> skip (already cleaned up)
- `StatusNeedsAttention` -> skip (requires human triage)
- `StatusQueued` (with `PRNumber == nil`) -> skip (agent still working, no PR yet)

### Coordination with Watcher 2

Both watchers poll PRs. They coordinate through job status and PR number:

- **Job is `StatusQueued`, `PRNumber == nil`**: Agent is still working on the initial session. Skip.
- **Job is `StatusQueued`, `PRNumber != nil`**: PR has been created. Watcher 3 checks merged/closed. Watcher 2 may also be processing comments on the same PR — if the PR merges during an active comment session, Watcher 2 detects merge via `IsMerged` checks (in `runPRQueue`) and kills the session, but does NOT perform resource cleanup. Watcher 3 picks up cleanup on the next poll after the comment session ends.
- **Job is `StatusClosed` or `StatusNeedsAttention`**: Skip — already terminal.

Race safety: Both watchers may see the same PR in the same poll cycle. Watcher 2 checks `IsMerged`/`IsClosed` before dispatching each comment. Watcher 3 checks the same status to trigger cleanup. The `SetJobStatus(StatusClosed)` call is idempotent — if both attempt it, the second is a no-op. `ReleaseSlot` clamps to 0, so a redundant call is safe.

### Slot Lifecycle

The concurrency slot acquired by Watcher 1 before `SpawnSession` is held throughout the entire PR lifecycle:
1. **Watcher 1** acquires slot → spawns initial session
2. **Callback router** `handleComplete` → creates PR, does NOT release slot
3. **Watcher 2** acquires/releases *separate* slots for each comment session
4. **Watcher 3** releases the original slot from step 1 during cleanup

For the failure path: `handleFailed` releases the slot immediately (no PR created, nothing for Watcher 3 to clean up).

### Tracking Label

Watcher 3 does NOT remove the tracking label from cleaned-up PRs. The label is inert on merged/closed PRs and doesn't affect `ListOpenPRs` (which only returns open PRs). No action needed.

### Known Limitations

**Orphaned branches**: If the worktree is manually deleted before Watcher 3 runs, `DeleteWorktree` will fail because it can't resolve the branch from the worktree. The branch may remain as an orphan. This is a minor hygiene issue. A future improvement could store the branch name in the `Job` struct to enable independent cleanup.

### Config Addition

- `ClosedPRTicketStatus string` — Linear status for PRs closed without merge. Default: `"cancelled"`. Configurable via `--closed-pr-status` flag / `ORCH_CLOSED_PR_STATUS` env var.

## Prerequisites

1. Add `ClosedPRTicketStatus string` field to `Config` struct in `internal/orchestrator/config.go`
2. Add `--closed-pr-status` CLI flag (default `"cancelled"`) with `ORCH_CLOSED_PR_STATUS` env var in `cmd/orchestrator/main.go` (in the `serve` subcommand's flag set)
3. Parse the field in `NewConfig()` — no validation needed (any string is a valid Linear status)

## Target File

- **Implementation**: `internal/orchestrator/watcher_pr.go` (new file)
- **Tests**: `internal/orchestrator/watcher_pr_test.go` (new file)
- **Wiring**: Replace `prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)` in `orchestrator.go` with the real constructor
- **Constructor pattern**: Method on `*Orchestrator` (matching `NewCommentWatcher` / `NewLinearPoller`)
- **Dependencies**: `o.store` (StateStore), `o.github` (github.Client), `o.linear` (linear.Client), `o.worktrees` (worktree.Manager), `o.cfg` (Config). Does NOT need `CommentDispatcher` or `SessionManager`.

## User Scenarios & Testing

### User Story 1 - Merged PR triggers full cleanup (Priority: P1)

When a PR is merged on GitHub, Watcher 3 detects it on the next poll cycle and performs cleanup: deletes the worktree and branch, marks the Linear ticket as "done", sets the job status to closed, and releases the concurrency slot.

**Why this priority**: This is the happy-path completion of the entire autonomous pipeline. Without it, worktrees, branches, and slots accumulate indefinitely.

**Independent Test**: Create a job in `StatusQueued` with a PR number, configure `IsMerged` to return true, run one poll cycle, verify worktree is deleted, ticket is "done", job is `StatusClosed`, and slot is released.

**Acceptance Scenarios**:

1. **Given** a job is `StatusQueued` with `PRNumber` set and the PR is merged, **When** Watcher 3 polls, **Then** `DeleteWorktree` is called, the Linear ticket is set to "done", the job status is set to `StatusClosed`, and `ReleaseSlot` is called
2. **Given** a merged PR's worktree was already deleted manually, **When** Watcher 3 runs cleanup, **Then** `DeleteWorktree` failure is logged but ticket status update, job status update, and slot release still proceed
3. **Given** a merged PR's job is already in `StatusClosed`, **When** Watcher 3 polls again, **Then** the job is skipped (terminal state)

---

### User Story 2 - Closed-without-merge PR triggers cleanup with configurable status (Priority: P1)

When a PR is closed without being merged, Watcher 3 detects it and performs the same cleanup sequence, but sets the Linear ticket to the configured non-merge status (default: "cancelled") instead of "done".

**Why this priority**: Closed PRs need the same resource cleanup as merged ones — the only difference is the ticket outcome.

**Independent Test**: Create a job in `StatusQueued` with a PR number, configure `IsMerged` to return false and `IsClosed` to return true, verify cleanup runs and ticket status matches the configured value.

**Acceptance Scenarios**:

1. **Given** a job is `StatusQueued` with `PRNumber` set and the PR is closed without merge, **When** Watcher 3 polls, **Then** `DeleteWorktree` is called, the Linear ticket is set to the configured `ClosedPRTicketStatus`, the job status is set to `StatusClosed`, and `ReleaseSlot` is called
2. **Given** `ClosedPRTicketStatus` is configured as "backlog", **When** a PR is closed without merging, **Then** the Linear ticket is set to "backlog" (not "cancelled")

---

### User Story 3 - Partial cleanup failure does not block remaining steps (Priority: P2)

When any individual cleanup step fails (e.g., worktree already deleted, Linear API error), the remaining steps still execute. All failures are logged.

**Why this priority**: Robustness. Manual intervention (deleting worktrees, force-closing PRs) should not break the cleanup pipeline.

**Independent Test**: Configure `DeleteWorktree` to return an error, verify remaining cleanup steps (ticket status, job status, slot release) still execute.

**Acceptance Scenarios**:

1. **Given** `DeleteWorktree` fails, **When** cleanup runs, **Then** ticket status update, job status update, and slot release still proceed; the error is logged
2. **Given** `SetTicketStatus` fails (Linear API unreachable), **When** cleanup runs, **Then** job status update and slot release still proceed; the Linear error is logged
3. **Given** `SetJobStatus` fails (database error), **When** cleanup runs, **Then** `ReleaseSlot` still proceeds; the error is logged

---

### User Story 4 - Graceful shutdown exits cleanly (Priority: P3)

When the orchestrator receives a shutdown signal, Watcher 3 stops polling and exits without starting new cleanup operations.

**Why this priority**: Required for clean deploys but not a primary behavior.

**Independent Test**: Cancel the context during a poll, verify the watcher exits cleanly.

**Acceptance Scenarios**:

1. **Given** the context is cancelled, **When** the watcher is between polls, **Then** it exits immediately
2. **Given** the context is cancelled, **When** the watcher is mid-cleanup for a PR, **Then** cleanup for the current PR completes (all remaining steps execute) and the watcher exits without processing further PRs

---

### Edge Cases

- PR merged and closed simultaneously (GitHub state): `IsMerged` is checked first, so the merge path is taken
- Job has no PR number (`PRNumber == nil`): Skipped — agent still working, no PR yet
- Multiple PRs detected as merged in the same poll: All cleaned up sequentially in a single poll cycle
- Worktree already deleted manually: `DeleteWorktree` fails, error logged, remaining cleanup proceeds. Branch may be orphaned (see Known Limitations)
- Linear API is unreachable: Logged, job status and slot release still proceed
- Re-poll after cleanup: Job is `StatusClosed`, skipped by FR-006
- Both Watcher 2 and Watcher 3 detect merge simultaneously: Both are safe — `SetJobStatus(StatusClosed)` and `ReleaseSlot` are idempotent

## Requirements

### Functional Requirements

- **FR-001**: System MUST poll `StateStore.ListJobsByStatus(ctx, StatusQueued)` at the configured poll interval, then filter in-memory for jobs where `PRNumber != nil`
- **FR-002**: System MUST check `IsMerged` then `IsClosed` for each tracked PR via `github.Client`
- **FR-003**: On merge detection, system MUST: (a) call `DeleteWorktree`; (b) set the Linear ticket to "done"; (c) set job status to `StatusClosed`; (d) call `ReleaseSlot`. Each step proceeds regardless of prior failures.
- **FR-004**: On close-without-merge detection, system MUST: (a) call `DeleteWorktree`; (b) set the Linear ticket to the configured `ClosedPRTicketStatus`; (c) set job status to `StatusClosed`; (d) call `ReleaseSlot`. Each step proceeds regardless of prior failures.
- **FR-005**: System MUST log errors from any failed cleanup step and continue with remaining steps
- **FR-006**: System MUST skip jobs in terminal state (`StatusClosed`, `StatusNeedsAttention`)
- **FR-007**: System MUST skip jobs where `PRNumber == nil` (agent still working, no PR created)
- **FR-008**: When context is cancelled mid-cleanup, system MUST complete all remaining cleanup steps for the PR currently being processed, then exit without processing further PRs
- **FR-009**: System MUST support configurable Linear ticket status for closed-without-merge PRs via `ClosedPRTicketStatus` config field (default: "cancelled")
- **FR-010**: System MUST be idempotent — `SetJobStatus(StatusClosed)` and `ReleaseSlot` are safe to call redundantly

### Key Entities

- **Job** (existing): `{TicketID, WorktreePath, CmuxSession, PRNumber, Status}` — Watcher 3 queries jobs where `Status == StatusQueued` and `PRNumber != nil`
- **Config.ClosedPRTicketStatus** (new): String config field for the Linear status to set when a PR is closed without merge

## Success Criteria

### Measurable Outcomes

- **SC-001**: Merged PRs are detected and cleaned up within one poll interval
- **SC-002**: Closed-without-merge PRs are detected and cleaned up within one poll interval
- **SC-003**: Partial cleanup failures do not prevent remaining cleanup steps from executing
- **SC-004**: No resource leaks — worktrees, branches, and concurrency slots are freed for all terminal PRs
- **SC-005**: Watcher 3 does not interfere with Watcher 2's active comment processing

## Testing Requirements

### Test Strategy

Integration tests exercising the PR lifecycle watcher against mock implementations of `github.Client`, `worktree.Manager`, `linear.Client`, and `state.StateStore`. Following the same test double pattern established in `watcher_linear_test.go` and `watcher_comment_test.go`.

Test doubles needed:
- Reuse `github.MockClient` from `internal/github/mock.go`
- Create `stubWorktree` implementing `worktree.Manager` — new for this test file since existing watcher tests don't stub the worktree manager
- Create `stubLinear` implementing `linear.Client` (can follow `watcher_linear_test.go`'s `stubLinear` pattern)
- Create `stubState` implementing `state.StateStore` (can follow existing `stubState` patterns)

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify polling loop queries `ListJobsByStatus(StatusQueued)` and filters for jobs with `PRNumber != nil` |
| FR-002 | Integration | Verify `IsMerged` and `IsClosed` are called for each job with a PR number |
| FR-003 | Integration | Verify merge cleanup: `DeleteWorktree` called, ticket set to "done", job `StatusClosed`, `ReleaseSlot` called |
| FR-004 | Integration | Verify close cleanup: `DeleteWorktree` called, ticket set to `ClosedPRTicketStatus`, job `StatusClosed`, `ReleaseSlot` called |
| FR-005 | Integration | Verify cleanup continues after `DeleteWorktree` failure; verify cleanup continues after `SetTicketStatus` failure |
| FR-006 | Integration | Verify jobs in `StatusClosed` and `StatusNeedsAttention` are skipped |
| FR-007 | Integration | Verify jobs with `PRNumber == nil` are skipped |
| FR-008 | Integration | Verify watcher exits cleanly on context cancellation; verify mid-cleanup PR completes before exit |
| FR-009 | Integration | Verify configurable `ClosedPRTicketStatus` is used for closed-without-merge PRs |
| FR-010 | Integration | Verify re-polling after cleanup produces no errors and no repeated side effects |

### Edge Case Coverage

- Worktree already deleted -> Verify `DeleteWorktree` error logged, remaining cleanup proceeds
- Linear API unreachable -> Verify error logged, job status and slot release proceed
- Database error on `SetJobStatus` -> Verify `ReleaseSlot` still called
- PR merged and closed simultaneously -> Verify merge path taken (`IsMerged` checked first)
- Job has no PR number -> Verify skipped
- Multiple PRs merged in same poll -> Verify all cleaned up sequentially
- Concurrent detection by Watcher 2 and Watcher 3 -> Verify idempotent operations produce no errors
