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

// run executes a git command from the repo directory and returns its stdout.
// Errors are classified by parsing stderr.
func (_m *GitManager) run(ctx context.Context, args ...string) (string, error) {
	return _m.runFrom(ctx, _m.repoDir, args...)
}

// runFrom executes a git command from the given directory and returns its stdout.
func (_m *GitManager) runFrom(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, _m.gitBin, args...)
	cmd.Dir = dir
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
//
// If the branch or worktree path already exists from a previous run, it
// cleans up the stale state and retries once. If the existing worktree has
// file conflicts that prevent cleanup, it returns an error.
func (_m *GitManager) CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error) {
	sanitized := sanitizeTitle(shortTitle)
	branch := ticketID + "/" + sanitized
	worktreePath := filepath.Join(_m.worktreesRoot, ticketID)

	if err := os.MkdirAll(_m.worktreesRoot, 0o755); err != nil {
		return "", fmt.Errorf("creating worktrees root: %w", err)
	}

	_, err := _m.run(ctx, "worktree", "add", "-b", branch, worktreePath)
	if err != nil {
		if !IsBranchExists(err) && !IsPathExists(err) {
			return "", err
		}

		// Stale worktree or branch from a previous run — clean up and retry.
		if cleanErr := _m.cleanStaleWorktree(ctx, worktreePath, branch); cleanErr != nil {
			return "", fmt.Errorf("cleaning stale worktree for %s: %w", ticketID, cleanErr)
		}

		if _, err := _m.run(ctx, "worktree", "add", "-b", branch, worktreePath); err != nil {
			return "", err
		}
	}

	if err := _m.copyFiles(worktreePath); err != nil {
		return worktreePath, fmt.Errorf("copy files: %w", err)
	}

	return worktreePath, nil
}

// cleanStaleWorktree removes a leftover worktree path and/or branch so
// CreateWorktree can retry. It handles three cases:
//  1. Worktree path exists in git's tracking — remove it, then delete its branch
//  2. Only the directory exists on disk (git lost track) — remove directory
//  3. Only the branch exists (worktree was removed but branch lingers) — delete branch
func (_m *GitManager) cleanStaleWorktree(ctx context.Context, path, branch string) error {
	// Resolve the branch currently associated with this worktree path,
	// which may differ from the new branch name we're trying to create.
	oldBranch, _ := _m.resolveBranch(ctx, path)

	// Try git worktree remove first (handles case 1).
	if _, err := _m.run(ctx, "worktree", "remove", "-f", path); err != nil {
		if IsWorktreeLocked(err) {
			return err
		}
		// Worktree not tracked by git — remove the directory if it exists (case 2).
		if IsNotAWorktree(err) {
			if rmErr := os.RemoveAll(path); rmErr != nil {
				return fmt.Errorf("removing stale directory %s: %w", path, rmErr)
			}
		}
	}

	// Delete branches that may be lingering. The old branch (from the
	// previous worktree) and the new branch (that we're about to create)
	// may be different — clean up both.
	for _, b := range uniqueNonEmpty(oldBranch, branch) {
		if _, err := _m.run(ctx, "branch", "-D", b); err != nil {
			if !IsBranchNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// uniqueNonEmpty returns the unique non-empty strings from the arguments.
func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]bool, len(values))
	var result []string
	for _, v := range values {
		if v != "" && !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
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
	}
	if err != nil {
		return "", fmt.Errorf("resolving worktree path: %w", err)
	}

	for _, block := range strings.Split(output, "\n\n") {
		var wtPath, branch string
		for _, line := range strings.Split(strings.TrimSpace(block), "\n") {
			if after, ok := strings.CutPrefix(line, "worktree "); ok {
				wtPath = after
			} else if after, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
				branch = after
			}
		}
		if wtPath == absPath && branch != "" {
			return branch, nil
		}
	}

	return "", newError(ErrNotAWorktree, fmt.Sprintf("%s is not a known worktree", path), nil)
}

// PushBranch pushes a branch from the given worktree to origin.
func (_m *GitManager) PushBranch(ctx context.Context, worktreePath, branch string) error {
	_, err := _m.runFrom(ctx, worktreePath, "push", "-u", "origin", branch)
	return err
}

// copyFiles copies configured files from the repo root into the worktree.
// Missing source files are skipped with a debug log. Copy errors are returned.
func (_m *GitManager) copyFiles(worktreePath string) error {
	for _, relPath := range _m.filesToCopy {
		src := filepath.Join(_m.repoDir, relPath)

		info, err := os.Stat(src)
		if err != nil {
			// Missing source is expected (file may be optional).
			continue
		}
		if info.IsDir() {
			continue
		}

		dst := filepath.Join(worktreePath, relPath)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", relPath, err)
		}

		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", relPath, err)
		}

		if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}
	return nil
}
