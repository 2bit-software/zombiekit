# Initiative: feature-dev-187-github-client-interface

**Type**: feature
**Status**: completed
**Created**: 2026-03-29
**ID**: 69c95e84-feature-feature-dev-187-github-client-interface

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-29 10:35 |
| plan | completed | 2026-03-29 10:50 |
| tasks | completed | 2026-03-29 10:55 |
| implement | completed | 2026-03-29 11:00 |

## Source

**Linear Ticket**: [DEV-187](https://linear.app/heinsight/issue/DEV-187/define-githubclient-interface-and-test-stub)
**Title**: Define GitHubClient interface and test stub

## Description

Define the `GitHubClient` Go interface with all methods the orchestrator needs for PR lifecycle management, along with input/output types, error types, and a configurable test mock.

## Completion

**Completed**: 2026-03-29
**Duration**: ~45 minutes

### Outcomes

- `internal/github/client.go` -- Client interface with 8 methods, 4 domain types (CreatePRInput, PRComment, PRSummary, CommentKind)
- `internal/github/errors.go` -- 4 error kinds (NotFound, RateLimited, API, Network) with constructors and predicates
- `internal/github/mock.go` -- MockClient with per-method Fn fields and call recording
- `internal/github/mock_test.go` -- 11 tests covering all 5 acceptance criteria

### Key Decisions

- Interface named `Client` (not `GitHubClient`) following Go `github.Client` convention
- `CommentKind` as `string` type matching `callback.EventKind` pattern
- `PostCommentReply` takes `CommentKind` parameter to distinguish issue vs review endpoints
- PR numbers as `int` in interface; callers convert to `int64` at state store boundary
