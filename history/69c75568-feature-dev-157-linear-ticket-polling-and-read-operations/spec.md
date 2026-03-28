# Feature Specification: Linear Ticket Polling and Read Operations

**Feature Branch**: `69c75568-feature-dev-157-linear-ticket-polling-and-read-operations`
**Created**: 2026-03-27
**Status**: Audited
**Input**: Linear ticket DEV-157: Implement Linear ticket polling and read operations

## User Scenarios & Testing

### User Story 1 - Poll for AI-Ready Tickets (Priority: P1)

The orchestrator calls `PollReadyTickets` to discover which Linear tickets are labeled `ai-ready` and have a non-empty description, so it can queue them for autonomous processing.

**Why this priority**: This is the primary entry point for the autonomous pipeline. Without polling, no work gets discovered.

**Independent Test**: Call `PollReadyTickets` against a workspace with known `ai-ready` tickets and verify the returned slice contains those tickets with ID, identifier, title, and description populated.

**Acceptance Scenarios**:

1. **Given** a valid API key and a workspace with one `ai-ready` ticket with a non-empty description, **When** `PollReadyTickets(ctx, "ai-ready")` is called, **Then** that ticket is returned with ID, identifier, title, and description populated
2. **Given** a workspace with no `ai-ready` tickets, **When** `PollReadyTickets(ctx, "ai-ready")` is called, **Then** an empty slice is returned without error
3. **Given** a workspace with an `ai-ready` ticket that has a null/empty description, **When** `PollReadyTickets(ctx, "ai-ready")` is called, **Then** that ticket is excluded from results

---

### User Story 2 - Fetch Full Ticket by ID (Priority: P1)

The orchestrator calls `GetTicket` to retrieve the full content of a specific ticket (including description) after discovering it via polling, so it can pass the requirements to an agent.

**Why this priority**: Equal to polling -- the orchestrator needs both poll and fetch to function. Polling discovers tickets; fetching retrieves the content to work on.

**Independent Test**: Call `GetTicket` with a known ticket identifier and verify all fields are populated correctly.

**Acceptance Scenarios**:

1. **Given** a valid ticket identifier (e.g., "DEV-157"), **When** `GetTicket(ctx, "DEV-157")` is called, **Then** a Ticket is returned with ID, identifier, title, description, status, labels, priority, and URL populated
2. **Given** an invalid/nonexistent ticket identifier, **When** `GetTicket(ctx, "DEV-99999")` is called, **Then** a `NotFoundError` is returned

---

### User Story 3 - Rate Limit Resilience (Priority: P2)

When Linear returns a rate limit response, the client automatically retries with exponential backoff so that transient throttling doesn't cause the orchestrator to drop work.

**Why this priority**: Important for production reliability but not needed for basic functionality. The happy path works without this.

**Independent Test**: Simulate a rate-limited response (HTTP 400 + RATELIMITED error code), verify the client retries and eventually succeeds or returns `RateLimitedError` after exhausting retries.

**Acceptance Scenarios**:

1. **Given** Linear returns a rate limit response on the first call, **When** the subsequent retry succeeds, **Then** the successful result is returned transparently
2. **Given** Linear returns rate limit responses on all retry attempts, **When** max retries are exhausted, **Then** a `RateLimitedError` is returned

---

### User Story 4 - API Key Validation at Init (Priority: P2)

When the client is constructed without a valid API key, initialization fails immediately with a clear error rather than failing opaquely on the first API call.

**Why this priority**: Fail-fast behavior prevents confusing runtime errors but isn't required for the core polling/fetch loop.

**Independent Test**: Construct the client with an empty API key and verify it returns an error.

**Acceptance Scenarios**:

1. **Given** no API key environment variable is set, **When** the client is initialized, **Then** initialization fails with an error message indicating the missing key
2. **Given** a valid API key is provided, **When** the client is initialized, **Then** initialization succeeds

---

### Edge Cases

- What happens when Linear returns a non-JSON response (e.g., 502 gateway error)? → Return `NetworkError`
- What happens when the GraphQL response contains partial data with errors? → Return the error, not partial data
- What happens when the API key is syntactically present but revoked? → Return `APIError` on first call (401 response maps to `APIError`)
- What happens when the network is unreachable? → Return `NetworkError`
- What happens when context is cancelled mid-request? → Return `ctx.Err()` wrapped in appropriate error type

## Requirements

### Functional Requirements

- **FR-001**: System MUST query Linear GraphQL API for issues filtered by label name and non-null description. The query MUST fetch all matching results by paginating through cursor-based pages (first: 50, using `after` cursor) until `hasNextPage` is false
- **FR-002**: System MUST apply client-side filter to exclude issues with empty (but non-null) descriptions
- **FR-003**: System MUST fetch a single issue by identifier string (e.g., "DEV-157") including full description
- **FR-004**: System MUST authenticate requests using `Authorization: <API_KEY>` header (no "Bearer" prefix). The constructor accepts the API key as a parameter; a factory function `NewClientFromEnv()` reads from `BRAINS_LINEAR_API_KEY`
- **FR-005**: System MUST fail initialization with a clear error when API key is missing or empty
- **FR-006**: System MUST detect rate limit responses (HTTP 400 with GraphQL error `extensions.code: "RATELIMITED"`) and retry with exponential backoff: base delay 1s, multiplier 2x, jitter up to 500ms, max 3 retries. If `X-RateLimit-Requests-Reset` header is present, use it to compute retry delay instead of exponential backoff
- **FR-007**: System MUST surface a `RateLimitedError` after retry exhaustion (not silently fail)
- **FR-008**: System MUST map Linear API errors to the existing error type system:
  - HTTP 400 + `extensions.code: "RATELIMITED"` → `RateLimitedError`
  - GraphQL error with "not found" in message → `NotFoundError`
  - HTTP 401 → `APIError` (revoked/invalid key)
  - HTTP 5xx or non-JSON response → `NetworkError`
  - Connection/DNS/timeout failures → `NetworkError`
  - Context cancellation → wrap `ctx.Err()` in `NetworkError`
- **FR-009**: System MUST respect context cancellation on all API calls
- **FR-010**: System MUST implement only `PollReadyTickets` and `GetTicket` from the `Client` interface (other methods return "not implemented" errors)

### Key Entities

- **Ticket**: Represents a Linear issue. Field mapping from GraphQL:
  - `ID` ← `id` (UUID string)
  - `Identifier` ← `identifier` (e.g., "DEV-157")
  - `Title` ← `title`
  - `Description` ← `description` (markdown string, nullable)
  - `Status` ← `state.name` (e.g., "In Progress")
  - `Labels` ← `labels.nodes[].name` (flattened to `[]string`)
  - `Priority` ← `priority` (Linear returns float 0-4, truncate to int)
  - `URL` ← `url`
- **LinearClient**: Real HTTP implementation of the existing `Client` interface, created via `NewClient(apiKey string, opts ...Option)` or `NewClientFromEnv()`

## Success Criteria

### Measurable Outcomes

- **SC-001**: `PollReadyTickets` returns correct tickets when run against a workspace with known `ai-ready` labeled issues
- **SC-002**: `GetTicket` returns complete ticket data including description for valid identifiers
- **SC-003**: Rate-limited requests are retried transparently up to the configured maximum
- **SC-004**: All error conditions produce the correct error type (verifiable via `IsNotFound`, `IsRateLimited`, etc.)

## Testing Requirements

### Test Strategy

- **Unit tests**: Test the GraphQL request construction, response parsing, error mapping, and retry logic using an `httptest.Server` that simulates Linear's API responses
- **Integration test**: Behind `//go:build integration` tag, test `PollReadyTickets` and `GetTicket` against the real Linear API with a valid API key
- **No mocking of `net/http`**: Use `httptest.Server` to simulate the real HTTP flow

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Unit | httptest server returns filtered issues; verify query contains label filter and description null check |
| FR-002 | Unit | httptest server returns issue with empty description; verify client filters it out |
| FR-003 | Unit | httptest server returns issue for identifier; verify all Ticket fields populated |
| FR-004 | Unit | Verify Authorization header is sent with correct format (no Bearer prefix) |
| FR-005 | Unit | Construct client with empty key; verify error |
| FR-006 | Unit | httptest server returns 400+RATELIMITED then 200; verify retry succeeds |
| FR-007 | Unit | httptest server returns 400+RATELIMITED on all attempts; verify RateLimitedError |
| FR-008 | Unit | httptest server returns various error responses; verify correct error types |
| FR-009 | Unit | Cancel context during request; verify context error propagated |
| FR-010 | Unit | Call unimplemented methods; verify "not implemented" error |

### Edge Case Coverage

- Empty description (non-null) → Unit test: filtered out by client-side check
- Non-JSON response body → Unit test: returns NetworkError
- Partial GraphQL response with errors → Unit test: returns APIError
- Context cancellation → Unit test: returns wrapped context error
- Revoked API key (401) → Unit test: returns APIError
