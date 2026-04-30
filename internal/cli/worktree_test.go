package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestRepo sets up a temp git repo with an initial commit and returns
// the repo directory.
func initTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "config", "user.email", "test@test.com")

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644))
	mustGit(t, dir, "add", ".gitkeep")
	mustGit(t, dir, "commit", "-m", "initial")

	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// runBrains invokes the brains CLI with the given arguments against a fresh
// app instance. It chdirs into runFromDir for the duration of the call so
// cwd-based config auto-detection picks up the right repo.
func runBrains(t *testing.T, runFromDir string, args ...string) error {
	t.Helper()
	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(runFromDir))
	defer func() { _ = os.Chdir(prev) }()

	app := NewApp(&version.BuildInfo{})
	return app.RunContext(context.Background(), append([]string{"brains"}, args...))
}

func TestWorktreeCreate_AutoDetect(t *testing.T) {
	repo := initTestRepo(t)

	err := runBrains(t, repo, "worktree", "create", "DEV-1", "first feature")
	require.NoError(t, err)

	wtPath := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-1")
	assert.DirExists(t, wtPath)

	// Branch DEV-1/first-feature should exist on the original repo.
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/DEV-1/first-feature")
	cmd.Dir = repo
	require.NoError(t, cmd.Run(), "expected branch DEV-1/first-feature to exist")
}

func TestWorktreeDelete_AutoDetect(t *testing.T) {
	repo := initTestRepo(t)

	require.NoError(t, runBrains(t, repo, "worktree", "create", "DEV-2", "doomed"))

	wtPath := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-2")
	require.DirExists(t, wtPath)

	require.NoError(t, runBrains(t, repo, "worktree", "delete", wtPath))
	assert.NoDirExists(t, wtPath)

	// Branch should also be gone.
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/DEV-2/doomed")
	cmd.Dir = repo
	assert.Error(t, cmd.Run(), "expected branch DEV-2/doomed to be deleted")
}

func TestWorktreeCleanBranch_AutoDetect(t *testing.T) {
	repo := initTestRepo(t)

	mustGit(t, repo, "branch", "DEV-3/orphan")

	require.NoError(t, runBrains(t, repo, "worktree", "clean-branch", "DEV-3/orphan"))

	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/DEV-3/orphan")
	cmd.Dir = repo
	assert.Error(t, cmd.Run(), "expected branch DEV-3/orphan to be deleted")
}

func TestWorktreeList_AutoDetect(t *testing.T) {
	repo := initTestRepo(t)
	require.NoError(t, runBrains(t, repo, "worktree", "create", "DEV-4", "listed"))

	// list returns nil on success; we mainly verify the command path doesn't error.
	require.NoError(t, runBrains(t, repo, "worktree", "list"))
}

func TestWorktreeCreate_MissingArgs(t *testing.T) {
	repo := initTestRepo(t)
	err := runBrains(t, repo, "worktree", "create", "DEV-5")
	require.Error(t, err)
}

func TestWorktreeCreate_NoGitRepo(t *testing.T) {
	tmp := t.TempDir()
	err := runBrains(t, tmp, "worktree", "create", "DEV-6", "should fail")
	require.Error(t, err)
}
