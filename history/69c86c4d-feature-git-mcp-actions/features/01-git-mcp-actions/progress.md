# Progress Log

### T001 - Create git command runner package
- Status: Complete
- Files: `internal/git/runner.go`
- Notes: Runner, RunSilent, Error type. Extracts exec pattern from worktree manager.

### T002 - Create git tool types and validation
- Status: Complete
- Files: `internal/mcp/tools/git/types.go`, `internal/mcp/tools/git/validation.go`
- Notes: 6 response types, ToolError, 3 validation functions, arg extraction helpers.

### T003 - Implement git tool read actions
- Status: Complete
- Files: `internal/mcp/tools/git/tool.go`
- Notes: status, log, diff actions. All read-only, no side effects.

### T004 - Implement git tool write actions
- Status: Complete
- Files: `internal/mcp/tools/git/tool.go` (same file)
- Notes: stage, commit, push actions. Commit uses temp file for message. Push validates branch protection.

### T005 - Create gh-pr tool types
- Status: Complete
- Files: `internal/mcp/tools/ghpr/types.go`
- Notes: ViewResponse, CreateResponse, CommentResponse.

### T006 - Implement gh-pr tool
- Status: Complete
- Files: `internal/mcp/tools/ghpr/tool.go`
- Notes: view, create, comment actions. Uses gh CLI via exec. Body passed via temp file.

### T007 - Register tools in config and server
- Status: Complete
- Files: `internal/config/tools.go`, `internal/mcp/server.go`, `internal/cli/serve.go`
- Notes: Added "git" and "gh-pr" to KnownTools. NewServer takes optional workDir variadic param (backward compatible). serve.go passes os.Getwd(). Tools nil-guarded if git/gh not available.

### T008 - Integration tests for git tool
- Status: Complete
- Files: `internal/git/runner_test.go`, `internal/mcp/tools/git/tool_test.go`
- Notes: 19 tests total. Real git repos in temp dirs. Push tested with local bare remote.

### T009 - Tests for gh-pr tool
- Status: Complete
- Files: `internal/mcp/tools/ghpr/tool_test.go`
- Notes: 8 tests. Validation-focused. Gracefully skips if gh not in PATH.

## Summary

- **Files created:** 9
- **Files modified:** 3 (config/tools.go, mcp/server.go, cli/serve.go)
- **Tests:** 27 total (4 runner + 15 git tool + 8 gh-pr)
- **All tests pass**, full project builds clean
