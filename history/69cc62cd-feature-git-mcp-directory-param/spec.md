# Feature Specification: Git MCP Tool — Optional Directory Parameter

**Feature Branch**: `DEV-228/git-mcp-tool-add-optional-directory-para`
**Created**: 2026-03-31
**Status**: Draft
**Input**: Linear ticket DEV-228

## User Scenarios & Testing

### User Story 1 - Specify directory for git operations (Priority: P1)

An MCP client (e.g., Claude Code) working across multiple repositories or worktrees needs to run git operations against a specific directory rather than the server's default working directory. The caller passes a `directory` parameter alongside the normal `action` parameter and receives results scoped to that directory.

**Why this priority**: This is the entire feature — without per-call directory targeting, the tool is limited to one repository per server instance.

**Independent Test**: Call the git tool with `action: "status"` and `directory: "/path/to/other/repo"` — receive status output from that repo, not the server default.

**Acceptance Scenarios**:

1. **Given** a git tool call with `action: "status"` and `directory: "/some/repo"`, **When** executed, **Then** returns the git status of `/some/repo`.
2. **Given** a git tool call with `action: "log"` and `directory: "/some/repo"`, **When** executed, **Then** returns commit log from `/some/repo`.
3. **Given** a git tool call with `action: "diff"` and `directory: "/some/repo"`, **When** executed, **Then** returns diffs from `/some/repo`.
4. **Given** a git tool call with `action: "stage"`, `files: "foo.txt"`, and `directory: "/some/repo"`, **When** executed, **Then** stages `foo.txt` in `/some/repo`.
5. **Given** a git tool call with `action: "commit"`, `message: "test"`, and `directory: "/some/repo"`, **When** executed, **Then** commits in `/some/repo`.
6. **Given** a git tool call with `action: "push"` and `directory: "/some/repo"`, **When** executed, **Then** pushes from `/some/repo` (subject to existing protected-branch rules).

---

### User Story 2 - Default behavior unchanged when directory omitted (Priority: P1)

When the caller does not provide the `directory` parameter, the tool behaves identically to today — using the server's configured working directory. No existing callers break.

**Why this priority**: Backward compatibility is non-negotiable; tied for P1 with the core feature.

**Independent Test**: Call the git tool with `action: "status"` and no `directory` parameter — behavior is identical to the current implementation.

**Acceptance Scenarios**:

1. **Given** a git tool call with `action: "status"` and no `directory`, **When** executed, **Then** returns status from the server's default working directory.
2. **Given** a git tool call with any valid action and no `directory`, **When** executed, **Then** the tool uses the runner's original working directory (not a directory from a previous call).

---

### User Story 3 - Invalid directory produces clear error (Priority: P2)

When a caller provides a `directory` value that doesn't exist, isn't a directory, or isn't a git repository, the tool returns a structured error with a useful hint.

**Why this priority**: Good error messages prevent debugging spirals, but the feature works without them (callers would get cryptic git errors).

**Independent Test**: Call with `directory: "/nonexistent"` and verify error response.

**Acceptance Scenarios**:

1. **Given** `directory: "/nonexistent"`, **When** any action is called, **Then** returns a ToolError with code `INVALID_DIRECTORY`.
2. **Given** `directory: "/tmp"` (exists but not a git repo), **When** any action is called, **Then** returns a ToolError with code `NOT_GIT_REPOSITORY`.
3. **Given** `directory` pointing to a file (not a directory), **When** any action is called, **Then** returns a ToolError with code `INVALID_DIRECTORY`.

---

### Edge Cases

- `directory` is an empty string: MUST behave as if omitted (covered by FR-003).
- `directory` is a relative path: MUST resolve relative to the server's default working directory (the `workDir` passed to `NewServer`, not `os.Getwd()`). See FR-008.
- `directory` points to a subdirectory within a git repo (not the repo root): Git handles this naturally — operations scope to the repo containing that directory. No special handling needed.
- `directory` points to a symlink: Follow it (standard OS behavior). No special handling needed.
- `directory` contains `~` or environment variables: Not expanded. Go does not do shell expansion. Callers must send absolute or relative paths only. This is consistent with all other MCP tools.
- Permission denied on `directory`: The `git rev-parse` validation check will fail, producing a git error. Return the git error as-is — no special error code needed.

## Scope

**In scope**: Adding `directory` parameter to the git MCP tool only.

**Out of scope**: The `gh-pr` tool does NOT gain a directory parameter in this ticket. If needed, that's a separate feature.

## Requirements

### Functional Requirements

- **FR-001**: The git tool MUST accept an optional `directory` parameter (string) on all actions (status, log, diff, stage, commit, push).
- **FR-002**: When `directory` is provided and non-empty, the tool MUST construct a new `git.Runner` for that directory and use it for all operations within that call. The server's default runner MUST NOT be mutated.
- **FR-003**: When `directory` is omitted or empty, the tool MUST use the server's default runner (current behavior).
- **FR-004**: When `directory` does not exist or is not a directory, the tool MUST return a ToolError with code `INVALID_DIRECTORY`, a descriptive message, and a hint.
- **FR-005**: When `directory` exists but is not inside a git repository, the tool MUST return a ToolError with code `NOT_GIT_REPOSITORY`, a descriptive message, and a hint.
- **FR-006**: File validation for the `stage` action MUST validate paths against the effective directory for the call (the provided `directory` if set, otherwise the server default). The `workDir` argument to `validateFiles` MUST be the effective directory, not `t.runner.WorkDir()` when a per-call directory is active.
- **FR-007**: The `directory` parameter MUST NOT appear in the `required` array — it is always optional.
- **FR-008**: When `directory` is a relative path, the tool MUST resolve it relative to the server's default working directory (`t.runner.WorkDir()`).
- **FR-009**: The `directory` parameter MUST be added to the MCP tool schema in `server.go`'s `registerGitTool` function using `mcp.WithString("directory", mcp.Description("Working directory for git operations. When omitted, uses the server's default working directory."))` without `mcp.Required()`.
- **FR-010**: The `NewTool` and `Execute` function signatures MUST NOT change. The `directory` parameter is passed through the `args` map like all other parameters.

### Key Entities

- **Runner** (`internal/git/runner.go`): Executes git commands with `cmd.Dir` set to its `workDir`. When `directory` is provided, a new Runner is constructed for that call. Runner construction is cheap (stores a path string, no connections or state).
- **Tool** (`internal/mcp/tools/git/tool.go`): MCP tool wrapper. Directory resolution happens once at the top of `Execute()`, producing the effective runner for that call. Action handlers receive the effective runner.

### Files to Modify

- `internal/mcp/tools/git/tool.go` — directory resolution logic at top of `Execute`, pass effective runner to handlers
- `internal/mcp/tools/git/tool_test.go` — new tests for directory parameter
- `internal/mcp/server.go` — add `directory` to MCP schema in `registerGitTool`
- `internal/mcp/tools/git/validation.go` — no changes expected (already takes `workDir` as arg)
- `internal/git/runner.go` — no changes expected (already supports arbitrary directories via constructor)

## Success Criteria

### Measurable Outcomes

- **SC-001**: All existing git tool tests pass without modification (backward compat).
- **SC-002**: New tests exercise all six actions (status, log, diff, stage, commit, push) with an explicit `directory` parameter pointing to a different repo.
- **SC-003**: Error cases (nonexistent directory, non-git directory, file-not-directory) are tested and return structured ToolError responses with the specified error codes.
- **SC-004**: The MCP tool schema includes `directory` as an optional string parameter.

## Testing Requirements

### Test Strategy

Integration tests using temporary git repositories (matching the existing test pattern in `tool_test.go`). Create two temp repos; call actions against both via the `directory` parameter to prove isolation. The tool is constructed with repo A as the default, and calls with `directory` pointing to repo B verify operations target B.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Call with `directory` param, verify it's accepted |
| FR-002 | Integration | Call status/log/diff/stage/commit/push against non-default dir, verify results come from that dir |
| FR-003 | Integration | Call without `directory`, verify default behavior unchanged |
| FR-004 | Integration | Call with nonexistent directory, verify ToolError with `INVALID_DIRECTORY` |
| FR-005 | Integration | Call with non-git directory, verify ToolError with `NOT_GIT_REPOSITORY` |
| FR-006 | Integration | Stage a file via `directory` param, verify path resolution uses that directory |
| FR-007 | Integration | Verify tool schema does not list `directory` as required |
| FR-008 | Integration | Call with relative path `directory`, verify resolution against server default |
| FR-009 | Integration | Verify MCP schema includes `directory` parameter |
| FR-010 | Integration | Verify existing `NewTool` constructor still works unchanged |

### Edge Case Coverage

- Empty string `directory` → uses default (test alongside FR-003)
- Relative path `directory` → resolves against server default workDir (test for FR-008)
- Subdirectory of a git repo → git resolves naturally (test that it works)
- Symlink directory → follows symlink (test that status returns valid output)
- `directory` is a file not a directory → returns `INVALID_DIRECTORY` error
