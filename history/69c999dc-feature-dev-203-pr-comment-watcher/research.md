---
status: complete
updated: 2026-03-29
---

# Research: Watcher 2 — PR Comment Queue

## Executive Summary

The orchestrator codebase already has all the infrastructure Watcher 2 needs: `StateStore` with comment watermarks, `github.Client` with `GetCommentsSince`/`ListOpenPRs`/`IsMerged`, `SessionManager` with `SpawnSession`/`KillSession`, and a `shutdown.Manager` with `ServiceFunc` contract. The key design decision is the per-PR serial queue pattern — a goroutine-per-active-PR with a buffered channel is the best fit for the codebase's channel-oriented conventions.

## Findings

### Codebase Context

**Watcher Pattern (from Watcher 1):**
- Watcher 1 is a method on `Orchestrator` returning `shutdown.ServiceFunc`
- Pattern: `time.NewTicker` + `select` on `<-ticker.C` and `<-ctx.Done()`
- Calls a `pollAndProcess` method on each tick
- Sequential processing per tick, rollback on failure
- File: `internal/orchestrator/watcher_linear.go`

**Existing Stubs:**
- `orchestrator.go:66-67` wires `prWatcher` and `commentWatcher` as `NewWatcherStub`
- Constants: `WatcherPRWatcher`, `WatcherCommentWatcher` in `watchers.go`

**Interfaces Already Available:**
- `SessionManager.SpawnSession(ctx, ticketID, title, worktreePath, env)` — spawns cmux workspace
- `SessionManager.KillSession(ctx, ticketID)` — closes workspace
- `StateStore.GetCommentWatermark(ctx, prNumber)` / `SetCommentWatermark(ctx, prNumber, commentID)`
- `StateStore.TryAcquireSlot(ctx, projectID, limit)` / `ReleaseSlot(ctx, projectID)`
- `StateStore.GetJob(ctx, ticketID)` / `ListJobsByStatus(ctx, statuses...)`
- `github.Client.ListOpenPRs(ctx, label)` — returns PRs with tracking label
- `github.Client.GetCommentsSince(ctx, prNumber, kind, afterID)` — filters by comment ID
- `github.Client.IsMerged(ctx, prNumber)` / `IsClosed(ctx, prNumber)`

**Callback Router (DEV-202):**
- `handleCommentResolved`: updates PR body, posts threaded reply, advances watermark, archives
- `handleFailed`: releases slot, sets needs-attention, posts failure to Linear
- Events arrive via `CallbackServer.Events()` channel (buffered at 64)

**Config:**
- `PollInterval` (default 30s), `ConcurrencyLimit` (default 1), `TrackingLabel` (default "ai-managed")
- Comment sessions count against the same concurrency limit

**Shutdown:**
- `shutdown.Manager.Run(services...)` — all services are `ServiceFunc`
- Context cancellation propagates to all services
- Two-phase: signal → drain with timeout → force exit

### Domain Knowledge

**GitHub API:**
- `since` parameter filters on `updated_at`, not `created_at` — unreliable as sole dedup
- ID-based watermarking (already in StateStore) is the correct approach — IDs are monotonically increasing
- Rate budget: ~720 requests/hour for 5 active PRs at 30s interval — well within 5,000/hr PAT limit
- `go-github-ratelimit` transport already handles rate limit backoff
- ETag caching could reduce cost further but is a follow-up optimization

**Per-PR Serial Queue Patterns (Go):**

| Approach | Pros | Cons | Fit |
|----------|------|------|-----|
| Goroutine-per-PR + channel | Natural serial processing, clean context cancellation, aligns with codebase patterns | Goroutine lifecycle management | Best |
| Worker pool + per-PR mutex | Bounded goroutines | Mutex held during long sessions, ordering not guaranteed | Poor |
| Single dispatcher goroutine | No concurrency in queue logic | Bottleneck, less idiomatic | OK |

**Cancellation Pattern:**
- Context hierarchy: shutdown → watcher → per-PR → per-session
- Check `IsMerged`/`IsClosed` before dispatching each comment
- On merge during session: cancel per-PR context, `KillSession`, drain queue

**Failure Pattern:**
- On session failure: advance watermark, drain queue, mark needs-attention
- Matches existing `markNeedsAttention` convention — no infinite retry loops

## Decision Points

- [x] **D1**: Per-PR queue pattern → Goroutine-per-PR with buffered channel
- [x] **D2**: Coordination with PR watcher → Merge PR state checking into comment watcher (avoid two-poller coordination)
- [ ] **D3**: Comment types to process — Review comments only, or both review + issue comments?
- [ ] **D4**: Bot comment filtering — Filter by author in poller (need configurable bot username)
- [ ] **D5**: Comment threading — If reviewer replies "that's not what I meant" after resolution, should the reply re-enter the queue? (Watermark says yes — higher ID)

## Recommendations

1. Use goroutine-per-active-PR with buffered channel. Lazily create on first comment, cancel on merge/close.
2. Keep ID-based watermarks as primary dedup. `since` parameter is optional optimization.
3. Check PR state (merged/closed) in the comment watcher itself rather than coordinating with a separate PR watcher.
4. On session failure: advance watermark, drain queue, mark needs-attention. No automatic retry.
5. Filter bot comments by author before enqueuing.
6. Defer ETag caching transport to follow-up.

## Sources

- Codebase: `internal/orchestrator/watcher_linear.go`, `router.go`, `orchestrator.go`, `watchers.go`
- Codebase: `internal/state/store.go`, `internal/github/client.go`, `internal/cmux/types.go`
- Codebase: `internal/callback/event.go`, `internal/shutdown/manager.go`
- GitHub REST API: PR review comments, issue comments, rate limits, conditional requests
- Go: effective Go concurrency patterns, errgroup, context hierarchy
