# Technical Requirements Research: DEV-156

## Implementation Preferences (from ticket)

- Interface is the primary deliverable -- design before implementation
- This interface is the mock boundary for Epic 4 (archival/auditing track)
- Can start in parallel with DEV-153 (no dependency)
- No actual HTTP calls, no auth, no attachment upload implementation

## Codebase Patterns to Follow

### Package Location
New package: `internal/linear/`

### Interface Pattern
From `internal/state/store.go`:
- `context.Context` as first parameter on all methods
- Error returns use `fmt.Errorf("operation: %w", err)` wrapping
- Pointer receivers on concrete implementations
- Compile-time assertion: `var _ LinearClient = (*MockLinearClient)(nil)`

### Error Pattern
From `internal/state/errors.go`:
- Codebase uses sentinel errors with `errors.New()`
- However, for an API client, a typed error struct with predicates is more appropriate (carries status codes, messages, wrapped errors)
- Predicate functions: `IsNotFound(err) bool`, etc.
- Use `errors.As` internally in predicates

### Test Stub Pattern
From `internal/state/store_test.go`:
- Hand-rolled stubs (no generated mocks)
- `testify/assert` and `testify/require`
- `t.Helper()` on helper functions
- `t.Cleanup()` for teardown
- Function-field pattern for configurable responses:
  ```go
  type MockLinearClient struct {
      GetTicketFn func(ctx context.Context, id string) (*Ticket, error)
      // ...
      Calls []Call
  }
  ```

### Type Naming
From `internal/memory/importer/types.go`:
- Input types: `CreateTicketInput`, `AttachmentInput`
- Domain types: `Ticket` (not `Issue`, to match ticket language)
- Optional fields: `*T` for pointer-based optionality, empty string for "omit"

## Linear API Notes (for future implementation, not this ticket)

- Linear API is GraphQL-only (no REST)
- Mutations require UUIDs, not names -- interface uses names, implementation resolves
- Statuses are `WorkflowState` entities, per-team
- Labels are many-to-many, modified via connect/disconnect mutations
- "Attachments" in Linear are URL links, not binary uploads
- Rate limiting is complexity-based (1500 points/min)
- Pagination is cursor-based (max 250 per page)
- No official Go SDK exists -- raw HTTP with string queries is standard
