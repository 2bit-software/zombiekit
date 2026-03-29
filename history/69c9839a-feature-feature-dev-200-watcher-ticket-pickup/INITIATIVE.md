# Initiative: feature-dev-200-watcher-ticket-pickup

**Type**: feature
**Status**: complete
**Created**: 2026-03-29
**ID**: 69c9839a-feature-feature-dev-200-watcher-ticket-pickup

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | complete | 2026-03-29 |
| plan | complete | 2026-03-29 |
| tasks | complete | 2026-03-29 |
| implement | complete | 2026-03-29 |

## Source

**Linear Ticket**: [DEV-200](https://linear.app/heinsight/issue/DEV-200/watcher-1-ready-ticket-pickup-and-session-spawning)
**Title**: Watcher 1 — ready ticket pickup and session spawning

## Completion

**Completed**: 2026-03-29
**Duration**: 1 day

### Outcomes
- Feature: watcher-ticket-pickup - Complete (all 14 FRs implemented and tested)

### Summary
- Implemented `NewLinearPoller` watcher replacing the stub in orchestrator
- Poll loop with configurable interval, context cancellation for graceful shutdown
- Full pickup pipeline: poll -> check existing -> acquire slot -> create worktree -> write ticket file -> spawn session -> record job -> update Linear
- Compensating transaction rollback on failure with needs-attention labeling
- 19 integration tests covering happy path, concurrency, rollback, and edge cases
- Extended Config with `ProjectID` and `RepoDir` fields
- Wired real Linear, worktree, and session manager clients in cmd/orchestrator

### Files Changed
- `internal/orchestrator/config.go` - 2 new fields + validation
- `internal/orchestrator/config_test.go` - 3 new tests
- `internal/orchestrator/orchestrator.go` - extended struct, replaced stub
- `internal/orchestrator/orchestrator_test.go` - updated constructor calls
- `internal/orchestrator/watcher_linear.go` - NEW (~175 LOC)
- `internal/orchestrator/watcher_linear_test.go` - NEW (~450 LOC, 19 tests)
- `cmd/orchestrator/main.go` - 2 new flags, real client wiring
