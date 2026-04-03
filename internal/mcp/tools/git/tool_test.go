package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalgit "github.com/2bit-software/zombiekit/internal/git"
	gittool "github.com/2bit-software/zombiekit/internal/mcp/tools/git"
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
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}

	// Create initial commit so HEAD exists
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644))
	for _, args := range [][]string{
		{"git", "add", "README.md"},
		{"git", "commit", "-m", "initial commit"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}
	return dir
}

func newTool(t *testing.T, dir string) *gittool.Tool {
	t.Helper()
	runner, err := internalgit.NewRunner(dir)
	require.NoError(t, err)
	return gittool.NewTool(runner)
}

func TestStatus(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"branch"`)
	assert.Contains(t, result, `"has_staged_changes"`)
}

func TestLog(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "log",
		"count":  float64(5),
	})
	require.NoError(t, err)
	assert.Contains(t, result, "initial commit")
	assert.Contains(t, result, `"count": 1`)
}

func TestDiffAll(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	// Modify a file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Updated"), 0644))

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "diff",
		"scope":  "all",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Updated")
}

func TestDiffStaged(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	// Create and stage a new file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("content"), 0644))
	cmd := exec.Command("git", "add", "new.txt")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "diff",
		"scope":  "staged",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "new.txt")
}

func TestDiffStatOnly(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Changed"), 0644))

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "diff",
		"scope":     "all",
		"stat_only": true,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "README.md")
	assert.Contains(t, result, `"stat_only": true`)
}

func TestStage(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	// Create a file to stage
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stage-me.txt"), []byte("data"), 0644))

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "stage",
		"files":  "stage-me.txt",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "stage-me.txt")
}

func TestStageRejectsFlags(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "stage",
		"files":  "--all",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "looks like a flag")
}

func TestStageRejectsNonexistentFile(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "stage",
		"files":  "nonexistent.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestStageDeletedFile(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	// Create and commit a file
	trackedFile := filepath.Join(dir, "tracked.txt")
	require.NoError(t, os.WriteFile(trackedFile, []byte("hello"), 0644))
	for _, args := range [][]string{
		{"git", "add", "tracked.txt"},
		{"git", "commit", "-m", "add tracked file"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}

	// Delete the file from disk
	require.NoError(t, os.Remove(trackedFile))

	// Stage the deletion via the tool
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "stage",
		"files":  "tracked.txt",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "tracked.txt")
}

func TestCommit(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	// Create and stage a file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "commit-me.txt"), []byte("data"), 0644))
	cmd := exec.Command("git", "add", "commit-me.txt")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":  "commit",
		"message": "test commit message",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "test commit message")
	assert.Contains(t, result, `"action": "commit"`)
}

func TestCommitRejectsEmptyMessage(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action":  "commit",
		"message": "  ",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestCommitRejectsNoStagedChanges(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action":  "commit",
		"message": "should fail",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nothing staged")
}

func TestInvalidAction(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "rebase",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_ACTION")
}

func TestMissingAction(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING_REQUIRED_PARAM")
}

// --- Directory parameter tests ---

func initSecondTestRepo(t *testing.T) string {
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
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}

	require.NoError(t, os.WriteFile(filepath.Join(dir, "SECOND.md"), []byte("# Second Repo"), 0644))
	for _, args := range [][]string{
		{"git", "add", "SECOND.md"},
		{"git", "commit", "-m", "second repo initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}

	// Create a distinct branch name
	cmd := exec.Command("git", "checkout", "-b", "second-branch")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	return dir
}

func TestStatusWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)
	tool := newTool(t, defaultDir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "status",
		"directory": secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "second-branch")
}

func TestLogWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)
	tool := newTool(t, defaultDir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "log",
		"directory": secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "second repo initial")
	assert.NotContains(t, result, "initial commit")
}

func TestDiffWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)
	tool := newTool(t, defaultDir)

	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "SECOND.md"), []byte("# Modified"), 0644))

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "diff",
		"scope":     "all",
		"directory": secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Modified")
}

func TestStageWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)
	tool := newTool(t, defaultDir)

	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "new-file.txt"), []byte("data"), 0644))

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "stage",
		"files":     "new-file.txt",
		"directory": secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "new-file.txt")
}

func TestCommitWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)
	tool := newTool(t, defaultDir)

	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "commit-file.txt"), []byte("data"), 0644))
	cmd := exec.Command("git", "add", "commit-file.txt")
	cmd.Dir = secondDir
	require.NoError(t, cmd.Run())

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "commit",
		"message":   "commit in second repo",
		"directory": secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "commit in second repo")
}

func TestPushWithDirectory(t *testing.T) {
	defaultDir := initTestRepo(t)
	secondDir := initSecondTestRepo(t)

	// Create bare remote for the second repo
	bareDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = bareDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "remote", "add", "origin", bareDir)
	cmd.Dir = secondDir
	require.NoError(t, cmd.Run())

	tool := newTool(t, defaultDir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":       "push",
		"set_upstream": true,
		"directory":    secondDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"success": true`)
	assert.Contains(t, result, "second-branch")
}

func TestDirectoryOmitted(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "log",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "initial commit")
}

func TestDirectoryEmpty(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "log",
		"directory": "",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "initial commit")
}

func TestDirectoryNotFound(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action":    "status",
		"directory": "/nonexistent/path/that/does/not/exist",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_DIRECTORY")
}

func TestDirectoryNotGitRepo(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	notGitDir := t.TempDir()

	_, err := tool.Execute(context.Background(), map[string]any{
		"action":    "status",
		"directory": notGitDir,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_GIT_REPOSITORY")
}

func TestDirectoryIsFile(t *testing.T) {
	dir := initTestRepo(t)
	tool := newTool(t, dir)

	tmpFile := filepath.Join(t.TempDir(), "somefile.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("not a dir"), 0644))

	_, err := tool.Execute(context.Background(), map[string]any{
		"action":    "status",
		"directory": tmpFile,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_DIRECTORY")
}

func TestDirectoryRelative(t *testing.T) {
	defaultDir := initTestRepo(t)

	// Create a subdirectory that is also a git repo
	subDir := filepath.Join(defaultDir, "subrepo")
	require.NoError(t, os.Mkdir(subDir, 0755))
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = subDir
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("sub"), 0644))
	for _, args := range [][]string{
		{"git", "add", "sub.txt"},
		{"git", "commit", "-m", "sub commit"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = subDir
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}

	tool := newTool(t, defaultDir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":    "log",
		"directory": "subrepo",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "sub commit")
}

func TestPushWithLocalBareRemote(t *testing.T) {
	// Create a bare repo as remote
	bareDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = bareDir
	require.NoError(t, cmd.Run())

	// Create working repo
	dir := initTestRepo(t)

	// Switch to feature branch
	cmd = exec.Command("git", "checkout", "-b", "feat/test")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", bareDir)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	tool := newTool(t, dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"action":       "push",
		"set_upstream": true,
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"success": true`)
	assert.Contains(t, result, "feat/test")
}
