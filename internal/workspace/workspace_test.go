package workspace

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/worktree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSandbox records every call so tests can assert on the prep/teardown
// sequence without invoking sbx.
type fakeSandbox struct {
	available    bool
	createErr    error
	createCalls  int32
	cleanupCalls int32
	lastName     string
}

func (f *fakeSandbox) Available() bool       { return f.available }
func (f *fakeSandbox) Name(id string) string { return "zk-" + id }
func (f *fakeSandbox) Create(_ context.Context, name, _ string, _ sandbox.Config) error {
	atomic.AddInt32(&f.createCalls, 1)
	f.lastName = name
	return f.createErr
}
func (f *fakeSandbox) Cleanup(_ context.Context, _ string) {
	atomic.AddInt32(&f.cleanupCalls, 1)
}

// fakeSpawner records SpawnSession/KillSession invocations.
type fakeSpawner struct {
	spawnErr   error
	killErr    error
	spawnCalls int32
	killCalls  int32
	lastTicket string
	lastEnv    map[string]string
}

func (f *fakeSpawner) SpawnSession(_ context.Context, ticketID, _, _ string, env map[string]string, _ string) (string, error) {
	atomic.AddInt32(&f.spawnCalls, 1)
	f.lastTicket = ticketID
	f.lastEnv = env
	if f.spawnErr != nil {
		return "", f.spawnErr
	}
	return "session-ref-" + ticketID, nil
}
func (f *fakeSpawner) KillSession(_ context.Context, _ string) error {
	atomic.AddInt32(&f.killCalls, 1)
	return f.killErr
}

// initRepo creates a fresh git repo with an initial commit and returns the
// repo dir, worktrees root, and a configured worktree.GitManager.
func initRepo(t *testing.T) (repoDir, worktreesRoot string, mgr *worktree.GitManager) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir = t.TempDir()
	mustGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	mustGit("init")
	mustGit("config", "user.name", "Test")
	mustGit("config", "user.email", "test@test.com")
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, ".gitkeep"), nil, 0o644))
	mustGit("add", ".gitkeep")
	mustGit("commit", "-m", "initial")

	worktreesRoot = filepath.Join(t.TempDir(), "worktrees")
	mgr, err := worktree.New(repoDir, worktree.WithWorktreesRoot(worktreesRoot))
	require.NoError(t, err)
	return repoDir, worktreesRoot, mgr
}

func TestPrep_NoSandbox_NoSpawn_WritesTicketAndMarker(t *testing.T) {
	_, _, wt := initRepo(t)
	sbx := &fakeSandbox{available: false}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx))
	result, err := m.Prep(context.Background(), PrepInput{
		TicketID:    "DEV-1",
		Title:       "first feature",
		Description: "describe the feature",
	})
	require.NoError(t, err)
	assert.Empty(t, result.SandboxName)
	assert.Empty(t, result.SessionRef)

	ticketBytes, err := os.ReadFile(filepath.Join(result.WorktreePath, ".ai", "ticket.md"))
	require.NoError(t, err)
	assert.Equal(t, "describe the feature", string(ticketBytes))

	marker, err := ReadMarker(result.WorktreePath)
	require.NoError(t, err)
	assert.Equal(t, "DEV-1", marker.TicketID)
	assert.Equal(t, "DEV-1/first-feature", marker.Branch)
	assert.False(t, marker.Spawned)
}

func TestPrep_WithSandbox_CreatesSandbox(t *testing.T) {
	_, _, wt := initRepo(t)
	sbx := &fakeSandbox{available: true}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx))
	result, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-2",
		Title:    "with sandbox",
		Sandbox:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "zk-DEV-2", result.SandboxName)
	assert.Equal(t, int32(1), atomic.LoadInt32(&sbx.createCalls))
}

func TestPrep_SandboxRequestedButUnavailable_SkipsSandbox(t *testing.T) {
	_, _, wt := initRepo(t)
	sbx := &fakeSandbox{available: false}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx))
	result, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-3",
		Title:    "no sbx",
		Sandbox:  true,
	})
	require.NoError(t, err)
	assert.Empty(t, result.SandboxName)
	assert.Equal(t, int32(0), atomic.LoadInt32(&sbx.createCalls))
}

func TestPrep_SandboxFails_RollsBackWorktree(t *testing.T) {
	_, root, wt := initRepo(t)
	sbx := &fakeSandbox{available: true, createErr: errors.New("docker oom")}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx))
	_, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-4",
		Title:    "doomed",
		Sandbox:  true,
	})
	require.Error(t, err)

	// Worktree should be gone after rollback.
	assert.NoDirExists(t, filepath.Join(root, "DEV-4"))
}

func TestPrep_SpawnFails_RollsBackSandboxAndWorktree(t *testing.T) {
	_, _, wt := initRepo(t)
	sbx := &fakeSandbox{available: true}
	sp := &fakeSpawner{spawnErr: errors.New("cmux down")}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx), WithSpawner(sp))
	_, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-5",
		Title:    "spawn fail",
		Sandbox:  true,
		Spawn:    &SpawnInput{Prompt: "go", Env: map[string]string{"K": "V"}},
	})
	require.Error(t, err)

	// Sandbox cleanup must have been called as part of rollback.
	assert.Equal(t, int32(1), atomic.LoadInt32(&sbx.cleanupCalls))
}

func TestPrep_WithSpawn_PassesEnvAndPrompt(t *testing.T) {
	_, _, wt := initRepo(t)
	sbx := &fakeSandbox{available: false}
	sp := &fakeSpawner{}

	m := NewManager(wt, sandbox.DefaultConfig(), WithSandbox(sbx), WithSpawner(sp))
	result, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-6",
		Title:    "spawn ok",
		Spawn:    &SpawnInput{Prompt: "begin", Env: map[string]string{"FOO": "BAR"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "session-ref-DEV-6", result.SessionRef)
	assert.Equal(t, "BAR", sp.lastEnv["FOO"])

	marker, err := ReadMarker(result.WorktreePath)
	require.NoError(t, err)
	assert.True(t, marker.Spawned)
	assert.Equal(t, "begin", marker.Prompt)
}

func TestTeardown_ReadsMarker_AndCleansAll(t *testing.T) {
	_, root, wt := initRepo(t)
	sbx := &fakeSandbox{available: true}
	sp := &fakeSpawner{}

	m := NewManager(wt, sandbox.DefaultConfig(),
		WithSandbox(sbx), WithSpawner(sp), WithWorktreesRoot(root))
	prep, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-7",
		Title:    "teardown me",
		Sandbox:  true,
		Spawn:    &SpawnInput{Prompt: "go"},
	})
	require.NoError(t, err)

	require.NoError(t, m.Teardown(context.Background(), "DEV-7", prep.WorktreePath))

	assert.Equal(t, int32(1), atomic.LoadInt32(&sp.killCalls))
	assert.GreaterOrEqual(t, atomic.LoadInt32(&sbx.cleanupCalls), int32(1))
	assert.NoDirExists(t, prep.WorktreePath)
}

func TestTeardown_NoMarker_FallsBackToConvention(t *testing.T) {
	_, root, wt := initRepo(t)
	sbx := &fakeSandbox{available: true}
	m := NewManager(wt, sandbox.DefaultConfig(),
		WithSandbox(sbx), WithWorktreesRoot(root))

	// Prepare a worktree manually (no marker).
	prep, err := m.Prep(context.Background(), PrepInput{
		TicketID: "DEV-8",
		Title:    "marker missing",
	})
	require.NoError(t, err)
	require.NoError(t, os.Remove(MarkerPath(prep.WorktreePath)))

	// Teardown by ticket only — should use rootGuess to find the worktree.
	require.NoError(t, m.Teardown(context.Background(), "DEV-8", ""))
	assert.NoDirExists(t, prep.WorktreePath)
}

func TestShortTitle_OrchestratorParity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Watcher 1 — ready ticket pickup", "watcher-1-ready-ticket-pickup"},
		{"Hello!! World", "hello-world"},
		{"   spaced  ", "spaced"},
		{"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzABCDEF", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, ShortTitle(tc.in))
		})
	}
}

func TestMarker_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ai"), 0o755))

	in := Marker{
		TicketID:     "DEV-9",
		Title:        "round trip",
		Branch:       "DEV-9/round-trip",
		WorktreePath: dir,
		SandboxName:  "zk-dev-9",
		Spawned:      true,
		Prompt:       "go",
	}
	require.NoError(t, writeMarker(dir, in))

	out, err := ReadMarker(dir)
	require.NoError(t, err)
	assert.Equal(t, in.TicketID, out.TicketID)
	assert.Equal(t, in.SandboxName, out.SandboxName)
	assert.Equal(t, in.Spawned, out.Spawned)
	assert.Equal(t, in.Prompt, out.Prompt)
}

func TestReadMarker_Missing_ReturnsErrNoMarker(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadMarker(dir)
	assert.ErrorIs(t, err, ErrNoMarker)
}
