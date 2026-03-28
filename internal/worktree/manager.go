package worktree

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// New creates a GitManager for the repository at repoDir.
// It eagerly validates that git is available on PATH and that repoDir
// is a git repository.
func New(repoDir string, opts ...Option) (*GitManager, error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return nil, newError(ErrGitUnavailable, "git not found on PATH", err)
	}

	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, fmt.Errorf("resolving repo path: %w", err)
	}

	m := &GitManager{
		repoDir:       absRepo,
		worktreesRoot: filepath.Join(absRepo, "..", "worktrees"),
		gitBin:        gitBin,
	}
	for _, opt := range opts {
		opt(m)
	}

	absRoot, err := filepath.Abs(m.worktreesRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving worktrees root: %w", err)
	}
	m.worktreesRoot = absRoot

	if _, err := m.run(context.Background(), "rev-parse", "--git-dir"); err != nil {
		return nil, newError(ErrNotARepository, fmt.Sprintf("%s is not a git repository", repoDir), err)
	}

	return m, nil
}

// run executes a git command and returns its stdout.
// Errors are classified by parsing stderr.
func (_m *GitManager) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, _m.gitBin, args...)
	cmd.Dir = _m.repoDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		kind := classifyError(stderrStr)
		return "", newError(kind, fmt.Sprintf("git %s: %s", args[0], stderrStr), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// CreateWorktree creates a new worktree at {worktreesRoot}/{ticketID} with
// a branch named {ticketID}/{sanitized-short-title}.
func (_m *GitManager) CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error) {
	sanitized := sanitizeTitle(shortTitle)
	branch := ticketID + "/" + sanitized
	worktreePath := filepath.Join(_m.worktreesRoot, ticketID)

	if err := os.MkdirAll(_m.worktreesRoot, 0o755); err != nil {
		return "", fmt.Errorf("creating worktrees root: %w", err)
	}

	if _, err := _m.run(ctx, "worktree", "add", "-b", branch, worktreePath); err != nil {
		return "", err
	}

	return worktreePath, nil
}

// DeleteWorktree removes a worktree directory and its associated branch.
// Dirty worktrees are force-removed. Locked worktrees return an error.
func (_m *GitManager) DeleteWorktree(ctx context.Context, path string) error {
	branch, err := _m.resolveBranch(ctx, path)
	if err != nil {
		return err
	}

	if _, err := _m.run(ctx, "worktree", "remove", "-f", path); err != nil {
		return err
	}

	if _, err := _m.run(ctx, "branch", "-D", branch); err != nil {
		if !IsBranchNotFound(err) {
			return err
		}
	}

	return nil
}

// CleanBranch force-deletes a local branch. Returns an error if the branch
// has an active worktree or does not exist.
func (_m *GitManager) CleanBranch(ctx context.Context, branch string) error {
	_, err := _m.run(ctx, "branch", "-D", branch)
	return err
}

// resolveBranch parses `git worktree list --porcelain` to find the branch
// associated with the given worktree path.
func (_m *GitManager) resolveBranch(ctx context.Context, path string) (string, error) {
	output, err := _m.run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}

	absPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		absPath, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolving worktree path: %w", err)
		}
	}

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

	return "", newError(ErrNotAWorktree, fmt.Sprintf("%s is not a known worktree", path), nil)
}
