package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

func setupEmbeddedFS(t *testing.T) func() {
	originalFS := workflow.GetEmbeddedFS()

	mockFS := fstest.MapFS{
		"workflows/new.md": &fstest.MapFile{
			Data: []byte(`---
name: new
description: Test workflow
---

Test workflow content.
`),
		},
	}
	workflow.SetEmbeddedFS(mockFS)

	return func() {
		if originalFS != nil {
			workflow.SetEmbeddedFS(originalFS)
		} else {
			workflow.ResetEmbeddedFS()
		}
	}
}

func TestTool_HandleCompose(t *testing.T) {
	t.Run("returns error when name is missing", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleCompose(ctx, map[string]interface{}{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleCompose(ctx, map[string]interface{}{
			"name": "",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("loads workflow from embedded", func(t *testing.T) {
		cleanup := setupEmbeddedFS(t)
		defer cleanup()

		tool := NewTool()
		ctx := context.Background()

		result, err := tool.HandleCompose(ctx, map[string]interface{}{
			"name": "new",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Test workflow content")
	})

	t.Run("loads workflow from local directory", func(t *testing.T) {
		cleanup := setupEmbeddedFS(t)
		defer cleanup()

		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, ".brains", "workflows")
		require.NoError(t, os.MkdirAll(workflowDir, 0o755))

		content := `---
name: local-workflow
description: Local workflow
---

Local workflow content.
`
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "local-workflow.md"), []byte(content), 0o644))

		tool := NewTool()
		ctx := context.Background()

		result, err := tool.HandleCompose(ctx, map[string]interface{}{
			"name":              "local-workflow",
			"working_directory": tempDir,
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Local workflow content")
	})

	t.Run("local workflow shadows embedded", func(t *testing.T) {
		cleanup := setupEmbeddedFS(t)
		defer cleanup()

		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, ".brains", "workflows")
		require.NoError(t, os.MkdirAll(workflowDir, 0o755))

		content := `---
name: new
description: Local override
---

Local override content.
`
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "new.md"), []byte(content), 0o644))

		tool := NewTool()
		ctx := context.Background()

		result, err := tool.HandleCompose(ctx, map[string]interface{}{
			"name":              "new",
			"working_directory": tempDir,
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Local override content")
		assert.NotContains(t, result, "Test workflow content")
	})

	t.Run("returns error for non-existent workflow", func(t *testing.T) {
		cleanup := setupEmbeddedFS(t)
		defer cleanup()

		tool := NewTool()
		ctx := context.Background()

		_, err := tool.HandleCompose(ctx, map[string]interface{}{
			"name": "nonexistent",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTool_Definition(t *testing.T) {
	tool := NewTool()
	def := tool.Definition()

	assert.Equal(t, "workflow-compose", def.Name)
	assert.NotEmpty(t, def.Description)
}
