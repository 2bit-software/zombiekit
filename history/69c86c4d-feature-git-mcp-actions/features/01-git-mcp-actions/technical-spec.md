# Technical Spec: Git MCP Actions

## Package Layout

```
internal/
  git/
    runner.go          # Git command execution (extracted from worktree)
    runner_test.go
  mcp/
    tools/
      git/
        tool.go        # Git MCP tool (status, log, diff, stage, commit, push)
        types.go       # Response structs, ToolError
        validation.go  # Input validation
        tool_test.go
      ghpr/
        tool.go        # GitHub PR MCP tool (view, create, comment)
        types.go       # Response structs
        tool_test.go
    server.go          # +registerGitTool(), +registerGHPRTool()
  config/
    tools.go           # +KnownTools entries
```

## Interfaces and Types

### git/runner.go

```go
package git

import (
    "bytes"
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"
)

// Runner executes git commands in a working directory.
type Runner struct {
    gitBin  string
    workDir string
}

// NewRunner creates a Runner for the given working directory.
// Returns error if git is not found in PATH.
func NewRunner(workDir string) (*Runner, error) {
    gitBin, err := exec.LookPath("git")
    if err != nil {
        return nil, fmt.Errorf("git not found: %w", err)
    }
    return &Runner{gitBin: gitBin, workDir: workDir}, nil
}

// Run executes a git command and returns stdout.
func (r *Runner) Run(ctx context.Context, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, r.gitBin, args...)
    cmd.Dir = r.workDir
    cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", &GitError{
            Args:    args,
            Stderr:  strings.TrimSpace(stderr.String()),
            Wrapped: err,
        }
    }
    return strings.TrimSpace(stdout.String()), nil
}

// GitError wraps a failed git command with stderr output.
type GitError struct {
    Args    []string
    Stderr  string
    Wrapped error
}

func (e *GitError) Error() string {
    return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

func (e *GitError) Unwrap() error {
    return e.Wrapped
}
```

### mcp/tools/git/types.go

```go
package git

type StatusResponse struct {
    Branch          string   `json:"branch"`
    StatusLines     []string `json:"status_lines"`
    HasStagedChanges bool    `json:"has_staged_changes"`
    TrackingInfo    string   `json:"tracking_info"`
}

type LogEntry struct {
    Hash    string `json:"hash"`
    Message string `json:"message"`
}

type LogResponse struct {
    Commits []LogEntry `json:"commits"`
    Count   int        `json:"count"`
}

type DiffResponse struct {
    Content      string `json:"content"`
    FilesChanged int    `json:"files_changed,omitempty"`
    StatOnly     bool   `json:"stat_only,omitempty"`
}

type StageResponse struct {
    StagedFiles []string       `json:"staged_files"`
    Status      StatusResponse `json:"status"`
}

type CommitResponse struct {
    Hash    string `json:"hash"`
    Branch  string `json:"branch"`
    Summary string `json:"summary"`
}

type PushResponse struct {
    Success  bool   `json:"success"`
    Remote   string `json:"remote"`
    Branch   string `json:"branch"`
    Output   string `json:"output"`
}

type ToolError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Hint    string `json:"hint,omitempty"`
}

func (e *ToolError) Error() string {
    if e.Hint != "" {
        return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Hint)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### mcp/tools/ghpr/types.go

```go
package ghpr

type ViewResponse struct {
    Exists bool   `json:"exists"`
    URL    string `json:"url,omitempty"`
    Title  string `json:"title,omitempty"`
    Number int    `json:"number,omitempty"`
    State  string `json:"state,omitempty"`
}

type CreateResponse struct {
    URL    string `json:"url"`
    Number int    `json:"number"`
    Title  string `json:"title"`
}

type CommentResponse struct {
    Success  bool `json:"success"`
    PRNumber int  `json:"pr_number"`
}
```

## Tool Parameter Schemas

### `git` Tool

```json
{
  "action": {
    "type": "string",
    "enum": ["status", "log", "diff", "stage", "commit", "push"],
    "required": true,
    "description": "Git operation to perform"
  },
  "base": {
    "type": "string",
    "description": "Base ref for log/diff (default: main)"
  },
  "count": {
    "type": "integer",
    "description": "Number of log entries (default: 10)"
  },
  "scope": {
    "type": "string",
    "enum": ["all", "staged", "unstaged"],
    "description": "Diff scope (required for diff action)"
  },
  "stat_only": {
    "type": "boolean",
    "description": "Return only file stat, not full diff"
  },
  "paths": {
    "type": "string",
    "description": "Comma-separated file paths to limit diff"
  },
  "files": {
    "type": "string",
    "description": "Comma-separated file paths to stage (required for stage action)"
  },
  "message": {
    "type": "string",
    "description": "Commit message (required for commit action)"
  },
  "set_upstream": {
    "type": "boolean",
    "description": "Set upstream tracking on push (default: false)"
  },
  "remote": {
    "type": "string",
    "description": "Remote name for push (default: origin)"
  }
}
```

### `gh-pr` Tool

```json
{
  "action": {
    "type": "string",
    "enum": ["view", "create", "comment"],
    "required": true,
    "description": "PR operation to perform"
  },
  "title": {
    "type": "string",
    "description": "PR title (required for create)"
  },
  "body": {
    "type": "string",
    "description": "PR body or comment text (required for create/comment)"
  },
  "base": {
    "type": "string",
    "description": "Base branch for PR (default: main)"
  },
  "draft": {
    "type": "boolean",
    "description": "Create PR as draft (default: false)"
  },
  "pr_number": {
    "type": "integer",
    "description": "PR number (required for comment)"
  }
}
```

## Validation Rules

### git-stage
- `files` must be non-empty
- Each file path must not start with `-` (flag injection)
- Each file must exist in the working tree (`os.Stat`)

### git-commit
- `message` must be non-empty after trimming
- `git diff --cached --quiet` must fail (meaning staged changes exist)
- No `--amend` or `--no-verify` support (by design)

### git-push
- Current branch must not be `main` or `master`
- `git log <remote>/<branch>..HEAD` must show at least one commit

### gh-pr create
- `title` must be non-empty
- `body` must be non-empty
- `gh pr view` must indicate no existing PR

### gh-pr comment
- `pr_number` must be positive integer
- `body` must be non-empty

## Error Response Format

All errors return via `mcp.NewToolResultError()` with structured messages:

```
VALIDATION_ERROR: files parameter is required for stage action
GIT_ERROR: git push: fatal: the current branch has no upstream branch
BRANCH_PROTECTED: cannot push to main (Use a feature branch)
NO_STAGED_CHANGES: nothing staged for commit (Stage files first with action=stage)
GH_NOT_FOUND: gh CLI not found in PATH (Install: https://cli.github.com)
PR_EXISTS: PR already exists for this branch (URL: https://...)
```

## Security Considerations

1. **No shell expansion**: All commands use `exec.CommandContext` with explicit args (no shell=true)
2. **Flag injection prevention**: File paths validated to not start with `-`
3. **Branch protection**: Push refuses main/master
4. **No force operations**: No `--force`, `--amend`, `--no-verify`
5. **Temp file cleanup**: Commit message temp files cleaned via `defer os.Remove()`
6. **GIT_TERMINAL_PROMPT=0**: Prevents interactive auth prompts from hanging
7. **PR body via temp file**: `gh pr create` body passed via `--body` arg (not shell expansion)
