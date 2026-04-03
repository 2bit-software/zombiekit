# Technical Requirements: Restore gh-pr Tool

## Approach

Restored from git history (commit 118828f) which already had the edit action implemented. The removal commit (6ed235d) deleted the entire package.

## Files Restored/Modified

- `internal/mcp/tools/ghpr/tool.go` — main tool implementation
- `internal/mcp/tools/ghpr/types.go` — response types and helpers
- `internal/mcp/tools/ghpr/tool_test.go` — validation tests
- `internal/mcp/server.go` — import, struct field, initialization, registration
- `internal/config/tools.go` — added "gh-pr" to KnownTools

## Lint Fixes Applied

The restored code had errcheck violations (unchecked os.Remove and tmpFile.Close calls). Fixed with `_ = os.Remove(...)` and `_ = tmpFile.Close()` patterns.
