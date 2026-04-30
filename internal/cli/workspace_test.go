package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2bit-software/zombiekit/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePrep_NoSandboxNoSpawn_TextFormat(t *testing.T) {
	repo := initTestRepo(t)

	out := captureStdout(t, func() {
		err := runBrains(t, repo,
			"workspace", "prep",
			"--title", "demo prep",
			"--description", "the description",
			"--no-sandbox",
			"DEV-101",
		)
		require.NoError(t, err)
	})

	wtPath := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-101")
	assert.DirExists(t, wtPath)

	ticketBytes, err := os.ReadFile(filepath.Join(wtPath, ".ai", "ticket.md"))
	require.NoError(t, err)
	assert.Equal(t, "the description", string(ticketBytes))

	marker, err := workspace.ReadMarker(wtPath)
	require.NoError(t, err)
	assert.Equal(t, "DEV-101", marker.TicketID)

	assert.Contains(t, out, "worktree:")
	assert.Contains(t, out, "branch:")
}

func TestWorkspacePrep_JSONFormat(t *testing.T) {
	repo := initTestRepo(t)

	out := captureStdout(t, func() {
		err := runBrains(t, repo,
			"workspace", "prep",
			"--title", "json test",
			"--no-sandbox",
			"--format", "json",
			"DEV-102",
		)
		require.NoError(t, err)
	})

	out = strings.TrimSpace(out)
	var got workspace.PrepResult
	require.NoError(t, json.Unmarshal([]byte(out), &got), "output: %s", out)
	assert.Equal(t, "DEV-102/json-test", got.Branch)
	assert.NotEmpty(t, got.WorktreePath)
}

func TestWorkspacePrep_DescriptionFile(t *testing.T) {
	repo := initTestRepo(t)
	descFile := filepath.Join(t.TempDir(), "desc.md")
	require.NoError(t, os.WriteFile(descFile, []byte("from a file"), 0o644))

	err := runBrains(t, repo,
		"workspace", "prep",
		"--title", "file desc",
		"--description-file", descFile,
		"--no-sandbox",
		"DEV-103",
	)
	require.NoError(t, err)

	wtPath := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-103")
	ticketBytes, err := os.ReadFile(filepath.Join(wtPath, ".ai", "ticket.md"))
	require.NoError(t, err)
	assert.Equal(t, "from a file", string(ticketBytes))
}

func TestWorkspacePrep_MissingTitle_Errors(t *testing.T) {
	repo := initTestRepo(t)
	err := runBrains(t, repo, "workspace", "prep", "DEV-104", "--no-sandbox")
	require.Error(t, err)
}

func TestWorkspaceTeardown_RemovesWorktree(t *testing.T) {
	repo := initTestRepo(t)

	require.NoError(t, runBrains(t, repo,
		"workspace", "prep",
		"--title", "cleanup me", "--no-sandbox",
		"DEV-105",
	))

	wtPath := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-105")
	require.DirExists(t, wtPath)

	require.NoError(t, runBrains(t, repo, "workspace", "teardown", "DEV-105"))
	assert.NoDirExists(t, wtPath)
}

func TestWorkspaceTeardown_NoWorktreeWithoutForce_Errors(t *testing.T) {
	repo := initTestRepo(t)
	err := runBrains(t, repo, "workspace", "teardown", "DEV-DOES-NOT-EXIST")
	require.Error(t, err)
}

func TestWorkspaceGC_DryRunReportsOrphans(t *testing.T) {
	repo := initTestRepo(t)

	// Make a fake orphan (worktree dir with no .ai/workspace.json marker).
	orphan := filepath.Join(filepath.Dir(repo), "worktrees", "DEV-ORPHAN")
	require.NoError(t, os.MkdirAll(orphan, 0o755))

	out := captureStdout(t, func() {
		err := runBrains(t, repo, "workspace", "gc")
		require.NoError(t, err)
	})

	assert.Contains(t, out, "DEV-ORPHAN")
	assert.Contains(t, out, "(dry-run")
	assert.DirExists(t, orphan, "dry-run should not delete")
}

func TestFindOrphanWorktrees_SkipsMarkedDirs(t *testing.T) {
	root := t.TempDir()

	marked := filepath.Join(root, "DEV-OK")
	require.NoError(t, os.MkdirAll(filepath.Join(marked, ".ai"), 0o755))
	require.NoError(t, os.WriteFile(workspace.MarkerPath(marked), []byte(`{"ticket_id":"DEV-OK"}`), 0o644))

	orphan := filepath.Join(root, "DEV-ORPHAN")
	require.NoError(t, os.MkdirAll(orphan, 0o755))

	got, err := findOrphanWorktrees(root)
	require.NoError(t, err)
	assert.Equal(t, []string{orphan}, got)
}

func TestFindOrphanWorktrees_MissingRoot_NoError(t *testing.T) {
	got, err := findOrphanWorktrees(filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	assert.Empty(t, got)
}
