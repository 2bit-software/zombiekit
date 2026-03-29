# Implementation Plan: Implement GitHubClient

## Dependencies

- `github.com/google/go-github/v84/github`
- `github.com/gofri/go-github-ratelimit/v2/github_ratelimit`
- `golang.org/x/oauth2` (transitive via go-github)

## Implementation Waves

### Wave 1: Foundation (no dependencies between items)

**T001: Add dependencies to go.mod**
- `go get github.com/google/go-github/v84`
- `go get github.com/gofri/go-github-ratelimit/v2`
- Verify compilation

**T002: Create `internal/github/options.go`**
- `Option` type and all functional options
- `WithEndpoint`, `WithHTTPClient`, `WithRetryTiming`, `WithRateLimitThreshold`
- Follows Linear client `Option func(*httpClient)` pattern

### Wave 2: Core client (depends on Wave 1)

**T003: Create `internal/github/http_client.go` — struct + constructors**
- `httpClient` struct wrapping `*github.Client`
- `NewClient(token, owner, repo string, opts ...Option) (*httpClient, error)`
- `NewClientFromEnv(owner, repo string, opts ...Option) (*httpClient, error)`
- Internal go-github + rate-limiter transport setup
- Pre-emptive rate limit checker: `checkRateLimit(resp *github.Response)`

**T004: Error mapping helper**
- `mapError(err error) error` — converts go-github errors to `github.Error` types
- Maps `*github.RateLimitError` → `ErrRateLimited`
- Maps `*github.AbuseRateLimitError` → `ErrRateLimited`
- Maps 404 → `ErrNotFound`
- Maps other API errors → `ErrAPI`
- Maps network/context errors → `ErrNetwork`

### Wave 3: Simple methods (depends on Wave 2, all independent of each other)

**T005: `CreatePR`**
- Delegates to `client.PullRequests.Create()`
- Maps `CreatePRInput` → `github.NewPullRequest`
- Returns PR number

**T006: `UpdatePRBody`**
- Delegates to `client.PullRequests.Edit()`
- Passes only `Body` field

**T007: `IsMerged`**
- Delegates to `client.PullRequests.IsMerged()`
- go-github returns `(bool, *Response, error)` — 404 becomes `false`, not error

**T008: `IsClosed`**
- Delegates to `client.PullRequests.Get()`
- Checks `state == "closed" && !merged`

**T009: `ApplyLabel`**
- Delegates to `client.Issues.AddLabelsToIssue()`
- Idempotent by API design

### Wave 4: Paginated + branching methods (depends on Wave 2)

**T010: `GetCommentsSince`**
- Dispatches on `CommentKind`:
  - `CommentKindIssue` → `client.Issues.ListComments()` with `per_page=100`, traditional pagination loop
  - `CommentKindReview` → `client.PullRequests.ListComments()` with `per_page=100`, traditional pagination loop
- Client-side filter: `id > afterID`
- Maps go-github comment types → `PRComment`
- Calls `checkRateLimit(resp)` on each page

**T011: `PostCommentReply`**
- Dispatches on `CommentKind`:
  - `CommentKindIssue` → `client.Issues.CreateComment()` — new top-level comment
  - `CommentKindReview` → `client.PullRequests.CreateCommentInReplyTo()` — threaded reply
- Guard: `CommentKindReview` with `commentID == 0` → return `ErrNotFound`

**T012: `ListOpenPRs`**
- `client.PullRequests.List()` with `State: "open"`, `per_page=100`, traditional pagination loop
- Client-side filter: PR labels contain target label
- Maps `*github.PullRequest` → `PRSummary`
- Calls `checkRateLimit(resp)` on each page

### Wave 5: Tests (depends on Wave 3 + 4)

**T013: Unit tests — `internal/github/http_client_test.go`**
- Use `httptest.NewServer` to mock GitHub API responses
- Point go-github at test server via `WithEndpoint`
- Test each method happy path
- Test error classification (404, 429, 403-rate-limit, 403-permission, 5xx)
- Test pre-emptive rate limit delay
- Test `GetCommentsSince` pagination + ID filtering
- Test `PostCommentReply` kind dispatching + `commentID=0` guard
- Test `ListOpenPRs` label filtering
- Test context cancellation during retry

## File Map

| File | Purpose |
|------|---------|
| `internal/github/client.go` | Interface (exists) |
| `internal/github/errors.go` | Error types (exists) |
| `internal/github/mock.go` | Mock client (exists) |
| `internal/github/options.go` | Functional options (new) |
| `internal/github/http_client.go` | Implementation (new) |
| `internal/github/http_client_test.go` | Tests (new) |

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| go-github test server setup complexity | Linear client tests already demonstrate the httptest pattern |
| `GetCommentsSince` correctness with large comment sets | Test with multi-page mock responses, verify ID boundary |
| Rate limiter transport interaction with test server | Use `WithHTTPClient` to bypass rate limiter in unit tests |
