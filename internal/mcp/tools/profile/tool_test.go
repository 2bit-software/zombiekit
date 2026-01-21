package profile

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/profile"
)

// Helper to set up embedded profiles for testing
func setupEmbeddedFS(_ *testing.T) func() {
	originalFS := profile.GetEmbeddedFS()

	mockFS := fstest.MapFS{
		"profiles/embedded-test.md": &fstest.MapFile{
			Data: []byte(`---
name: embedded-test
description: Test profile from embedded
---

Embedded test content.
`),
		},
		"profiles/mcp-test.md": &fstest.MapFile{
			Data: []byte(`---
name: mcp-test
description: MCP test profile
---

MCP test content for tools.
`),
		},
	}
	profile.SetEmbeddedFS(mockFS)

	return func() {
		profile.SetEmbeddedFS(originalFS)
	}
}


// T025: Unit test for MCP profile-compose with embedded profile
func TestMCPTool_ComposeWithEmbedded(t *testing.T) {
	cleanup := setupEmbeddedFS(t)
	defer cleanup()

	tool := NewTool()
	ctx := context.Background()

	// Compose an embedded profile
	result, err := tool.HandleCompose(ctx, map[string]interface{}{
		"profiles": []interface{}{"embedded-test"},
	})

	require.NoError(t, err)
	assert.Contains(t, result, "Embedded test content")
}

// T026: Unit test for MCP profile-list returning embedded profiles
func TestMCPTool_ListIncludesEmbedded(t *testing.T) {
	cleanup := setupEmbeddedFS(t)
	defer cleanup()

	tool := NewTool()
	ctx := context.Background()

	result, err := tool.HandleList(ctx, map[string]interface{}{})

	require.NoError(t, err)
	// Check that embedded profiles are listed
	assert.Contains(t, result, "embedded-test")
	assert.Contains(t, result, "mcp-test")
	assert.Contains(t, result, "(embedded)")
}

func TestMCPTool_ShowEmbedded(t *testing.T) {
	cleanup := setupEmbeddedFS(t)
	defer cleanup()

	tool := NewTool()
	ctx := context.Background()

	result, err := tool.HandleShow(ctx, map[string]interface{}{
		"name": "embedded-test",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "Embedded test content")
}

func TestMCPTool_ValidateIncludesEmbedded(t *testing.T) {
	cleanup := setupEmbeddedFS(t)
	defer cleanup()

	tool := NewTool()
	ctx := context.Background()

	result, err := tool.HandleValidate(ctx, map[string]interface{}{})

	require.NoError(t, err)
	// Should validate successfully (no missing includes)
	assert.Contains(t, strings.ToLower(result), "validated successfully")
}

func TestMCPTool_HandleSave(t *testing.T) {
	t.Run("returns error when name is missing", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleSave(ctx, map[string]interface{}{
			"content":  "test content",
			"location": "local",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("returns error when content is missing", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":     "test",
			"location": "local",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "content is required")
	})

	t.Run("returns error when location is missing", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":    "test",
			"content": "test content",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "location is required")
	})

	t.Run("returns error for invalid location", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":     "test",
			"content":  "test content",
			"location": "invalid",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "local")
	})

	t.Run("writes profile successfully", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()
		tempDir := t.TempDir()

		content := `---
name: handler-test
description: Test profile
---

Handler test content.
`
		result, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":              "handler-test",
			"content":           content,
			"location":          "local",
			"working_directory": tempDir,
		})

		require.NoError(t, err)
		assert.Contains(t, result, `"success": true`)
		assert.Contains(t, result, "handler-test.md")
	})

	t.Run("returns PROFILE_EXISTS error without overwrite", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()
		tempDir := t.TempDir()

		content := "---\nname: exists-test\n---\nContent"

		// Write first time
		_, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":              "exists-test",
			"content":           content,
			"location":          "local",
			"working_directory": tempDir,
		})
		require.NoError(t, err)

		// Write second time without overwrite
		result, err := tool.HandleSave(ctx, map[string]interface{}{
			"name":              "exists-test",
			"content":           "new content",
			"location":          "local",
			"working_directory": tempDir,
		})

		// Should return success=false response, not error
		require.NoError(t, err)
		assert.Contains(t, result, `"success": false`)
		assert.Contains(t, result, "PROFILE_EXISTS")
	})
}

