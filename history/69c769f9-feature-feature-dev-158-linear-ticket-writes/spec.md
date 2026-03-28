# Feature Specification: Linear Ticket Writes

**Feature Branch**: `morganhein/dev-158-implement-linear-ticket-writes-status-labels-and-ticket`
**Created**: 2026-03-27
**Status**: Draft (post-audit revision)
**Input**: DEV-158 — Implement Linear ticket writes (status, labels, ticket creation)

## User Scenarios & Testing

### User Story 1 - Update Ticket Status (Priority: P1)

An automation agent transitions a Linear ticket through workflow states (e.g., "Backlog" -> "In Progress" -> "Done") by providing the ticket identifier and a human-readable status name.

**Why this priority**: Status transitions are the most fundamental write operation — they reflect work progress and are required by both orchestrator-core and friction-auditor.

**Independent Test**: Call `SetTicketStatus` with a valid ticket and status name; verify the ticket's status changes in Linear.

**ID parameter convention**: The `id` parameter across all write methods accepts whatever Linear's API accepts — both UUIDs and human-readable identifiers (e.g., "DEV-158") work. No client-side resolution is needed; Linear resolves identifiers internally.

**Team context**: `SetTicketStatus` needs the ticket's team ID to resolve the status name to a workflow state ID. The implementation MUST fetch the issue first (using the existing `issue` query, extended to include `team { id }`) to derive the team. This requires adding a `TeamID` field to the `Ticket` struct and including `team { id }` in all issue queries.

**Acceptance Scenarios**:

1. **Given** a valid ticket ID and a status name that exists in the workspace, **When** `SetTicketStatus` is called, **Then** the ticket's workflow status is updated
2. **Given** a status name that does not exist for the ticket's team, **When** `SetTicketStatus` is called, **Then** a descriptive error is returned (not a silent no-op)
3. **Given** a ticket ID that does not exist, **When** `SetTicketStatus` is called, **Then** a not-found error is returned

---

### User Story 2 - Manage Labels on Tickets (Priority: P1)

An automation agent applies or removes labels on Linear tickets by name, enabling categorization (e.g., applying "improvements" label to friction-auditor-generated tickets).

**Why this priority**: Label management is required for the friction-auditor's workflow — tickets it creates need specific labels for filtering and polling.

**Independent Test**: Call `ApplyLabel` on a ticket without the label; verify it appears. Call again; verify no error. Call `RemoveLabel`; verify it's gone. Call again; verify no error.

**Acceptance Scenarios**:

1. **Given** a label not on the ticket, **When** `ApplyLabel` is called, **Then** the label is applied
2. **Given** a label already on the ticket, **When** `ApplyLabel` is called, **Then** no error is returned (idempotent)
3. **Given** a label on the ticket, **When** `RemoveLabel` is called, **Then** the label is removed
4. **Given** a label not on the ticket, **When** `RemoveLabel` is called, **Then** no error is returned (idempotent)
5. **Given** a label name that does not exist in the workspace, **When** `ApplyLabel` or `RemoveLabel` is called, **Then** a descriptive error is returned
6. **Given** a label name that matches multiple labels (e.g., same name at team and workspace scope), **When** `ApplyLabel` or `RemoveLabel` is called, **Then** an error is returned indicating ambiguous match

**Label resolution scope**: Labels are resolved via the `issueLabels` query filtered by exact name (workspace-wide). If the query returns exactly one result, use it. If it returns zero results, return a not-found error. If it returns multiple results, return an ambiguity error. No team-scoping of the label query — the name must be globally unique.

---

### User Story 3 - Create New Tickets (Priority: P2)

The friction-auditor creates new Linear tickets in a specified project with title, description, and labels — filing improvement tickets automatically.

**Why this priority**: Ticket creation is needed by the friction-auditor but is a less frequent operation than status/label management.

**Independent Test**: Call `CreateTicket` with valid input; verify a ticket is created and the returned `Ticket` has a valid ID and identifier.

**`CreateTicketInput` struct changes**: The existing struct needs a `ProjectID string` field added. The struct takes pre-resolved UUIDs for `StateID`, `LabelIDs`, and `ProjectID` — this is intentional. Unlike `SetTicketStatus`/`ApplyLabel`/`RemoveLabel` which accept human-readable names and resolve internally (single-value convenience), `CreateTicket` takes structured input where the caller provides IDs. Callers that need name-to-ID resolution (e.g., friction-auditor resolving "improvements" label) should use the label/state resolution helpers exposed as internal methods or call `ApplyLabel` after creation.

**Acceptance Scenarios**:

1. **Given** valid `CreateTicketInput` with team, title, description, and labels, **When** `CreateTicket` is called, **Then** the ticket is created and the new ticket (with ID) is returned
2. **Given** an invalid project ID, **When** `CreateTicket` is called, **Then** an error is returned
3. **Given** label IDs that don't exist, **When** `CreateTicket` is called, **Then** the API error is propagated

---

### Edge Cases

- What happens when a ticket's team has multiple workflow states with similar names? Error on exact match failure.
- How does the system handle concurrent label operations on the same ticket? `addedLabelIds`/`removedLabelIds` are safe for concurrent use — Linear handles merging.
- What happens when the API key has read-only permissions? API error is propagated.
- What happens on network failure mid-mutation? Existing retry logic handles rate limits; other errors propagate immediately.

## Requirements

### Functional Requirements

- **FR-001**: System MUST update a ticket's workflow status by fetching the ticket to derive its team, resolving the status name to a workflow state ID via `workflowStates` query (filtered by team and name), and calling `issueUpdate` with the resolved `stateId`
- **FR-002**: System MUST return a descriptive error when a status name cannot be resolved, including the attempted name and the ticket's team context
- **FR-003**: System MUST apply a label to a ticket by resolving the label name to an ID via `issueLabels` query (filtered by exact name, workspace-wide), then calling `issueUpdate` with `addedLabelIds`. Idempotent: no error if already applied
- **FR-004**: System MUST remove a label from a ticket by resolving the label name to an ID via `issueLabels` query, then calling `issueUpdate` with `removedLabelIds`. Idempotent: no error if not present
- **FR-005**: System MUST return a descriptive error when a label name cannot be resolved (zero matches) or is ambiguous (multiple matches), including the attempted name and match count
- **FR-006**: System MUST create a new ticket via `issueCreate` with `CreateTicketInput` fields (TeamID required; Title, Description, StateID, LabelIDs, ProjectID, Priority, AssigneeID optional), returning the created ticket with its ID and identifier
- **FR-007**: System MUST propagate API errors from ticket creation (invalid project, missing permissions, etc.)
- **FR-008**: The `Ticket` struct MUST be extended with a `TeamID` field, and all issue queries MUST include `team { id }` in the response selection
- **FR-009**: The `CreateTicketInput` struct MUST be extended with a `ProjectID string` field

### Key Entities

- **WorkflowState**: Team-scoped status with ID, name, and type (triage/backlog/unstarted/started/completed/canceled)
- **IssueLabel**: Team-scoped or workspace-scoped label with ID and name
- **Ticket**: Existing entity — created tickets return full Ticket representation

## Success Criteria

### Measurable Outcomes

- **SC-001**: All 4 write operations implemented and passing unit tests with httptest server mocks
- **SC-002**: Integration tests pass against real Linear API (gated behind env var)
- **SC-003**: Error messages for failed name resolution include the attempted name
- **SC-004**: Label operations are idempotent — repeated calls produce no errors

## Testing Requirements

### Test Strategy

- **Unit tests**: httptest.Server mocking GraphQL responses, testing query construction, response parsing, error mapping, and name resolution logic
- **Integration tests**: Real Linear API calls behind `BRAINS_LINEAR_API_KEY` env var, testing actual mutations
- **Framework**: `testing` + `testify/assert` + `net/http/httptest` (consistent with existing test suite)

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Unit + Integration | Verify status update mutation sent with correct stateId after name resolution |
| FR-002 | Unit | Verify error returned when status name not found in mock response |
| FR-003 | Unit + Integration | Verify addedLabelIds mutation sent; idempotent on already-applied label |
| FR-004 | Unit + Integration | Verify removedLabelIds mutation sent; idempotent on absent label |
| FR-005 | Unit | Verify error returned when label name not found in mock response |
| FR-006 | Unit + Integration | Verify issueCreate mutation with correct variables; ticket returned |
| FR-007 | Unit | Verify API error propagation from creation failures |

### Edge Case Coverage

- Status name not found -> descriptive error with attempted name (unit test with empty workflowStates response)
- Label name not found -> descriptive error with attempted name (unit test with empty issueLabels response)
- Ticket not found during status update -> not-found error (unit test with 404-like GraphQL error)
- Rate limit during mutation -> retry with backoff (unit test with RATELIMITED then success)
- Multiple labels with same name -> error on ambiguous match (unit test with multiple nodes)
- Ticket fetch fails during SetTicketStatus -> error propagated (unit test with issue query error)

## Resolved Decisions

1. **Team context for name resolution**: Fetch the issue first to derive team context. The interface signature `SetTicketStatus(ctx, id, status)` stays unchanged. Implementation fetches the issue internally to get `TeamID`, then resolves the status name against the team's workflow states.

2. **Label ambiguity**: Error on ambiguous match (multiple labels with the same name). The caller must disambiguate. This prevents silent wrong-label bugs.

3. **`CreateTicketInput` uses IDs, not names**: Unlike the convenience methods (`SetTicketStatus`, `ApplyLabel`, `RemoveLabel`) which accept human-readable names, `CreateTicket` takes pre-resolved UUIDs. This is intentional — structured input for batch operations. Callers needing name resolution can use the internal resolution helpers or call `ApplyLabel` after creation.

4. **`id` parameter convention**: All write methods accept whatever Linear's API accepts (both UUIDs and human-readable identifiers like "DEV-158"). No client-side ID resolution needed.
