package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bit-software/zombiekit/internal/git"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "setup command failed: %v", args)
	}
	return dir
}

func TestNewRunner(t *testing.T) {
	dir := initTestRepo(t)
	runner, err := git.NewRunner(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, runner.WorkDir())
}

func TestRunSimpleCommand(t *testing.T) {
	dir := initTestRepo(t)
	runner, err := git.NewRunner(dir)
	require.NoError(t, err)

	out, err := runner.Run(context.Background(), "status")
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestRunReturnsErrorOnFailure(t *testing.T) {
	dir := initTestRepo(t)
	runner, err := git.NewRunner(dir)
	require.NoError(t, err)

	_, err = runner.Run(context.Background(), "log")
	assert.Error(t, err, "git log on empty repo should fail")

	var gitErr *git.Error
	assert.ErrorAs(t, err, &gitErr)
	assert.Contains(t, gitErr.Stderr, "does not have any commits")
}

func TestRunSilent(t *testing.T) {
	dir := initTestRepo(t)
	runner, err := git.NewRunner(dir)
	require.NoError(t, err)

	// Create and stage a file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644))
	_, err = runner.Run(context.Background(), "add", "test.txt")
	require.NoError(t, err)

	// RunSilent should return error because staged changes exist (diff --cached --quiet fails)
	err = runner.RunSilent(context.Background(), "diff", "--cached", "--quiet")
	assert.Error(t, err, "should detect staged changes")

	// Commit, then RunSilent should succeed (no staged changes)
	_, err = runner.Run(context.Background(), "commit", "-m", "init")
	require.NoError(t, err)

	err = runner.RunSilent(context.Background(), "diff", "--cached", "--quiet")
	assert.NoError(t, err, "should have no staged changes after commit")
}
