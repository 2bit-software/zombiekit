// Package worktree manages git worktree lifecycle for agent sessions.
//
// Each worktree gets a dedicated branch, enabling parallel ticket work
// without checkout conflicts. The orchestrator creates worktrees when
// spawning agents and deletes them on completion or failure.
//
// # Usage
//
//	mgr, err := worktree.New("/path/to/repo")
//	if err != nil {
//	    // ErrGitUnavailable or ErrNotARepository
//	}
//
//	// Create a worktree for a ticket
//	path, err := mgr.CreateWorktree(ctx, "DEV-185", "git worktree manager")
//	// path: /path/to/worktrees/DEV-185
//	// branch: DEV-185/git-worktree-manager
//
//	// Clean up when done
//	err = mgr.DeleteWorktree(ctx, path)
//
//	// Delete orphaned branch (worktree already removed)
//	err = mgr.CleanBranch(ctx, "DEV-185/git-worktree-manager")
//
// # Worktree Layout
//
// Worktrees are created at {worktrees-root}/{ticket-id} with branches
// named {ticket-id}/{sanitized-short-title}. The worktrees root defaults
// to ../worktrees relative to the repository root.
//
// # Error Handling
//
// All errors are returned as *Error with a classified ErrorKind.
// Use the Is* helper functions for programmatic error checks:
//
//	if worktree.IsPathExists(err) {
//	    // worktree already exists for this ticket
//	}
package worktree
