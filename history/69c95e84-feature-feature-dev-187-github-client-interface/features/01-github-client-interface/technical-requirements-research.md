# Technical Requirements & Implementation Hints

## Source

Extracted from DEV-187 ticket and DEV-188 (follow-on) context.

## Interface Design Constraints

- Interface is the **mock boundary** for orchestrator-core (Epic 3, DEV-150)
- Design the interface before the implementation -- the interface is the primary deliverable
- Must follow existing LinearClient patterns exactly (see research-summary.md)

## Existing Codebase Patterns to Follow

### Package Structure

```
internal/github/
  client.go              # Interface + domain types
  errors.go              # Error types + predicates
  mock.go                # MockClient with call recording
  mock_test.go           # Mock verification tests
```

Note: `http_client.go` and integration tests are DEV-188 scope, not this ticket.

### Error Pattern

Use typed errors with `ErrorKind` enum + predicate functions, matching `internal/linear/errors.go`:

```go
type ErrorKind int
const (
    ErrNotFound    ErrorKind = iota + 1
    ErrRateLimited
    ErrAPI
    ErrNetwork
)
```

With `Error()`, `Unwrap()`, `NewXyzError()` constructors, and `IsXyz()` predicates.

### Mock Pattern

Match `internal/linear/mock.go`:
- Compile-time assertion: `var _ Client = (*MockClient)(nil)`
- One `*Fn` field per interface method
- `Calls []Call` recording (context stripped from recorded args)
- Default to error if function not configured

### Method Signature Pattern

Match LinearClient conventions:
- All methods take `ctx context.Context` as first parameter
- Return `(T, error)` or just `error`
- Complex inputs bundled in typed structs (e.g., `CreatePRInput`)

## GitHub API Notes (for DEV-188 implementation)

- PRs are Issues in GitHub's data model -- label and comment operations use the Issues API
- The Pulls list endpoint has no label filter; ListOpenPRs must use Issues API + client-side filter
- `GetCommentsSince` watermark uses `since` timestamp param + client-side `id > watermark` filter
- `IsMerged` check: `GET .../pulls/{n}/merge` returns 204 (merged) or 404 (not merged)
- CreatePR endpoint does not accept labels -- requires separate ApplyLabel call
- Auth: `GITHUB_TOKEN` env var, `repo` scope for private repos
- Rate limit: `X-RateLimit-Remaining` + `X-RateLimit-Reset` headers on every response

## Integration Points

- **State store** (`internal/state/`): Jobs reference PRs via `PRNumber *int64`
- **Callback server** (`internal/callback/`): Events reference `CommentID string`
- **Orchestrator** (DEV-150): Will call GitHubClient methods for PR lifecycle management
