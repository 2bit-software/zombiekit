# Research Summary: GitHubClient Interface Design

## Codebase Research

### LinearClient Pattern (internal/linear/)

The LinearClient is the direct precedent. Key patterns:

**Interface** (`client.go`):
- All methods accept `context.Context` first
- Returns `(T, error)` or `error`
- Complex inputs use typed structs (`CreateTicketInput`, `AttachmentInput`)
- Optional fields use pointers (`Priority *int`)

**Errors** (`errors.go`):
- `ErrorKind` int enum: `ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork`
- `Error` struct with `Kind`, `Message`, `Err` fields
- Implements `Error()` and `Unwrap()` for Go error chain
- Constructor functions: `NewNotFoundError(msg, cause)`
- Predicate functions: `IsNotFound(err)` using `errors.As()`

**Mock** (`mock.go`):
- Compile-time check: `var _ Client = (*MockClient)(nil)`
- Per-method function fields: `GetTicketFn`, `ApplyLabelFn`, etc.
- Call recording: `Calls []Call` with `Call{Method string, Args []any}`
- Context excluded from recorded args
- Unconfigured methods return error

**Testing** (`http_client_test.go`):
- `testify/assert` and `testify/require`
- `httptest.Server` for HTTP stubbing
- Helper: `newTestClient(t, handler)` returns `(*httpClient, *httptest.Server)`
- Fast retry timing injected via options

### State Store Integration (internal/state/)

- Jobs track PRs via `PRNumber *int64`
- Comment watermarks: `GetCommentWatermark(prNumber int64) (int64, error)`
- Watermark stored as `int64` comment ID

### Callback Server Integration (internal/callback/)

- Events reference `CommentID string` and `Branch string`
- `EventComplete`, `EventCommentResolved`, `EventFailed` kinds

### Existing GitHub Code (internal/mcp/tools/ghpr/)

- Shells out to `gh` CLI -- different approach than what we're building
- No `google/go-github` dependency in go.mod
- Confirms this will be a new, clean implementation

## Domain Research: GitHub REST API

### Endpoint Mapping

| Operation | Endpoint | Notes |
|-----------|----------|-------|
| CreatePR | `POST /repos/{o}/{r}/pulls` | No labels param; separate call needed |
| UpdatePRBody | `PATCH /repos/{o}/{r}/pulls/{n}` | Partial update, only changed fields |
| GetCommentsSince (issue) | `GET /repos/{o}/{r}/issues/{n}/comments?since={ts}` | `since` filters by `updated_at` |
| GetCommentsSince (review) | `GET /repos/{o}/{r}/pulls/{n}/comments?since={ts}` | Same `since` caveat |
| PostCommentReply (issue) | `POST /repos/{o}/{r}/issues/{n}/comments` | No threading concept |
| PostCommentReply (review) | `POST /repos/{o}/{r}/pulls/{n}/comments/{id}/replies` | Threaded reply |
| ApplyLabel | `POST /repos/{o}/{r}/issues/{n}/labels` | Labels must pre-exist on repo |
| IsMerged | `GET /repos/{o}/{r}/pulls/{n}/merge` | 204=yes, 404=no |
| IsClosed | `GET /repos/{o}/{r}/pulls/{n}` | Check `state` + `merged` fields |
| ListOpenPRs | `GET /repos/{o}/{r}/issues?state=open&labels={l}` | Filter for `pull_request` key |

### Key Domain Insights

1. **PRs are Issues** in GitHub's model. Label ops, issue comments, and listing-by-label all go through the Issues API.

2. **Two comment systems**: Issue comments (top-level PR conversation) and review comments (inline diff). The orchestrator's Watcher 2 likely needs **both** -- issue comments for agent-human coordination, review comments for code feedback.

3. **Watermark pagination**: `since` parameter filters by `updated_at`, not creation order. Correct strategy: store `(last_id, last_timestamp)`, query with `since=timestamp`, filter `id > last_id` client-side.

4. **Label filtering**: Pulls endpoint has no label filter. Use Issues endpoint + filter for `pull_request` key presence.

5. **Merge check is elegant**: Pure 204/404 status code, no body parsing.

6. **IsMerged + IsClosed optimization**: Single PR fetch gives `state` and `merged` fields, deriving both states. Could combine into one `GetPRStatus` method.

7. **Rate limits**: 5000 req/hr authenticated. Headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset` (Unix epoch), `Retry-After` (seconds, on 403).

8. **Auth scopes**: `repo` scope (classic PAT) or `pull_requests: write` + `issues: write` + `contents: read` (fine-grained).

### Design Decisions for Interface

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Comment type for GetCommentsSince | Both issue + review | Orchestrator needs both for Watcher 2 |
| Comment ID type | `int64` | Matches state store watermark type |
| ListOpenPRs approach | Issues API + filter | Only way to filter by label |
| IsMerged/IsClosed | Separate methods per spec | Matches ticket requirements; impl can optimize |
| Label pre-existence | Caller's responsibility | Keeps interface simple, matches Linear pattern |
