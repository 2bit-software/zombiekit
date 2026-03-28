# Implementation Plan: DEV-158 Linear Ticket Writes

## Overview

Extend the existing Linear HTTP client with four write operations: SetTicketStatus, ApplyLabel, RemoveLabel, CreateTicket. Implementation follows existing patterns (GraphQL query constants, response structs, `doWithRetry`).

## Phase 1: Struct and Query Prerequisites

**Goal**: Extend existing types and queries to support write operations.

**Changes**:

1. **`client.go`** — Add `TeamID string` to `Ticket` struct, add `ProjectID string` to `CreateTicketInput`
2. **`http_client.go`** — Add `team { id }` to `issueNode` struct, update `getTicketQuery` and `pollReadyTicketsQuery` to include `team { id }`, update `toTicket()` to populate `TeamID`

**Dependencies**: None
**Traces to**: FR-008, FR-009

## Phase 2: Name Resolution Helpers

**Goal**: Create internal methods for resolving human-readable names to Linear UUIDs.

**Changes in `http_client.go`**:

1. **`resolveWorkflowStateID(ctx, teamID, statusName) (string, error)`**
   - GraphQL: `workflowStates(filter: { team: { id: { eq: $teamID } }, name: { eq: $name } })`
   - Returns the state ID if exactly one match
   - Returns `NewNotFoundError` if zero matches (include attempted name in message)
   - Response struct: `workflowStatesResponse` with `nodes []{ id, name }`

2. **`resolveLabelID(ctx, labelName) (string, error)`**
   - GraphQL: `issueLabels(filter: { name: { eq: $name } })`
   - Returns the label ID if exactly one match
   - Returns `NewNotFoundError` if zero matches (include attempted name)
   - Returns `NewAPIError` if multiple matches (ambiguity error with count)
   - Response struct: `issueLabelsResponse` with `nodes []{ id, name }`

**Dependencies**: None (uses existing `doWithRetry`)
**Traces to**: FR-001, FR-002, FR-003, FR-004, FR-005

## Phase 3: Write Operations

**Goal**: Implement the four stub methods.

### 3a: `SetTicketStatus(ctx, id, status)`

1. Fetch the issue with `GetTicket` to get `TeamID`
2. Call `resolveWorkflowStateID(ctx, ticket.TeamID, status)`
3. Call `issueUpdate` mutation with `{ stateId: resolvedID }`
4. Check `success` field in response

**GraphQL mutation**:
```graphql
mutation($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
  }
}
```

**Dependencies**: Phase 1 (TeamID on Ticket), Phase 2 (resolveWorkflowStateID)
**Traces to**: FR-001, FR-002

### 3b: `ApplyLabel(ctx, id, label)`

1. Call `resolveLabelID(ctx, label)`
2. Call `issueUpdate` mutation with `{ addedLabelIds: [resolvedID] }`
3. Check `success` field

**Dependencies**: Phase 2 (resolveLabelID)
**Traces to**: FR-003, FR-005

### 3c: `RemoveLabel(ctx, id, label)`

1. Call `resolveLabelID(ctx, label)`
2. Call `issueUpdate` mutation with `{ removedLabelIds: [resolvedID] }`
3. Check `success` field

**Dependencies**: Phase 2 (resolveLabelID)
**Traces to**: FR-004, FR-005

### 3d: `CreateTicket(ctx, input)`

1. Map `CreateTicketInput` fields to `issueCreate` variables
2. Call `issueCreate` mutation
3. Parse response into `Ticket`

**GraphQL mutation**:
```graphql
mutation($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue {
      id identifier title description url priority
      state { name }
      labels { nodes { name } }
      team { id }
    }
  }
}
```

**Dependencies**: Phase 1 (ProjectID on CreateTicketInput, TeamID on issueNode)
**Traces to**: FR-006, FR-007

## Phase 4: Unit Tests

**Goal**: Test all write operations with httptest.Server mocks.

**Test cases per operation**:

| Method | Test | Mock Response |
|--------|------|---------------|
| SetTicketStatus | happy path | issue query → ticket with team, workflowStates → one match, issueUpdate → success |
| SetTicketStatus | status not found | issue query → ticket, workflowStates → empty nodes |
| SetTicketStatus | ticket not found | issue query → not found error |
| ApplyLabel | happy path | issueLabels → one match, issueUpdate → success |
| ApplyLabel | already applied (idempotent) | issueLabels → one match, issueUpdate → success (Linear handles idempotency) |
| ApplyLabel | label not found | issueLabels → empty nodes |
| ApplyLabel | ambiguous label | issueLabels → multiple matches |
| RemoveLabel | happy path | issueLabels → one match, issueUpdate → success |
| RemoveLabel | already absent (idempotent) | issueLabels → one match, issueUpdate → success (Linear handles idempotency) |
| RemoveLabel | label not found | issueLabels → empty nodes |
| RemoveLabel | ambiguous label | issueLabels → multiple matches |
| CreateTicket | happy path | issueCreate → success with full issue |
| CreateTicket | API error | issueCreate → error response |
| resolveWorkflowStateID | found | workflowStates → one node |
| resolveWorkflowStateID | not found | workflowStates → empty |
| resolveLabelID | found | issueLabels → one node |
| resolveLabelID | not found | issueLabels → empty |
| resolveLabelID | ambiguous | issueLabels → multiple nodes |

**Idempotency assumption**: Linear's `addedLabelIds` and `removedLabelIds` are idempotent at the API level — applying an already-present label or removing an absent label returns `success: true`. Unit tests verify this by mocking `success: true` for these cases. Integration tests confirm real API behavior.

**Test infrastructure**: Reuse `newTestClient`, `gqlSuccess`, `gqlError` helpers. Multi-step tests (SetTicketStatus) need a handler that dispatches based on the query body.

**Dependencies**: Phase 3
**Traces to**: SC-001, SC-003, SC-004

## Phase 5: Integration Tests

**Goal**: Verify against real Linear API.

**Gating**: Skip when `BRAINS_LINEAR_API_KEY` not set.

**Tests**:
- SetTicketStatus on a known test ticket
- ApplyLabel / RemoveLabel round-trip
- CreateTicket with cleanup (or in a test project)

**Dependencies**: Phase 4 (unit tests passing first)
**Traces to**: SC-002

## Implementation Order

```
Phase 1 (structs/queries) → Phase 2 (resolvers) → Phase 3a-d (write ops, parallelizable) → Phase 4 (unit tests) → Phase 5 (integration tests)
```

## Risk Register

| Risk | Mitigation |
|------|-----------|
| Linear API mutation field names differ from research | Schema was verified against official GitHub source; low risk |
| `issueUpdate` on non-existent ticket returns unexpected error shape | Unit test with mock; integration test confirms real behavior |
| Label ambiguity never triggers in practice | Still implement error path for correctness; test with mock |
| GraphQL variable types (`String!` vs `ID!`) may differ from schema | Research verified against official schema; `String!` is canonical for Linear mutations. Integration tests will catch type mismatches. |
