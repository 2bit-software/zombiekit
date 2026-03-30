# Initiative: feature-dev-201-pr-lifecycle-watcher

**Type**: feature
**Status**: completed
**Created**: 2026-03-29
**ID**: 69c9d94f-feature-feature-dev-201-pr-lifecycle-watcher

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-29 19:25 |
| plan | completed | 2026-03-29 19:35 |
| tasks | completed | 2026-03-29 19:38 |
| implement | completed | 2026-03-29 19:40 |

## Source

**Linear Ticket**: [DEV-201](https://linear.app/heinsight/issue/DEV-201/watcher-3-pr-lifecycle-detection-and-cleanup)
**Title**: Watcher 3 — PR lifecycle detection and cleanup

## Description

Watcher 3 — PR lifecycle detection and cleanup. Polling loop to detect merged/closed PRs and perform cleanup (worktree deletion, branch cleanup, ticket status update, slot release).

## Completion

**Completed**: 2026-03-29
**Duration**: ~40 minutes (single session)

### Outcomes
- Feature: Watcher 3 — PR lifecycle detection and cleanup - Complete

### Files Changed
- `internal/orchestrator/config.go` — added `ClosedPRTicketStatus` field
- `cmd/orchestrator/main.go` — added `--closed-pr-status` flag
- `internal/orchestrator/watcher_pr.go` — new (~115 LOC)
- `internal/orchestrator/watcher_pr_test.go` — new (~300 LOC, 12 tests)
- `internal/orchestrator/orchestrator.go` — replaced stub with real watcher

### Key Decisions
- Detection via `ListJobsByStatus(StatusQueued)` + `PRNumber != nil` filter (not `ListOpenPRs`)
- Jobs stay `StatusQueued` until Watcher 3 transitions to `StatusClosed` (router doesn't set `StatusComplete`)
- `ReleaseSlot` called unconditionally (safe — clamps to 0)
- `context.Background()` used in `cleanupPR` so mid-shutdown cleanup completes
- No `CleanBranch` fallback (orphaned branches accepted as known limitation)
