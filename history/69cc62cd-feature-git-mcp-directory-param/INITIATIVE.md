# Initiative: git-mcp-directory-param

**Type**: feature
**Status**: completed
**Created**: 2026-03-31
**ID**: 69cc62cd-feature-git-mcp-directory-param

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-31 17:12 |
| plan | completed | 2026-03-31 17:13 |
| tasks | completed | 2026-03-31 17:14 |
| implement | completed | 2026-03-31 17:15 |

## Source

**Linear Ticket**: [DEV-228](https://linear.app/heinsight/issue/DEV-228/git-mcp-tool-add-optional-directory-parameter-for-working-directory)
**Title**: Git MCP tool: add optional "directory" parameter for working directory

## Description

Add an optional `directory` parameter to the MCP git tool, enabling callers to target git operations at a specific working directory instead of the server's default.

## Completion

**Completed**: 2026-03-31
**Duration**: Same day

### Outcomes
- Feature: git-mcp-directory-param - Complete
  - Added optional `directory` parameter to MCP schema
  - Added `resolveRunner` method for per-call directory resolution
  - Updated all 6 handler signatures to accept runner parameter
  - 12 new tests (happy path, error cases, edge cases)
  - All 26 tests pass (14 existing + 12 new)

### Files Changed
- `internal/mcp/server.go` — schema registration
- `internal/mcp/tools/git/tool.go` — core implementation
- `internal/mcp/tools/git/tool_test.go` — tests
