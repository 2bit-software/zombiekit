package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestFS(subdir, name, content string) fstest.MapFS {
	return fstest.MapFS{
		subdir + "/" + name + ".md": &fstest.MapFile{Data: []byte(content)},
	}
}

func TestTool_HandleLoad(t *testing.T) {
	t.Run("returns error when name is missing", func(t *testing.T) {
		tool := NewTool(nil, nil)
		ctx := context.Background()

		_, err := tool.HandleLoad(ctx, map[string]any{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		tool := NewTool(nil, nil)
		ctx := context.Background()

		_, err := tool.HandleLoad(ctx, map[string]any{
			"name": "",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("returns error for invalid type", func(t *testing.T) {
		tool := NewTool(nil, nil)
		ctx := context.Background()

		_, err := tool.HandleLoad(ctx, map[string]any{
			"name": "foo",
			"type": "invalid",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "type must be")
	})

	t.Run("loads workflow from embedded", func(t *testing.T) {
		wfFS := makeTestFS("workflows", "new", "---\nname: new\ndescription: Test\n---\n\nTest workflow content.\n")
		tool := NewTool(nil, wfFS)
		ctx := context.Background()

		result, err := tool.HandleLoad(ctx, map[string]any{
			"name": "new",
			"type": "workflow",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Test workflow content")
	})

	t.Run("loads command from embedded", func(t *testing.T) {
		cmdFS := makeTestFS("commands", "next", "---\nname: next\ndescription: Test\n---\n\nTest command content.\n")
		tool := NewTool(cmdFS, nil)
		ctx := context.Background()

		result, err := tool.HandleLoad(ctx, map[string]any{
			"name": "next",
			"type": "command",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Test command content")
	})

	t.Run("loads workflow from local directory", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, ".brains", "workflows")
		require.NoError(t, os.MkdirAll(workflowDir, 0o755))

		content := "---\nname: local-workflow\ndescription: Local\n---\n\nLocal workflow content.\n"
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "local-workflow.md"), []byte(content), 0o644))

		tool := NewTool(nil, nil)
		ctx := context.Background()

		result, err := tool.HandleLoad(ctx, map[string]any{
			"name":              "local-workflow",
			"type":              "workflow",
			"working_directory": tempDir,
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Local workflow content")
	})

	t.Run("local workflow shadows embedded", func(t *testing.T) {
		wfFS := makeTestFS("workflows", "new", "---\nname: new\ndescription: Test\n---\n\nEmbedded content.\n")

		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, ".brains", "workflows")
		require.NoError(t, os.MkdirAll(workflowDir, 0o755))

		content := "---\nname: new\ndescription: Local override\n---\n\nLocal override content.\n"
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "new.md"), []byte(content), 0o644))

		tool := NewTool(nil, wfFS)
		ctx := context.Background()

		result, err := tool.HandleLoad(ctx, map[string]any{
			"name":              "new",
			"type":              "workflow",
			"working_directory": tempDir,
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Local override content")
		assert.NotContains(t, result, "Embedded content")
	})

	t.Run("returns error for non-existent workflow", func(t *testing.T) {
		tool := NewTool(nil, nil)
		ctx := context.Background()

		_, err := tool.HandleLoad(ctx, map[string]any{
			"name": "nonexistent",
			"type": "workflow",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTool_Definition(t *testing.T) {
	tool := NewTool(nil, nil)
	def := tool.Definition()

	assert.Equal(t, "workflow-load", def.Name)
	assert.NotEmpty(t, def.Description)
}
