---
status: complete
updated: 2026-03-29
---

# Research: Watcher 1 — Ready Ticket Pickup

## Executive Summary

All four dependency interfaces (LinearClient, WorktreeManager, SessionManager, StateStore) are fully implemented and tested. The orchestrator scaffold exists with stub watchers that need to be replaced with real polling logic. The watcher is a pure coordination layer — no new domain logic is needed, only orchestration of existing components.

## Findings

### Codebase Context

- **Orchestrator scaffold**: `internal/orchestrator/` contains `Orchestrator` struct with `Run()` method that currently instantiates three watcher stubs via `NewWatcherStub()`. The linear poller stub needs to be replaced with real implementation.
- **Service lifecycle**: Uses a shutdown manager pattern where each service is a `func(ctx context.Context) error`. The watcher must conform to this signature.
- **Dependency injection**: `Orchestrator` currently holds `cfg *Config` and `store state.StateStore`. Needs to be extended with `linear.Client`, `worktree.Manager`, and `cmux.SessionManager`.
- **Config**: All necessary fields already exist in `orchestrator.Config` — `PollInterval`, `ConcurrencyLimit`, `CallbackPort`, `WorktreesRoot`, `LinearAPIKey`.
- **cmd/orchestrator/main.go**: Currently only creates the SQLite store. Needs to also create the Linear client, worktree manager, and session manager.

### Interface Availability (Verified)

| Component | Interface | Implementation | Status |
|-----------|-----------|----------------|--------|
| Linear client | `linear.Client` | `linear.HTTPClient` | Complete, tested |
| Worktree manager | `worktree.Manager` | `worktree.GitManager` | Complete, tested |
| Session manager | `cmux.SessionManager` | `cmux.CmuxManager` | Complete, tested |
| State store | `state.StateStore` | `state.SQLiteStore` | Complete, tested |
| Callback server | `callback.CallbackServer` | Concrete | Complete, tested |

### Domain Knowledge

- **Compensating transactions**: The rollback pattern (delete worktree + release slot on failure) is a standard compensating transaction. Order matters — clean up in reverse order of creation.
- **Poll-based architecture**: Simpler than webhooks for MVP. Linear rate limits make configurable intervals necessary (default likely 30s-60s).
- **Ticket content delivery**: Writing to `.ai/ticket.md` in the worktree is a filesystem convention — the agent knows to look there. No IPC needed.

## Decision Points

- [x] **D1**: All interfaces exist — no new interfaces needed
- [x] **D2**: Config has all required fields — no schema changes needed
- [ ] **D3**: Watcher function signature — use `ServiceFunc` pattern matching existing stubs
- [ ] **D4**: Where to write the watcher — replace stub in `watchers.go` or new file? (Recommend new file, keep stubs for other watchers)
- [ ] **D5**: How to handle duplicate ticket pickup across polls — rely on CreateJob idempotency or check job existence first?

## Recommendations

1. **Create a dedicated `watcher_linear.go`** for the real implementation, keeping the stub pattern available for the other two watchers (PR watcher, comment watcher).
2. **Use `time.NewTicker` with `select` on context.Done()** for the poll loop — standard Go pattern for cancellable periodic work.
3. **Process tickets sequentially within a poll** — no need for goroutine-per-ticket since we're already rate-limited by Linear and concurrency slots.
4. **Log-and-continue on individual ticket failures** — one bad ticket shouldn't block others in the same poll batch.

## Sources

- Linear ticket DEV-200
- Agent Context Brief (Linear document)
- Codebase exploration: `internal/orchestrator/`, `internal/linear/`, `internal/worktree/`, `internal/cmux/`, `internal/state/`
