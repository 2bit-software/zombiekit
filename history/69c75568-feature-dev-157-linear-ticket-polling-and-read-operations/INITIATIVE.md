# Initiative: dev-157-linear-ticket-polling-and-read-operations

**Type**: feature
**Status**: completed
**Created**: 2026-03-27
**ID**: 69c75568-feature-dev-157-linear-ticket-polling-and-read-operations

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-27 21:25 |
| plan | completed | 2026-03-27 21:30 |
| tasks | completed | 2026-03-27 21:33 |
| implement | completed | 2026-03-27 21:36 |

## Source

**Linear Ticket**: [DEV-157](https://linear.app/heinsight/issue/DEV-157/implement-linear-ticket-polling-and-read-operations)
**Title**: Implement Linear ticket polling and read operations

## Description

<!-- Add a description of this initiative -->

## Goals

<!-- Define the goals for this initiative -->

## Progress

<!-- Track progress here -->

## Completion

**Completed**: 2026-03-27 21:36
**Duration**: ~25 minutes (21:13 - 21:36)

### Outcomes

- **Feature: Linear HTTP Client** - Complete
  - Real `Client` implementation against Linear GraphQL API
  - `PollReadyTickets` with label filtering, pagination, client-side empty-description filter
  - `GetTicket` by identifier string
  - Rate limit detection (HTTP 400 + RATELIMITED) with exponential backoff and reset header support
  - Error mapping to existing `NotFoundError`, `RateLimitedError`, `APIError`, `NetworkError` types
  - API key auth via `Authorization: <key>` (no Bearer prefix)
  - Constructor with functional options + `NewClientFromEnv()` factory

### Files Created
- `internal/linear/http_client.go` (~250 lines)
- `internal/linear/http_client_test.go` (19 unit tests with httptest.Server)
- `internal/linear/http_client_integration_test.go` (3 integration tests, `//go:build integration`)

### Architectural Decisions
- Hand-rolled GraphQL over `net/http` (no client library -- only community option was dead)
- Retry timing injectable via `WithRetryTiming` for fast tests
- All unimplemented `Client` methods return descriptive "not implemented" errors
