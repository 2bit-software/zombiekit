# Implementation Plan: DEV-156 LinearClient Interface and Test Stub

## Overview

Create the `internal/linear/` package with a `Client` interface, domain types, error types with predicates, and a `MockClient` test stub.

**Spike assessment**: None needed. No external APIs, no uncertain integration points, all types fully specified.

## File Structure

```
internal/linear/
├── client.go       # Client interface + Ticket, CreateTicketInput, AttachmentInput types
├── errors.go       # Error struct, ErrorKind, predicates, constructors
├── mock.go         # MockClient struct, Call struct
└── mock_test.go    # Tests for mock behavior, error predicates, call recording
```

## Implementation Steps

### Step 1: Create `internal/linear/errors.go`

**FR mapping**: FR-005
**Dependencies**: None (leaf node)

Contents:
- `ErrorKind` type (`int`) with constants: `ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork`
- `Error` struct with `Kind`, `Message`, `Err` fields
- `Error()` method returning `Message`
- `Unwrap()` method returning `Err`
- 4 predicate functions using `errors.As` internally
- 4 constructor functions (`NewNotFoundError`, etc.)

**Verification**: Predicates compile and return expected values (tested in step 4).

### Step 2: Create `internal/linear/client.go`

**FR mapping**: FR-001, FR-002, FR-003, FR-004
**Dependencies**: Step 1 (errors.go must exist for package to compile, but no direct import needed within same package)

Contents:
- `Ticket` struct with 8 fields (ID, Identifier, Title, Description, Status, Labels, Priority, URL)
- `CreateTicketInput` struct with 7 fields (TeamID, Title, Description, StateID, LabelIDs, Priority, AssigneeID)
- `AttachmentInput` struct with 3 fields (URL, Title, Subtitle)
- `Client` interface with 7 methods

No json/db tags needed — these are domain types, not serialization types. Tags can be added when the real client implementation needs them.

**Verification**: Package compiles.

### Step 3: Create `internal/linear/mock.go`

**FR mapping**: FR-006, FR-007, FR-008, FR-009
**Dependencies**: Step 2 (references Client interface and domain types)

Contents:
- `Call` struct with `Method string` and `Args []any`
- `MockClient` struct with 7 `XxxFn` function fields and `Calls []Call`
- 7 method implementations, each:
  1. Appends to `Calls` with method name and args (excluding ctx)
  2. Delegates to `XxxFn` if non-nil
  3. Returns `fmt.Errorf("MockClient.Xxx not configured")` if `XxxFn` is nil
- Compile-time assertion: `var _ Client = (*MockClient)(nil)`

**Design note on Args**: Exclude `ctx` from recorded args — it's infrastructure, not business data. Tests care about "was GetTicket called with id='DEV-156'", not "was a context passed".

**Verification**: Compile-time assertion passes.

### Step 4: Create `internal/linear/mock_test.go`

**FR mapping**: All FRs (integration verification)
**Dependencies**: Steps 1-3

Test cases:

1. **TestMockClient_InterfaceCompliance** — Compile-time assertion (already in mock.go, but verify it builds)

2. **TestMockClient_ConfiguredResponse_PollReadyTickets** — Configure `PollReadyTicketsFn` to return 2 tickets, call it, assert exactly 2 returned with correct fields (SC-002)

3. **TestMockClient_ConfiguredResponse_GetTicket** — Configure `GetTicketFn` to return a specific ticket, call with matching ID, assert that ticket is returned (User Story 2, scenario 2)

4. **TestMockClient_UnconfiguredMethod** — Call `GetTicket` without configuring `GetTicketFn`, assert non-nil error containing "MockClient.GetTicket not configured" (FR-009, SC-005)

5. **TestMockClient_CallRecording_AllMethods** — Table-driven test calling all 7 methods, verify each records the correct method name and args in order (FR-008, SC-003). Exercises every method to prove full coverage.

6. **TestMockClient_ErrorPredicates** — Table-driven: create each of the 4 error kinds via constructors, verify the corresponding predicate returns true and the other 3 return false (FR-005, SC-004)

7. **TestMockClient_ErrorPredicates_NilAndForeign** — Verify all 4 predicates return false for nil error and `errors.New("unrelated")` (edge case)

8. **TestMockClient_ErrorUnwrap** — Create error with wrapped cause, verify `errors.Unwrap` returns the cause (Error.Unwrap behavior)

9. **TestMockClient_ConfiguredError** — Configure `GetTicketFn` to return a NotFoundError, call it, verify return is `(nil, error)` where `IsNotFound(err) == true` (User Story 4, FR-010)

10. **TestMockClient_ConsumerWiring** — Define a consumer function that accepts `Client` interface, wire in `MockClient`, call a method through the consumer, verify it executes and the mock records the call (SC-001, FR-006)

11. **TestMockClient_CallAccumulation** — Use the same mock across multiple calls without reset, verify `Calls` slice accumulates all calls (edge case: no implicit reset)

## Dependency Graph

```
errors.go (Step 1) ──┐
                      ├── mock.go (Step 3) ── mock_test.go (Step 4)
client.go (Step 2) ──┘
```

Steps 1 and 2 are independent and can be implemented in parallel. Step 3 depends on both. Step 4 depends on Step 3.

## Verification Checklist

- [ ] `go build ./internal/linear/` succeeds
- [ ] `go test ./internal/linear/` passes all tests
- [ ] `go vet ./internal/linear/` reports no issues
- [ ] Compile-time assertion `var _ Client = (*MockClient)(nil)` in mock.go
- [ ] All 7 interface methods callable on MockClient
- [ ] All 4 error predicates work correctly
- [ ] Unconfigured methods return descriptive errors
- [ ] Call recording captures method name and args in order
