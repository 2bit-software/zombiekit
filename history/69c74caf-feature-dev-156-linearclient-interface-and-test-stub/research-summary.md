# Research Summary: DEV-156

## Codebase Research

### Existing Patterns
- **Interfaces**: Defined in main package file (e.g., `store.go`), all methods take `context.Context` first, return `(T, error)`. Examples: `StateStore`, `Storage` (memory), `Storage` (recall).
- **Errors**: Sentinel errors via `errors.New()` at package level. Checked with `errors.Is()`. Wrapped with `fmt.Errorf("op: %w", err)`.
- **Test stubs**: Hand-rolled with testify. Helper functions use `t.Helper()`. Compile-time interface assertions. No mock generation tools.
- **Types**: Input types use `-Input`/`-Options` suffix. Optional fields use `*T`. Domain types are plain structs with json/db tags.

### Package Structure
No existing `linear` or `client` package. Natural home: `internal/linear/`. Follows pattern of `internal/state/`, `internal/memory/`, `internal/recall/`.

### No Existing Linear Code
Zero Linear-related Go code in the codebase. This is greenfield.

## Domain Research

### Linear API Characteristics
- GraphQL-only (single endpoint: `api.linear.app/graphql`)
- Issues have `WorkflowState` (not "status"), `IssueLabel` (many-to-many), `Attachment` (URL links, not binary)
- All mutations require entity UUIDs, not human-readable names
- Complexity-based rate limiting (1500 points/min)
- Cursor-based pagination (max 250 per page)
- No official Go SDK

### Go Interface Best Practices
- `context.Context` first param (universal convention)
- Small interfaces preferred, but a comprehensive interface at the provider level is acceptable when consumers can implicitly satisfy subsets
- Typed errors with predicate functions preferred over sentinels for API clients
- Function-field stubs preferred over generated mocks for narrow interfaces

## Key Tensions Resolved

| Tension | Resolution |
|---------|------------|
| Codebase uses sentinels vs domain suggests typed errors | Typed errors with predicates -- API client errors carry more context than DB errors |
| Ticket says 4 error types vs single struct | Single struct with 4 predicates -- less surface area, same capability |
| Ticket omits `context.Context` in signatures | Add it -- universal codebase convention, required for future implementation |
| Ticket uses "Ticket" vs Linear uses "Issue" | Keep "Ticket" -- matches ticket language and existing `TicketID` field in `Job` struct |
| Single interface vs ISP small interfaces | Single `LinearClient` interface in `linear` package -- consumers define narrow interfaces at their usage sites |
