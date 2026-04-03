# Initiative: git-tool-deleted-files

**Type**: bug
**Status**: in_progress
**Created**: 2026-04-02
**ID**: 69cf4091-bug-git-tool-deleted-files

## Steps

| Step | Status | Updated |
|------|--------|--------|
| investigate | complete | 2026-04-02 |
| fix | complete | 2026-04-02 |
| verify | complete | 2026-04-02 |

## Description

MCP git tool's `stage` action rejected deleted files because `validateFiles` used `os.Stat` which fails for files removed from disk. Added git-aware fallback using `git ls-files` to allow staging tracked-but-deleted files.

## Completion

**Completed**: 2026-04-02
**Status**: complete

### Outcomes
- Bug: stage-deleted-files - Fixed

### Changes
- `internal/mcp/tools/git/validation.go` — Added `context.Context` and `*git.Runner` params; falls back to `git ls-files` when `os.Stat` fails
- `internal/mcp/tools/git/tool.go` — Updated `validateFiles` call site
- `internal/mcp/tools/git/tool_test.go` — Added `TestStageDeletedFile`
