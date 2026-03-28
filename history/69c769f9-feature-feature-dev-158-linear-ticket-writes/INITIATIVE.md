# Initiative: feature-dev-158-linear-ticket-writes

**Type**: feature
**Status**: completed
**Created**: 2026-03-27
**ID**: 69c769f9-feature-feature-dev-158-linear-ticket-writes

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-28 00:00 |
| plan | completed | 2026-03-28 00:10 |
| tasks | completed | 2026-03-28 00:15 |
| implement | completed | 2026-03-28 00:30 |

## Source

**Linear Ticket**: [DEV-158](https://linear.app/heinsight/issue/DEV-158/implement-linear-ticket-writes-status-labels-and-ticket-creation)
**Title**: Implement Linear ticket writes — status, labels, and ticket creation

## Description

Implement Linear ticket write operations: SetTicketStatus, ApplyLabel/RemoveLabel, and CreateTicket. These operations enable the orchestrator-core and friction-auditor to modify Linear state programmatically.

## Completion

**Completed**: 2026-03-28
**Duration**: 1 day (2026-03-27 to 2026-03-28)

### Outcomes

- **SetTicketStatus**: Implemented with team-derived workflow state resolution
- **ApplyLabel / RemoveLabel**: Implemented with workspace-wide label resolution, ambiguity detection, idempotent behavior
- **CreateTicket**: Implemented with full input mapping (TeamID, Title, Description, StateID, LabelIDs, ProjectID, Priority, AssigneeID)
- **Struct updates**: Added `TeamID` to `Ticket`, `ProjectID` to `CreateTicketInput`, `team { id }` to all issue queries
- **Unit tests**: 19 new test cases covering happy paths, error paths, idempotency, and ambiguity
- **Integration tests**: 7 new tests gated behind `integration` build tag

### Architectural Decisions

1. `SetTicketStatus` fetches the issue internally to derive team context (extra API call per status update)
2. Label resolution is workspace-wide; ambiguous matches (same name, multiple scopes) are hard errors
3. `CreateTicketInput` takes pre-resolved UUIDs; convenience methods (`SetTicketStatus`, `ApplyLabel`, `RemoveLabel`) resolve names internally
4. Hand-rolled GraphQL mutations (no SDK), consistent with DEV-157 read operations

### Files Changed

- `internal/linear/client.go` — Struct updates
- `internal/linear/http_client.go` — 4 queries/mutations, 4 response types, 2 resolver methods, 4 write method implementations
- `internal/linear/http_client_test.go` — 19 new unit tests + queryDispatcher helper
- `internal/linear/http_client_integration_test.go` — 7 new integration tests
