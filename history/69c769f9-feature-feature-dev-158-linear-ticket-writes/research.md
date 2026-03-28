# Research Summary: DEV-158 Linear Ticket Writes

## Executive Summary

Linear's GraphQL API uses `issueUpdate` for status changes and label management, and `issueCreate` for ticket creation. Both workflow states and labels are referenced by UUID, requiring name-to-ID resolution queries. The existing HTTP client has all infrastructure needed (retry, auth, error mapping) — four stub methods need implementation.

## Findings

### Codebase Context

- **Interface** (`client.go`): 4 write methods defined, all accept human-readable names (not UUIDs)
- **HTTP client** (`http_client.go`): `do`/`doWithRetry` infrastructure, rate limit handling, error mapping — all reusable
- **Stubs**: `SetTicketStatus`, `ApplyLabel`, `RemoveLabel`, `CreateTicket` return "not implemented"
- **Mock client** (`mock.go`): Already has function fields for all 4 methods
- **Types**: `CreateTicketInput` (TeamID, Title, Description, StateID, LabelIDs, Priority, AssigneeID), `Ticket` (ID, Identifier, Title, Description, Status, Labels, Priority, URL)
- **Test patterns**: httptest.Server, `newTestClient` helper, `gqlSuccess`/`gqlError` response builders

### Domain Knowledge

**`issueUpdate(id: String!, input: IssueUpdateInput!): IssuePayload!`**
- Status: `input.stateId` (workflow state UUID)
- Add labels: `input.addedLabelIds` (append, no clobber)
- Remove labels: `input.removedLabelIds` (subtract, no clobber)
- Never use `input.labelIds` — it replaces all labels

**`issueCreate(input: IssueCreateInput!): IssuePayload!`**
- Only `teamId` required; `title`, `description`, `stateId`, `labelIds`, `projectId`, `priority` optional
- Defaults to Backlog state if `stateId` omitted

**Name resolution queries:**
- Workflow states: `workflowStates(filter: { team: { id: { eq: $teamId } }, name: { eq: $name } })`
- Labels: `issueLabels(filter: { name: { eq: $name } })`

**Rate limiting**: 5,000 req/hr, leaky bucket, HTTP 400 with `RATELIMITED` code (not 429). Existing retry logic already handles this.

## Decision Points

- [x] **D1**: Team context for status resolution — derive from issue via existing `GetTicket` (interface already supports `SetTicketStatus(ctx, id, status)` without team param)
- [ ] **D2**: Label name ambiguity — if same name at team + workspace scope, error or prefer one?

## Recommendations

1. Derive team context from the issue for `SetTicketStatus` (fetch issue → get team → resolve state name)
2. Use `addedLabelIds`/`removedLabelIds` for label operations, never `labelIds`
3. Return descriptive errors on resolution failure including the attempted name
4. Follow existing query/response struct patterns for new GraphQL operations
5. Consider caching resolved IDs as a future optimization, not in initial implementation

## Sources

- Linear GraphQL Schema: `github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql`
- Linear Rate Limiting: `linear.app/developers/rate-limiting`
- Linear GraphQL Docs: `linear.app/developers/graphql`
- Existing codebase: `internal/linear/` package
