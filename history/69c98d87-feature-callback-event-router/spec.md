# Feature Specification: Callback Event Router

**Feature Branch**: `morganhein/dev-202-callback-event-router-and-post-session-processing`
**Created**: 2026-03-29
**Status**: Draft
**Input**: Linear ticket DEV-202

## Assumptions

- **Single-project mode**: The orchestrator manages one project at a time. `Config.ProjectID` is the sole project identifier for slot management.
- **Review comments only**: PR comments addressed by agents are always review comments (`CommentKindReview`), not top-level issue comments.
- **Base branch from config**: PRs target a configurable base branch (default `main`).
- **Commit-on-success semantics**: Each step's result is persisted immediately. If `CreatePR` succeeds, `SetPR` is called before attempting `ApplyLabel`. If a later step fails, earlier results are preserved.
- **GetJob returns (nil, nil) for unknown tickets**: The router checks for nil `Job`, not a sentinel error.
- **"needs-attention" means both systems**: Moving a ticket to `needs-attention` updates BOTH `LinearClient.SetTicketStatus` and `StateStore.SetJobStatus` unless the job is unknown (in which case only Linear is updated if possible).
- **FailureEvent.CommentID is ignored**: When a FailureEvent carries a CommentID, this ticket does not act on it. The failure is handled identically regardless.
- **Job status stays `in-progress` after PR creation**: The job status does not change to `complete` on PR creation. It remains `in-progress` until the PR is merged (handled by the PR watcher in a future ticket).

## User Scenarios & Testing

### User Story 1 - Successful Completion Creates PR (Priority: P1)

When an agent session finishes work on a ticket, the orchestrator automatically creates a GitHub PR with the agent's description, applies a tracking label, and records the PR number for future reference.

**Why this priority**: This is the core happy path of the entire pipeline. Without it, completed work has no way to reach code review.

**Independent Test**: Can be tested by sending a CompletionEvent to the callback server with a valid ticket ID that has a job in state, a `.ai/pr-description.md` file on disk, and verifying a PR is created.

**Acceptance Scenarios**:

1. **Given** a CompletionEvent for a known ticket ID with `.ai/pr-description.md` in the worktree, **When** the router processes it, **Then** a GitHub PR is created with the file contents as body, the tracking label is applied, and the PR number is stored in state via `SetPR`.
2. **Given** a CompletionEvent where `.ai/pr-description.md` does not exist, **When** the router processes it, **Then** the error is logged with full context and the ticket is moved to `needs-attention` in both Linear and state -- no panic.
3. **Given** a CompletionEvent for an unknown ticket ID (no job in state), **When** the router processes it, **Then** the error is logged and the event is discarded without crashing.

---

### User Story 2 - Failure Handling Preserves Debugging Context (Priority: P1)

When an agent session fails, the orchestrator moves the ticket to a human-visible status, posts the failure reason to Linear for triage, preserves the worktree for debugging, and releases the concurrency slot.

**Why this priority**: Tied with P1 -- without failure handling, failed tickets are silently lost and slots leak, eventually starving the system.

**Independent Test**: Can be tested by sending a FailureEvent and verifying the ticket status changes, a Linear comment is posted, the slot is released, and the worktree is NOT deleted.

**Acceptance Scenarios**:

1. **Given** a FailureEvent with a reason for a known ticket, **When** the router processes it, **Then** the ticket status is set to `needs-attention` in both Linear and state, the reason is posted as a Linear comment, the concurrency slot is released, and the worktree is preserved.
2. **Given** the Linear API is unreachable when processing a FailureEvent, **When** the status update fails, **Then** the error is logged with full context (ticket ID, reason, partial progress) and the slot is still released.

---

### User Story 3 - Comment Resolution Updates PR (Priority: P2)

When an agent resolves a PR review comment, the orchestrator updates the PR description with any changes, posts the agent's resolution as a reply, and advances the watermark so the same comment isn't reprocessed.

**Why this priority**: Important for the review loop but only relevant after P1 (PR must exist first).

**Independent Test**: Can be tested by sending a CommentResolvedEvent for a ticket that has a PR recorded in state, verifying the PR body is updated, a reply is posted, and the watermark advances.

**Acceptance Scenarios**:

1. **Given** a CommentResolvedEvent for a ticket with a recorded PR, **When** processed, **Then** the PR body is updated from `.ai/pr-description.md`, a reply is posted to the original comment with the resolution text, and the comment watermark advances.
2. **Given** a CommentResolvedEvent where the job has no PR number recorded, **When** processed, **Then** the error is logged and the ticket is moved to `needs-attention` in both Linear and state.

---

### User Story 4 - Stub Interfaces Wired for Future Expansion (Priority: P3)

Archival and friction auditor interfaces are defined with correct signatures and called at the right points. No-op implementations are used now; Epic 5 swaps in real implementations without touching router code.

**Why this priority**: Infrastructure for future work. Must be wired correctly but produces no user-visible behavior yet.

**Independent Test**: Can be verified by checking that the router calls the archival interface after each event type and the friction auditor interface after CompletionEvent and CommentResolvedEvent, without errors.

**Acceptance Scenarios**:

1. **Given** any event is processed successfully, **When** the router finishes, **Then** the archival stub is called with the relevant context.
2. **Given** real implementations are provided at construction time, **When** the router runs, **Then** it uses them without any code changes to the router.

---

### Edge Cases

- **Event channel closed** (server shutting down): Router exits its loop cleanly, returns nil.
- **Partial failure in handler**: Error is logged with full context (ticket ID, event kind, step that failed) and ticket moves to `needs-attention` in both Linear and state.
- **GetJob returns nil for unknown ticket**: Event is logged and discarded -- no crash, no status change.
- **CreatePR fails** (e.g., branch doesn't exist on remote): Ticket moves to `needs-attention` with error context.
- **Event.CommentID fails to parse as int64**: Error is logged, ticket moves to `needs-attention`.
- **Job.PRNumber is nil when PR is expected** (CommentResolvedEvent): Error is logged, ticket moves to `needs-attention`.
- **Linear API unreachable during failure handling**: Slot is still released; error is logged but does not block slot cleanup.

## Requirements

### Functional Requirements

- **FR-001**: System MUST consume events from `callbackServer.Events()` channel using a `select` loop that also listens on `ctx.Done()`.
- **FR-002**: On `EventComplete`, system MUST:
  1. Look up the job via `StateStore.GetJob(ctx, event.TicketID)`. If nil, log and discard.
  2. Read `.ai/pr-description.md` from `filepath.Join(job.WorktreePath, ".ai", "pr-description.md")`. If missing, move ticket to `needs-attention` in both systems.
  3. Fetch ticket details via `LinearClient.GetTicket(ctx, event.TicketID)` to obtain the title.
  4. Call `GitHubClient.CreatePR(ctx, CreatePRInput{Title: ticket.Identifier + ": " + ticket.Title, Body: fileContents, Head: event.Branch, Base: config.BaseBranch})`. On failure, move to `needs-attention`.
  5. Call `StateStore.SetPR(ctx, event.TicketID, int64(prNumber))`.
  6. Call `GitHubClient.ApplyLabel(ctx, prNumber, config.TrackingLabel)`.
  7. Call `Archiver.Archive(ctx, event.TicketID, event.Kind)`.
  8. Call `Auditor.Audit(ctx, event.TicketID, event.Kind)`.
- **FR-003**: On `EventComplete`, if `.ai/pr-description.md` does not exist, system MUST log the error with full context and move the ticket to `needs-attention` in both `LinearClient.SetTicketStatus` and `StateStore.SetJobStatus`.
- **FR-004**: On `EventFailed`, system MUST:
  1. Look up the job via `StateStore.GetJob(ctx, event.TicketID)`. Job may be nil (unknown ticket).
  2. Set ticket status to `needs-attention` via `LinearClient.SetTicketStatus(ctx, event.TicketID, "needs-attention")`. This always runs regardless of whether the job exists.
  3. If job exists: call `StateStore.SetJobStatus(ctx, event.TicketID, state.StatusNeedsAttention)`. If job is nil: skip this step.
  4. Post the failure reason as a comment on the Linear ticket via `LinearClient.PostComment(ctx, event.TicketID, event.Reason)`.
  5. Release the concurrency slot via `StateStore.ReleaseSlot(ctx, config.ProjectID)`. This MUST happen even if earlier steps fail (use deferred or ensure-release pattern).
  6. Do NOT delete the worktree.
  7. Call `Archiver.Archive(ctx, event.TicketID, event.Kind)`.
- **FR-005**: On `EventCommentResolved`, system MUST:
  1. Look up the job via `StateStore.GetJob(ctx, event.TicketID)`. If nil, log and discard.
  2. Verify `job.PRNumber` is non-nil. If nil, move to `needs-attention`.
  3. Parse `event.CommentID` as `int64`. If parsing fails, move to `needs-attention`.
  4. Read `.ai/pr-description.md` from the job's worktree. If missing, move to `needs-attention`.
  5. Call `GitHubClient.UpdatePRBody(ctx, int(*job.PRNumber), fileContents)`.
  6. Call `GitHubClient.PostCommentReply(ctx, int(*job.PRNumber), github.CommentKindReview, commentID, event.Resolution)`.
  7. Call `StateStore.SetCommentWatermark(ctx, *job.PRNumber, commentID)`.
  8. Call `Archiver.Archive(ctx, event.TicketID, event.Kind)`.
  9. Call `Auditor.Audit(ctx, event.TicketID, event.Kind)`.
- **FR-006**: After each event handler completes (all three types), system MUST call `Archiver.Archive`. (This is already embedded in the per-handler steps above for clarity.)
- **FR-007**: After `EventComplete` and `EventCommentResolved`, system MUST call `Auditor.Audit`. Rationale: friction auditing is only meaningful for successful outcomes, not failures.
- **FR-008**: If any handler fails partway through, system MUST log the error with full context (ticket ID, event kind, step name, error) and move the ticket to `needs-attention` in both systems (skipping `SetJobStatus` if the job is unknown). Earlier successful steps are preserved (commit-on-success).
- **FR-009**: The event router MUST run as a `shutdown.ServiceFunc`, exiting cleanly when the event channel closes or context is cancelled.

### Interface Requirements (Gaps to Fill)

- **IR-001**: Add `PostComment(ctx context.Context, issueID string, body string) error` to the `linear.Client` interface in `internal/linear/client.go`. Implement in `HTTPClient` using the `commentCreate` GraphQL mutation.
- **IR-002**: Add `github.Client` as a field on the `Orchestrator` struct. Update `New()` constructor to accept it.
- **IR-003**: Add `BaseBranch string` and `TrackingLabel string` fields to `orchestrator.Config`. Default `BaseBranch` to `"main"`, default `TrackingLabel` to `"ai-managed"`.
- **IR-004**: Define `Archiver` interface in `internal/archival/archiver.go`:
  ```
  type Archiver interface {
      Archive(ctx context.Context, ticketID string, eventKind callback.EventKind) error
  }
  ```
  Provide `NoopArchiver` that returns nil.
- **IR-005**: Define `Auditor` interface in `internal/friction/auditor.go`:
  ```
  type Auditor interface {
      Audit(ctx context.Context, ticketID string, eventKind callback.EventKind) error
  }
  ```
  Provide `NoopAuditor` that returns nil.

### Router Architecture

The router is a standalone struct in `internal/orchestrator/router.go`:

```
type Router struct {
    events   <-chan callback.Event
    store    state.StateStore
    github   github.Client
    linear   linear.Client
    archiver archival.Archiver
    auditor  friction.Auditor
    cfg      *Config
    logger   *slog.Logger
}
```

Constructor: `NewRouter(events <-chan callback.Event, store state.StateStore, gh github.Client, lc linear.Client, arch archival.Archiver, aud friction.Auditor, cfg *Config, logger *slog.Logger) *Router`

The `Orchestrator.Run()` method creates the router and passes `router.Run` to the shutdown manager as a `ServiceFunc`.

### Key Entities

- **Event**: Value type from `callback.Event` -- carries ticket ID, kind, and kind-specific fields.
- **Job**: State record from `state.Job` -- maps ticket ID to worktree path, PR number, status.
- **Archiver**: Interface for conversation archival (stub for now).
- **Auditor**: Interface for friction auditing (stub for now).

## Success Criteria

### Measurable Outcomes

- **SC-001**: A CompletionEvent with valid `.ai/pr-description.md` results in a GitHub PR within the same process cycle.
- **SC-002**: A FailureEvent results in ticket status `needs-attention` (both systems), a Linear comment with the reason, and a released slot -- all within the same process cycle.
- **SC-003**: A CommentResolvedEvent results in updated PR body, posted reply, and advanced watermark.
- **SC-004**: No event processing failure causes a panic or silently drops the ticket.
- **SC-005**: Swapping archival/friction stubs for real implementations requires zero changes to router code.

## Testing Requirements

### Test Strategy

Integration tests using mock implementations of GitHubClient, LinearClient, StateStore, Archiver, and Auditor. Each handler is tested through the router's public entry point (processing an event), not by calling internal methods directly. Tests verify the sequence and correctness of calls made to dependencies.

The router's `Run` method is tested by feeding events into a channel and asserting on mock call sequences.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Router dispatches events to correct handler based on Kind |
| FR-002 | Integration | CompletionEvent with valid pr-description: creates PR with correct Title/Head/Base/Body, applies label, stores PR number, calls stubs |
| FR-003 | Integration | CompletionEvent with missing pr-description: moves ticket to needs-attention in both systems |
| FR-004 | Integration | FailureEvent: sets status in both systems (skips SetJobStatus if unknown), posts comment, releases slot with config.ProjectID, preserves worktree, calls Archive |
| FR-005 | Integration | CommentResolvedEvent: updates PR body, posts review reply with parsed int64 commentID, advances watermark, calls stubs |
| FR-006/007 | Integration | Stubs are called at correct points: Archive for all three, Auditor for completions and comment resolutions only |
| FR-008 | Integration | Partial failure: logs context, moves to needs-attention (skipping SetJobStatus if unknown), earlier steps preserved |
| FR-009 | Integration | Router exits cleanly on channel close and on context cancellation |

### Edge Case Coverage

- Missing `.ai/pr-description.md` -> ticket moved to needs-attention in both systems, no panic
- Unknown ticket ID (GetJob returns nil) -> event logged and discarded
- GitHub API failure during CreatePR -> ticket moved to needs-attention
- Linear API failure during FailureEvent -> slot still released, error logged
- CommentResolvedEvent with no PR recorded (nil PRNumber) -> ticket moved to needs-attention
- Event.CommentID fails int64 parse -> ticket moved to needs-attention
- Event channel closed -> router returns nil
- Context cancelled -> router returns nil
