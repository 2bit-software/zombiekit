---
status: complete
updated: 2026-03-27
---

# Research: Linear Ticket Polling and Read Operations

## Executive Summary

Linear's GraphQL API at `https://api.linear.app/graphql` supports both required operations with single queries each. No usable Go client library exists -- hand-rolling with `net/http` is standard practice. The existing `Client` interface and error types in `internal/linear/` align cleanly with the API's response shapes. Key gotchas: API key auth uses bare `Authorization: <key>` (no Bearer prefix), and rate limiting returns HTTP 400 (not 429) with a `RATELIMITED` error code in the response body.

## Findings

### Codebase Context

- **Existing interface**: `Client` in `internal/linear/client.go` already defines `PollReadyTickets(ctx, label) ([]Ticket, error)` and `GetTicket(ctx, id) (*Ticket, error)`
- **Error types**: `internal/linear/errors.go` provides `ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork` with constructors and predicates -- ready to use
- **Mock**: `internal/linear/mock.go` has full mock with call recording -- consumer tests are covered
- **HTTP patterns**: Codebase uses `net/http` directly (see `internal/recall/embedder.go`), no HTTP client abstraction library
- **Config pattern**: Env vars loaded via `os.Getenv()` with defaults (see `internal/config/storage.go`), naming convention `BRAINS_*`
- **No GraphQL**: Zero GraphQL usage in the codebase -- this is the first integration
- **Dependencies**: Go 1.24.1, no GraphQL library in go.mod

### Domain Knowledge

**Authentication:**
- Endpoint: `https://api.linear.app/graphql`
- API key auth: `Authorization: <API_KEY>` (no "Bearer" prefix)
- OAuth auth: `Authorization: Bearer <TOKEN>` (with prefix)

**PollReadyTickets query:**
```graphql
query($label: String!, $after: String) {
  issues(
    filter: {
      labels: { name: { eq: $label } }
      description: { null: false }
    }
    first: 50
    after: $after
  ) {
    nodes { id identifier title description url priority state { name } labels { nodes { name } } }
    pageInfo { hasNextPage endCursor }
  }
}
```

**GetTicket query:**
```graphql
query($id: String!) {
  issue(id: $id) {
    id identifier title description url priority
    state { name }
    labels { nodes { name } }
  }
}
```

The `id` parameter accepts both UUIDs and human-readable identifiers (e.g., "DEV-157").

**Rate limiting:**
- HTTP 400 (not 429) with `RATELIMITED` error code in response body
- Leaky bucket: 5,000 requests/hour, 3,000,000 complexity/hour for API key auth
- Headers: `X-RateLimit-Requests-Remaining`, `X-RateLimit-Requests-Reset` (UTC epoch ms), etc.

**Filtering:**
- Multiple fields at same level are implicitly ANDed
- `description: { null: false }` checks for non-null (not non-empty string)
- Linear UI typically sets description to null when empty, but client-side `len > 0` is cheap insurance

## Decision Points

- [x] **D1**: Use existing Go Linear client library vs hand-roll? → **Hand-roll.** Only community library (`guillermo/linear`) has 3 stars, 1 commit, v0.0.0. Not production-worthy.
- [x] **D2**: Add a GraphQL client library? → **No.** Two queries don't justify the dependency. `net/http` + `encoding/json` suffice.
- [x] **D3**: Combined server-side filter or client-side post-filter? → **Both.** Server-side `labels + description: { null: false }` filter, plus client-side `len(description) > 0` safety net.
- [ ] **D4**: Env var name for API key → Suggest `BRAINS_LINEAR_API_KEY` to match existing `BRAINS_*` convention

## Recommendations

1. Create a `linearClient` struct in `internal/linear/` implementing the existing `Client` interface (only `PollReadyTickets` and `GetTicket` for this ticket)
2. Internal `graphqlDo(ctx, query, variables, target)` helper (~50 lines) for auth, POST, error parsing, rate limit detection
3. Map GraphQL errors to existing `Error` types: not-found → `ErrNotFound`, rate limited → `ErrRateLimited`, server errors → `ErrAPI`, connection failures → `ErrNetwork`
4. Exponential backoff with jitter for rate limit retries, configurable max retries
5. Integration test behind `//go:build integration` tag against real Linear API

## Sources

- [Linear Developers - Getting Started](https://linear.app/developers/graphql)
- [Linear Developers - Filtering](https://linear.app/developers/filtering)
- [Linear Developers - Rate Limiting](https://linear.app/developers/rate-limiting)
- [Linear GraphQL Schema (GitHub)](https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql)
- [guillermo/linear on pkg.go.dev](https://pkg.go.dev/github.com/guillermo/linear/linear-api)
