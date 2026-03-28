# Implementation Plan: DEV-157 Linear Ticket Polling and Read Operations

**Spec**: [spec.md](./spec.md)
**Research**: [research.md](./research.md)
**Technical Constraints**: [technical-requirements-research.md](./technical-requirements-research.md)

## Overview

Implement a real HTTP `Client` for the Linear GraphQL API, covering `PollReadyTickets` and `GetTicket`. All other interface methods return "not implemented" errors. The implementation lives in `internal/linear/` alongside the existing interface, types, errors, and mock.

## File Plan

| File | Action | Purpose |
|------|--------|---------|
| `internal/linear/http_client.go` | Create | Real HTTP implementation of `Client` interface |
| `internal/linear/http_client_test.go` | Create | Unit tests using `httptest.Server` |
| `internal/linear/http_client_integration_test.go` | Create | Integration tests behind `//go:build integration` |

No existing files are modified.

## Implementation Steps

### Step 1: Constructor and GraphQL Transport (FR-004, FR-005)

Create the `httpClient` struct and constructors.

```go
type httpClient struct {
    apiKey     string
    endpoint   string
    httpClient *http.Client
}
```

- `NewClient(apiKey string, opts ...Option) (*httpClient, error)` -- validates key non-empty, default endpoint `https://api.linear.app/graphql`, default `http.Client` with 30s timeout
- `NewClientFromEnv() (*httpClient, error)` -- reads `BRAINS_LINEAR_API_KEY`, calls `NewClient`
- `Option` functional options: `WithEndpoint(url)` (for httptest), `WithHTTPClient(c)`

**FR trace**: FR-004 (auth), FR-005 (fail on missing key)

### Step 2: GraphQL Request/Response Layer (FR-008)

Internal `do(ctx, query, variables, target)` method:

1. Marshal `{"query": query, "variables": variables}` to JSON
2. Create POST request to `endpoint` with `Authorization: <apiKey>` header and `Content-Type: application/json`
3. Execute with context
4. Read response body
5. Parse into generic GraphQL response:
   ```go
   type graphqlResponse struct {
       Data   json.RawMessage `json:"data"`
       Errors []graphqlError  `json:"errors"`
   }
   type graphqlError struct {
       Message    string `json:"message"`
       Extensions struct {
           Code string `json:"code"`
       } `json:"extensions"`
   }
   ```
6. Error mapping:
   - HTTP 401 → `NewAPIError("unauthorized", nil)`
   - HTTP 400 + `extensions.code == "RATELIMITED"` → `NewRateLimitedError(...)`
   - GraphQL errors with "not found" in message → `NewNotFoundError(...)`
   - HTTP 5xx or non-JSON body → `NewNetworkError(...)`
   - Connection/DNS/timeout → `NewNetworkError(..., err)`
   - `ctx.Err() != nil` → `NewNetworkError("request cancelled", ctx.Err())`
7. Unmarshal `data` into `target`

**FR trace**: FR-008 (error mapping), FR-009 (context)

### Step 3: Retry Wrapper (FR-006, FR-007)

Internal `doWithRetry(ctx, query, variables, target)` method:

1. Call `do()`, check error
2. If `IsRateLimited(err)`:
   - Check `X-RateLimit-Requests-Reset` header → compute `time.Until(resetTime)`
   - Otherwise: exponential backoff (1s base, 2x multiplier, random jitter 0-500ms)
   - Max 3 retries
   - Sleep with context awareness (`select` on `ctx.Done()` and timer)
3. If not rate limited or retries exhausted: return result/error

Note: The `do()` method needs to return the response headers alongside errors for reset time extraction. Consider returning a `doResult` or storing last response headers on the struct.

**FR trace**: FR-006 (retry), FR-007 (exhaustion error)

### Step 4: PollReadyTickets (FR-001, FR-002)

```go
func (c *httpClient) PollReadyTickets(ctx context.Context, label string) ([]Ticket, error)
```

1. Build GraphQL query with `issues(filter: { labels: { name: { eq: $label } }, description: { null: false } }, first: 50, after: $after)`
2. Loop: call `doWithRetry`, append results, check `pageInfo.hasNextPage`, update `after` cursor
3. Client-side filter: exclude tickets where `len(Description) == 0`
4. Map GraphQL response nodes to `[]Ticket`:
   - `Priority`: truncate float to int
   - `Status`: extract from `state.name`
   - `Labels`: flatten `labels.nodes[].name`

**FR trace**: FR-001 (query + paginate), FR-002 (client-side filter)

### Step 5: GetTicket (FR-003)

```go
func (c *httpClient) GetTicket(ctx context.Context, id string) (*Ticket, error)
```

1. Build GraphQL query with `issue(id: $id)`
2. Call `doWithRetry`
3. Map response to `*Ticket` (same field mapping as Step 4)
4. If GraphQL returns not-found error → already mapped by Step 2

**FR trace**: FR-003

### Step 6: Unimplemented Stubs (FR-010)

Implement remaining `Client` interface methods:
- `SetTicketStatus` → `return fmt.Errorf("SetTicketStatus: not implemented")`
- `ApplyLabel` → same pattern
- `RemoveLabel` → same pattern
- `CreateTicket` → `return nil, fmt.Errorf("CreateTicket: not implemented")`
- `UploadAttachment` → same pattern

**FR trace**: FR-010

## Dependency Order

```
Step 1 (constructor) ──► Step 2 (GraphQL transport) ──► Step 3 (retry) ──► Step 4 (poll)
                                                                       └──► Step 5 (get)
Step 6 (stubs) has no dependencies
```

Steps 4 and 5 can be implemented in parallel after Step 3. Step 6 is independent.

## Test Plan

### Unit Tests (httptest.Server)

Each test creates an `httptest.Server` that simulates Linear's API:

| Test | Simulates | Verifies |
|------|-----------|----------|
| `TestNewClient_MissingAPIKey` | N/A | Returns error for empty key |
| `TestNewClientFromEnv` | N/A | Reads env var, creates client |
| `TestPollReadyTickets_Success` | Returns 2 matching issues | Correct Ticket field mapping |
| `TestPollReadyTickets_EmptyResult` | Returns empty nodes | Returns `[]Ticket{}`, no error |
| `TestPollReadyTickets_FiltersEmptyDescription` | Returns issue with `""` description | Excluded from results |
| `TestPollReadyTickets_Pagination` | Page 1 has `hasNextPage: true`, page 2 has results | Aggregates both pages |
| `TestPollReadyTickets_AuthHeader` | Inspects request header | `Authorization: <key>`, no Bearer |
| `TestGetTicket_Success` | Returns full issue | Correct Ticket field mapping |
| `TestGetTicket_NotFound` | Returns GraphQL not-found error | `IsNotFound(err) == true` |
| `TestRetry_RateLimitThenSuccess` | 400+RATELIMITED first, 200 second | Returns success transparently |
| `TestRetry_RateLimitExhausted` | 400+RATELIMITED on all attempts | `IsRateLimited(err) == true` |
| `TestRetry_UsesResetHeader` | 400 with X-RateLimit-Requests-Reset | Waits until reset time |
| `TestDo_HTTPError500` | Returns 500 | `IsNetworkError(err) == true` |
| `TestDo_NonJSONResponse` | Returns HTML | `IsNetworkError(err) == true` |
| `TestDo_ContextCancelled` | Slow response + cancelled ctx | Error wraps `context.Canceled` |
| `TestDo_Unauthorized` | Returns 401 | `IsAPIError(err) == true` |
| `TestUnimplemented_Methods` | N/A | All return "not implemented" error |

### Integration Tests (//go:build integration)

Require `BRAINS_LINEAR_API_KEY` env var:

| Test | Does |
|------|------|
| `TestIntegration_PollReadyTickets` | Calls real API, verifies non-error response shape |
| `TestIntegration_GetTicket` | Fetches known ticket, verifies fields populated |
| `TestIntegration_GetTicket_NotFound` | Fetches nonexistent ID, verifies NotFoundError |

## Design Decisions

1. **Single file** (`http_client.go`): All production code in one file. The implementation is ~200-250 lines -- splitting would be premature.
2. **Functional options**: `WithEndpoint` enables httptest injection without global state. `WithHTTPClient` enables custom transports.
3. **No pagination limit**: Poll fetches all pages. If this becomes a problem (hundreds of ai-ready tickets), add a configurable max later.
4. **Retry in transport layer**: Retry wraps the GraphQL `do()` method, not individual business methods. This keeps poll/get logic clean.
5. **Headers stored on struct**: After each `do()` call, store the last response headers so retry logic can read rate limit headers without threading them through return values.
