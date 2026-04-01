# Implementation Plan: Git MCP Tool — Optional Directory Parameter

## Overview

Add an optional `directory` parameter to the git MCP tool. When provided, operations target that directory instead of the server default. When omitted, behavior is unchanged.

## Implementation Steps

### Step 1: Add `directory` to MCP schema (`server.go`)

**File**: `internal/mcp/server.go`
**Change**: Add `mcp.WithString("directory", ...)` to the `registerGitTool` function, after the existing parameters.

```go
mcp.WithString("directory",
    mcp.Description("Working directory for git operations. When omitted, uses the server's default working directory."),
),
```

No `mcp.Required()` — this is optional.

**Also update**: The tool description in `tool.go` `Definition()` — change "All operations run in the server's working directory." to "Operations run in the server's working directory by default, or in the specified directory."

### Step 2: Add directory resolution to `Execute` (`tool.go`)

**File**: `internal/mcp/tools/git/tool.go`
**Change**: At the top of `Execute()`, after extracting `action`, resolve the effective runner:

1. Extract `directory` from args using `getStringArg`.
2. If empty → use `t.runner` (default behavior).
3. If non-empty:
   a. If relative path → resolve against `t.runner.WorkDir()` using `filepath.Join`.
   b. Validate the path exists and is a directory (`os.Stat` + `IsDir()`).
   c. Validate it's a git repo by creating a `NewRunner` and running `rev-parse --git-dir`.
   d. Use the new runner for all operations in this call.

**Add helper function** `resolveRunner`:

```go
func (t *Tool) resolveRunner(args map[string]any) (*internalgit.Runner, error) {
    dir := getStringArg(args, "directory")
    if dir == "" {
        return t.runner, nil
    }

    // Resolve relative paths against default workDir
    if !filepath.IsAbs(dir) {
        dir = filepath.Join(t.runner.WorkDir(), dir)
    }

    // Validate directory exists and is a directory
    info, err := os.Stat(dir)
    if err != nil {
        return nil, &ToolError{
            Code:    "INVALID_DIRECTORY",
            Message: fmt.Sprintf("directory does not exist: %s", dir),
            Hint:    "Provide a valid absolute or relative directory path",
        }
    }
    if !info.IsDir() {
        return nil, &ToolError{
            Code:    "INVALID_DIRECTORY",
            Message: fmt.Sprintf("path is not a directory: %s", dir),
            Hint:    "Provide a path to a directory, not a file",
        }
    }

    // Create runner and verify it's a git repo
    runner, err := internalgit.NewRunner(dir)
    if err != nil {
        return nil, fmt.Errorf("creating runner: %w", err)
    }

    if _, err := runner.Run(context.Background(), "rev-parse", "--git-dir"); err != nil {
        return nil, &ToolError{
            Code:    "NOT_GIT_REPOSITORY",
            Message: fmt.Sprintf("not a git repository: %s", dir),
            Hint:    "Provide a path to a directory inside a git repository",
        }
    }

    return runner, nil
}
```

### Step 3: Update handler signatures (`tool.go`)

**File**: `internal/mcp/tools/git/tool.go`
**Change**: Each handler changes from using `t.runner` to accepting a runner parameter.

Before:
```go
func (t *Tool) handleStatus(ctx context.Context) (string, error) {
    branch, err := t.runner.Run(ctx, "branch", "--show-current")
```

After:
```go
func (t *Tool) handleStatus(ctx context.Context, runner *internalgit.Runner) (string, error) {
    branch, err := runner.Run(ctx, "branch", "--show-current")
```

All handlers get `runner *internalgit.Runner` added to their signature:
- `handleStatus(ctx, runner)` — currently no `args`, add `runner`
- `handleLog(ctx, args, runner)` — add `runner`
- `handleDiff(ctx, args, runner)` — add `runner`
- `handleStage(ctx, args, runner)` — add `runner`, change `t.runner.WorkDir()` → `runner.WorkDir()`
- `handleCommit(ctx, args, runner)` — add `runner`
- `handlePush(ctx, args, runner)` — add `runner`, change `t.runner.WorkDir()` → `runner.WorkDir()`

Update the switch in `Execute` to call `resolveRunner` first and pass it:

```go
func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
    action := getStringArg(args, "action")
    if action == "" {
        return "", &ToolError{...}
    }

    runner, err := t.resolveRunner(args)
    if err != nil {
        return "", err
    }

    switch action {
    case "status":
        return t.handleStatus(ctx, runner)
    case "log":
        return t.handleLog(ctx, args, runner)
    // ... etc
    }
}
```

### Step 4: Add tests (`tool_test.go`)

**File**: `internal/mcp/tools/git/tool_test.go`

Add a new helper `initSecondTestRepo` or reuse `initTestRepo` to create a second temp repo with distinguishable content.

**New tests**:

1. `TestStatusWithDirectory` — tool points to repo A, call with `directory` pointing to repo B. Verify branch/status comes from repo B.
2. `TestLogWithDirectory` — call log with `directory` pointing to repo B. Verify commits from B, not A.
3. `TestDiffWithDirectory` — modify a file in repo B, call diff with `directory`. Verify diff shows B's changes.
4. `TestStageWithDirectory` — create a file in repo B, stage via `directory`. Verify it's staged in B.
5. `TestCommitWithDirectory` — stage + commit in repo B via `directory`. Verify commit lands in B.
6. `TestPushWithDirectory` — set up bare remote for repo B, push via `directory`.
7. `TestDirectoryNotFound` — call with nonexistent directory. Verify `INVALID_DIRECTORY` error.
8. `TestDirectoryNotGitRepo` — call with a temp dir that's not a git repo. Verify `NOT_GIT_REPOSITORY` error.
9. `TestDirectoryIsFile` — call with a file path. Verify `INVALID_DIRECTORY` error.
10. `TestDirectoryEmpty` — call with `directory: ""`. Verify default behavior.
11. `TestDirectoryRelative` — call with relative path. Verify resolution against default workDir.

### Step 5: Run tests and verify

Run `go test ./internal/mcp/tools/git/...` and verify:
- All existing tests still pass (SC-001)
- All new tests pass (SC-002, SC-003)

## Dependency Order

```
Step 1 (schema) ──┐
                   ├── Step 5 (test & verify)
Step 2 (resolve) ──┤
Step 3 (handlers) ─┤
Step 4 (tests) ────┘
```

Steps 1, 2, 3 are implementation (must be sequential in code, but logically parallel).
Step 4 can be written alongside 2-3.
Step 5 comes last.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| `NewRunner` calls `exec.LookPath` each time | Negligible cost for MCP tool call frequency |
| `rev-parse --git-dir` validation adds latency | One extra git call per request with `directory`; acceptable |
| Context not passed to `resolveRunner`'s `rev-parse` | Pass `ctx` from `Execute` to `resolveRunner` |

## No Spike Needed

All technical assumptions verified against actual code:
- `NewRunner(dir)` accepts any directory string ✓
- `validateFiles` takes `workDir` as first arg ✓
- Handler signatures are private methods, safe to change ✓
- `handlePush` uses `t.runner.WorkDir()` — will use passed runner instead ✓
