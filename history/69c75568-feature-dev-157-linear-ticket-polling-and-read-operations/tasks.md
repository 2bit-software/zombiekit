# Tasks: DEV-157 Linear Ticket Polling and Read Operations

**Plan**: [implementation-plan.md](./implementation-plan.md)
**Spec**: [spec.md](./spec.md)
**Complexity**: Simple (3 new files, 0 existing files modified)

## Dependency Graph

```
T001 ──► T002 ──► T003 [P]──► T006 ──► T007
  │                     [P]──► T004
  └──► T005 [P]────────────────┘
```

## Tasks

- [ ] T001 [US4] Constructor, options, and GraphQL transport layer — `internal/linear/http_client.go`
  - Create `httpClient` struct with `apiKey`, `endpoint`, `httpClient` fields
  - Implement `Option` type: `WithEndpoint(url)`, `WithHTTPClient(c)`
  - Implement `NewClient(apiKey string, opts ...Option) (*httpClient, error)` — validates key non-empty, defaults endpoint to `https://api.linear.app/graphql`, defaults `http.Client` with 30s timeout
  - Implement `NewClientFromEnv() (*httpClient, error)` — reads `BRAINS_LINEAR_API_KEY`
  - Define internal response types: `graphqlRequest`, `graphqlResponse`, `graphqlError`
  - Implement `do(ctx, query, variables, target) error` — POST with `Authorization: <apiKey>` (no Bearer), parse response, map errors per FR-008
  - **Acceptance**: `NewClient("key")` succeeds; `NewClient("")` returns error; `do()` sends correct headers and parses responses
  - **FR trace**: FR-004, FR-005, FR-008, FR-009

- [ ] T002 Retry wrapper with exponential backoff — `internal/linear/http_client.go`
  - Implement `doWithRetry(ctx, query, variables, target) error`
  - On `RateLimitedError`: check `X-RateLimit-Requests-Reset` header for reset time, fall back to exponential backoff (1s base, 2x multiplier, 0-500ms jitter)
  - Max 3 retries, context-aware sleep via `select` on `ctx.Done()` + timer
  - Note: `do()` must expose last response headers (store on struct or return via result)
  - **Acceptance**: Rate-limited call retries and succeeds on next attempt; exhausted retries return `RateLimitedError`
  - **FR trace**: FR-006, FR-007
  - **Depends on**: T001

- [ ] T003 [P] [US1] PollReadyTickets implementation — `internal/linear/http_client.go`
  - GraphQL query: `issues(filter: { labels: { name: { eq: $label } }, description: { null: false } }, first: 50, after: $after)`
  - Paginate: loop until `pageInfo.hasNextPage` is false
  - Client-side filter: exclude tickets where `len(Description) == 0`
  - Map response: `id`→`ID`, `identifier`→`Identifier`, `title`→`Title`, `description`→`Description`, `state.name`→`Status`, `labels.nodes[].name`→`Labels`, `priority`(float→int)→`Priority`, `url`→`URL`
  - **Acceptance**: Returns correct `[]Ticket` for matching issues; empty slice for no matches; excludes empty-description tickets; paginates beyond first page
  - **FR trace**: FR-001, FR-002
  - **Depends on**: T002

- [ ] T004 [P] [US2] GetTicket implementation — `internal/linear/http_client.go`
  - GraphQL query: `issue(id: $id)` — accepts both UUID and identifier strings
  - Same field mapping as T003
  - Not-found error already handled by `do()` error mapping
  - **Acceptance**: Returns correct `*Ticket` for valid identifier; returns `NotFoundError` for nonexistent ID
  - **FR trace**: FR-003
  - **Depends on**: T002

- [ ] T005 [P] Unimplemented method stubs — `internal/linear/http_client.go`
  - `SetTicketStatus` → `fmt.Errorf("SetTicketStatus: not implemented")`
  - `ApplyLabel` → `fmt.Errorf("ApplyLabel: not implemented")`
  - `RemoveLabel` → `fmt.Errorf("RemoveLabel: not implemented")`
  - `CreateTicket` → `return nil, fmt.Errorf("CreateTicket: not implemented")`
  - `UploadAttachment` → `fmt.Errorf("UploadAttachment: not implemented")`
  - **Acceptance**: Each method returns descriptive "not implemented" error
  - **FR trace**: FR-010
  - **Depends on**: T001

- [ ] T006 Unit tests with httptest.Server — `internal/linear/http_client_test.go`
  - Create shared `newTestServer(handler)` helper that returns `httptest.Server` + client configured with `WithEndpoint`
  - Tests (17 total, per implementation plan test table):
    - Constructor: `TestNewClient_MissingAPIKey`, `TestNewClientFromEnv`
    - Auth: `TestPollReadyTickets_AuthHeader` (verify `Authorization: <key>`, no Bearer)
    - PollReadyTickets: `_Success`, `_EmptyResult`, `_FiltersEmptyDescription`, `_Pagination`
    - GetTicket: `_Success`, `_NotFound`
    - Retry: `TestRetry_RateLimitThenSuccess`, `TestRetry_RateLimitExhausted`, `TestRetry_UsesResetHeader`
    - Errors: `TestDo_HTTPError500`, `TestDo_NonJSONResponse`, `TestDo_ContextCancelled`, `TestDo_Unauthorized`
    - Stubs: `TestUnimplemented_Methods`
  - **Acceptance**: All tests pass with `go test ./internal/linear/ -count=1`
  - **Depends on**: T001-T005

- [ ] T007 Integration tests — `internal/linear/http_client_integration_test.go`
  - Build tag: `//go:build integration`
  - Requires: `BRAINS_LINEAR_API_KEY` env var
  - Tests: `TestIntegration_PollReadyTickets`, `TestIntegration_GetTicket`, `TestIntegration_GetTicket_NotFound`
  - **Acceptance**: Tests pass against real Linear API
  - **Depends on**: T006

## Validation Matrix

| FR | Tasks |
|----|-------|
| FR-001 | T003, T006 |
| FR-002 | T003, T006 |
| FR-003 | T004, T006 |
| FR-004 | T001, T006 |
| FR-005 | T001, T006 |
| FR-006 | T002, T006 |
| FR-007 | T002, T006 |
| FR-008 | T001, T006 |
| FR-009 | T001, T006 |
| FR-010 | T005, T006 |

All 10 FRs covered. No orphan tasks.

## Execution Order

1. T001 (constructor + transport)
2. T002 (retry) + T005 (stubs) [parallel]
3. T003 (poll) + T004 (get) [parallel]
4. T006 (unit tests)
5. T007 (integration tests)

**Total**: 7 tasks, 2 parallel opportunities, critical path = 5 sequential steps.
