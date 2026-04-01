# Tasks: Git MCP Tool — Optional Directory Parameter

**Complexity**: Simple (3 files, ~120 LOC)
**Critical Path**: T001 → T002 → T003 → T004 → T005

## Tasks

- [ ] T001 [P] [FR-009] Add `directory` parameter to MCP schema in `internal/mcp/server.go`
  - Add `mcp.WithString("directory", mcp.Description("Working directory for git operations. When omitted, uses the server's default working directory."))` to `registerGitTool` after existing params
  - No `mcp.Required()`
  - **Acceptance**: Schema includes optional `directory` string parameter

- [ ] T002 [FR-002,FR-003,FR-004,FR-005,FR-006,FR-008] Add `resolveRunner` and update all handlers in `internal/mcp/tools/git/tool.go`
  - Add `resolveRunner(ctx, args) (*internalgit.Runner, error)` method
  - Resolve empty → `t.runner`, relative → `filepath.Join(t.runner.WorkDir(), dir)`, absolute → use as-is
  - Validate: `os.Stat` + `IsDir()` → `INVALID_DIRECTORY`; `rev-parse --git-dir` → `NOT_GIT_REPOSITORY`
  - Add `runner *internalgit.Runner` parameter to all 6 handlers (handleStatus, handleLog, handleDiff, handleStage, handleCommit, handlePush)
  - Replace all `t.runner` references inside handlers with the passed `runner` parameter
  - Update `Execute()` to call `resolveRunner` before the switch, pass result to handlers
  - Update `handleStage`: pass `runner.WorkDir()` to `validateFiles` instead of `t.runner.WorkDir()`
  - Update `handlePush`: use `runner.WorkDir()` for `filepath.Abs` call
  - Update `Definition()` description to mention directory parameter
  - **Acceptance**: Code compiles; calling with/without `directory` both work

- [ ] T003 [US1,US2,SC-002] Add happy-path tests in `internal/mcp/tools/git/tool_test.go`
  - Create second test repo with distinct content (different branch name or commit message)
  - Test all 6 actions with explicit `directory` param pointing to second repo
  - Test default behavior (no `directory`) still uses first repo
  - Tests: TestStatusWithDirectory, TestLogWithDirectory, TestDiffWithDirectory, TestStageWithDirectory, TestCommitWithDirectory, TestPushWithDirectory, TestDirectoryOmitted
  - **Acceptance**: All new tests pass; verify results come from correct repo

- [ ] T004 [US3,SC-003] Add error and edge-case tests in `internal/mcp/tools/git/tool_test.go`
  - TestDirectoryNotFound: nonexistent path → `INVALID_DIRECTORY`
  - TestDirectoryNotGitRepo: temp dir (no git init) → `NOT_GIT_REPOSITORY`
  - TestDirectoryIsFile: file path → `INVALID_DIRECTORY`
  - TestDirectoryEmpty: `directory: ""` → default behavior
  - TestDirectoryRelative: relative path resolves against server default
  - **Acceptance**: All error tests verify correct ToolError codes

- [ ] T005 [SC-001] Run full test suite and verify backward compatibility
  - Run `go test ./internal/mcp/tools/git/...`
  - All pre-existing tests pass without modification
  - All new tests pass
  - **Acceptance**: Zero test failures

## Dependency Graph

```
T001 ──┐
       ├── T003 ──┐
T002 ──┤          ├── T005
       ├── T004 ──┘
       │
       └── (T001 is parallelizable with T002)
```

## FR Traceability

| FR | Task(s) |
|----|---------|
| FR-001 | T001, T002 |
| FR-002 | T002 |
| FR-003 | T002, T003, T004 |
| FR-004 | T002, T004 |
| FR-005 | T002, T004 |
| FR-006 | T002, T003 |
| FR-007 | T001 |
| FR-008 | T002, T004 |
| FR-009 | T001 |
| FR-010 | T002 (signatures unchanged) |
