# Initiative: callback-event-router

**Type**: feature
**Status**: completed
**Created**: 2026-03-29
**ID**: 69c98d87-feature-callback-event-router

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-29 13:55 |
| plan | completed | 2026-03-29 14:10 |
| tasks | completed | 2026-03-29 14:15 |
| implement | completed | 2026-03-29 14:20 |

## Source

**Linear Ticket**: [DEV-202](https://linear.app/heinsight/issue/DEV-202/callback-event-router-and-post-session-processing)
**Title**: Callback event router and post-session processing

## Description

Callback event router and post-session processing - subscribe to callback server events and handle CompletionEvent, CommentResolvedEvent, and FailureEvent with appropriate post-session actions.

## Completion

**Completed**: 2026-03-29
**Duration**: Same day (spec through implementation)

### Outcomes
- Feature: Callback event router - Complete
- Interface: LinearClient.PostComment - Complete
- Interface: Archiver stub (internal/archival) - Complete
- Interface: Auditor stub (internal/friction) - Complete
- Config: GitHubOwner, GitHubRepo, BaseBranch, TrackingLabel - Complete
- Wiring: github.Client in Orchestrator + Router in shutdown manager - Complete

### Files Changed
- `internal/orchestrator/router.go` (NEW) - Router struct, 3 handlers, markNeedsAttention
- `internal/orchestrator/router_test.go` (NEW) - 11 integration tests
- `internal/archival/archiver.go` (NEW) - Archiver interface + NoopArchiver
- `internal/friction/auditor.go` (NEW) - Auditor interface + NoopAuditor
- `internal/orchestrator/config.go` - 4 new config fields
- `internal/orchestrator/config_test.go` - 4 new validation tests
- `internal/orchestrator/orchestrator.go` - github.Client field, router wiring
- `internal/linear/client.go` - PostComment added to interface
- `internal/linear/http_client.go` - PostComment via commentCreate mutation
- `internal/linear/mock.go` - PostCommentFn added
- `cmd/orchestrator/main.go` - CLI flags, github client creation
- `internal/orchestrator/watcher_linear_test.go` - Fixed stubs

### Tests
- 11 new router integration tests (all passing)
- 4 new config validation tests (all passing)
- Full test suite green
