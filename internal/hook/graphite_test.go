package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGraphiteAvailable(t *testing.T) {
	// This test reflects the actual environment.
	// If gt is installed, it returns true; otherwise false.
	if _, err := exec.LookPath("gt"); err == nil {
		assert.True(t, isGraphiteAvailable())
	} else {
		assert.False(t, isGraphiteAvailable())
	}
}

func TestIsGraphiteInitialized(t *testing.T) {
	t.Run("returns false when no .graphite directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.False(t, isGraphiteInitialized(tmpDir))
	})

	t.Run("returns true when .graphite directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".graphite"), 0o755))
		assert.True(t, isGraphiteInitialized(tmpDir))
	})

	t.Run("returns false when .graphite is a file not directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".graphite"), []byte("not a dir"), 0o644))
		assert.False(t, isGraphiteInitialized(tmpDir))
	})
}

func TestIsGraphiteTracked(t *testing.T) {
	t.Run("returns false in non-graphite directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.False(t, isGraphiteTracked(tmpDir))
	})
}

func TestDetectGraphiteStatus(t *testing.T) {
	t.Run("not initialized returns correct status", func(t *testing.T) {
		if _, err := exec.LookPath("gt"); err != nil {
			t.Skip("gt not available")
		}

		tmpDir := t.TempDir()
		status := DetectGraphiteStatus(tmpDir)
		assert.Equal(t, "graphite: available, not initialized", status)
	})

	t.Run("initialized but not tracked returns correct status", func(t *testing.T) {
		if _, err := exec.LookPath("gt"); err != nil {
			t.Skip("gt not available")
		}

		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".graphite"), 0o755))

		status := DetectGraphiteStatus(tmpDir)
		assert.Equal(t, "graphite: available, initialized", status)
	})
}
