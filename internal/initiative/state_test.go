package initiative

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateManager(t *testing.T) {
	t.Run("creates manager with explicit path", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		expected := filepath.Join(tmpDir, BrainsDir, StateFileName)
		assert.Equal(t, expected, mgr.Path())
	})

	t.Run("uses current directory if empty", func(t *testing.T) {
		// Save current dir and restore after test
		cwd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(cwd)

		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))

		mgr, err := NewStateManager("")
		require.NoError(t, err)

		// The path should contain the expected components, accounting for symlinks
		assert.Contains(t, mgr.Path(), BrainsDir)
		assert.Contains(t, mgr.Path(), StateFileName)
		assert.True(t, filepath.IsAbs(mgr.Path()), "path should be absolute")
	})
}

func TestFileStateManager_Load(t *testing.T) {
	t.Run("returns empty state when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		state, err := mgr.Load()
		require.NoError(t, err)
		assert.True(t, state.IsEmpty())
	})

	t.Run("returns empty state for empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		brainsDir := filepath.Join(tmpDir, BrainsDir)
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, StateFileName), []byte{}, 0644))

		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		state, err := mgr.Load()
		require.NoError(t, err)
		assert.True(t, state.IsEmpty())
	})

	t.Run("loads valid state", func(t *testing.T) {
		tmpDir := t.TempDir()
		brainsDir := filepath.Join(tmpDir, BrainsDir)
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		stateJSON := `{
			"initiative": "history/675d8a3f-feature-user-auth",
			"cycle": "history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth",
			"current_step": "specify"
		}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, StateFileName), []byte(stateJSON), 0644))

		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		state, err := mgr.Load()
		require.NoError(t, err)
		assert.False(t, state.IsEmpty())
		assert.Equal(t, "history/675d8a3f-feature-user-auth", state.Initiative)
		assert.Equal(t, "history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth", state.Cycle)
		assert.Equal(t, "specify", state.CurrentStep)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		brainsDir := filepath.Join(tmpDir, BrainsDir)
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, StateFileName), []byte("invalid json"), 0644))

		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		_, err = mgr.Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing state file")
	})
}

func TestFileStateManager_Save(t *testing.T) {
	t.Run("creates directory and file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		state := &InitiativeState{
			Initiative:  "history/675d8a3f-feature-user-auth",
			Cycle:       "history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth",
			Started:     time.Now(),
			CurrentStep: "specify",
		}

		err = mgr.Save(state)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(mgr.Path())
		assert.NoError(t, err)

		// Verify content
		loaded, err := mgr.Load()
		require.NoError(t, err)
		assert.Equal(t, state.Initiative, loaded.Initiative)
		assert.Equal(t, state.Cycle, loaded.Cycle)
		assert.Equal(t, state.CurrentStep, loaded.CurrentStep)
	})

	t.Run("updates last_activity on save", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		oldTime := time.Now().Add(-1 * time.Hour)
		state := &InitiativeState{
			Initiative:   "history/test",
			LastActivity: oldTime,
		}

		err = mgr.Save(state)
		require.NoError(t, err)

		loaded, err := mgr.Load()
		require.NoError(t, err)
		assert.True(t, loaded.LastActivity.After(oldTime))
	})

	t.Run("overwrites existing state", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		// Save first state
		state1 := &InitiativeState{
			Initiative: "history/first",
		}
		require.NoError(t, mgr.Save(state1))

		// Save second state
		state2 := &InitiativeState{
			Initiative: "history/second",
		}
		require.NoError(t, mgr.Save(state2))

		// Verify second state is loaded
		loaded, err := mgr.Load()
		require.NoError(t, err)
		assert.Equal(t, "history/second", loaded.Initiative)
	})
}

func TestFileStateManager_Lock(t *testing.T) {
	t.Run("acquires and releases lock", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		unlock, err := mgr.Lock()
		require.NoError(t, err)
		require.NotNil(t, unlock)

		// Lock file should exist
		lockPath := mgr.Path() + ".lock"
		_, err = os.Stat(lockPath)
		assert.NoError(t, err)

		// Release lock
		unlock()
	})

	t.Run("creates brains directory if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		// Directory shouldn't exist yet
		brainsDir := filepath.Join(tmpDir, BrainsDir)
		_, err = os.Stat(brainsDir)
		assert.True(t, os.IsNotExist(err))

		// Lock should create it
		unlock, err := mgr.Lock()
		require.NoError(t, err)
		defer unlock()

		_, err = os.Stat(brainsDir)
		assert.NoError(t, err)
	})
}

func TestFileStateManager_Clear(t *testing.T) {
	t.Run("removes state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		// Save state
		state := &InitiativeState{Initiative: "history/test"}
		require.NoError(t, mgr.Save(state))

		// Clear
		err = mgr.Clear()
		require.NoError(t, err)

		// File should be gone
		_, err = os.Stat(mgr.Path())
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("no error if file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr, err := NewStateManager(tmpDir)
		require.NoError(t, err)

		err = mgr.Clear()
		assert.NoError(t, err)
	})
}

func TestInitiativeState_IsEmpty(t *testing.T) {
	t.Run("returns true for empty state", func(t *testing.T) {
		state := &InitiativeState{}
		assert.True(t, state.IsEmpty())
	})

	t.Run("returns false when initiative is set", func(t *testing.T) {
		state := &InitiativeState{Initiative: "history/test"}
		assert.False(t, state.IsEmpty())
	})
}
