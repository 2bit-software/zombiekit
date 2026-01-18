package initiative

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, svc)
		assert.Equal(t, tmpDir, svc.workDir)
	})

	t.Run("uses current directory if empty", func(t *testing.T) {
		svc, err := NewService("")
		require.NoError(t, err)
		require.NotNil(t, svc)
		cwd, _ := os.Getwd()
		assert.Equal(t, cwd, svc.workDir)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("creates initiative folder with correct structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		init, err := svc.Create(TypeFeature, "user-auth")
		require.NoError(t, err)

		// Check initiative properties
		assert.Equal(t, TypeFeature, init.Type)
		assert.Equal(t, "user-auth", init.Name)
		assert.Equal(t, StatusActive, init.Status)
		assert.NotEmpty(t, init.ID)
		assert.Contains(t, init.ID, "feature")
		assert.Contains(t, init.ID, "user-auth")

		// Check folder was created
		_, err = os.Stat(init.Path)
		assert.NoError(t, err)

		// Check INITIATIVE.md was created
		initMDPath := filepath.Join(init.Path, "INITIATIVE.md")
		_, err = os.Stat(initMDPath)
		assert.NoError(t, err)

		// Check content of INITIATIVE.md
		content, err := os.ReadFile(initMDPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "user-auth")
		assert.Contains(t, string(content), "feature")
	})

	t.Run("sets initiative as active", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		init, err := svc.Create(TypeBug, "login-crash")
		require.NoError(t, err)

		// Check state was updated (pointer only - no type/name/status stored in state)
		state, err := svc.stateManager.Load()
		require.NoError(t, err)
		assert.False(t, state.IsEmpty())
		assert.Contains(t, state.Initiative, init.ID)
	})

	t.Run("creates history folder if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		_, err = svc.Create(TypeRefactor, "extract-service")
		require.NoError(t, err)

		// Check history folder was created
		historyDir := filepath.Join(tmpDir, "history")
		info, err := os.Stat(historyDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("validates initiative type", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		_, err = svc.Create(InitiativeType("invalid"), "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_TYPE")
	})

	t.Run("validates initiative name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Empty name
		_, err = svc.Create(TypeFeature, "")
		assert.Error(t, err)

		// Name that normalizes to empty (only special chars)
		_, err = svc.Create(TypeFeature, "!!!")
		assert.Error(t, err)
	})
}

func TestService_List(t *testing.T) {
	t.Run("returns all initiatives from history folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		// Create some initiative folders manually
		historyDir := filepath.Join(tmpDir, "history")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		init1Dir := filepath.Join(historyDir, "abc12345-feature-auth")
		init2Dir := filepath.Join(historyDir, "def67890-bug-crash")
		require.NoError(t, os.MkdirAll(init1Dir, 0755))
		require.NoError(t, os.MkdirAll(init2Dir, 0755))

		// Create INITIATIVE.md files
		require.NoError(t, os.WriteFile(filepath.Join(init1Dir, "INITIATIVE.md"), []byte("# Auth"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(init2Dir, "INITIATIVE.md"), []byte("# Crash"), 0644))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		initiatives, err := svc.List()
		require.NoError(t, err)
		assert.Len(t, initiatives, 2)
	})

	t.Run("returns empty list if no history folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		initiatives, err := svc.List()
		require.NoError(t, err)
		assert.Empty(t, initiatives)
	})
}

func TestService_GetActive(t *testing.T) {
	t.Run("returns active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		created, err := svc.Create(TypeFeature, "test-feature")
		require.NoError(t, err)

		active, err := svc.GetActive()
		require.NoError(t, err)
		assert.Equal(t, created.ID, active.ID)
		assert.Equal(t, created.Path, active.Path)
	})

	t.Run("returns nil if no active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		active, err := svc.GetActive()
		require.NoError(t, err)
		assert.Nil(t, active)
	})
}

func TestService_SetActive(t *testing.T) {
	t.Run("sets specified initiative as active", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		// Create two initiatives
		historyDir := filepath.Join(tmpDir, "history")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		init1Dir := filepath.Join(historyDir, "abc12345-feature-first")
		init2Dir := filepath.Join(historyDir, "def67890-feature-second")
		require.NoError(t, os.MkdirAll(init1Dir, 0755))
		require.NoError(t, os.MkdirAll(init2Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(init1Dir, "INITIATIVE.md"), []byte("# First"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(init2Dir, "INITIATIVE.md"), []byte("# Second"), 0644))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		err = svc.SetActive("def67890-feature-second")
		require.NoError(t, err)

		state, err := svc.stateManager.Load()
		require.NoError(t, err)
		assert.Contains(t, state.Initiative, "def67890-feature-second")
	})

	t.Run("returns error for non-existent initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		err = svc.SetActive("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "INITIATIVE_NOT_FOUND")
	})
}

func TestService_Complete(t *testing.T) {
	t.Run("marks active initiative as completed", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create an initiative
		_, err = svc.Create(TypeFeature, "test-feature")
		require.NoError(t, err)

		// Complete it
		err = svc.Complete()
		require.NoError(t, err)

		// State should be empty (no active initiative)
		state, err := svc.stateManager.Load()
		require.NoError(t, err)
		assert.True(t, state.IsEmpty())
	})

	t.Run("returns error when no active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		err = svc.Complete()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NO_ACTIVE_INITIATIVE")
	})
}

func TestService_GenerateID(t *testing.T) {
	t.Run("generates IDs with correct format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		id := svc.generateID(TypeFeature, "test")

		// Check format: {hex-timestamp}-{type}-{name}
		assert.Contains(t, id, "feature")
		assert.Contains(t, id, "test")
		assert.Regexp(t, `^[0-9a-f]{8}-feature-test$`, id)
	})

	t.Run("generates unique IDs for different names", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		id1 := svc.generateID(TypeFeature, "first")
		id2 := svc.generateID(TypeFeature, "second")

		assert.NotEqual(t, id1, id2)
	})
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user-auth", "user-auth"},
		{"User Auth", "user-auth"},
		{"MY_FEATURE", "my-feature"},
		{"feature 123", "feature-123"},
		{"feature--double", "feature-double"},
		{" trimmed ", "trimmed"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"valid-name", true},
		{"also-valid-123", true},
		{"", false},           // empty
		{" ", false},          // whitespace
		{"has space", false},  // contains space
		{"has_under", false},  // contains underscore (after normalize it's fine, but raw fails)
		{"Valid-Name", false}, // uppercase (after normalize it's fine, but raw fails)
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			err := validateName(tc.input)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestService_FindActiveByNameAndType(t *testing.T) {
	t.Run("returns nil when no active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		found, err := svc.FindActiveByNameAndType("test", TypeFeature)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns nil when active initiative has different name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create an initiative with name "foo"
		_, err = svc.Create(TypeFeature, "foo")
		require.NoError(t, err)

		// Look for "bar" - should not find it
		found, err := svc.FindActiveByNameAndType("bar", TypeFeature)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns nil when active initiative has different type", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create a feature initiative
		_, err = svc.Create(TypeFeature, "test")
		require.NoError(t, err)

		// Look for bug type - should not find it
		found, err := svc.FindActiveByNameAndType("test", TypeBug)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("returns initiative when name and type match", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create an initiative
		created, err := svc.Create(TypeFeature, "my-feature")
		require.NoError(t, err)

		// Look for same name+type - should find it
		found, err := svc.FindActiveByNameAndType("my-feature", TypeFeature)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Name, found.Name)
		assert.Equal(t, created.Type, found.Type)
	})

	t.Run("normalizes name before comparison", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		// Create an initiative with normalized name "user-auth"
		created, err := svc.Create(TypeFeature, "user-auth")
		require.NoError(t, err)

		// Look with non-normalized name - should still find it
		found, err := svc.FindActiveByNameAndType("User Auth", TypeFeature)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)

		// Also try with underscores
		found2, err := svc.FindActiveByNameAndType("user_auth", TypeFeature)
		require.NoError(t, err)
		require.NotNil(t, found2)
		assert.Equal(t, created.ID, found2.ID)
	})
}
