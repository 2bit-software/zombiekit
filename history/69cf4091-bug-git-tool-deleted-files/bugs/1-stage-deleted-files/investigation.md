# Investigation

## Relevant Files

- `internal/mcp/tools/git/validation.go` — `validateFiles()` function (line 10-42)
- `internal/mcp/tools/git/tool.go` — `handleStage()` function (line 245-276)
- `internal/git/runner.go` — `Runner` type used to execute git commands

## Execution Flow

1. `handleStage()` parses file list from args
2. Calls `validateFiles(runner.WorkDir(), files)`
3. `validateFiles` iterates files, checks for flag-like paths, then calls `os.Stat(path)` at line 33
4. For deleted files, `os.Stat` fails → returns `ToolError{Code: "VALIDATION_ERROR", Message: "file does not exist"}`
5. `git add` is never reached

## Root Cause

`validateFiles` assumes all files to be staged must exist on disk. This is incorrect — `git add` can stage deletions of tracked files. The validation is overly strict.

## Key Insight

The validation needs a fallback: if `os.Stat` fails, check whether the file is known to git (tracked). `git ls-files <path>` returns output for tracked files even if deleted from disk. If the file is both absent from disk AND not tracked by git, then it's genuinely nonexistent and should be rejected.
