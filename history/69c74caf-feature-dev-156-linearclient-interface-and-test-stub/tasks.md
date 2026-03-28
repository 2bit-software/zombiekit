# Tasks: DEV-156 LinearClient Interface and Test Stub

**Complexity**: Simple (4 files, ~250 LOC, 0 cross-module deps)
**Critical path**: T001/T002 (parallel) -> T003 -> T004

## Tasks

### Phase 1: Foundation (parallel)

- [ ] T001 [P] [FR-005] Create `internal/linear/errors.go` — Define `ErrorKind` type with 4 constants (`ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork`), `Error` struct with `Kind`/`Message`/`Err` fields, `Error()` and `Unwrap()` methods, 4 predicate functions (`IsNotFound`, `IsRateLimited`, `IsAPIError`, `IsNetworkError`) using `errors.As`, and 4 constructors (`NewNotFoundError`, `NewRateLimitedError`, `NewAPIError`, `NewNetworkError`). Predicates must return false for nil and non-Linear errors.

- [ ] T002 [P] [FR-001,FR-002,FR-003,FR-004] Create `internal/linear/client.go` — Define `Client` interface with 7 methods (all taking `context.Context` first), `Ticket` struct (8 fields: ID, Identifier, Title, Description, Status, Labels, Priority, URL), `CreateTicketInput` struct (7 fields with `*int` for Priority), `AttachmentInput` struct (3 fields). No json tags needed.

### Phase 2: Mock (depends on Phase 1)

- [ ] T003 [FR-006,FR-007,FR-008,FR-009] Create `internal/linear/mock.go` — Define `Call` struct (`Method string`, `Args []any`), `MockClient` struct with 7 `XxxFn` function fields and `Calls []Call`. Implement all 7 interface methods: each appends to `Calls` (args exclude ctx), delegates to `XxxFn` if non-nil, returns `fmt.Errorf("MockClient.Xxx not configured")` if nil. Add compile-time assertion `var _ Client = (*MockClient)(nil)`.

### Phase 3: Tests (depends on Phase 2)

- [ ] T004 [All FRs, All SCs] Create `internal/linear/mock_test.go` — 11 test cases using testify:
  1. `TestMockClient_InterfaceCompliance` — verify build succeeds with assertion
  2. `TestMockClient_ConfiguredResponse_PollReadyTickets` — 2 tickets returned (SC-002)
  3. `TestMockClient_ConfiguredResponse_GetTicket` — specific ticket returned (US2-S2)
  4. `TestMockClient_UnconfiguredMethod` — error containing "not configured" (FR-009, SC-005)
  5. `TestMockClient_CallRecording_AllMethods` — table-driven, all 7 methods record correctly (SC-003)
  6. `TestMockClient_ErrorPredicates` — table-driven, each kind true/others false (SC-004)
  7. `TestMockClient_ErrorPredicates_NilAndForeign` — false for nil and foreign errors
  8. `TestMockClient_ErrorUnwrap` — unwrap returns wrapped cause
  9. `TestMockClient_ConfiguredError` — GetTicket returns (nil, NotFoundError), IsNotFound true (FR-010)
  10. `TestMockClient_ConsumerWiring` — consumer func accepts Client, mock wired in (SC-001)
  11. `TestMockClient_CallAccumulation` — calls accumulate without reset

### Phase 4: Verify

- [ ] T005 Run `go build ./internal/linear/`, `go test ./internal/linear/`, `go vet ./internal/linear/` — all must pass with zero errors.

## FR Traceability

| FR | Task(s) |
|----|---------|
| FR-001 | T002 |
| FR-002 | T002 |
| FR-003 | T002 |
| FR-004 | T002 |
| FR-005 | T001 |
| FR-006 | T003 |
| FR-007 | T003 |
| FR-008 | T003 |
| FR-009 | T003 |
| FR-010 | T004 (test 9) |

## SC Traceability

| SC | Task(s) |
|----|---------|
| SC-001 | T004 (test 10) |
| SC-002 | T004 (test 2) |
| SC-003 | T004 (test 5) |
| SC-004 | T004 (test 6) |
| SC-005 | T004 (test 4) |

## Execution Order

```
T001 ──┐
       ├── T003 ── T004 ── T005
T002 ──┘
```

T001 and T002 can execute in parallel. Everything else is sequential.
