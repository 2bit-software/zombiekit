# Initiative: brains-new-active-check

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-02
**ID**: 69cf4f58-feature-brains-new-active-check

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-02 22:35 |
| plan | completed | 2026-04-02 22:45 |
| tasks | completed | 2026-04-02 22:50 |
| implement | completed | 2026-04-02 23:00 |

## Completion

**Completed**: 2026-04-02
**Duration**: <1 day

### Outcomes
- Feature: abandon action for initiative MCP tool — Complete
- Feature: active initiative conflict detection in /brains.new — Complete
- Tests: 3 new integration tests for abandon action — Complete

### Files Changed
- `internal/initiative/service.go` — Added `AbandonResult` type and `Abandon()` method
- `internal/mcp/tools/initiative/types.go` — Added `AbandonResponse` struct
- `internal/mcp/tools/initiative/tool.go` — Added `abandon` action to MCP schema and handler
- `internal/mcp/tools/initiative/tool_test.go` — 3 new tests
- `embed/commands/new.md` — Pre-classification initiative check with 3-option prompt
