# Implementation Plan: Git MCP Actions

## Design Decision: 2 Tools with Action Routing

The business spec proposed 6 separate tools. After reviewing the codebase, the initiative tool pattern (single tool, action-based routing) is the established convention. Two tools give a clean domain split:

| Tool | Category | Actions | Side Effects |
|------|----------|---------|--------------|
| `git` | `git` | `status`, `log`, `diff`, `stage`, `commit`, `push` | Yes (stage, commit, push) |
| `gh-pr` | `gh` | `view`, `create`, `comment` | Yes (create, comment) |

Config toggle: `tools.git.enabled` / `tools.gh.enabled` controls each independently.

Tradeoff: No per-action config (can't disable push but keep commit). If that granularity becomes needed, split into `git-read` / `git-write` later.

## Implementation Steps

### Step 1: Git Command Runner

**File:** `internal/git/runner.go`

Extract a reusable git command runner from `worktree/manager.go`'s `run()` method. The worktree manager's runner is private and tied to the manager struct. We need a standalone version.

```go
type Runner struct {
    gitBin  string
    workDir string
}

func NewRunner(workDir string) (*Runner, error)
func (r *Runner) Run(ctx context.Context, args ...string) (string, error)
func (r *Runner) RunCombined(ctx context.Context, args ...string) (string, string, error) // stdout, stderr, error
```

Key behaviors:
- `exec.CommandContext` with `GIT_TERMINAL_PROMPT=0`
- Capture stdout and stderr separately
- Error classification (reuse `worktree.classifyError` pattern or share it)
- Work directory set via `cmd.Dir`

Why extract? Both `worktree/manager.go` and the new git tool need the same exec pattern. The worktree manager can be refactored to use this runner later (out of scope for this PR).

### Step 2: Git Tool Implementation

**File:** `internal/mcp/tools/git/tool.go`

```go
type Tool struct {
    runner *gitrunner.Runner
}

func NewTool(runner *gitrunner.Runner) *Tool
func (t *Tool) Definition() ToolDefinition
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error)
```

**Actions:**

**`status`** -- Returns combined repo context
- Runs: `git branch --show-current`, `git status --short`, `git status -sb` (for tracking info), `git diff --cached --quiet`
- Returns: `{ branch, status_lines[], has_staged_changes, tracking_info }`

**`log`** -- Returns recent commits
- Params: `base` (optional, default "main"), `count` (optional, default 10), `range` (optional, e.g. "main..HEAD")
- Runs: `git log --oneline -N` or `git log --oneline base..HEAD`
- Returns: `{ commits: [{hash, message}] }`

**`diff`** -- Returns diff content
- Params: `scope` (required: "all", "staged", "unstaged"), `base` (optional), `stat_only` (optional boolean), `paths` (optional string array)
- Runs: `git diff HEAD` / `git diff --cached` / `git diff` / `git diff base...HEAD --stat`
- Returns: `{ diff_text, files_changed }` (or stat output)

**`stage`** -- Stages specific files
- Params: `files` (required string array)
- Validation: reject flags (`-*`), validate each file exists
- Runs: `git add -- file1 file2 ...`
- Returns: updated status (calls status internally)

**`commit`** -- Creates a commit
- Params: `message` (required string)
- Validation: non-empty message, staged changes must exist
- Runs: writes message to temp file, `git commit -F <tempfile>`, cleanup
- Returns: `{ hash, branch, summary }`

**`push`** -- Pushes to remote
- Params: `set_upstream` (optional bool), `remote` (optional, default "origin")
- Validation: refuse main/master, check commits ahead
- Runs: `git push [-u] <remote> <branch>`
- Returns: `{ success, remote, branch, url }`

**File:** `internal/mcp/tools/git/types.go`

Response and error types:
```go
type StatusResponse struct { ... }
type LogResponse struct { ... }
type DiffResponse struct { ... }
type CommitResponse struct { ... }
type PushResponse struct { ... }
type ToolError struct { Code, Message, Hint string }
```

**File:** `internal/mcp/tools/git/validation.go`

Input validation helpers:
```go
func validateFiles(files []string) error      // reject flags, check existence
func validateBranch(branch string) error       // refuse main/master for push
func validateMessage(message string) error     // non-empty
```

### Step 3: GitHub PR Tool Implementation

**File:** `internal/mcp/tools/ghpr/tool.go`

```go
type Tool struct {
    ghBin   string  // path to gh binary
    workDir string
}

func NewTool(workDir string) (*Tool, error)  // LookPath for gh
func (t *Tool) Definition() ToolDefinition
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error)
```

**Actions:**

**`view`** -- Check if PR exists for current branch
- Runs: `gh pr view --json url,title,number,state`
- Returns: `{ exists, url, title, number, state }` or `{ exists: false }`

**`create`** -- Create a new PR
- Params: `title` (required), `body` (required), `base` (optional, default "main"), `draft` (optional bool)
- Validation: non-empty title/body, check no existing PR
- Runs: `gh pr create --base <base> --title <title> --body <body> [--draft]`
- Returns: `{ url, number, title }`

**`comment`** -- Add comment to PR
- Params: `pr_number` (required int), `body` (required string)
- Validation: non-empty body, positive pr_number
- Runs: writes body to temp file, `gh pr comment <number> --body-file <tempfile>`
- Returns: `{ success, pr_number }`

### Step 4: Config Registration

**File:** `internal/config/tools.go`

Add to `KnownTools`:
```go
"git",
"gh-pr",
```

Category derivation handles this automatically:
- `"git"` → category `"git"` (no hyphen, full name)
- `"gh-pr"` → category `"gh"`

### Step 5: Server Integration

**File:** `internal/mcp/server.go`

1. Add fields to `Server` struct:
   ```go
   gitTool  *gittool.Tool
   ghPRTool *ghpr.Tool
   ```

2. In `NewServer()`, create tool instances (conditional on git/gh availability)

3. Add `registerGitTool()` and `registerGHPRTool()` methods

4. Call from `registerTools()`

Registration follows the initiative tool pattern exactly -- `mcp.NewTool()` with parameter definitions, `s.mcpServer.AddTool()` with handler.

### Step 6: Tests

**File:** `internal/mcp/tools/git/tool_test.go`

Integration tests using a real git repo (temp dir):
- `TestGitStatus` -- init repo, check status
- `TestGitStage` -- create file, stage it, verify
- `TestGitCommit` -- stage + commit, verify hash
- `TestGitDiff` -- modify file, check diff output
- `TestGitPush` -- skip in CI (needs remote), or use local bare repo
- `TestValidation` -- flag rejection, empty message, main branch protection

**File:** `internal/mcp/tools/ghpr/tool_test.go`

Limited testing (gh requires auth):
- `TestGHPRViewNoAuth` -- verify graceful error when gh not authenticated
- Mock-based tests if needed for create/comment logic

## Dependency Graph

```
Step 1 (git runner) ← Step 2 (git tool)
                    ← Step 3 (gh-pr tool) [independent of runner, uses own exec]
Step 4 (config)     ← Step 5 (server integration)
Step 2 + Step 3     ← Step 5 (server integration)
Step 2 + Step 3     ← Step 6 (tests)
```

Steps 2, 3, and 4 can be implemented in parallel after Step 1.

## Remaining Uncertainties

1. **Worktree manager refactor**: Should we refactor `worktree/manager.go` to use the new `git/runner.go`? Deferred -- out of scope for this PR but noted as follow-up.

2. **Working directory resolution**: The MCP server runs as a long-lived process. How does the git tool know which directory to operate in? Options:
   - Pass `workDir` as a parameter on every call (explicit, flexible)
   - Set at tool creation time (simpler, but assumes single repo)
   - Use the MCP server's working directory

   Recommendation: Set at tool creation time via `NewTool(runner)`, matching the worktree manager pattern. The MCP server is typically started in a specific project directory.

3. **Large diff truncation**: `git diff` can produce massive output. Should we truncate? Add a `max_lines` parameter? For now, return full output -- the MCP protocol handles large responses and the LLM will context-manage.
