package worktree

import "context"

// Manager defines the worktree lifecycle operations.
type Manager interface {
	CreateWorktree(ctx context.Context, ticketID, shortTitle string) (string, error)
	DeleteWorktree(ctx context.Context, path string) error
	CleanBranch(ctx context.Context, branch string) error
}

// GitManager implements Manager by shelling out to the git CLI.
type GitManager struct {
	repoDir       string
	worktreesRoot string
	gitBin        string
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
