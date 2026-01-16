package profile

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T018: Unit test verifying local profile shadows embedded profile with same name
func TestBrainsSource_LocalShadowsEmbedded(t *testing.T) {
	// Save original embedded FS
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	// Create temp directory with local profile
	tmpDir, err := os.MkdirTemp("", "brains-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .brains/profiles directory structure
	profilesDir := filepath.Join(tmpDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0755))

	// Create local profile with same name as embedded
	localContent := `---
name: test
description: Local test profile
---

Local content - this should take precedence.
`
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "test.md"), []byte(localContent), 0644))

	// Set up embedded FS with a profile of the same name
	embeddedFS := fstest.MapFS{
		"profiles/test.md": &fstest.MapFile{
			Data: []byte(`---
name: test
description: Embedded test profile
---

Embedded content - this should be shadowed.
`),
		},
	}
	SetEmbeddedFS(embeddedFS)

	// Create BrainsSource
	source, err := NewBrainsSource(tmpDir)
	require.NoError(t, err)

	// Find directories
	dirs, err := source.FindProfileDirs()
	require.NoError(t, err)

	// Load profiles
	profiles, err := source.LoadProfiles(dirs)
	require.NoError(t, err)

	// Verify local profile shadows embedded
	p, exists := profiles["test"]
	require.True(t, exists, "profile 'test' should exist")
	assert.Equal(t, SourceLocal, p.Source, "profile should be from local source")
	assert.Equal(t, "Local test profile", p.Description)
	assert.Contains(t, p.Body, "Local content")
}

// T019: Unit test verifying global profile shadows embedded profile with same name
func TestBrainsSource_GlobalShadowsEmbedded(t *testing.T) {
	// Save original embedded FS
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	// Create temp directory structure to simulate global
	tmpDir, err := os.MkdirTemp("", "brains-global-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a fake home directory
	homeDir := filepath.Join(tmpDir, "home")
	globalProfilesDir := filepath.Join(homeDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(globalProfilesDir, 0755))

	// Create global profile
	globalContent := `---
name: global-test
description: Global test profile
---

Global content - this should shadow embedded.
`
	require.NoError(t, os.WriteFile(filepath.Join(globalProfilesDir, "global-test.md"), []byte(globalContent), 0644))

	// Set up embedded FS with a profile of the same name
	embeddedFS := fstest.MapFS{
		"profiles/global-test.md": &fstest.MapFile{
			Data: []byte(`---
name: global-test
description: Embedded global-test profile
---

Embedded content.
`),
		},
	}
	SetEmbeddedFS(embeddedFS)

	// Create a workdir (without local profiles)
	workDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Use custom resolver with our fake home
	source := &BrainsSource{
		resolver: &Resolver{
			workingDir: workDir,
			homeDir:    homeDir,
		},
		workingDir: workDir,
		homeDir:    homeDir,
	}

	// Find directories
	dirs, err := source.FindProfileDirs()
	require.NoError(t, err)

	// Load profiles
	profiles, err := source.LoadProfiles(dirs)
	require.NoError(t, err)

	// Verify global profile shadows embedded
	p, exists := profiles["global-test"]
	require.True(t, exists, "profile 'global-test' should exist")
	assert.Equal(t, SourceGlobal, p.Source, "profile should be from global source")
	assert.Equal(t, "Global test profile", p.Description)
	assert.Contains(t, p.Body, "Global content")
}

// T020: Unit test verifying precedence order: local > parent > global > embedded
func TestBrainsSource_FullPrecedenceOrder(t *testing.T) {
	// Save original embedded FS
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	// Set up embedded FS
	embeddedFS := fstest.MapFS{
		"profiles/precedence-test.md": &fstest.MapFile{
			Data: []byte(`---
name: precedence-test
description: Embedded profile
---

Embedded.
`),
		},
	}
	SetEmbeddedFS(embeddedFS)

	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "brains-precedence-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create project with local profiles
	projectDir := filepath.Join(tmpDir, "project")
	localProfilesDir := filepath.Join(projectDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(localProfilesDir, 0755))

	localContent := `---
name: precedence-test
description: Local profile - highest precedence
---

Local.
`
	require.NoError(t, os.WriteFile(filepath.Join(localProfilesDir, "precedence-test.md"), []byte(localContent), 0644))

	// Create BrainsSource
	source, err := NewBrainsSource(projectDir)
	require.NoError(t, err)

	// Find and load
	dirs, err := source.FindProfileDirs()
	require.NoError(t, err)
	profiles, err := source.LoadProfiles(dirs)
	require.NoError(t, err)

	// Verify local takes precedence
	p := profiles["precedence-test"]
	require.NotNil(t, p)
	assert.Equal(t, SourceLocal, p.Source)
	assert.Equal(t, "Local profile - highest precedence", p.Description)
}

// Test that LoadAllProfiles includes both local and embedded versions
func TestBrainsSource_LoadAllProfilesIncludesEmbedded(t *testing.T) {
	// Save original embedded FS
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	// Create temp directory with local profile
	tmpDir, err := os.MkdirTemp("", "brains-all-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create local profile
	profilesDir := filepath.Join(tmpDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0755))

	localContent := `---
name: shadowed
description: Local version
---

Local.
`
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "shadowed.md"), []byte(localContent), 0644))

	// Set up embedded FS
	embeddedFS := fstest.MapFS{
		"profiles/shadowed.md": &fstest.MapFile{
			Data: []byte(`---
name: shadowed
description: Embedded version
---

Embedded.
`),
		},
		"profiles/embedded-only.md": &fstest.MapFile{
			Data: []byte(`---
name: embedded-only
description: Only in embedded
---

Embedded only.
`),
		},
	}
	SetEmbeddedFS(embeddedFS)

	// Create source and load all
	source, err := NewBrainsSource(tmpDir)
	require.NoError(t, err)

	dirs, err := source.FindProfileDirs()
	require.NoError(t, err)

	allProfiles, err := source.LoadAllProfiles(dirs)
	require.NoError(t, err)

	// Verify shadowed profile has both versions
	shadowedVersions := allProfiles["shadowed"]
	require.Len(t, shadowedVersions, 2, "should have local and embedded versions")

	// First should be local (higher precedence)
	assert.Equal(t, SourceLocal, shadowedVersions[0].Source)
	// Second should be embedded
	assert.Equal(t, SourceEmbedded, shadowedVersions[1].Source)

	// Verify embedded-only profile exists
	embeddedOnlyVersions := allProfiles["embedded-only"]
	require.Len(t, embeddedOnlyVersions, 1)
	assert.Equal(t, SourceEmbedded, embeddedOnlyVersions[0].Source)
}
