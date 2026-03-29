# Feature Specification: Watcher 1 — Ready Ticket Pickup and Session Spawning

**Feature Branch**: `feat/feature-dev-200-watcher-ticket-pickup`
**Created**: 2026-03-29
**Status**: Approved
**Input**: Linear ticket DEV-200

## User Scenarios & Testing

### User Story 1 - Single Ticket Pickup (Priority: P1)

The orchestrator polls Linear for tickets labeled "ai-ready". When it finds one and a concurrency slot is available, it creates a git worktree, writes the ticket content to a well-known file, spawns an agent session with the correct environment, records the job in state, and updates the ticket status in Linear.

**Why this priority**: This is the core happy path. Without it, no work gets done.

**Independent Test**: Can be fully tested by creating a mock Linear ticket with the "ai-ready" label and verifying the full pipeline executes: worktree exists, session is running, job is recorded, ticket status is "in-progress", and the "ai-ready" label is removed.

**Acceptance Scenarios**:

1. **Given** one `ai-ready` ticket and a free concurrency slot, **When** the watcher polls, **Then** a worktree is created, ticket content is written to `.ai/ticket.md` in the worktree, a session is spawned with `WORK_CALLBACK_URL` in its environment, the job is recorded in state, and the ticket status is set to "in-progress" with the "ai-ready" label removed.
2. **Given** no `ai-ready` tickets, **When** the watcher polls, **Then** no action is taken and the watcher waits for the next poll interval.

---

### User Story 2 - Concurrency Enforcement (Priority: P1)

When multiple tickets are ready but the concurrency limit is reached, the watcher picks up only as many as slots allow and leaves the rest for the next poll cycle.

**Why this priority**: Without concurrency control, the system could overwhelm the machine or hit rate limits. This is a safety constraint, not a nice-to-have.

**Independent Test**: Configure concurrency limit to 1, create two "ai-ready" tickets, verify only one session is spawned. After the first slot is released, verify the second is picked up on the next poll.

**Acceptance Scenarios**:

1. **Given** two `ai-ready` tickets and a concurrency limit of 1, **When** the watcher polls, **Then** only one session is spawned; the second ticket remains "ai-ready".
2. **Given** the concurrency limit is reached, **When** the watcher polls, **Then** it skips ticket pickup entirely and retries on the next interval.
3. **Given** a slot is released between polls, **When** the next poll fires, **Then** the remaining ticket is picked up.

---

### User Story 3 - Error Rollback (Priority: P1)

If any step in the pickup pipeline fails, the system must clean up in reverse order and mark the ticket as needing human attention.

**Why this priority**: Without rollback, failures leak resources (orphan worktrees, stuck slots). Without "needs-attention" labeling, broken tickets retry forever.

**Independent Test**: Inject a failure at session spawning (e.g., mock returns error). Verify the worktree is deleted, the slot is released, and the ticket is labeled "needs-attention" with "ai-ready" removed.

**Acceptance Scenarios**:

1. **Given** worktree creation succeeds but session spawning fails, **When** the error occurs, **Then** the worktree is deleted, the slot is released, and the ticket is labeled "needs-attention" with "ai-ready" removed.
2. **Given** worktree creation succeeds, session spawning succeeds, but state recording fails, **When** the error occurs, **Then** the session is killed, the worktree is deleted, the slot is released, and the ticket is labeled "needs-attention".
3. **Given** slot acquisition fails (at capacity), **When** the failure occurs, **Then** no worktree is created and the ticket is untouched (still "ai-ready" — this is a normal deferral, not an error).
4. **Given** the job is created but `SetTicketStatus` fails, **When** the error occurs, **Then** the error is logged and the job continues running. The ticket status will be stale in Linear but correct in the state store.

---

### User Story 4 - Graceful Shutdown (Priority: P2)

When the orchestrator receives a stop signal during a poll cycle, the current iteration completes (no partial work) and no new sessions are spawned.

**Why this priority**: Important for operational safety, but the system can function without it during development — a hard kill is acceptable in early iterations.

**Independent Test**: Start a poll cycle, close the stop channel mid-iteration, verify the current ticket finishes processing but no additional tickets are started.

**Acceptance Scenarios**:

1. **Given** the stop channel is closed mid-poll, **When** the signal fires, **Then** the current ticket's pipeline completes and no new sessions are spawned.
2. **Given** the stop channel is closed between polls, **When** the next poll would fire, **Then** the watcher exits cleanly.

---

### Edge Cases

- What happens when Linear API returns an error during polling? The watcher logs the error and retries on the next interval — it does not crash.
- What happens when the ticket content (description) is empty? The watcher still creates `.ai/ticket.md` with whatever content exists (even if empty) and proceeds.
- What happens when the worktree directory already exists for a ticket ID? `CreateWorktree` should handle this — the watcher does not need to check.
- What happens when the same ticket appears in consecutive polls (race between poll and label removal)? The watcher checks if a job already exists for this ticket ID (via `StateStore.GetJob`) before attempting pickup. If a job exists, the ticket is skipped.
- What happens when worktree creation fails after slot acquisition (e.g., disk full)? The slot is released. No worktree cleanup is needed since creation failed.
- What happens when `SetTicketStatus` or `RemoveLabel` fails after the job is running? Log the error and continue — the job is already running, state store is source of truth, and FR-012 prevents duplicate pickup.

## Pipeline Order (Explicit)

The pickup pipeline for each ticket executes in this exact order:

```
1. TryAcquireSlot(ctx, projectID, concurrencyLimit)  → skip ticket if false
2. CreateWorktree(ctx, ticket.Identifier, shortTitle)  → rollback: ReleaseSlot
3. Write ticket.Description to {worktreePath}/.ai/ticket.md (create .ai/ dir if needed)
4. SpawnSession(ctx, ticket.Identifier, ticket.Title, worktreePath, envMap)  → rollback: DeleteWorktree + ReleaseSlot
5. CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef)  → rollback: KillSession + DeleteWorktree + ReleaseSlot
6. SetTicketStatus(ctx, ticket.ID, "In Progress")  → on failure: log and continue (job is running)
7. RemoveLabel(ctx, ticket.ID, "ai-ready")  → on failure: log and continue (job is running)
```

**Field usage**: `ticket.Identifier` (e.g., "DEV-200") is used for worktree naming, session naming, and job keys. `ticket.ID` (UUID) is used for Linear API calls (`SetTicketStatus`, `RemoveLabel`).

**`projectID` for slot acquisition**: Uses `Config.ProjectID` — a new required config field identifying the Linear project being watched (e.g., a project slug or ID).

**`callbackPort`**: Read from `Config.CallbackPort` (already exists).

## Requirements

### Functional Requirements

- **FR-001**: System MUST poll Linear for tickets with a configurable label (default: "ai-ready") at a configurable interval.
- **FR-002**: System MUST enforce a per-project concurrency limit when picking up tickets. Tickets exceeding the limit are deferred to the next poll.
- **FR-003**: System MUST create a git worktree for each picked-up ticket using `worktree.Manager.CreateWorktree(ctx, ticket.Identifier, shortTitle)`.
- **FR-004**: System MUST write the ticket content (`ticket.Description`) to `.ai/ticket.md` within the worktree before spawning the session, creating the `.ai/` directory if needed.
- **FR-005**: System MUST construct `WORK_CALLBACK_URL` as `http://localhost:{Config.CallbackPort}/{ticket.Identifier}` and pass it to the session environment.
- **FR-006**: System MUST spawn an agent session via `cmux.SessionManager.SpawnSession` with `ticket.Identifier`, `ticket.Title`, worktree path, and environment map.
- **FR-007**: System MUST record the job via `state.StateStore.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef)`.
- **FR-008**: System MUST set the ticket status to "In Progress" via `linear.Client.SetTicketStatus(ctx, ticket.ID, "In Progress")` after successful job creation.
- **FR-009**: System MUST remove the pickup label via `linear.Client.RemoveLabel(ctx, ticket.ID, "ai-ready")` after setting status.
- **FR-010**: System MUST roll back on failure in reverse pipeline order: kill session (if spawned via `KillSession`), delete worktree (if created via `DeleteWorktree`), and release slot (if acquired via `ReleaseSlot`). Rollback failures are logged but do not propagate.
- **FR-011**: System MUST respect context cancellation for graceful shutdown — complete current ticket's pipeline, start no new tickets.
- **FR-012**: System MUST check for existing jobs (via `StateStore.GetJob`) before attempting pickup, to prevent duplicate processing of the same ticket across polls.
- **FR-013**: On pipeline failure (after rollback), system MUST apply the "needs-attention" label to the ticket via `linear.Client.ApplyLabel(ctx, ticket.ID, "needs-attention")` and remove the "ai-ready" label. If these Linear calls also fail, log and move on.
- **FR-014**: If `SetTicketStatus` or `RemoveLabel` fails after the job is successfully created (steps 6-7), system MUST log the error and continue — the job is running, state store is source of truth.

### Key Entities

- **Job**: Associates a Linear ticket ID with a worktree path and cmux session reference. Tracks the lifecycle of a single unit of agent work.
- **Slot**: A concurrency token scoped to a project. Limits how many jobs can run simultaneously.
- **Ticket**: A Linear issue with an "ai-ready" label, containing the work specification in its description.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A single "ai-ready" ticket is picked up, processed, and marked "in-progress" within one poll interval.
- **SC-002**: Concurrency limits are never exceeded — at no point do more jobs exist than the configured limit.
- **SC-003**: After a mid-pipeline failure, zero orphan worktrees or leaked slots remain.
- **SC-004**: Graceful shutdown completes without orphaned partial work.

## Testing Requirements

### Test Strategy

Integration tests using test doubles (stub implementations of the interfaces). The watcher orchestrates calls across four interfaces — the value is in testing the coordination logic, not the individual implementations (which have their own tests).

Test doubles should be simple in-memory implementations that record calls and can be configured to return errors at specific points for rollback testing.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Watcher calls PollReadyTickets on each tick with configured label |
| FR-002 | Integration | With limit=1 and 2 tickets, only 1 session spawned per poll |
| FR-003 | Integration | CreateWorktree called with ticket ID and short title |
| FR-004 | Integration | `.ai/ticket.md` exists in worktree path with ticket description |
| FR-005 | Integration | SpawnSession receives env map containing correct WORK_CALLBACK_URL |
| FR-006 | Integration | SpawnSession called with correct ticketID, title, worktreePath |
| FR-007 | Integration | CreateJob called with ticket ID, worktree path, session ref |
| FR-008 | Integration | SetTicketStatus called with "in-progress" after CreateJob |
| FR-009 | Integration | RemoveLabel called with "ai-ready" after SetTicketStatus |
| FR-010 | Integration | On SpawnSession error: KillSession (if needed) + DeleteWorktree + ReleaseSlot called in reverse order |
| FR-011 | Integration | Context cancellation stops new work; in-flight work completes |
| FR-012 | Integration | Ticket with existing job in state store is skipped |
| FR-013 | Integration | On pipeline failure: "needs-attention" label applied, "ai-ready" removed |
| FR-014 | Integration | SetTicketStatus failure after CreateJob: error logged, job continues |

### Edge Case Coverage

- Poll returns empty list → no calls to downstream services
- Poll returns error → logged, retry on next interval
- Duplicate ticket across polls → idempotent handling (skip or fail gracefully)
- Slot acquisition returns false → ticket skipped, no cleanup needed
- SetTicketStatus/RemoveLabel fails after job created → behavior per Decision D1
- Worktree creation fails after slot acquired → ReleaseSlot called, no worktree cleanup needed

## Resolved Decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| D1 | Post-job-creation Linear API failure | **(a) Log and continue** | Job is running, state store is source of truth. FR-012 prevents duplicate pickup. |
| D2 | Source of `projectID` for slot acquisition | **(a) New `ProjectID` config field** | Explicit config, matches single-project-per-orchestrator deployment model. |
| D3 | Failure behavior for ticket | **(b) Move to "needs-attention"** | Aligns with Agent Context Brief. Prevents infinite retry on persistent failures. |
