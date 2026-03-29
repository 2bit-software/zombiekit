# Technical Requirements & Implementation Hints

Extracted from DEV-200 and the Agent Context Brief.

## From the Ticket

- Polling loop calling `LinearClient.PollReadyTickets("ai-ready")` on configurable interval
- For each ready ticket:
  1. `TryAcquireSlot` — skip if at concurrency limit, retry next poll
  2. `GitManager.CreateWorktree`
  3. Construct `WORK_CALLBACK_URL` as `http://localhost:{port}/{ticket-id}`
  4. `SessionManager.SpawnSession` with env map including `WORK_CALLBACK_URL` and ticket content
  5. `StateStore.CreateJob` — record ticket->worktree->session association
  6. `LinearClient.SetTicketStatus` to in-progress
  7. `LinearClient.RemoveLabel("ai-ready")`
- Error handling: if any step fails after worktree creation, delete worktree and release slot
- Ticket content (description/spec) written to `.ai/ticket.md` in the worktree before spawning
- Poll interval must be configurable (tight loop hits Linear rate limits)
- Stop channel / context cancellation for graceful shutdown mid-poll

## From the Agent Context Brief

- Orchestrator knows infra, agent knows code (hard boundary)
- `WORK_CALLBACK_URL` injected as env var when spawning session
- No auto-retry on failure — ticket moved to needs-attention
- Concurrency enforcement per-project (configurable, default 1)
- Shared filesystem between orchestrator and agent (same machine)

## Interface Dependencies

These interfaces are referenced but may or may not exist yet:
- `LinearClient` — `PollReadyTickets`, `SetTicketStatus`, `RemoveLabel`
- `GitManager` — `CreateWorktree`, `DeleteWorktree`
- `SessionManager` — `SpawnSession`
- `StateStore` — `CreateJob`, `TryAcquireSlot`, `ReleaseSlot`

## Technical Preferences (Inferred)

- Replace the watcher stub in `internal/orchestrator/watchers.go` with real implementation
- Follow the same ServiceFunc pattern (`func(ctx context.Context) error`)
- Inject dependencies via the Orchestrator struct or watcher constructor
- Use `time.Ticker` for poll loop with context cancellation
