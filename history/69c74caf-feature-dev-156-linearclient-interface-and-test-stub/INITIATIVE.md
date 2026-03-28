# Initiative: dev-156-linearclient-interface-and-test-stub

**Type**: feature
**Status**: completed
**Created**: 2026-03-27
**ID**: 69c74caf-feature-dev-156-linearclient-interface-and-test-stub

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-27 20:56 |
| plan | completed | 2026-03-27 21:04 |
| tasks | completed | 2026-03-27 21:07 |
| implement | completed | 2026-03-27 21:10 |

## Source

**Linear Ticket**: [DEV-156](https://linear.app/heinsight/issue/DEV-156/define-linearclient-interface-and-test-stub)
**Title**: Define LinearClient interface and test stub

## Description

Define a Go `LinearClient` interface with all methods the orchestrator needs, along with input/output types, error types, and a configurable test stub/mock implementation.

## Completion

**Completed**: 2026-03-27
**Duration**: Same day (spec through implementation)

### Outcomes

- `internal/linear/errors.go` -- ErrorKind enum, Error struct, 4 predicate functions, 4 constructors
- `internal/linear/client.go` -- Client interface (7 methods), Ticket, CreateTicketInput, AttachmentInput types
- `internal/linear/mock.go` -- MockClient with function-field stubs and call recording
- `internal/linear/mock_test.go` -- 11 tests covering all FRs and SCs, all passing

### Key Decisions

- Named `linear.Client` (not `LinearClient`) to avoid Go stutter
- Unified `Error` struct with `ErrorKind` + predicate functions instead of 4 separate error types
- Function-field stubs with `XxxFn` pattern, matching codebase conventions
- `context.Context` on all methods (codebase convention)
- Human-readable names in interface; UUID resolution deferred to real implementation
- Mock in exported `mock.go` (not `_test.go`) so other packages can import it
