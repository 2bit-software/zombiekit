# Technical Spec: Git MCP Tool — Optional Directory Parameter

## Architecture

The change stays within the git tool layer. No changes to `internal/git/runner.go` or the MCP server handler pattern.

```
┌─────────────────────────────────────────────────┐
│ server.go: registerGitTool()                    │
│  + mcp.WithString("directory", ...)             │
│  handleGit() — unchanged                        │
└──────────────────────┬──────────────────────────┘
                       │ args map[string]any
                       ▼
┌─────────────────────────────────────────────────┐
│ tool.go: Execute()                              │
│  1. Extract action                              │
│  2. resolveRunner(ctx, args) → effective runner │
│  3. Switch on action, pass runner to handler    │
└──────────────────────┬──────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
    handleStatus  handleStage  handlePush
    (uses runner) (uses runner) (uses runner)
```

## API Change

### MCP Tool Schema

New optional parameter added to the git tool:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `directory` | string | No | Working directory for git operations. When omitted, uses the server's default working directory. |

### Behavior Matrix

| `directory` value | Behavior |
|-------------------|----------|
| omitted / `""` | Use `t.runner` (server default) |
| absolute path | Validate, create new runner for that path |
| relative path | Resolve against `t.runner.WorkDir()`, then validate and create new runner |
| nonexistent path | Return `INVALID_DIRECTORY` ToolError |
| path to a file | Return `INVALID_DIRECTORY` ToolError |
| path not in git repo | Return `NOT_GIT_REPOSITORY` ToolError |

## Error Codes

| Code | When | Hint |
|------|------|------|
| `INVALID_DIRECTORY` | Path doesn't exist or is not a directory | "Provide a valid absolute or relative directory path" |
| `NOT_GIT_REPOSITORY` | Path exists but is not inside a git repo | "Provide a path to a directory inside a git repository" |

## Function Signatures

### New function

```go
// resolveRunner returns the effective runner for this call.
// If directory is specified in args, creates a new runner for that directory.
// Otherwise returns the tool's default runner.
func (t *Tool) resolveRunner(ctx context.Context, args map[string]any) (*internalgit.Runner, error)
```

### Modified signatures

All handlers gain a `runner *internalgit.Runner` parameter:

```go
func (t *Tool) handleStatus(ctx context.Context, runner *internalgit.Runner) (string, error)
func (t *Tool) handleLog(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error)
func (t *Tool) handleDiff(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error)
func (t *Tool) handleStage(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error)
func (t *Tool) handleCommit(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error)
func (t *Tool) handlePush(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error)
```

### Unchanged

- `NewTool(runner *internalgit.Runner) *Tool` — no change
- `Execute(ctx context.Context, args map[string]any) (string, error)` — no signature change
- `validateFiles(workDir string, files []string) error` — no change (already takes workDir)
- `server.go` handler: `handleGit(ctx, req)` — no change

## Concurrency

Each call to `Execute` with a `directory` parameter creates its own `Runner` instance. No shared mutable state. Safe for concurrent calls with different directories.

## Files Modified

| File | Change |
|------|--------|
| `internal/mcp/server.go` | Add `directory` parameter to schema |
| `internal/mcp/tools/git/tool.go` | Add `resolveRunner`, update handler signatures, update `Definition` description |
| `internal/mcp/tools/git/tool_test.go` | Add tests for directory parameter |

## Files NOT Modified

| File | Why |
|------|-----|
| `internal/git/runner.go` | Already supports arbitrary directories |
| `internal/mcp/tools/git/types.go` | No new response types needed |
| `internal/mcp/tools/git/validation.go` | Already takes `workDir` as parameter |
