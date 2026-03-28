# Feature Specification: LinearClient Interface and Test Stub

**Feature Branch**: `morganhein/dev-156-define-linearclient-interface-and-test-stub`
**Created**: 2026-03-27
**Status**: Draft
**Input**: DEV-156 - Define LinearClient interface and test stub

## User Scenarios & Testing

### User Story 1 - Interface Compiles and Wires Into Consumers (Priority: P1)

A developer defines a consumer (e.g., an orchestrator or reconciler) that depends on the `linear.Client` interface. The consumer compiles, and calls route through the interface without any real API calls.

**Why this priority**: The interface is the primary deliverable. If it doesn't compile or can't be wired into consumers, nothing else matters.

**Independent Test**: Create a consumer function that accepts the interface, wire in the mock, call a method, and verify compilation + execution.

**Acceptance Scenarios**:

1. **Given** the `linear.Client` interface is defined, **When** a consumer accepts it as a dependency, **Then** the consumer compiles and method calls route through without real API calls
2. **Given** the mock implementation exists, **When** it is passed to a consumer as a `linear.Client`, **Then** it satisfies the interface at compile time (via `var _ LinearClient = ...` assertion)

---

### User Story 2 - Mock Returns Configurable Canned Responses (Priority: P1)

A test author configures the mock to return specific data (e.g., 2 tickets from PollReadyTickets). When the method is called, exactly that data is returned.

**Why this priority**: The mock's value is in making consumer tests deterministic. Without configurable responses, the mock is useless.

**Independent Test**: Configure the mock's PollReadyTickets to return 2 tickets, call it, assert exactly those 2 are returned.

**Acceptance Scenarios**:

1. **Given** the mock's PollReadyTickets is configured to return 2 tickets, **When** called, **Then** exactly those 2 tickets are returned in the defined type
2. **Given** the mock's GetTicket is configured to return a specific ticket, **When** called with the matching ID, **Then** that ticket is returned
3. **Given** a method is NOT configured on the mock, **When** called, **Then** an explicit error is returned (not a silent zero value)

---

### User Story 3 - Mock Records Calls for Verification (Priority: P1)

A test author calls SetTicketStatus through the mock, then inspects the mock to verify the call was made with the expected arguments.

**Why this priority**: Call verification is essential for testing side-effecting operations (status changes, label mutations) where the return value alone doesn't confirm correct behavior.

**Independent Test**: Call SetTicketStatus on the mock, then assert the call was recorded with the correct arguments.

**Acceptance Scenarios**:

1. **Given** the mock's SetTicketStatus is called, **When** the test inspects recorded calls, **Then** the call is present with the correct ticket ID and status
2. **Given** multiple methods are called, **When** the test inspects the call log, **Then** all calls are recorded in order with their arguments

---

### User Story 4 - Mock Returns Configurable Errors (Priority: P2)

A test author configures the mock to return a specific error type (e.g., NotFoundError). When the method is called, the configured error is returned and is checkable via error predicates.

**Why this priority**: Error path testing is important but secondary to the happy path. Consumers need to verify they handle Linear API failures correctly.

**Independent Test**: Configure the mock to return a NotFoundError, call the method, assert the error satisfies the IsNotFound predicate.

**Acceptance Scenarios**:

1. **Given** the mock is configured to return a NotFoundError, **When** the method is called, **Then** the returned error satisfies `IsNotFound(err) == true`
2. **Given** the mock is configured to return a RateLimitError, **When** the method is called, **Then** the returned error satisfies `IsRateLimited(err) == true`
3. **Given** the mock is configured to return an APIError, **When** the method is called, **Then** the returned error satisfies `IsAPIError(err) == true`
4. **Given** the mock is configured to return a NetworkError, **When** the method is called, **Then** the returned error satisfies `IsNetworkError(err) == true`

---

### Edge Cases

- What happens when a mock method is called but never configured? -> Must return an explicit, descriptive error.
- What happens when the same mock is used across multiple test cases without reset? -> Call log accumulates. Document this behavior.
- What happens when nil is passed as the context? -> Interface accepts it (Go convention), behavior is implementation-defined.

## Requirements

### Functional Requirements

- **FR-001**: System MUST define a `LinearClient` interface in package `internal/linear/` with the exact signatures below
- **FR-002**: All interface methods MUST accept `context.Context` as the first parameter
- **FR-003**: System MUST define input types: `CreateTicketInput`, `AttachmentInput` with the exact fields below
- **FR-004**: System MUST define output type: `Ticket` with the exact fields below
- **FR-005**: System MUST define an `Error` struct and 4 predicate functions: `IsNotFound`, `IsRateLimited`, `IsAPIError`, `IsNetworkError`
- **FR-006**: System MUST provide a `MockClient` struct in a non-test file (`mock.go`) so other packages can import it
- **FR-007**: The mock MUST have one `XxxFn` function field per interface method (e.g., `GetTicketFn`)
- **FR-008**: The mock MUST record all calls in a `Calls []Call` field using the `Call` struct defined below
- **FR-009**: The mock MUST return `fmt.Errorf("MockClient.MethodName not configured")` for unconfigured methods (no silent zero values)
- **FR-010**: `GetTicket` MUST return `(nil, error)` wrapping a not-found error when the ticket doesn't exist — never `(nil, nil)`

### Type Definitions (Go)

**Package**: `internal/linear/` (package name: `linear`)

#### Interface

```go
type Client interface {
    PollReadyTickets(ctx context.Context, label string) ([]Ticket, error)
    GetTicket(ctx context.Context, id string) (*Ticket, error)
    SetTicketStatus(ctx context.Context, id string, status string) error
    ApplyLabel(ctx context.Context, id string, label string) error
    RemoveLabel(ctx context.Context, id string, label string) error
    CreateTicket(ctx context.Context, input CreateTicketInput) (*Ticket, error)
    UploadAttachment(ctx context.Context, ticketID string, input AttachmentInput) error
}
```

Note: Named `Client` (not `LinearClient`) to avoid stutter (`linear.Client` not `linear.LinearClient`).

#### Domain Types

```go
type Ticket struct {
    ID          string   // Linear UUID
    Identifier  string   // Human-readable (e.g., "DEV-156")
    Title       string
    Description string   // Empty string if unset (not pointer)
    Status      string   // Workflow state name (e.g., "In Progress")
    Labels      []string // Label names; empty slice (not nil) if none
    Priority    int      // 0=none, 1=urgent, 2=high, 3=medium, 4=low
    URL         string   // Linear web URL
}

type CreateTicketInput struct {
    TeamID      string   // Required
    Title       string   // Required
    Description string   // Optional (empty string = omit)
    StateID     string   // Optional (empty string = omit)
    LabelIDs    []string // Optional (nil = omit)
    Priority    *int     // Optional (nil = omit, 0 is meaningful: "no priority")
    AssigneeID  string   // Optional (empty string = omit)
}

type AttachmentInput struct {
    URL      string // Required
    Title    string // Required
    Subtitle string // Optional (empty string = omit)
}
```

#### Error Type

```go
type ErrorKind int

const (
    ErrNotFound    ErrorKind = iota + 1
    ErrRateLimited
    ErrAPI
    ErrNetwork
)

type Error struct {
    Kind    ErrorKind
    Message string
    Err     error // Wrapped underlying error (may be nil)
}

func (e *Error) Error() string   // Returns Message
func (e *Error) Unwrap() error   // Returns Err

// Predicate functions (package-level, accept any error)
func IsNotFound(err error) bool    // true when errors.As finds *Error with Kind == ErrNotFound
func IsRateLimited(err error) bool // true when errors.As finds *Error with Kind == ErrRateLimited
func IsAPIError(err error) bool    // true when errors.As finds *Error with Kind == ErrAPI
func IsNetworkError(err error) bool // true when errors.As finds *Error with Kind == ErrNetwork

// Constructors
func NewNotFoundError(msg string, cause error) *Error
func NewRateLimitedError(msg string, cause error) *Error
func NewAPIError(msg string, cause error) *Error
func NewNetworkError(msg string, cause error) *Error
```

Predicate functions MUST return `false` for `nil` errors and non-Linear errors.

#### Mock Type

File: `internal/linear/mock.go` (exported, importable by other packages' tests)

```go
type Call struct {
    Method string
    Args   []any
}

type MockClient struct {
    PollReadyTicketsFn  func(ctx context.Context, label string) ([]Ticket, error)
    GetTicketFn         func(ctx context.Context, id string) (*Ticket, error)
    SetTicketStatusFn   func(ctx context.Context, id string, status string) error
    ApplyLabelFn        func(ctx context.Context, id string, label string) error
    RemoveLabelFn       func(ctx context.Context, id string, label string) error
    CreateTicketFn      func(ctx context.Context, input CreateTicketInput) (*Ticket, error)
    UploadAttachmentFn  func(ctx context.Context, ticketID string, input AttachmentInput) error

    Calls []Call
}
```

Each method implementation: append to `Calls`, then delegate to `XxxFn` if non-nil, else return `fmt.Errorf("MockClient.Xxx not configured")`.

Compile-time assertion: `var _ Client = (*MockClient)(nil)`

## Scope Boundaries

### In Scope

- Interface definition with all 7 methods
- Input/output types for all methods
- Error types with predicate functions
- Test stub with configurable responses and call recording

Note: This interface is also the mock boundary for Epic 4 (archival/auditing track). It must be broad enough to serve both orchestration and archival consumers.

### Out of Scope

- Any actual HTTP/GraphQL calls to the Linear API
- OAuth or API key authentication setup (DEV-5)
- Attachment upload implementation (DEV-7, though the interface method signature IS in scope)
- Rate limiting, retry logic, or pagination handling (implementation concerns)

## Success Criteria

### Measurable Outcomes

- **SC-001**: A consumer function accepting `linear.Client` compiles and executes with the mock wired in
- **SC-002**: Mock configured with 2 tickets returns exactly those 2 tickets from PollReadyTickets
- **SC-003**: All 7 interface methods are callable on the mock and record calls
- **SC-004**: All 4 error predicates correctly identify their respective error types
- **SC-005**: Unconfigured mock methods return a descriptive error, not a zero value

## Testing Requirements

### Test Strategy

Integration tests using the mock implementation. Since this is an interface + mock deliverable (no real API calls), all tests exercise the mock itself to verify it correctly satisfies the interface contract, records calls, and returns configured responses/errors.

Framework: `testify/assert` and `testify/require` (matching existing codebase patterns).

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Compile-time | `var _ Client = (*MockClient)(nil)` assertion |
| FR-002 | Compile-time | Methods won't compile without context parameter |
| FR-003 | Compile-time | Types exist and are usable in method signatures |
| FR-004 | Integration | Create a Ticket, verify all fields are populated and accessible |
| FR-005 | Integration | Create each error type, verify predicates return true/false correctly |
| FR-006 | Integration | Wire mock into a consumer function, verify it executes |
| FR-007 | Integration | Configure mock responses, call methods, verify returns |
| FR-008 | Integration | Call methods, inspect Calls slice for correct recording |
| FR-009 | Integration | Call unconfigured method, verify non-nil error returned |

### Edge Case Coverage

- Unconfigured method call -> explicit error (not panic, not nil)
- Multiple calls recorded in order -> verify call log ordering
- Error predicate on non-Linear error -> returns false
- Error predicate on nil -> returns false

## Key Decisions for Review

| # | Decision | Rationale | Alternatives Considered |
|---|----------|-----------|------------------------|
| D1 | Single `linear.Client` interface (not segregated) | Ticket specifies one interface as the mock boundary. Go consumers can still define narrow interfaces implicitly. | Consumer-defined small interfaces only (rejected: ticket explicitly asks for this interface) |
| D2 | Function-field stubs (not generated mocks) | Matches codebase pattern (hand-rolled stubs with testify), more explicit, zero tooling dependency | mockgen/moq (rejected: adds tooling), map-based stubs (rejected: can't test error paths) |
| D3 | Unified error struct with predicates (not 4 separate types) | Cleaner API, errors carry context (message, underlying error). Predicates (`IsNotFound`) are stable public API. | 4 sentinel errors (rejected: no context), 4 struct types (rejected: more surface area for same capability) |
| D4 | `context.Context` on all methods | Universal codebase convention. Required for cancellation, deadlines, tracing when real implementation arrives. | No context (rejected: breaks codebase convention, limits future implementation) |
| D5 | Human-readable names in interface (not UUIDs) | Interface uses "status name" and "label name", not UUIDs. ID resolution is an implementation concern for the real client. | UUID-only (rejected: consumers think in names, not UUIDs) |
