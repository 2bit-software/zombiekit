# Technical Requirements Research: Watcher 2 — PR Comment Queue

## Implementation Hints from Ticket

- Use `GitHubClient.ListOpenPRs` to find orchestrator-owned PRs (identified by tracking label applied during PR creation in DEV-202/T3)
- Use `GetCommentsSince(watermark)` to fetch new comments past the stored watermark
- Comment watermark read from `StateStore` before polling, advanced by T3 (callback router, DEV-202) after each successful resolution via `SetCommentWatermark`
- `SessionManager.SpawnSession` for fresh session per comment — pass original spec + single comment via worktree file or env
- `SessionManager.KillSession` for abort path when PR merges during active session
- Comment-resolution sessions count against the concurrency slot limit
- Per-PR serial queues can be goroutines with channels or in-memory queue — keep it simple
- Watcher 2 and Watcher 3 both poll the same PR list — coordinate via state: check job status in state store before processing

## Coordination with Other Components

- **DEV-199** (Scaffold): Provides config, startup, graceful shutdown — Watcher 2 integrates into this lifecycle
- **DEV-200** (Watcher 1): Ticket pickup and session spawning — establishes the Watcher pattern
- **DEV-202** (Callback Router): Handles `CommentResolvedEvent` and `FailureEvent` post-processing — Watcher 2 waits for these callbacks
- **DEV-201** (Watcher 3): PR lifecycle/cleanup — polls same PR list, must coordinate via state store

## Context Document

Before implementation: read the [Agent Context Brief](https://linear.app/heinsight/document/agent-context-brief-autonomous-dev-pipeline-35c9c49c6532)
