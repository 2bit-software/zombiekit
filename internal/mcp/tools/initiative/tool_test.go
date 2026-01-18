package initiative

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalInit "github.com/zombiekit/brains/internal/initiative"
)

// testEmbeddedFS returns a mock embedded filesystem with templates for testing.
func testEmbeddedFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte("# Spec Template\n\nThis is a test spec template."),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte("# Research Template\n\nThis is a test research template."),
		},
	}
}

func TestCopyTemplateIfNotExists(t *testing.T) {
	templateContent := []byte("# Template Content\n\nThis is a template.")

	t.Run("copies template when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "spec.md")

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		require.NoError(t, err)
		assert.True(t, copied)

		// Verify file was created with template content
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, templateContent, content)
	})

	t.Run("skips copy when file has content", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "spec.md")

		// Create file with custom content
		customContent := []byte("# My Custom Spec\n\nThis should not be overwritten.")
		require.NoError(t, os.WriteFile(destPath, customContent, 0644))

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		require.NoError(t, err)
		assert.False(t, copied)

		// Verify file was not modified
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, customContent, content)
	})

	t.Run("copies template when file is empty (0 bytes)", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "spec.md")

		// Create empty file
		require.NoError(t, os.WriteFile(destPath, []byte{}, 0644))

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		require.NoError(t, err)
		assert.True(t, copied)

		// Verify file now has template content
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, templateContent, content)
	})

	t.Run("copies template when file contains only whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "spec.md")

		// Create file with only whitespace
		whitespaceContent := []byte("   \n\t\n   ")
		require.NoError(t, os.WriteFile(destPath, whitespaceContent, 0644))

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		require.NoError(t, err)
		assert.True(t, copied)

		// Verify file now has template content
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, templateContent, content)
	})

	t.Run("returns error when cannot write to destination", func(t *testing.T) {
		// Try to write to a path that doesn't exist (parent dir missing)
		destPath := "/nonexistent/directory/spec.md"

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		assert.Error(t, err)
		assert.False(t, copied)
		assert.Contains(t, err.Error(), "writing template")
	})

	t.Run("preserves file with minimal content", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "spec.md")

		// Create file with minimal non-whitespace content
		minimalContent := []byte("x")
		require.NoError(t, os.WriteFile(destPath, minimalContent, 0644))

		copied, err := copyTemplateIfNotExists(templateContent, destPath)
		require.NoError(t, err)
		assert.False(t, copied)

		// Verify file was not modified
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, minimalContent, content)
	})
}

// Integration tests for handleCreate idempotency

func TestHandleCreate_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	tool := NewTool()
	tool.SetEmbeddedFS(testEmbeddedFS())
	ctx := context.Background()

	// First create - should succeed
	args1 := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "feature",
		"name":   "foo",
	}

	result1, err := tool.Execute(ctx, args1)
	require.NoError(t, err)

	var resp1 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result1), &resp1))
	assert.False(t, resp1.AlreadyExisted)
	assert.NotEmpty(t, resp1.InitiativeID)

	// Write custom content to spec.md
	specPath := filepath.Join(resp1.CyclePath, "spec.md")
	customContent := []byte("# My Custom Spec\n\nThis is my custom content that should be preserved.")
	require.NoError(t, os.WriteFile(specPath, customContent, 0644))

	// Second create with same name+type - should return existing
	result2, err := tool.Execute(ctx, args1)
	require.NoError(t, err)

	var resp2 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result2), &resp2))

	// Should return same initiative with AlreadyExisted=true
	assert.True(t, resp2.AlreadyExisted)
	assert.Equal(t, resp1.InitiativeID, resp2.InitiativeID)

	// spec.md should be in skipped files
	assert.Contains(t, resp2.SkippedFiles, "spec.md")

	// Verify spec.md content was NOT overwritten
	finalContent, err := os.ReadFile(specPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, finalContent)
}

func TestHandleCreate_DifferentInitiativeActive(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	tool := NewTool()
	tool.SetEmbeddedFS(testEmbeddedFS())
	ctx := context.Background()

	// First create initiative "foo"
	args1 := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "feature",
		"name":   "foo",
	}

	result1, err := tool.Execute(ctx, args1)
	require.NoError(t, err)

	var resp1 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result1), &resp1))
	assert.False(t, resp1.AlreadyExisted)

	// Try to create different initiative "bar" - should fail
	args2 := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "feature",
		"name":   "bar",
	}

	_, err = tool.Execute(ctx, args2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INITIATIVE_ALREADY_ACTIVE")
}

func TestHandleCreate_SameNameDifferentType(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	tool := NewTool()
	tool.SetEmbeddedFS(testEmbeddedFS())
	ctx := context.Background()

	// Create feature initiative "foo"
	args1 := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "feature",
		"name":   "foo",
	}

	result1, err := tool.Execute(ctx, args1)
	require.NoError(t, err)

	var resp1 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result1), &resp1))
	assert.False(t, resp1.AlreadyExisted)

	// Try to create bug initiative with same name "foo" - should fail
	// (name+type must both match for idempotency)
	args2 := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "bug",
		"name":   "foo",
	}

	_, err = tool.Execute(ctx, args2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INITIATIVE_ALREADY_ACTIVE")
}

func TestHandleCreate_AfterComplete(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	tool := NewTool()
	tool.SetEmbeddedFS(testEmbeddedFS())
	ctx := context.Background()

	// Create initiative
	createArgs := map[string]interface{}{
		"action": "create",
		"dir":    tmpDir,
		"type":   "feature",
		"name":   "foo",
	}

	result1, err := tool.Execute(ctx, createArgs)
	require.NoError(t, err)

	var resp1 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result1), &resp1))
	firstID := resp1.InitiativeID

	// Complete the initiative
	initSvc, err := internalInit.NewService(tmpDir)
	require.NoError(t, err)
	require.NoError(t, initSvc.Complete())

	// Sleep briefly to ensure different timestamp for ID generation
	// (IDs use unix timestamp which has 1-second resolution)
	time.Sleep(1100 * time.Millisecond)

	// Create again with same name+type - should create NEW initiative
	result2, err := tool.Execute(ctx, createArgs)
	require.NoError(t, err)

	var resp2 CreateResponse
	require.NoError(t, json.Unmarshal([]byte(result2), &resp2))

	// Should be a new initiative (different ID), not idempotent return
	assert.False(t, resp2.AlreadyExisted)
	assert.NotEqual(t, firstID, resp2.InitiativeID)
}
