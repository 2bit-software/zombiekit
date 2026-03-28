# Tasks: DEV-158 Linear Ticket Writes

**Complexity**: Simple (3 files, ~350 lines)
**Total tasks**: 11
**Critical path**: T001 → T002 → T003 → T008

## Dependency Graph

```
T001 (structs/queries)
├── T002 (resolvers + mutation constants)
│   ├── T003 (SetTicketStatus)      → T008 (tests)
│   ├── T004 (ApplyLabel)      ─┐
│   └── T005 (RemoveLabel)     ─┤→ T009 (tests)
└── T006 (CreateTicket)             → T010 (tests)
                                         │
T007 (resolver tests) ──────────────────┤
                                         └── T011 (integration tests)
```

**Parallel opportunities**: T003+T004+T005+T006 after T002; T007+T008+T009+T010 after implementations.

## Tasks

### Phase 1: Prerequisites

- [ ] **T001** [FR-008, FR-009] Add `TeamID` to `Ticket` struct and `ProjectID` to `CreateTicketInput` in `internal/linear/client.go`. Add `Team` field to `issueNode` struct, update `toTicket()` to populate `TeamID`, add `team { id }` to `getTicketQuery` and `pollReadyTicketsQuery` in `internal/linear/http_client.go`. Run existing tests to verify no regressions.
  - **Acceptance**: `go test ./internal/linear/...` passes. `Ticket` has `TeamID string`. `CreateTicketInput` has `ProjectID string`. Both issue queries include `team { id }`.

### Phase 2: Resolution Helpers

- [ ] **T002** [FR-001, FR-005] Add `resolveWorkflowStateQuery` and `resolveLabelQuery` GraphQL constants, `workflowStatesResponse` and `issueLabelsResponse` response types, `resolveWorkflowStateID(ctx, teamID, name)` and `resolveLabelID(ctx, name)` methods, and `issueUpdateMutation` constant with `issueUpdateResponse` type in `internal/linear/http_client.go`.
  - **Acceptance**: Both resolver methods compile. `resolveWorkflowStateID` returns `NewNotFoundError` on zero matches. `resolveLabelID` returns `NewNotFoundError` on zero matches and `NewAPIError` on multiple matches.

### Phase 3: Write Operations

- [ ] **T003** [FR-001, FR-002] Implement `SetTicketStatus` in `internal/linear/http_client.go`: fetch issue via `GetTicket` → `resolveWorkflowStateID` → `issueUpdate` with `stateId`.
  - **Acceptance**: Method compiles. Error from `GetTicket` or resolver propagates. `issueUpdate` called with `map[string]any{"stateId": resolvedID}`.

- [ ] **T004** [P] [FR-003, FR-005] Implement `ApplyLabel` in `internal/linear/http_client.go`: `resolveLabelID` → `issueUpdate` with `addedLabelIds`.
  - **Acceptance**: Method compiles. Resolver errors propagate. `issueUpdate` called with `map[string]any{"addedLabelIds": []string{labelID}}`.

- [ ] **T005** [P] [FR-004, FR-005] Implement `RemoveLabel` in `internal/linear/http_client.go`: `resolveLabelID` → `issueUpdate` with `removedLabelIds`.
  - **Acceptance**: Method compiles. Resolver errors propagate. `issueUpdate` called with `map[string]any{"removedLabelIds": []string{labelID}}`.

- [ ] **T006** [P] [FR-006, FR-007] Implement `CreateTicket` in `internal/linear/http_client.go`: add `issueCreateMutation` constant, `issueCreateResponse` type, build variables from `CreateTicketInput` (only include non-zero fields), parse response `issueNode` into `Ticket`.
  - **Acceptance**: Method compiles. Returns `*Ticket` with populated fields. Errors from `doWithRetry` propagate. Optional fields (StateID, LabelIDs, ProjectID, Priority, AssigneeID) only included when non-zero.

### Phase 4: Unit Tests

- [ ] **T007** [P] [SC-001] Add `queryDispatcher` test helper and unit tests for `resolveWorkflowStateID` and `resolveLabelID` in `internal/linear/http_client_test.go`. Test cases: found, not found, ambiguous (label only).
  - **Acceptance**: 5 test cases pass. Dispatcher correctly routes based on query content.

- [ ] **T008** [SC-001, SC-003] Add unit tests for `SetTicketStatus` in `internal/linear/http_client_test.go`. Test cases: happy path (issue fetch → state resolve → mutation), status not found, ticket not found.
  - **Acceptance**: 3 test cases pass. Error messages include attempted status name.

- [ ] **T009** [SC-001, SC-004] Add unit tests for `ApplyLabel` and `RemoveLabel` in `internal/linear/http_client_test.go`. Test cases: happy path, already applied (idempotent), label not found, ambiguous label — for each method.
  - **Acceptance**: 8 test cases pass. Idempotent cases return nil error. Error messages include attempted label name.

- [ ] **T010** [SC-001] Add unit tests for `CreateTicket` in `internal/linear/http_client_test.go`. Test cases: happy path (full input), minimal input (only TeamID+Title), API error propagation.
  - **Acceptance**: 3 test cases pass. Returned Ticket has correct fields populated.

### Phase 5: Integration Tests

- [ ] **T011** [SC-002] Add integration tests for all write operations in `internal/linear/http_client_test.go`, gated behind `BRAINS_LINEAR_API_KEY` env var. Tests: SetTicketStatus round-trip, ApplyLabel/RemoveLabel round-trip, CreateTicket.
  - **Acceptance**: Tests skip when env var not set. Tests pass when env var is set with valid key.

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T002, T003, T008 |
| FR-002 | T003, T008 |
| FR-003 | T004, T009 |
| FR-004 | T005, T009 |
| FR-005 | T002, T004, T005, T007, T009 |
| FR-006 | T006, T010 |
| FR-007 | T006, T010 |
| FR-008 | T001 |
| FR-009 | T001 |

## Execution Order

```
T001 → T002 → T003, T004, T005, T006 (parallel) → T007, T008, T009, T010 (parallel) → T011
```
