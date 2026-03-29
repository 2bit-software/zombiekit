package cmux

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireCmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cmux"); err != nil {
		t.Skip("cmux not available")
	}
	cmd := exec.Command("cmux", "ping")
	if err := cmd.Run(); err != nil {
		t.Skip("cmux not running")
	}
}

// cleanupWorkspace closes a workspace by ref during test cleanup.
func cleanupWorkspace(t *testing.T, ref string) {
	t.Helper()
	t.Cleanup(func() {
		cmd := exec.Command("cmux", "close-workspace", "--workspace", ref)
		_ = cmd.Run()
	})
}

func TestNew_CmuxAvailable(t *testing.T) {
	requireCmux(t)

	mgr, err := New()
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, "claude", mgr.command)
}

func TestNew_WithCommand(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo hello"))
	require.NoError(t, err)
	assert.Equal(t, "echo hello", mgr.command)
}

func TestSpawnSession_Success(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo test-session"))
	require.NoError(t, err)

	ref, err := mgr.SpawnSession(context.Background(), "TEST-SPAWN-001", "spawn test", "/tmp", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ref, "workspace:"))
	cleanupWorkspace(t, ref)

	exists, err := mgr.SessionExists(context.Background(), "TEST-SPAWN-001")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSpawnSession_Duplicate(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo test-dup"))
	require.NoError(t, err)

	ref, err := mgr.SpawnSession(context.Background(), "TEST-DUP-001", "dup test", "/tmp", nil)
	require.NoError(t, err)
	cleanupWorkspace(t, ref)

	_, err = mgr.SpawnSession(context.Background(), "TEST-DUP-001", "dup test again", "/tmp", nil)
	assert.Error(t, err)
	assert.True(t, IsSessionExists(err))
}

func TestSpawnSession_WithEnv(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo env-test"))
	require.NoError(t, err)

	env := map[string]string{
		"WORK_CALLBACK_URL": "http://localhost:8666/TEST-ENV-001",
	}
	ref, err := mgr.SpawnSession(context.Background(), "TEST-ENV-001", "env test", "/tmp", env)
	require.NoError(t, err)
	cleanupWorkspace(t, ref)
}

func TestKillSession_Success(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo test-kill"))
	require.NoError(t, err)

	ref, err := mgr.SpawnSession(context.Background(), "TEST-KILL-001", "kill test", "/tmp", nil)
	require.NoError(t, err)
	// Don't use cleanupWorkspace here -- we're testing KillSession itself
	_ = ref

	err = mgr.KillSession(context.Background(), "TEST-KILL-001")
	require.NoError(t, err)

	exists, err := mgr.SessionExists(context.Background(), "TEST-KILL-001")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestKillSession_NotFound(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo noop"))
	require.NoError(t, err)

	err = mgr.KillSession(context.Background(), "TEST-NONEXISTENT-001")
	assert.Error(t, err)
	assert.True(t, IsSessionNotFound(err))
}

func TestSessionExists_Running(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo test-exists"))
	require.NoError(t, err)

	ref, err := mgr.SpawnSession(context.Background(), "TEST-EXISTS-001", "exists test", "/tmp", nil)
	require.NoError(t, err)
	cleanupWorkspace(t, ref)

	exists, err := mgr.SessionExists(context.Background(), "TEST-EXISTS-001")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSessionExists_NotRunning(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo noop"))
	require.NoError(t, err)

	exists, err := mgr.SessionExists(context.Background(), "TEST-NOEXIST-001")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSessionExists_StaleTracking(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo test-stale"))
	require.NoError(t, err)

	ref, err := mgr.SpawnSession(context.Background(), "TEST-STALE-001", "stale test", "/tmp", nil)
	require.NoError(t, err)

	// Close workspace directly via cmux (simulating manual close)
	closeCmd := exec.Command("cmux", "close-workspace", "--workspace", ref)
	require.NoError(t, closeCmd.Run())

	// Manager still has internal tracking, but cmux says it's gone
	exists, err := mgr.SessionExists(context.Background(), "TEST-STALE-001")
	require.NoError(t, err)
	assert.False(t, exists)

	// Internal tracking should be cleaned up -- SpawnSession should work now
	ref2, err := mgr.SpawnSession(context.Background(), "TEST-STALE-001", "stale respawn", "/tmp", nil)
	require.NoError(t, err)
	cleanupWorkspace(t, ref2)
}

func TestConcurrent_DifferentTickets(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo concurrent"))
	require.NoError(t, err)

	tickets := []string{"TEST-CONC-001", "TEST-CONC-002", "TEST-CONC-003"}
	refs := make([]string, len(tickets))
	errs := make([]error, len(tickets))

	var wg sync.WaitGroup
	for i, ticket := range tickets {
		wg.Add(1)
		go func(idx int, tid string) {
			defer wg.Done()
			refs[idx], errs[idx] = mgr.SpawnSession(context.Background(), tid, "concurrent "+tid, "/tmp", nil)
		}(i, ticket)
	}
	wg.Wait()

	for i, ticket := range tickets {
		require.NoError(t, errs[i], "spawn failed for %s", ticket)
		cleanupWorkspace(t, refs[i])
	}
}

func TestConcurrent_SameTicket(t *testing.T) {
	requireCmux(t)

	mgr, err := New(WithCommand("echo race"))
	require.NoError(t, err)

	const ticketID = "TEST-RACE-001"
	const goroutines = 5

	refs := make([]string, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			refs[idx], errs[idx] = mgr.SpawnSession(context.Background(), ticketID, "race test", "/tmp", nil)
		}(i)
	}
	wg.Wait()

	// Exactly one should succeed
	successCount := 0
	var successRef string
	for i := range goroutines {
		if errs[i] == nil {
			successCount++
			successRef = refs[i]
		} else {
			assert.True(t, IsSessionExists(errs[i]), "non-success error should be ErrSessionExists, got: %v", errs[i])
		}
	}
	assert.Equal(t, 1, successCount, "exactly one goroutine should succeed")
	if successRef != "" {
		cleanupWorkspace(t, successRef)
	}
}
