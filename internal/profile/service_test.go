package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/profile"
)

func TestProfileService_Write(t *testing.T) {
	t.Run("writes profile to local directory", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		content := `---
name: test-profile
description: A test profile
type: domain
---

# Test Profile

This is test content.
`
		path, err := svc.Write("test-profile", content, "local", false)
		require.NoError(t, err)

		expectedPath := filepath.Join(tempDir, ".brains", "profiles", "test-profile.md")
		assert.Equal(t, expectedPath, path)

		// Verify file was written
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("writes profile to global directory", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir := filepath.Join(tempDir, "home")
		require.NoError(t, os.MkdirAll(homeDir, 0o755))

		// Set HOME for this test
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		content := `---
name: global-profile
description: A global profile
---

Global content.
`
		path, err := svc.Write("global-profile", content, "global", false)
		require.NoError(t, err)

		expectedPath := filepath.Join(homeDir, ".brains", "profiles", "global-profile.md")
		assert.Equal(t, expectedPath, path)

		// Verify file was written
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("creates directory if it does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		// Directory should not exist yet
		profileDir := filepath.Join(tempDir, ".brains", "profiles")
		_, err = os.Stat(profileDir)
		assert.True(t, os.IsNotExist(err))

		content := "---\nname: new-profile\n---\nContent"
		_, err = svc.Write("new-profile", content, "local", false)
		require.NoError(t, err)

		// Directory should exist now
		info, err := os.Stat(profileDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("returns error if profile exists without overwrite", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		content := "---\nname: existing\n---\nContent"

		// Write first time
		_, err = svc.Write("existing", content, "local", false)
		require.NoError(t, err)

		// Write second time without overwrite should fail
		_, err = svc.Write("existing", "new content", "local", false)
		require.Error(t, err)

		var existsErr *profile.ProfileExistsError
		assert.ErrorAs(t, err, &existsErr)
		assert.Equal(t, "existing", existsErr.Name)
	})

	t.Run("overwrites existing profile with overwrite flag", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		original := "---\nname: overwrite-test\n---\nOriginal"
		updated := "---\nname: overwrite-test\n---\nUpdated"

		// Write first time
		path, err := svc.Write("overwrite-test", original, "local", false)
		require.NoError(t, err)

		// Overwrite
		_, err = svc.Write("overwrite-test", updated, "local", true)
		require.NoError(t, err)

		// Verify content was updated
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, updated, string(data))
	})

	t.Run("normalizes profile name", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		content := "---\nname: normalized\n---\nContent"

		// Name with spaces and uppercase should be normalized
		path, err := svc.Write("My Test Profile", content, "local", false)
		require.NoError(t, err)

		expectedPath := filepath.Join(tempDir, ".brains", "profiles", "my-test-profile.md")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("returns error for invalid location", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		_, err = svc.Write("test", "content", "invalid", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid location")
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		tempDir := t.TempDir()
		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		_, err = svc.Write("", "content", "local", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestProfileService_Compose(t *testing.T) {
	t.Run("composes profiles by name", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, ".brains", "profiles")
		require.NoError(t, os.MkdirAll(profileDir, 0o755))

		content := `---
name: test
description: Test profile
---

Test profile content.
`
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "test.md"), []byte(content), 0o644))

		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		result, err := svc.Compose([]string{"test"})
		require.NoError(t, err)
		assert.Contains(t, result.Content, "Test profile content")
	})

	t.Run("composes multiple profiles", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, ".brains", "profiles")
		require.NoError(t, os.MkdirAll(profileDir, 0o755))

		content1 := `---
name: first
---

First profile content.
`
		content2 := `---
name: second
---

Second profile content.
`
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "first.md"), []byte(content1), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "second.md"), []byte(content2), 0o644))

		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		result, err := svc.Compose([]string{"first", "second"})
		require.NoError(t, err)
		assert.Contains(t, result.Content, "First profile content")
		assert.Contains(t, result.Content, "Second profile content")
	})

	t.Run("returns error for non-existent profile", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := filepath.Join(tempDir, ".brains", "profiles")
		require.NoError(t, os.MkdirAll(profileDir, 0o755))

		svc, err := profile.NewService(tempDir)
		require.NoError(t, err)

		_, err = svc.Compose([]string{"nonexistent"})
		require.Error(t, err)
	})
}
