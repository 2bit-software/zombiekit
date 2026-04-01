package worktree

import "context"

// Manager defines the worktree lifecycle operations.
type Manager interface {
	CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error)
	DeleteWorktree(ctx context.Context, path string) error
	CleanBranch(ctx context.Context, branch string) error
	PushBranch(ctx context.Context, worktreePath, branch string) error
}

// GitManager implements Manager by shelling out to the git CLI.
type GitManager struct {
	repoDir       string
	worktreesRoot string
	gitBin        string
	filesToCopy   []string
}

// Option configures a GitManager.
type Option func(*GitManager)

// WithWorktreesRoot overrides the default worktrees root directory.
// The path is resolved to an absolute path during construction.
func WithWorktreesRoot(path string) Option {
	return func(m *GitManager) {
		m.worktreesRoot = path
	}
}

// WithCopyFiles specifies files to copy from the repo root into each new
// worktree. Paths are relative to the repo root (e.g., ".env", ".mcp.json").
// Missing source files are skipped with a debug log.
func WithCopyFiles(files []string) Option {
	return func(m *GitManager) {
		m.filesToCopy = files
	}
}
