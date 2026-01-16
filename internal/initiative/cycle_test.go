package initiative

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateCycle(t *testing.T) {
	t.Run("creates cycle folder with correct structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create an initiative first
		init, err := svc.Create(TypeFeature, "test-feature")
		require.NoError(t, err)

		// Create a cycle within the initiative
		cycle, err := svc.CreateCycle(init.Path, CycleFeat, "test-feature")
		require.NoError(t, err)

		// Check cycle properties
		assert.Equal(t, CycleFeat, cycle.Type)
		assert.Equal(t, "test-feature", cycle.Name)
		assert.Equal(t, CycleStatusTemplate, cycle.Status)
		assert.Equal(t, 1, cycle.Number)
		assert.NotEmpty(t, cycle.ID)
		assert.Contains(t, cycle.ID, "feat")
		assert.Contains(t, cycle.ID, "test-feature")

		// Check folder was created
		_, err = os.Stat(cycle.Path)
		assert.NoError(t, err)

		// Check audit folder was created
		auditPath := filepath.Join(cycle.Path, "audit")
		_, err = os.Stat(auditPath)
		assert.NoError(t, err)
	})

	t.Run("validates cycle type", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		init, err := svc.Create(TypeFeature, "test")
		require.NoError(t, err)

		_, err = svc.CreateCycle(init.Path, CycleType("invalid"), "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_CYCLE_TYPE")
	})

	t.Run("validates cycle name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		init, err := svc.Create(TypeFeature, "test")
		require.NoError(t, err)

		_, err = svc.CreateCycle(init.Path, CycleFeat, "")
		assert.Error(t, err)
	})

	t.Run("increments cycle number", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		init, err := svc.Create(TypeFeature, "test")
		require.NoError(t, err)

		cycle1, err := svc.CreateCycle(init.Path, CycleFeat, "first")
		require.NoError(t, err)
		assert.Equal(t, 1, cycle1.Number)

		cycle2, err := svc.CreateCycle(init.Path, CycleRef, "second")
		require.NoError(t, err)
		assert.Equal(t, 2, cycle2.Number)

		cycle3, err := svc.CreateCycle(init.Path, CycleFix, "third")
		require.NoError(t, err)
		assert.Equal(t, 3, cycle3.Number)
	})
}

func TestGenerateCycleID(t *testing.T) {
	t.Run("generates IDs with correct format", func(t *testing.T) {
		id := generateCycleID(CycleFeat, "test")

		// Check format: {hex-timestamp}-{cycle-type}-{name}
		assert.Contains(t, id, "feat")
		assert.Contains(t, id, "test")
		assert.Regexp(t, `^[0-9a-f]{8}-feat-test$`, id)
	})

	t.Run("generates correct prefix for each type", func(t *testing.T) {
		featID := generateCycleID(CycleFeat, "name")
		assert.True(t, strings.Contains(featID, "-feat-"))

		refID := generateCycleID(CycleRef, "name")
		assert.True(t, strings.Contains(refID, "-ref-"))

		fixID := generateCycleID(CycleFix, "name")
		assert.True(t, strings.Contains(fixID, "-fix-"))
	})
}

func TestGetNextCycleNumber(t *testing.T) {
	t.Run("returns 1 for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		initPath := filepath.Join(tmpDir, "history", "test-init")
		require.NoError(t, os.MkdirAll(initPath, 0755))

		num, err := svc.getNextCycleNumber(initPath)
		require.NoError(t, err)
		assert.Equal(t, 1, num)
	})

	t.Run("returns 1 for non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		num, err := svc.getNextCycleNumber(filepath.Join(tmpDir, "nonexistent"))
		require.NoError(t, err)
		assert.Equal(t, 1, num)
	})

	t.Run("counts existing subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		initPath := filepath.Join(tmpDir, "history", "test-init")
		require.NoError(t, os.MkdirAll(filepath.Join(initPath, "cycle1"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(initPath, "cycle2"), 0755))

		num, err := svc.getNextCycleNumber(initPath)
		require.NoError(t, err)
		assert.Equal(t, 3, num)
	})

	t.Run("excludes audit directory from count", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		initPath := filepath.Join(tmpDir, "history", "test-init")
		require.NoError(t, os.MkdirAll(filepath.Join(initPath, "cycle1"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(initPath, "audit"), 0755))

		num, err := svc.getNextCycleNumber(initPath)
		require.NoError(t, err)
		assert.Equal(t, 2, num) // audit directory should not be counted
	})
}
