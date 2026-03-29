# Tasks: Implement GitHubClient

**Complexity**: Medium (5 files, ~900 LOC)
**Critical path**: T001 → T002 → T003 → T004-T009 (parallel) → T010-T013 (parallel)

## Dependency Graph

```
T001 (deps)
  └─ T002 (options)
       └─ T003 (core infra)
            ├─ T004 [P] (CreatePR, UpdatePRBody)
            ├─ T005 [P] (IsMerged, IsClosed)
            ├─ T006 [P] (ApplyLabel)
            ├─ T007 [P] (GetCommentsSince)
            ├─ T008 [P] (PostCommentReply)
            └─ T009 [P] (ListOpenPRs)
                 ├─ T010 [P] (constructor + error tests)
                 ├─ T011 [P] (simple method tests)
                 ├─ T012 [P] (GetCommentsSince tests)
                 └─ T013 [P] (PostCommentReply, ListOpenPRs, retry, rate limit tests)
```

## Tasks

### Wave 1: Foundation

- [ ] **T001** Add go-github v84 and go-github-ratelimit v2 dependencies
  - Files: `go.mod`, `go.sum`
  - Run: `go get github.com/google/go-github/v84` and `go get github.com/gofri/go-github-ratelimit/v2`
  - Verify: `go build ./...` compiles
  - Traces to: NFR-1, NFR-3

- [ ] **T002** Create functional options in `internal/github/options.go`
  - File: `internal/github/options.go`
  - Implement: `Option` type, `WithEndpoint`, `WithHTTPClient`, `WithRetryTiming`, `WithRateLimitThreshold`
  - Follow Linear client pattern (`Option func(*httpClient)`)
  - Reference: technical-spec.md "File: `internal/github/options.go`" section
  - Traces to: FR-1

### Wave 2: Core Infrastructure

- [ ] **T003** Create core client infrastructure in `internal/github/http_client.go`
  - File: `internal/github/http_client.go`
  - Implement:
    - Constants: `defaultTimeout`, `defaultRateLimitThresh`, `maxRetries`, `retryBaseDelay`, `maxJitter`, `maxPreemptiveSleep`
    - `httpClient` struct with fields: `ghClient`, `owner`, `repo`, `endpoint`, `customHTTPClient`, `retryBase`, `retryMaxJitter`, `rateLimitThreshold`
    - Compile guard: `var _ Client = (*httpClient)(nil)`
    - `NewClient(token, owner, repo string, opts ...Option) (*httpClient, error)` — validates inputs, applies options, sets up go-github client with rate-limiter transport (or custom HTTP client), handles `WithEnterpriseURLs` for endpoint override
    - `NewClientFromEnv(owner, repo string, opts ...Option) (*httpClient, error)` — reads `BRAINS_GITHUB_TOKEN`
    - `checkRateLimit(ctx, resp)` — sleeps when `resp.Rate.Remaining < threshold`, capped at 60s
    - `mapError(err)` — maps go-github errors (`RateLimitError`, `AbuseRateLimitError`, `ErrorResponse`) to `github.Error` types
    - `doWithRetry(ctx, op)` — retry wrapper for rate limits + 5xx, max 3 retries, exponential backoff + jitter
    - `isServerError(err)` — checks for 5xx `ErrorResponse`
    - `retryDelay(attempt)` — exponential backoff with jitter
  - Reference: technical-spec.md "Constants and struct", "Constructors", "Pre-emptive rate limit check", "Error mapping", "Retry wrapper" sections
  - Traces to: FR-1, NFR-1, NFR-2, NFR-3, NFR-4, AC-8, AC-9, AC-10, AC-20

### Wave 3: Method Implementations (all parallel, depend on T003)

- [ ] **T004** [P] Implement `CreatePR` and `UpdatePRBody`
  - File: `internal/github/http_client.go`
  - `CreatePR`: delegate to `ghClient.PullRequests.Create()`, map `CreatePRInput` → `gh.NewPullRequest`, return PR number. Error already mapped by `doWithRetry` — do NOT double-map.
  - `UpdatePRBody`: delegate to `ghClient.PullRequests.Edit()`, pass only `Body` field via `gh.Ptr(body)`
  - Both call `checkRateLimit(ctx, resp)` after success
  - Reference: technical-spec.md "CreatePR" and "UpdatePRBody" sections
  - Traces to: FR-2, FR-3, AC-1, AC-2

- [ ] **T005** [P] Implement `IsMerged` and `IsClosed`
  - File: `internal/github/http_client.go`
  - `IsMerged`: delegate to `ghClient.PullRequests.IsMerged()`. Defensive guard: if go-github returns a 404 error, treat as `(false, nil)` per FR-7 instead of propagating `ErrNotFound`.
  - `IsClosed`: delegate to `ghClient.PullRequests.Get()`, check `state == "closed" && !merged`. Single API call — do NOT compose with `IsMerged`.
  - Reference: technical-spec.md "IsMerged" and "IsClosed" sections
  - Traces to: FR-7, FR-8, AC-6, AC-7, AC-16, AC-17, AC-18

- [ ] **T006** [P] Implement `ApplyLabel`
  - File: `internal/github/http_client.go`
  - Delegate to `ghClient.Issues.AddLabelsToIssue()` with `[]string{label}`. Idempotent by API design.
  - Reference: technical-spec.md "ApplyLabel" section
  - Traces to: FR-6, AC-14, AC-15

- [ ] **T007** [P] Implement `GetCommentsSince` with pagination and ID filtering
  - File: `internal/github/http_client.go`
  - Implement dispatcher: `GetCommentsSince` switches on `CommentKind`, delegates to `getIssueCommentsSince` or `getReviewCommentsSince`
  - `getIssueCommentsSince`: paginate `ghClient.Issues.ListComments()` with `per_page=100`, `Sort: "created"`, `Direction: "asc"`. Client-side filter: `id > afterID`. Map `*gh.IssueComment` → `PRComment` (ID, Author=login, Body, CreatedAt).
  - `getReviewCommentsSince`: paginate `ghClient.PullRequests.ListComments()` with same settings. Map `*gh.PullRequestComment` → `PRComment` (+ Path, DiffHunk, InReplyToID).
  - Both use traditional pagination loop (`resp.NextPage`), call `checkRateLimit` per page
  - Note: `IssueListCommentsOptions.Sort` and `Direction` are `*string` (use `gh.Ptr`), but `PullRequestListCommentsOptions.Sort` and `Direction` are plain `string`
  - Reference: technical-spec.md "GetCommentsSince" section
  - Traces to: FR-4, AC-3, AC-4, AC-5

- [ ] **T008** [P] Implement `PostCommentReply` with kind dispatching
  - File: `internal/github/http_client.go`
  - `CommentKindIssue`: delegate to `ghClient.Issues.CreateComment()`, return `comment.GetID()`
  - `CommentKindReview`: guard `commentID == 0` → return `ErrNotFound`. Otherwise delegate to `ghClient.PullRequests.CreateCommentInReplyTo()`, return `comment.GetID()`
  - Unknown kind → return `ErrAPI`
  - Reference: technical-spec.md "PostCommentReply" section
  - Traces to: FR-5, AC-11, AC-12, AC-13

- [ ] **T009** [P] Implement `ListOpenPRs` with client-side label filtering
  - File: `internal/github/http_client.go`
  - Paginate `ghClient.PullRequests.List()` with `State: "open"`, `per_page=100`
  - Client-side filter: `hasLabel(pr.Labels, label)`
  - Map matching PRs → `PRSummary{Number, Title, Head: pr.GetHead().GetRef(), Base: pr.GetBase().GetRef(), Labels: labelNames(pr.Labels)}`
  - Helper functions: `hasLabel(labels []*gh.Label, target string) bool` and `labelNames(labels []*gh.Label) []string`
  - Reference: technical-spec.md "ListOpenPRs" section
  - Traces to: FR-9, AC-19

### Wave 4: Tests (all parallel, depend on Wave 3)

- [ ] **T010** [P] Write unit tests for constructors and error classification
  - File: `internal/github/http_client_test.go`
  - Test helper: `newTestClient(t, handler)` using `httptest.NewServer` + `WithEndpoint` + `WithHTTPClient` + `WithRetryTiming`
  - Constructor tests: empty token → error, empty owner → error, empty repo → error, valid inputs → success
  - `NewClientFromEnv` tests: env var missing → error with message naming `BRAINS_GITHUB_TOKEN` (use `t.Setenv`)
  - `mapError` tests: `nil` → `nil`, 404 response → `ErrNotFound`, `RateLimitError` → `ErrRateLimited`, `AbuseRateLimitError` → `ErrRateLimited`, other `ErrorResponse` → `ErrAPI`, context cancelled → `ErrNetwork`, generic error → `ErrNetwork`
  - Important: go-github's `WithEnterpriseURLs` appends `/api/v3/` — test handlers must register routes at `/api/v3/repos/owner/repo/...`, or set `ghClient.BaseURL` directly
  - Reference: technical-spec.md "Testing Strategy" section
  - Traces to: AC-10

- [ ] **T011** [P] Write unit tests for simple methods
  - File: `internal/github/http_client_test.go`
  - `CreatePR`: mock returns 201 with `{"number": 42}`, verify returned number. Mock returns 422 → `ErrAPI`.
  - `UpdatePRBody`: mock returns 200 with PR JSON, verify no error. Mock returns 404 → `ErrNotFound`.
  - `IsMerged`: mock returns 204 → `true`. Mock returns 404 → `false, nil` (not error). Mock returns 500 → error.
  - `IsClosed`: mock returns PR with `state: "closed", merged: false` → `true`. PR with `state: "closed", merged: true` → `false`. PR with `state: "open"` → `false`.
  - `ApplyLabel`: mock returns 200 with labels array → no error. Call again (idempotent) → no error.
  - Traces to: AC-1, AC-2, AC-6, AC-7, AC-14, AC-15, AC-16, AC-17, AC-18

- [ ] **T012** [P] Write unit tests for `GetCommentsSince`
  - File: `internal/github/http_client_test.go`
  - Issue comments: 3 comments with `afterID=0` → all 3 returned in order (AC-3)
  - Issue comments: 5 comments with `afterID` of comment 3 → only comments 4,5 returned (AC-4)
  - Issue comments: multi-page (set `per_page=2` for test, return 5 comments across 3 pages) → all returned (AC-5). Use `Link` header or `resp.NextPage` mock.
  - Review comments: verify `Path`, `DiffHunk`, `InReplyToID` fields populated
  - ID boundary: comment with `id == afterID` must NOT be included (strict `>`, not `>=`)
  - Traces to: AC-3, AC-4, AC-5

- [ ] **T013** [P] Write unit tests for PostCommentReply, ListOpenPRs, retry, and rate limiting
  - File: `internal/github/http_client_test.go`
  - `PostCommentReply`:
    - Issue kind → mock `POST /issues/{n}/comments` returns 201, verify returned ID (AC-11)
    - Review kind with valid commentID → mock `POST /pulls/{n}/comments/{id}/replies` returns 201 (AC-12)
    - Review kind with `commentID=0` → returns `ErrNotFound` without HTTP call (AC-13)
  - `ListOpenPRs`:
    - 3 open PRs, 2 have target label → only 2 returned with correct fields (AC-19)
    - Multi-page pagination with label filtering
  - Retry tests:
    - 429 response → retries and succeeds on second attempt (AC-9)
    - 404 response → no retry, returns immediately
    - 5xx response → retries (AC-20)
    - Context cancelled during retry → returns `ErrNetwork`
  - Rate limit tests:
    - Response with `Rate.Remaining < threshold` → verify delay occurs (AC-8)
    - Response with `Rate.Remaining >= threshold` → no delay
  - Traces to: AC-8, AC-9, AC-11, AC-12, AC-13, AC-19, AC-20

## Traceability Matrix

| Requirement | Tasks |
|-------------|-------|
| FR-1 | T002, T003 |
| FR-2 | T004 |
| FR-3 | T004 |
| FR-4 | T007 |
| FR-5 | T008 |
| FR-6 | T006 |
| FR-7 | T005 |
| FR-8 | T005 |
| FR-9 | T009 |
| NFR-1 | T001, T003 |
| NFR-2 | T003 |
| NFR-3 | T003 |
| NFR-4 | T003 |
| AC-1..AC-20 | T010-T013 |

## Execution Summary

- **Total tasks**: 13
- **Sequential tasks**: 3 (T001 → T002 → T003)
- **Parallel opportunities**: Wave 3 (6 tasks) + Wave 4 (4 tasks)
- **Max parallelism**: 6 (Wave 3)
- **Critical path length**: 5 steps (T001 → T002 → T003 → any Wave 3 → any Wave 4)
