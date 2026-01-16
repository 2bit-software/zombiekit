package profile

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a mock embedded FS for testing
func createMockEmbeddedFS() fstest.MapFS {
	return fstest.MapFS{
		"profiles/test.md": &fstest.MapFile{
			Data: []byte(`---
name: test
description: A test profile
type: action
---

Test content here.
`),
		},
		"profiles/research.md": &fstest.MapFile{
			Data: []byte(`---
name: research
description: Research profile
inherits: true
---

Research content.
`),
		},
	}
}

// T008: Unit test for SetEmbeddedFS/GetEmbeddedFS/HasEmbeddedProfiles
func TestEmbeddedFSRegistry(t *testing.T) {
	// Clean up after test
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	t.Run("GetEmbeddedFS returns nil when not set", func(t *testing.T) {
		SetEmbeddedFS(nil)
		assert.Nil(t, GetEmbeddedFS())
	})

	t.Run("SetEmbeddedFS stores filesystem", func(t *testing.T) {
		mockFS := createMockEmbeddedFS()
		SetEmbeddedFS(mockFS)
		assert.NotNil(t, GetEmbeddedFS())
	})

	t.Run("HasEmbeddedProfiles returns false when nil", func(t *testing.T) {
		SetEmbeddedFS(nil)
		assert.False(t, HasEmbeddedProfiles())
	})

	t.Run("HasEmbeddedProfiles returns true when profiles exist", func(t *testing.T) {
		mockFS := createMockEmbeddedFS()
		SetEmbeddedFS(mockFS)
		assert.True(t, HasEmbeddedProfiles())
	})

	t.Run("HasEmbeddedProfiles returns false for empty FS", func(t *testing.T) {
		emptyFS := fstest.MapFS{
			"profiles/.keep": &fstest.MapFile{Data: []byte{}},
		}
		SetEmbeddedFS(emptyFS)
		assert.False(t, HasEmbeddedProfiles())
	})
}

// T009: Unit test for loadEmbeddedProfiles() returning correct profiles
func TestLoadEmbeddedProfiles(t *testing.T) {
	// Clean up after test
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	t.Run("returns nil when no FS registered", func(t *testing.T) {
		SetEmbeddedFS(nil)
		profiles := loadEmbeddedProfiles()
		assert.Nil(t, profiles)
	})

	t.Run("loads all valid profiles", func(t *testing.T) {
		mockFS := createMockEmbeddedFS()
		SetEmbeddedFS(mockFS)

		profiles := loadEmbeddedProfiles()
		require.Len(t, profiles, 2)

		// Check that both profiles were loaded
		names := make(map[string]bool)
		for _, p := range profiles {
			names[p.Name] = true
		}
		assert.True(t, names["test"])
		assert.True(t, names["research"])
	})

	t.Run("returns map from loadProfilesFromEmbedded", func(t *testing.T) {
		mockFS := createMockEmbeddedFS()
		SetEmbeddedFS(mockFS)

		profiles := loadProfilesFromEmbedded()
		require.Len(t, profiles, 2)
		assert.NotNil(t, profiles["test"])
		assert.NotNil(t, profiles["research"])
	})

	t.Run("skips invalid profiles gracefully", func(t *testing.T) {
		invalidFS := fstest.MapFS{
			"profiles/valid.md": &fstest.MapFile{
				Data: []byte(`---
name: valid
---

Valid content.
`),
			},
			"profiles/invalid.md": &fstest.MapFile{
				// Invalid YAML frontmatter (unclosed quotes)
				Data: []byte(`---
name: "invalid
---

Content.
`),
			},
		}
		SetEmbeddedFS(invalidFS)

		profiles := loadEmbeddedProfiles()
		// Should have 1 valid profile
		assert.Len(t, profiles, 1)
		assert.Equal(t, "valid", profiles[0].Name)
	})

	t.Run("skips non-markdown files", func(t *testing.T) {
		mixedFS := fstest.MapFS{
			"profiles/readme.txt": &fstest.MapFile{
				Data: []byte("not a profile"),
			},
			"profiles/valid.md": &fstest.MapFile{
				Data: []byte(`---
name: valid
---

Content.
`),
			},
		}
		SetEmbeddedFS(mixedFS)

		profiles := loadEmbeddedProfiles()
		assert.Len(t, profiles, 1)
	})
}

// T010: Unit test verifying embedded profiles have source=SourceEmbedded and path="[embedded]/name.md"
func TestEmbeddedProfileMetadata(t *testing.T) {
	// Clean up after test
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	mockFS := createMockEmbeddedFS()
	SetEmbeddedFS(mockFS)

	profiles := loadEmbeddedProfiles()
	require.NotEmpty(t, profiles)

	for _, p := range profiles {
		t.Run("profile_"+p.Name, func(t *testing.T) {
			// Verify source is SourceEmbedded
			assert.Equal(t, SourceEmbedded, p.Source, "profile should have SourceEmbedded")
			assert.Equal(t, "embedded", p.Source.String())

			// Verify path format is [embedded]/<name>.md
			expectedPath := "[embedded]/" + p.Name + ".md"
			assert.Equal(t, expectedPath, p.Path, "profile path should be [embedded]/<name>.md")

			// Verify content was parsed
			assert.NotEmpty(t, p.Body)
		})
	}
}

func TestEmbeddedProfileContent(t *testing.T) {
	// Clean up after test
	originalFS := GetEmbeddedFS()
	defer SetEmbeddedFS(originalFS)

	mockFS := createMockEmbeddedFS()
	SetEmbeddedFS(mockFS)

	profiles := loadProfilesFromEmbedded()

	t.Run("test profile has correct metadata", func(t *testing.T) {
		p := profiles["test"]
		require.NotNil(t, p)
		assert.Equal(t, "A test profile", p.Description)
		assert.Equal(t, "action", p.Type)
	})

	t.Run("research profile has correct metadata", func(t *testing.T) {
		p := profiles["research"]
		require.NotNil(t, p)
		assert.Equal(t, "Research profile", p.Description)
		assert.True(t, p.Inherits)
	})
}
