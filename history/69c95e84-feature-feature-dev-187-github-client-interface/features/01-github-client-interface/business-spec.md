# Business Specification: GitHubClient Interface and Test Stub

## Purpose

Define a Go interface that encapsulates all GitHub operations the autonomous orchestrator needs for PR lifecycle management. This interface serves as the mock boundary for orchestrator-core development (DEV-150), allowing the orchestrator to be built and tested against a configurable mock without real GitHub API calls.

## Scope

### In Scope

- `Client` interface with all methods the orchestrator requires (named `Client`, not `GitHubClient`, following Go convention -- the package name `github` provides the qualifier, yielding `github.Client`, consistent with `linear.Client`)
- Input/output types for all interface methods
- Error types matching the project's existing error classification pattern
- Test mock implementation with configurable canned responses and call recording

### Out of Scope

- Real GitHub API calls (DEV-188)
- Authentication setup (DEV-188)
- Webhook handling
- PR merge operations (humans merge; orchestrator observes)

## Interface Methods

### 1. CreatePR

Create a pull request in the target repository.

**Input**: Structured input containing title, body (markdown), head branch name, and base branch name.

**Output**: The PR number (integer) assigned by GitHub.

**Error conditions**: API errors, network failures, authentication failures, rate limiting.

**Note**: Labels are applied separately via ApplyLabel after PR creation.

### 2. UpdatePRBody

Update the description/body of an existing pull request.

**Input**: PR number and new body content (markdown string).

**Output**: None (success/error).

**Error conditions**: PR not found, API errors, network failures, rate limiting.

### 3. GetCommentsSince

Retrieve PR comments created after a known watermark, supporting both issue comments (top-level conversation) and review comments (inline diff feedback).

**Input**: PR number, comment kind (issue or review), and the ID of the last-seen comment (watermark). A zero/empty watermark means "fetch all."

**Output**: Ordered list of comment objects, each containing: comment ID (int64), author login, body text, creation timestamp, and -- for review comments -- the file path and diff context.

**Behavior**: Returns only comments with IDs greater than the watermark, in chronological order. The caller is responsible for storing the new watermark (the ID of the last returned comment).

**Pagination**: Transparent to the caller. Regardless of how many pages GitHub requires, the caller receives one complete slice.

**Error conditions**: PR not found, API errors, network failures, rate limiting.

### 4. PostCommentReply

Post a reply to a PR conversation.

**Input**: PR number, comment kind (issue or review), the comment ID being replied to (int64), and the reply body (markdown string).

- When `kind` is `CommentKindIssue` and `commentID` is 0: posts a new top-level issue comment (not a reply).
- When `kind` is `CommentKindIssue` and `commentID` is non-zero: posts a new top-level issue comment (GitHub issue comments have no threading; the commentID is recorded for caller context but not used in the API call).
- When `kind` is `CommentKindReview` and `commentID` is non-zero: posts a threaded reply to the specified review comment.
- When `kind` is `CommentKindReview` and `commentID` is 0: returns `ErrNotFound` -- review replies require a parent comment.

**Output**: The ID of the newly created comment (int64).

**Error conditions**: PR not found, parent comment not found (review reply with invalid ID), API errors, network failures, rate limiting.

### 5. ApplyLabel

Add a label to a pull request. Idempotent -- applying a label that already exists is a no-op success.

**Input**: PR number and label name (string).

**Output**: None (success/error).

**Precondition**: The label must already exist on the repository. The interface does not create labels.

**Error conditions**: PR not found, label does not exist on repo (validation error), API errors, network failures, rate limiting.

### 6. IsMerged

Check whether a pull request has been merged.

**Input**: PR number.

**Output**: Boolean (true if merged, false otherwise).

**Error conditions**: PR not found, API errors, network failures, rate limiting.

### 7. IsClosed

Check whether a pull request is closed (without being merged). A merged PR is NOT considered "closed" by this method.

**Input**: PR number.

**Output**: Boolean (true if closed-and-not-merged, false otherwise).

**Error conditions**: PR not found, API errors, network failures, rate limiting.

### 8. ListOpenPRs

List open pull requests that carry a specific label.

**Input**: Label name (string).

**Output**: List of PR summary objects, each containing: PR number, title, head branch, base branch, and labels.

**Behavior**: Returns only open PRs. Filtering by label is exact-match on label name.

**Pagination**: Transparent to the caller.

**Error conditions**: API errors, network failures, rate limiting.

## Domain Types

### CreatePRInput

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Title | string | yes | PR title |
| Body | string | yes | PR description (markdown) |
| Head | string | yes | Source branch name |
| Base | string | yes | Target branch name |

### CommentKind

String-typed enum distinguishing between issue comments (top-level PR conversation) and review comments (inline diff feedback). Follows the `callback.EventKind` pattern.

```go
type CommentKind string

const (
    CommentKindIssue  CommentKind = "issue"
    CommentKindReview CommentKind = "review"
)
```

### PRComment

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | GitHub comment ID (used as watermark) |
| Author | string | GitHub login of commenter |
| Body | string | Comment text (markdown) |
| CreatedAt | time.Time | When the comment was posted |
| Path | string | File path (review comments only, empty for issue comments) |
| DiffHunk | string | Diff context (review comments only, empty for issue comments) |
| InReplyToID | int64 | Parent comment ID set by GitHub on review comments to indicate threading (0 if top-level). Not the same as the `commentID` parameter in PostCommentReply. |

### PRSummary

| Field | Type | Description |
|-------|------|-------------|
| Number | int | PR number (callers convert to int64 when passing to state store's SetPR) |
| Title | string | PR title |
| Head | string | Head branch name |
| Base | string | Base branch name |
| Labels | []string | Label names on the PR |

## Error Types

Four error kinds, matching the existing classification pattern in the project:

| Kind | When |
|------|------|
| NotFound | PR, comment, or label does not exist |
| RateLimited | GitHub rate limit exceeded or approaching |
| API | GitHub returned a non-success response (4xx/5xx other than 404/rate limit) |
| Network | Connection failure, timeout, DNS resolution error |

Each error carries a human-readable message and wraps the underlying cause error for chain inspection.

Predicate functions (`IsNotFound`, `IsRateLimited`, `IsAPIError`, `IsNetworkError`) allow callers to classify errors without type assertions.

## Test Mock

The mock implementation satisfies the `Client` interface with:

1. **Configurable responses**: Each method has a corresponding function field. When set, calling the method delegates to that function. When not set, the method returns a descriptive error.

2. **Call recording**: Every method invocation is recorded with the method name and arguments (context excluded). Callers can inspect the `Calls` slice to verify invocation count, argument values, and call ordering.

3. **Compile-time verification**: A package-level assertion ensures the mock always satisfies the interface, catching drift immediately.

## Acceptance Criteria

1. Given the interface definition, when a mock implementation is wired into a consumer, then the consumer compiles and calls through without real API calls.
2. Given the mock's `CreatePR` is configured to return a specific PR number, when called, then that PR number is returned.
3. Given the mock's `GetCommentsSince` is configured to return 2 comments, when called, then exactly those 2 comments are returned in order.
4. Given an error is configured on the mock, when the corresponding method is called, then the configured error type is returned.
5. Given any method on the mock is called, then the call is recorded and verifiable in tests (call count, args).

## Conventions

All methods accept `context.Context` as the first parameter, matching the `linear.Client` pattern. Methods return `(T, error)` or just `error`. Complex inputs use typed structs.

## Integration Context

This interface is consumed by:
- **Orchestrator core** (DEV-150): Creates PRs after agent completion, monitors comments via Watcher 2, detects merges via Watcher 3
- **State store** (DEV-154): Correlates PR numbers with jobs via `SetPR(ticketID, prNumber int64)` -- callers convert `int` PR numbers to `int64`
- **Callback server** (DEV-184): References `CommentID` as `string` in events -- callers convert between `string` and `int64` as needed

### Watermark Tracking

The current `comment_watermarks` table stores a single watermark per PR number. Since issue comments and review comments use separate GitHub ID spaces, the orchestrator must track two watermarks per PR. Options:
1. Add a `comment_kind` column to the watermarks table (migration in a future ticket)
2. Track separate watermarks in application code using composite keys (e.g., `pr_number * 2 + kind_offset`)

This is an orchestrator-level decision (DEV-150 scope), not a GitHubClient concern. The interface is correct as-is -- it accepts a watermark per call and returns comments filtered by that watermark.
