# Progress Log

### T001 - Add directory param to MCP schema
- Status: Complete
- Files: `internal/mcp/server.go`
- Notes: Added `mcp.WithString("directory", ...)` to `registerGitTool`

### T002 - Add resolveRunner and update handlers
- Status: Complete
- Files: `internal/mcp/tools/git/tool.go`
- Notes: Added `resolveRunner` method; updated all 6 handler signatures to accept `runner` parameter; updated `Definition` description

### T003 - Add happy-path tests
- Status: Complete
- Files: `internal/mcp/tools/git/tool_test.go`
- Notes: 7 new tests (status, log, diff, stage, commit, push with directory; omitted directory)

### T004 - Add error and edge-case tests
- Status: Complete
- Files: `internal/mcp/tools/git/tool_test.go`
- Notes: 5 new tests (not found, not git repo, is file, empty string, relative path)

### T005 - Run full test suite
- Status: Complete
- Notes: 26/26 tests pass (14 existing + 12 new). All existing tests pass without modification.
