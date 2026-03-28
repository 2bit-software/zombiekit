# Technical Specification: Git Worktree Manager

## Package: `internal/worktree`

### Types

```go
// Manager defines the worktree lifecycle operations.
type Manager interface {
    CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error)
    DeleteWorktree(ctx context.Context, path string) error
    CleanBranch(ctx context.Context, branch string) error
}

// Option configures a GitManager.
type Option func(*GitManager)

// WithWorktreesRoot overrides the default worktrees root directory.
func WithWorktreesRoot(path string) Option {
    return func(m *GitManager) {
        m.worktreesRoot = path
    }
}
```

### Error Types

```go
type ErrorKind int

const (
    ErrPathExists     ErrorKind = iota + 1
    ErrBranchExists
    ErrNotAWorktree
    ErrWorktreeLocked
    ErrBranchInUse
    ErrBranchNotFound
    ErrGitUnavailable
    ErrNotARepository
    ErrGitCommand
)

type Error struct {
    Kind    ErrorKind
    Message string
    Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }
```

Error classification by parsing stderr:

```go
func classifyError(stderr string) ErrorKind {
    switch {
    case strings.Contains(stderr, "already used by worktree"):
        return ErrBranchExists  // branch checked out in another worktree
    case strings.Contains(stderr, "a branch named") && strings.Contains(stderr, "already exists"):
        return ErrBranchExists
    case strings.Contains(stderr, "already exists"):
        return ErrPathExists    // path already exists
    case strings.Contains(stderr, "is not a working tree"):
        return ErrNotAWorktree
    case strings.Contains(stderr, "cannot remove a locked"):
        return ErrWorktreeLocked
    case strings.Contains(stderr, "contains modified or untracked"):
        return ErrGitCommand    // dirty state — shouldn't reach here with -f
    case strings.Contains(stderr, "cannot delete branch"):
        return ErrBranchInUse
    case strings.Contains(stderr, "not found"):
        return ErrBranchNotFound
    default:
        return ErrGitCommand
    }
}
```

### Core Struct

```go
type GitManager struct {
    repoDir       string
    worktreesRoot string
    gitBin        string // cached path to git binary
}

func New(repoDir string, opts ...Option) (*GitManager, error) {
    gitBin, err := exec.LookPath("git")
    if err != nil {
        return nil, &Error{Kind: ErrGitUnavailable, Message: "git not found on PATH", Err: err}
    }

    m := &GitManager{
        repoDir:       repoDir,
        worktreesRoot: filepath.Join(repoDir, "..", "worktrees"),
        gitBin:        gitBin,
    }
    for _, opt := range opts {
        opt(m)
    }

    // Resolve to absolute paths
    m.repoDir, _ = filepath.Abs(m.repoDir)
    m.worktreesRoot, _ = filepath.Abs(m.worktreesRoot)

    // Validate repo
    if _, err := m.run(context.Background(), "rev-parse", "--git-dir"); err != nil {
        return nil, &Error{Kind: ErrNotARepository, Message: fmt.Sprintf("%s is not a git repository", repoDir), Err: err}
    }

    return m, nil
}
```

### Git Runner

```go
func (_m *GitManager) run(ctx context.Context, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, _m.gitBin, args...)
    cmd.Dir = _m.repoDir
    cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    if err != nil {
        kind := classifyError(stderr.String())
        return "", &Error{
            Kind:    kind,
            Message: fmt.Sprintf("git %s: %s", args[0], strings.TrimSpace(stderr.String())),
            Err:     err,
        }
    }
    return strings.TrimSpace(stdout.String()), nil
}
```

### CreateWorktree

```go
func (_m *GitManager) CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error) {
    sanitized := sanitizeTitle(shortTitle)
    branch := ticketID + "/" + sanitized
    worktreePath := filepath.Join(_m.worktreesRoot, ticketID)

    // Ensure worktrees root exists
    if err := os.MkdirAll(_m.worktreesRoot, 0o755); err != nil {
        return "", fmt.Errorf("creating worktrees root: %w", err)
    }

    // git worktree add -b <branch> <path>
    if _, err := _m.run(ctx, "worktree", "add", "-b", branch, worktreePath); err != nil {
        return "", err // already classified by run()
    }

    return worktreePath, nil
}
```

### DeleteWorktree

```go
func (_m *GitManager) DeleteWorktree(ctx context.Context, path string) error {
    // Resolve branch from worktree list
    branch, err := _m.resolveBranch(ctx, path)
    if err != nil {
        return err
    }

    // Force-remove worktree (handles dirty state)
    if _, err := _m.run(ctx, "worktree", "remove", "-f", path); err != nil {
        return err
    }

    // Force-delete branch
    if _, err := _m.run(ctx, "branch", "-D", branch); err != nil {
        // If branch is already gone (e.g., manual cleanup), that's fine
        if !IsBranchNotFound(err) {
            return err
        }
    }

    return nil
}
```

### Branch Resolution

```go
// resolveBranch parses `git worktree list --porcelain` to find the branch
// associated with the given worktree path.
func (_m *GitManager) resolveBranch(ctx context.Context, path string) (string, error) {
    output, err := _m.run(ctx, "worktree", "list", "--porcelain")
    if err != nil {
        return "", err
    }

    absPath, _ := filepath.Abs(path)
    blocks := strings.Split(output, "\n\n")
    for _, block := range blocks {
        lines := strings.Split(strings.TrimSpace(block), "\n")
        var wtPath, branch string
        for _, line := range lines {
            if after, ok := strings.CutPrefix(line, "worktree "); ok {
                wtPath = after
            }
            if after, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
                branch = after
            }
        }
        if wtPath == absPath && branch != "" {
            return branch, nil
        }
    }

    return "", &Error{
        Kind:    ErrNotAWorktree,
        Message: fmt.Sprintf("%s is not a known worktree", path),
    }
}
```

### CleanBranch

```go
func (_m *GitManager) CleanBranch(ctx context.Context, branch string) error {
    if _, err := _m.run(ctx, "branch", "-D", branch); err != nil {
        return err
    }
    return nil
}
```

### Sanitization

```go
func sanitizeTitle(title string) string {
    // 1. Lowercase
    s := strings.ToLower(title)
    // 2. Replace spaces with hyphens
    s = strings.ReplaceAll(s, " ", "-")
    // 3. Strip non-ASCII and non-alphanumeric (keep hyphens, underscores)
    var b strings.Builder
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
            b.WriteRune(r)
        }
    }
    s = b.String()
    // 4. Collapse consecutive hyphens
    for strings.Contains(s, "--") {
        s = strings.ReplaceAll(s, "--", "-")
    }
    // 5. Trim leading/trailing hyphens
    s = strings.Trim(s, "-")
    // 6. Truncate to 40 chars
    if len(s) > 40 {
        s = s[:40]
    }
    // 7. Trim trailing hyphens after truncation
    s = strings.TrimRight(s, "-")
    // 8. Fallback
    if s == "" {
        return "untitled"
    }
    return s
}
```

## Design Decisions

1. **Single `run()` method** — all git operations go through one execution path, making error classification consistent and testable.
2. **Cached `gitBin` path** — `exec.LookPath` called once in constructor, not per operation.
3. **Force-delete in DeleteWorktree** — dirty state is force-removed because the agent is responsible for committing before signaling completion. Locked worktrees are NOT force-removed (caller must unlock).
4. **BranchNotFound suppressed in DeleteWorktree** — if the branch is already gone during delete, that's not an error. The goal (branch gone) is achieved.
5. **Porcelain parsing for branch resolution** — stable format across git versions, no regex needed.
6. **No mutex** — the orchestrator serializes operations per ticket ID.
