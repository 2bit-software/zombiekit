# Research Summary: Implement GitHubClient

## Codebase Patterns

### Linear Client as Reference Implementation

The existing `internal/linear/http_client.go` establishes the pattern:

- **Struct**: unexported `httpClient` with fields: token, endpoint, `*http.Client`, lastHeaders, retry timing
- **Constructors**: `NewClient(token string, opts ...Option)` with validation + `NewClientFromEnv(opts ...Option)` reading `BRAINS_LINEAR_API_KEY`
- **Functional options**: `WithEndpoint`, `WithHTTPClient`, `WithRetryTiming`
- **Request flow**: `do()` for single request, `doWithRetry()` wrapping with exponential backoff
- **Retry**: Only on `IsRateLimited()` errors. Max 3 retries. Exponential backoff (1s base, 2x multiplier) + jitter (500ms max). Respects reset headers.
- **Error classification**: HTTP status -> `ErrorKind` mapping. Already defined in `internal/github/errors.go`.
- **Testing**: `httptest.NewServer` with `WithEndpoint(srv.URL)` + `WithRetryTiming(10ms, 5ms)` for fast tests

### Env Var Convention

Project uses `BRAINS_` prefix. GitHub token should be `BRAINS_GITHUB_TOKEN`.

### No External GitHub Library in go.mod

No `google/go-github` currently — clean decision point.

## GitHub REST API Mapping

### Endpoint Summary

| Method | HTTP | Endpoint | Notes |
|--------|------|----------|-------|
| CreatePR | POST | `/repos/{o}/{r}/pulls` | Returns `number` |
| UpdatePRBody | PATCH | `/repos/{o}/{r}/pulls/{n}` | Partial update, body only |
| GetCommentsSince (issue) | GET | `/repos/{o}/{r}/issues/{n}/comments` | Paginated, flat comments |
| GetCommentsSince (review) | GET | `/repos/{o}/{r}/pulls/{n}/comments` | Paginated, diff-attached |
| PostCommentReply (issue) | POST | `/repos/{o}/{r}/issues/{n}/comments` | New top-level comment |
| PostCommentReply (review) | POST | `/repos/{o}/{r}/pulls/{n}/comments/{id}/replies` | Must reference top-level comment |
| ApplyLabel | POST | `/repos/{o}/{r}/issues/{n}/labels` | Idempotent (re-add is no-op) |
| IsMerged | GET | `/repos/{o}/{r}/pulls/{n}/merge` | 204=merged, 404=not |
| IsClosed | GET | `/repos/{o}/{r}/pulls/{n}` | Check `state=="closed" && !merged` |
| ListOpenPRs | GET | `/repos/{o}/{r}/issues?state=open&labels=X` | Issues API; filter `IsPullRequest()` |

### Key Design Decisions

**1. GetCommentsSince — ID filtering gap**

The GitHub API only supports `since` (timestamp) filtering, not ID-based. The interface specifies `afterID int64`. Implementation must paginate all comments and filter client-side where `id > afterID`. Comment IDs are monotonically increasing, so this is reliable.

**2. ListOpenPRs — label filtering gap**

The Pulls API has no label filter parameter. Three options:
- **(A) Issues API**: `GET /issues?labels=X&state=open` + filter `IsPullRequest()`. Efficient but Issues response lacks `head`/`base` branch info — requires follow-up GET per PR.
- **(B) Pulls API**: `GET /pulls?state=open` + client-side label filter. Gets full PR data but fetches all open PRs regardless of label.
- **(C) Search API**: `GET /search/issues?q=is:pr+is:open+label:X`. Full data but stricter rate limit (30/min vs 5000/hour).

**3. Library choice: `google/go-github` vs hand-rolled**

| Aspect | go-github | Hand-rolled |
|--------|-----------|-------------|
| Typed structs | All GitHub types pre-defined | Must define response types |
| Pagination | `ListOptions` + `Response.NextPage` | Manual `Link` header parsing |
| Rate limit info | `Response.Rate` struct | Manual header parsing |
| Auth | `oauth2.StaticTokenSource` | Manual `Authorization` header |
| Error types | `RateLimitError`, `AbuseRateLimitError` | Manual HTTP status mapping |
| Dep count | +1 library + oauth2 | 0 new deps |
| Pattern consistency | Different from Linear client | Matches Linear pattern exactly |

### Rate Limiting

- **Primary**: 5,000 requests/hour (PAT). Headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset` (epoch seconds)
- **Secondary**: Abuse detection on rapid requests. Returns 429 or 403 with `Retry-After` header.
- **Pre-emptive**: Check `X-RateLimit-Remaining` after each response; slow down when below threshold (e.g., 10).
- **Companion library**: `gofri/go-github-ratelimit` handles secondary limits as HTTP transport middleware.

### Token Scopes

- **Classic PAT**: `repo` scope covers all 8 operations on private repos
- **Fine-grained PAT**: `Pull requests: write` + `Issues: write` on target repo
