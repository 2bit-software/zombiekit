# Technical Requirements Research: DEV-158

These are implementation preferences and technical hints extracted from the ticket description and research. They inform HOW to build, but are not business requirements.

## From Ticket

- Status names are workspace-configurable — accept string names and resolve at call time, don't hardcode status IDs
- `CreateTicket` is used by friction auditor to file tickets in AI-Enabled Dev project — `improvements` label name should be configurable, not hardcoded
- DEV-157 and DEV-158 both depend on DEV-156 and can be built in parallel
- Integration tests against real Linear API behind build flag

## From Research

- Hand-rolled GraphQL client (no SDK) — consistent with DEV-157 implementation
- Use `addedLabelIds`/`removedLabelIds` for incremental label operations, never `labelIds`
- Workflow state resolution requires team context — derive from issue or pass explicitly
- Consider caching workflow states and labels (they change rarely)
- Per-endpoint rate limit headers (`X-RateLimit-Endpoint-*`) available but not currently used
- Existing `doWithRetry` handles rate limiting — reuse for mutations

## Architectural Alignment

- Follow existing patterns: GraphQL query strings as constants, response structs per operation
- Error mapping: use existing `NewNotFoundError`, `NewAPIError` for resolution failures
- Mock client already has function fields — no interface changes needed
- Test pattern: httptest.Server with `newTestClient` helper, fast retry timing
