# Business Specification: Implement GitHubClient

## Overview

Implement a real HTTP client satisfying the `github.Client` interface (defined in DEV-187) to perform PR lifecycle operations against the GitHub REST API. This client is the runtime counterpart to the existing `MockClient` and will be used by the orchestrator to manage pull requests autonomously.

## Actors

- **Orchestrator**: The automated system that calls the GitHubClient to create, update, and observe pull requests
- **Human reviewer**: Merges PRs and leaves comments — the client only observes these actions, never performs them

## Functional Requirements

### FR-1: Client Construction

The client must be constructed with:
- A GitHub personal access token (required, non-empty)
- A repository owner and name (required — the client operates on a single repo)
- An HTTP client (optional, for testing)
- A base URL (optional, for GitHub Enterprise or testing)

Constructor signatures (concrete type is unexported `httpClient`):
```
NewClient(token, owner, repo string, opts ...Option) (*httpClient, error)
NewClientFromEnv(owner, repo string, opts ...Option) (*httpClient, error)
```

Functional options follow the Linear client pattern: `WithEndpoint(url)`, `WithHTTPClient(hc)`, `WithRetryTiming(base, maxJitter)`.

`NewClientFromEnv` reads the token from `BRAINS_GITHUB_TOKEN`. If the variable is unset or empty, construction must fail with a clear error message naming the variable.

### FR-2: Create Pull Request

Given a title, body, head branch, and base branch, the client creates a PR on GitHub and returns its number. If the head branch does not exist or the base branch is invalid, the error must be classified appropriately.

### FR-3: Update PR Description

Given a PR number and a new body string, the client replaces the PR's description. Only the body is updated — title and other fields are untouched.

### FR-4: Get Comments Since Watermark

Given a PR number, a comment kind (issue or review), and a watermark ID (`afterID`):
- Returns all comments with `id > afterID`, in chronological order
- When `afterID` is 0, returns all comments
- Transparently handles pagination — the caller receives the complete list regardless of how many pages the API requires
- For review comments, each result includes the file path, diff hunk, and reply-to ID
- All `PRComment` fields must be populated: ID, Author (GitHub login), Body, CreatedAt

**Implementation constraint**: The GitHub API only supports `since` (timestamp) filtering, not ID-based filtering. The implementation must paginate through all comments (using `per_page=100`) and filter client-side to return only those with `id > afterID`. Do not use the `since` query parameter — always paginate from the beginning and filter by ID. Comment IDs are monotonically increasing per-endpoint.

This is the highest-precision requirement. An off-by-one error here causes duplicate or skipped comment processing in the downstream watcher.

### FR-5: Post Comment Reply

Given a PR number, comment kind, comment ID, and body text:
- For **issue comments**: Posts a new top-level comment on the PR. The `commentID` parameter is recorded for caller context but not used in the API call.
- For **review comments**: Posts a threaded reply to the specified review comment. `commentID` must be non-zero; if zero, return `ErrNotFound`.

Returns the ID of the newly created comment.

### FR-6: Apply Label

Given a PR number and label name, adds the label to the PR. Must be idempotent — applying a label that already exists must not error.

### FR-7: Check Merge Status

Given a PR number, returns whether the PR has been merged (`true`/`false`). Must not return an error for unmerged PRs — a `false` return is the expected response, not a 404 error.

### FR-8: Check Closed Status

Given a PR number, returns whether the PR is closed without being merged. Returns `false` for both open PRs and merged PRs. Implement via a single `GET /pulls/{n}` call checking `state == "closed" && merged == false` — do not compose with `IsMerged`.

### FR-9: List Open PRs by Label

Given a label name, returns all open PRs carrying that label. Each result includes the PR number, title, head branch, base branch, and all labels.

## Non-Functional Requirements

### NFR-1: Rate Limit Handling

The client must handle GitHub API rate limits at two levels:

**Pre-emptive slowdown**: After each API response, read `X-RateLimit-Remaining`. When the value drops below a configurable low-water threshold (default: 10), sleep until `X-RateLimit-Reset` before the next request. Cap the sleep at 60 seconds — if the reset is further away, proceed and let the reactive backoff handle it. The threshold is configurable via a functional option.

**Reactive backoff**: On HTTP 429, or 403 with `Retry-After` header or "rate limit" in response body:
- Read the `Retry-After` or `X-RateLimit-Reset` header
- Wait the indicated duration (or exponential backoff if no header)
- Retry the request
- Maximum 3 retries before surfacing a rate-limit error to the caller
- A 403 without rate-limit indicators is classified as `ErrAPI`, not retried

**Transient server errors**: 5xx responses (502, 503, etc.) should be retried with the same exponential backoff strategy as rate-limit responses, up to the same maximum retry count.

### NFR-2: Error Classification

All errors returned by the client must be classified using the existing `github.Error` types:
- `ErrNotFound`: 404 responses
- `ErrRateLimited`: 429 or rate-limit 403 responses
- `ErrAPI`: Other 4xx/5xx responses (401, 422, 500, etc.)
- `ErrNetwork`: Connection failures, timeouts, context cancellation

### NFR-3: Authentication

The client authenticates via `Authorization: Bearer <token>` header on every request. All requests must include `Accept: application/vnd.github+json`. The token requires `repo` scope (classic PAT) or `Pull requests: write` + `Issues: write` (fine-grained PAT).

### NFR-4: Context Propagation

All methods accept `context.Context` and must respect cancellation and timeouts — including during retry waits.

## Scope Exclusions

- Webhook handling — polling only for now
- GitHub Actions or CI integration
- PR merge operations — the human merges, the orchestrator only observes
- Creating repository labels — labels are assumed to exist
- GraphQL API — REST only

## Acceptance Criteria

| ID | Given | When | Then |
|----|-------|------|------|
| AC-1 | Valid token and repo | `CreatePR` called with branch and body | PR created, number returned |
| AC-2 | PR number and new body | `UpdatePRBody` called | PR description updated |
| AC-3 | PR with 3 comments | `GetCommentsSince` with `afterID=0` | All 3 comments returned chronologically |
| AC-4 | PR with 5 comments, watermark after #3 | `GetCommentsSince` with `afterID` of comment 3 | Only comments 4 and 5 returned |
| AC-5 | PR with >1 page of comments | `GetCommentsSince` called | All pages fetched, full list returned |
| AC-6 | Merged PR | `IsMerged` called | Returns `true` |
| AC-7 | Open PR | `IsMerged` called | Returns `false` |
| AC-8 | `X-RateLimit-Remaining` below threshold | Any method called | Client delays before making request |
| AC-9 | 429 response | Any method called | Client backs off and retries |
| AC-10 | No `BRAINS_GITHUB_TOKEN` env var | Client initialized via `NewClientFromEnv` | Initialization fails with clear error |
| AC-11 | Existing PR | `PostCommentReply` with `CommentKindIssue` | New issue comment created, ID returned |
| AC-12 | Existing review comment | `PostCommentReply` with `CommentKindReview` and valid `commentID` | Reply posted in thread, ID returned |
| AC-13 | `CommentKindReview` with `commentID=0` | `PostCommentReply` called | Returns `ErrNotFound` |
| AC-14 | PR without label | `ApplyLabel` called | Label added, no error |
| AC-15 | PR already has the label | `ApplyLabel` called again | No error returned |
| AC-16 | PR closed without merge | `IsClosed` called | Returns `true` |
| AC-17 | Merged PR | `IsClosed` called | Returns `false` |
| AC-18 | Open PR | `IsClosed` called | Returns `false` |
| AC-19 | Multiple open PRs with label | `ListOpenPRs` called | All matching PRs returned with number, title, head, base, labels |
| AC-20 | 5xx response from GitHub | Any method called | Client retries with backoff before surfacing error |

## Design Decisions (Resolved)

### DD-1: Library Choice — Hybrid (Option C)

Use `google/go-github` internally for typed structs, pagination, and rate-limit parsing. Wrap in the same functional-options constructor pattern as the Linear client (`NewClient`, `WithEndpoint`, `WithHTTPClient`, etc.) for API consistency across the codebase.

### DD-2: ListOpenPRs — Pulls API + client-side label filter (Option B)

Fetch all open PRs via `GET /pulls?state=open`, paginate, filter by label client-side. Gets full PR data (including head/base branches) without follow-up calls. Webhook integration will eventually replace this polling approach.

### DD-3: Rate Limiting — HTTP transport middleware (Option B)

Use `gofri/go-github-ratelimit` as the HTTP transport layer for `google/go-github`. Handles secondary rate limits (abuse detection) with automatic sleep+retry. Primary rate limit handling (pre-emptive slowdown from `X-RateLimit-Remaining`) is implemented in the client's `do()` wrapper using `Response.Rate` from go-github.
