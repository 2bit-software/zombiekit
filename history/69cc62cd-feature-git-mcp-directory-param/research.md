---
status: complete
updated: 2026-03-31
---

# Research: Git MCP Tool — Directory Parameter

## Executive Summary

The zombiekit MCP git tool operates against a single working directory fixed at server initialization. Other MCP tools in the project (profile, workflow, initiative) already accept per-call directory parameters using a consistent pattern. Adding an optional `directory` parameter to the git tool follows established conventions and requires changes to the tool layer, schema registration, and validation — the git runner abstraction already supports arbitrary directories.

## Findings

### Codebase Context

- **Git tool location**: `internal/mcp/tools/git/tool.go` (Tool struct, Execute, action handlers)
- **Git runner**: `internal/git/runner.go` (command execution, sets `cmd.Dir`)
- **Schema registration**: `internal/mcp/server.go` lines 503-564 (`registerGitTool`)
- **Validation**: `internal/mcp/tools/git/validation.go` (`validateFiles` takes workDir as first arg)
- **Types/helpers**: `internal/mcp/tools/git/types.go` (response types, `getStringArg`, `getIntArg`, `getBoolArg`)
- **Tests**: `internal/mcp/tools/git/tool_test.go` (full coverage of all actions)

- **Existing directory parameter patterns**:
  - `profile` tool: optional `working_directory` via `getWorkingDir(args)` helper
  - `workflow` tool: optional `working_directory` via same pattern
  - `initiative` tool: required `dir` parameter

- **Runner design**: `git.NewRunner(workDir)` creates a runner; `runner.Run(ctx, args...)` executes `git <args>` with `cmd.Dir = workDir`. Construction is cheap — no connections, no state beyond the path string.

- **Server initialization**: `NewServer()` creates one runner if `workDir != ""`, passes it to `gittool.NewTool(runner)`. The tool stores this as `t.runner` and uses `t.runner.WorkDir()` throughout.

### Domain Knowledge

- MCP tools commonly support per-call directory targeting for multi-repo workflows (Claude Code worktrees, monorepos).
- The mcp-go framework (v0.43.2) supports optional string parameters via `mcp.WithString()` without `mcp.Required()`.
- Git itself handles subdirectory paths gracefully — operations resolve to the containing repository.

## Decision Points

- [x] **D1**: Parameter name — `directory` (matches ticket wording, concise)
- [x] **D2**: Implementation strategy — create new Runner per-call when directory is provided (cheapest, least invasive)
- [x] **D3**: Relative path handling — resolve relative to server's default working directory

## Recommendations

1. Add optional `directory` string parameter to the git tool schema in `server.go`.
2. In `tool.go`, resolve the effective working directory at the start of `Execute()`: use `directory` if provided, else fall back to `t.runner.WorkDir()`.
3. When `directory` is provided, validate it exists and is a git repo, then create a temporary `git.NewRunner(directory)` for that call.
4. Pass the resolved runner through to action handlers (or resolve once and use throughout the call).
5. Update `validateFiles` calls to use the resolved directory.
6. Add tests: two temp repos, call actions with explicit directory, verify isolation.

## Sources

- `internal/mcp/tools/git/tool.go` — current tool implementation
- `internal/mcp/tools/git/types.go` — helper functions and response types
- `internal/mcp/tools/git/validation.go` — file path validation
- `internal/mcp/tools/git/tool_test.go` — existing test patterns
- `internal/git/runner.go` — git command execution
- `internal/mcp/server.go` — tool registration and schema
- `internal/mcp/tools/profile/tool.go` — reference pattern for `working_directory` param
