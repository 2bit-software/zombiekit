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

func TestPushRejectsMainBranch(t *testing.T) {
	dir := initTestRepo(t)

	// Ensure we're on main
	cmd := exec.Command("git", "branch", "-m", "main")
	cmd.Dir = dir
	_ = cmd.Run() // May already be on main

	tool := newTool(t, dir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "push",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BRANCH_PROTECTED")
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
