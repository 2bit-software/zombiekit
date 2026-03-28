package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestRepo creates a temporary git repository with an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "user.email", "test@test.com")

	// Create initial commit so HEAD exists
	emptyFile := filepath.Join(dir, ".gitkeep")
	require.NoError(t, os.WriteFile(emptyFile, nil, 0o644))
	runGit(t, dir, "add", ".gitkeep")
	runGit(t, dir, "commit", "-m", "initial")

	return dir
}

// runGit executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}

// branchExists checks whether a branch exists in the repository.
func branchExists(t *testing.T, repoDir, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/"+branch)
	cmd.Dir = repoDir
	return cmd.Run() == nil
}

// worktreeExists checks whether a worktree path is listed.
func worktreeExists(t *testing.T, repoDir, path string) bool {
	t.Helper()
	absPath, _ := filepath.EvalSymlinks(path)
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	for _, line := range splitLines(string(out)) {
		if line == "worktree "+absPath {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range splitByNewline(s) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitByNewline(s string) []string {
	result := []string{}
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// --- Sanitization Tests ---

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal input", "git worktree manager", "git-worktree-manager"},
		{"special chars", "hello!! world@@", "hello-world"},
		{"all special", "!!@@##", "untitled"},
		{"empty string", "", "untitled"},
		{"long title", "this-is-a-very-long-title-that-exceeds-the-forty-character-limit-by-a-lot", "this-is-a-very-long-title-that-exceeds-t"},
		{"leading trailing hyphens", "-hello-world-", "hello-world"},
		{"consecutive hyphens", "hello---world", "hello-world"},
		{"underscores preserved", "hello_world", "hello_world"},
		{"mixed case", "Hello World", "hello-world"},
		{"unicode stripped", "uberflu\u00df", "uberflu"},
		{"numbers preserved", "dev 123 test", "dev-123-test"},
		{"truncation exposes trailing hyphen", "abcdefghijklmnopqrstuvwxyz-abcdefghijklm-nop", "abcdefghijklmnopqrstuvwxyz-abcdefghijklm"},
		{"only spaces", "   ", "untitled"},
		{"hyphens and spaces", "- - -", "untitled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeTitle(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// --- Constructor Tests ---

func TestNew_ValidRepo(t *testing.T) {
	repoDir := initTestRepo(t)

	mgr, err := New(repoDir)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Equal(t, filepath.Join(filepath.Dir(repoDir), "worktrees"), mgr.worktreesRoot)
}

func TestNew_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()

	_, err := New(dir)
	require.Error(t, err)
	assert.True(t, IsNotARepository(err))
}

func TestNew_CustomWorktreesRoot(t *testing.T) {
	repoDir := initTestRepo(t)
	customRoot := filepath.Join(t.TempDir(), "custom-worktrees")

	mgr, err := New(repoDir, WithWorktreesRoot(customRoot))
	require.NoError(t, err)
	assert.Equal(t, customRoot, mgr.worktreesRoot)
}

func TestNew_DefaultRootResolution(t *testing.T) {
	repoDir := initTestRepo(t)

	mgr, err := New(repoDir)
	require.NoError(t, err)

	expected := filepath.Join(filepath.Dir(repoDir), "worktrees")
	assert.Equal(t, expected, mgr.worktreesRoot)
}

// --- CreateWorktree Tests ---

func TestCreateWorktree_HappyPath(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	path, err := mgr.CreateWorktree(context.Background(), "DEV-100", "my feature")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(root, "DEV-100"), path)
	assert.DirExists(t, path)
	assert.True(t, branchExists(t, repoDir, "DEV-100/my-feature"))
	assert.True(t, worktreeExists(t, repoDir, path))
}

func TestCreateWorktree_DuplicateTicketID(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	_, err = mgr.CreateWorktree(context.Background(), "DEV-100", "first")
	require.NoError(t, err)

	_, err = mgr.CreateWorktree(context.Background(), "DEV-100", "second")
	require.Error(t, err)
	assert.True(t, IsPathExists(err), "expected ErrPathExists, got: %v", err)
}

func TestCreateWorktree_BranchCollision(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	// Pre-create the branch that CreateWorktree would use
	runGit(t, repoDir, "branch", "DEV-200/my-feature")

	_, err = mgr.CreateWorktree(context.Background(), "DEV-200", "my feature")
	require.Error(t, err)
	assert.True(t, IsBranchExists(err), "expected ErrBranchExists, got: %v", err)
}

func TestCreateWorktree_AutoCreatesRoot(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "nested", "deep", "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	path, err := mgr.CreateWorktree(context.Background(), "DEV-300", "test")
	require.NoError(t, err)
	assert.DirExists(t, path)
}

func TestCreateWorktree_ContextCancellation(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = mgr.CreateWorktree(ctx, "DEV-400", "cancelled")
	require.Error(t, err)
}

// --- DeleteWorktree Tests ---

func TestDeleteWorktree_HappyPath(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	path, err := mgr.CreateWorktree(context.Background(), "DEV-500", "to delete")
	require.NoError(t, err)

	err = mgr.DeleteWorktree(context.Background(), path)
	require.NoError(t, err)

	assert.NoDirExists(t, path)
	assert.False(t, branchExists(t, repoDir, "DEV-500/to-delete"))
	assert.False(t, worktreeExists(t, repoDir, path))
}

func TestDeleteWorktree_NonexistentPath(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	err = mgr.DeleteWorktree(context.Background(), filepath.Join(root, "nonexistent"))
	require.Error(t, err)
	assert.True(t, IsNotAWorktree(err), "expected ErrNotAWorktree, got: %v", err)
}

func TestDeleteWorktree_DirtyWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	path, err := mgr.CreateWorktree(context.Background(), "DEV-600", "dirty")
	require.NoError(t, err)

	// Create uncommitted file in the worktree
	require.NoError(t, os.WriteFile(filepath.Join(path, "dirty.txt"), []byte("dirty"), 0o644))

	err = mgr.DeleteWorktree(context.Background(), path)
	require.NoError(t, err, "dirty worktree should be force-removed")

	assert.NoDirExists(t, path)
}

func TestDeleteWorktree_LockedWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	path, err := mgr.CreateWorktree(context.Background(), "DEV-700", "locked")
	require.NoError(t, err)

	// Lock the worktree
	runGit(t, repoDir, "worktree", "lock", path)

	err = mgr.DeleteWorktree(context.Background(), path)
	require.Error(t, err)
	assert.True(t, IsWorktreeLocked(err), "expected ErrWorktreeLocked, got: %v", err)

	// Verify it was NOT removed
	assert.DirExists(t, path)
}

// --- CleanBranch Tests ---

func TestCleanBranch_HappyPath(t *testing.T) {
	repoDir := initTestRepo(t)
	mgr, err := New(repoDir)
	require.NoError(t, err)

	runGit(t, repoDir, "branch", "orphan-branch")
	assert.True(t, branchExists(t, repoDir, "orphan-branch"))

	err = mgr.CleanBranch(context.Background(), "orphan-branch")
	require.NoError(t, err)
	assert.False(t, branchExists(t, repoDir, "orphan-branch"))
}

func TestCleanBranch_BranchWithWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	mgr, err := New(repoDir, WithWorktreesRoot(root))
	require.NoError(t, err)

	_, err = mgr.CreateWorktree(context.Background(), "DEV-800", "active")
	require.NoError(t, err)

	err = mgr.CleanBranch(context.Background(), "DEV-800/active")
	require.Error(t, err)
	assert.True(t, IsBranchInUse(err), "expected ErrBranchInUse, got: %v", err)
}

func TestCleanBranch_NonexistentBranch(t *testing.T) {
	repoDir := initTestRepo(t)
	mgr, err := New(repoDir)
	require.NoError(t, err)

	err = mgr.CleanBranch(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.True(t, IsBranchNotFound(err), "expected ErrBranchNotFound, got: %v", err)
}
