# Initiative: git-mcp-actions

**Type**: feature
**Status**: completed
**Created**: 2026-03-28
**ID**: 69c86c4d-feature-git-mcp-actions

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-03-28 17:10 |
| plan | completed | 2026-03-28 17:20 |
| tasks | completed | 2026-03-28 17:30 |
| implement | completed | 2026-03-28 17:50 |

## Description

Add MCP tool endpoints for git and GitHub PR operations to eliminate shell script workarounds for Claude Code skill permissioning.

## Completion

**Completed**: 2026-03-28
**Duration**: Same session

### Outcomes
- Feature: `git` MCP tool (status, log, diff, stage, commit, push) - Complete
- Feature: `gh-pr` MCP tool (view, create, comment) - Complete
- Feature: `internal/git` runner package - Complete
- Feature: Config registration and server wiring - Complete
- Tests: 27 tests across 3 packages - All passing

### Files Changed
- **Created (9):** `internal/git/runner.go`, `internal/git/runner_test.go`, `internal/mcp/tools/git/tool.go`, `internal/mcp/tools/git/types.go`, `internal/mcp/tools/git/validation.go`, `internal/mcp/tools/git/tool_test.go`, `internal/mcp/tools/ghpr/tool.go`, `internal/mcp/tools/ghpr/types.go`, `internal/mcp/tools/ghpr/tool_test.go`
- **Modified (3):** `internal/config/tools.go`, `internal/mcp/server.go`, `internal/cli/serve.go`
