# Fix Plan

## Changes Required

### 1. `internal/mcp/tools/git/validation.go` — `validateFiles`

**Current**: `os.Stat(path)` fails → reject file.

**Fix**: Change signature to accept a `context.Context` and `*git.Runner`. When `os.Stat` fails, run `git ls-files <file>` via the runner. If `ls-files` returns output, the file is tracked (deleted from disk but known to git) — allow it. If `ls-files` returns empty, the file is genuinely nonexistent — reject.

```go
func validateFiles(ctx context.Context, runner *internalgit.Runner, files []string) error {
    // ... existing flag check ...

    path := resolvedPath
    if _, err := os.Stat(path); err != nil {
        // File not on disk — check if git tracks it
        out, gitErr := runner.Run(ctx, "ls-files", f)
        if gitErr != nil || strings.TrimSpace(out) == "" {
            return &ToolError{...}  // genuinely nonexistent
        }
        // File is tracked but deleted — allow staging
    }
}
```

### 2. `internal/mcp/tools/git/tool.go` — `handleStage`

Update the `validateFiles` call to pass `ctx` and `runner`:

```go
if err := validateFiles(ctx, runner, files); err != nil {
```

### 3. `internal/mcp/tools/git/tool_test.go` — Add test

Add `TestStageDeletedFile`:
1. Init repo, create+commit a file
2. Delete the file from disk
3. Call stage action with the deleted file
4. Assert success — file should be listed in staged files

Also verify existing `TestStageRejectsNonexistentFile` still passes (file that was never tracked).

## Verification

- `go test ./internal/mcp/tools/git/...` — all tests pass
- Manual test: delete a tracked file, stage via MCP tool, verify `git status` shows staged deletion
