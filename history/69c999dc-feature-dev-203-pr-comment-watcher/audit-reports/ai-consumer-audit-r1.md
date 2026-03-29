# AI-Consumer Audit — Round 1

## CRITICAL (2)

1. **No signaling mechanism between callback router and comment watcher** — `handleCommentResolved` advances watermark but doesn't notify per-PR goroutine to dispatch next comment. `handleFailed` doesn't notify watcher to drain queue. Implementing agent forced to invent architecture.
2. **No `GetJobByPR` or PR-to-ticket lookup** — `ListOpenPRs` returns `PRSummary{Number}` but `GetJob` takes `ticketID`. No method maps PR number to ticket ID. Watcher cannot check job status, acquire correct slot, or pass ticketID to SpawnSession.

## MAJOR (5)

3. **CommentKind unresolved** — `GetCommentsSince` requires `CommentKind` parameter. Which kind?
4. **Bot username not sourced** — Not in Config struct, not derivable from existing config.
5. **Slot acquisition/release ownership ambiguous** — Who acquires for comment sessions? `handleCommentResolved` does NOT call `ReleaseSlot` currently.
6. **Session payload delivery unspecified** — FR-004 says "pass original spec and single comment" but `SpawnSession` only takes `(ticketID, title, worktreePath, env)`.
7. **Type mismatch: `StateStore` uses `int64` for prNumber, `GetCommentsSince` uses `int`** — Will cause conversion bugs if not flagged.

## MINOR (4)

8. Open D5 (re-entry after resolution) — implicit answer but unconfirmed
9. Per-PR goroutine reaping trigger unspecified
10. Target file placement not stated
11. Shutdown timeout ownership unclear
