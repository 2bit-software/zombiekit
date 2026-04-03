# Reuse Audit

## Summary
- Duplicates: 0
- Overlaps: 2 (2 extend, 0 create new)
- Related: 1 (noted for consistency)
- No match: 1

## Findings

### OVERLAP

#### `Abandon()` service method
- **Existing**: `internal/initiative/service.go:Complete()`
- **Similarity**: Same state-check-then-clear pattern. Abandon adds `os.RemoveAll` before clearing state.
- **Decision**: Create new method (not extend Complete)
- **Rationale**: Complete and Abandon have different semantics (preserve vs destroy history). Adding a flag to Complete would muddy the API. Separate methods with shared internal patterns is cleaner.

#### `handleAbandon()` MCP tool handler
- **Existing**: `internal/mcp/tools/initiative/tool.go:handleComplete()`
- **Similarity**: Same structure: create service, check active, call service method, marshal response.
- **Decision**: Create new handler (not extend handleComplete)
- **Rationale**: Same reasoning — different semantics deserve separate handlers. The code duplication is minimal (~20 lines) and the handlers are independently testable.

### RELATED

#### Test setup pattern
- **Related code**: `internal/mcp/tools/initiative/tool_test.go` — `tmpDir`/`testEmbeddedFS()` pattern
- **Note**: Reuse the same test setup for abandon tests. No new test helpers needed.

### NONE

#### `new.md` conflict detection prompt
- No existing initiative-check-before-classification logic in the command markdown.

## Plan Changes

No changes to implementation-plan.md. All planned items are either new code or follow existing patterns without duplication.
