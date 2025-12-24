package step

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTool(t *testing.T) {
	t.Run("creates tool instance", func(t *testing.T) {
		tool := NewTool()
		require.NotNil(t, tool)
	})
}

func TestTool_Definition(t *testing.T) {
	t.Run("returns valid tool definition", func(t *testing.T) {
		tool := NewTool()
		def := tool.Definition()

		assert.Equal(t, "step", def.Name)
		assert.Contains(t, def.Description, "workflow step")

		// Check input schema has required properties
		schema := def.InputSchema
		assert.Equal(t, "object", schema["type"])

		props, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, props, "step")
		assert.Contains(t, props, "dir")
		assert.Contains(t, props, "initiative")

		required, ok := schema["required"].([]string)
		require.True(t, ok)
		assert.Contains(t, required, "step")
		assert.Contains(t, required, "dir")
	})
}

func TestTool_Execute(t *testing.T) {
	t.Run("executes step and returns response", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory with active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		// Create history folder
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte("# Spec"), 0644))

		// Set active initiative
		activeJSON := `{"initiative": "history/test-feature", "status": "active"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Create specification
files:
  - "spec.md"
---
Create the specification.`),
			},
		}

		tool := NewTool()
		tool.SetEmbeddedFS(embeddedFS)

		args := map[string]interface{}{
			"step": "specify",
			"dir":  tmpDir,
		}

		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		assert.Contains(t, result, "directive")
		assert.Contains(t, result, "history_folder")
		assert.Contains(t, result, "files_to_read")
	})

	t.Run("returns error for missing step parameter", func(t *testing.T) {
		tool := NewTool()

		args := map[string]interface{}{
			"dir": "/some/path",
		}

		_, err := tool.Execute(context.Background(), args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step")
	})

	t.Run("returns error for missing dir parameter", func(t *testing.T) {
		tool := NewTool()

		args := map[string]interface{}{
			"step": "specify",
		}

		_, err := tool.Execute(context.Background(), args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dir")
	})

	t.Run("returns NOT_INITIALIZED error when no .brains folder", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
---
Directive`),
			},
		}

		tool := NewTool()
		tool.SetEmbeddedFS(embeddedFS)

		args := map[string]interface{}{
			"step": "specify",
			"dir":  tmpDir,
		}

		_, err := tool.Execute(context.Background(), args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NOT_INITIALIZED")
	})

	t.Run("returns UNKNOWN_STEP error for invalid step", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		tool := NewTool()

		args := map[string]interface{}{
			"step": "nonexistent",
			"dir":  tmpDir,
		}

		_, err := tool.Execute(context.Background(), args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})

	t.Run("accepts initiative override parameter", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		// Create two history folders
		historyDir1 := filepath.Join(tmpDir, "history", "first")
		historyDir2 := filepath.Join(tmpDir, "history", "second")
		require.NoError(t, os.MkdirAll(historyDir1, 0755))
		require.NoError(t, os.MkdirAll(historyDir2, 0755))

		// Active is first
		activeJSON := `{"initiative": "history/first", "status": "active"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
---
Directive`),
			},
		}

		tool := NewTool()
		tool.SetEmbeddedFS(embeddedFS)

		args := map[string]interface{}{
			"step":       "specify",
			"dir":        tmpDir,
			"initiative": "second",
		}

		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		assert.Contains(t, result, "second")
	})

	t.Run("init step works without active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory without active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		embeddedFS := fstest.MapFS{
			"steps/init.md": &fstest.MapFile{
				Data: []byte(`---
name: init
description: Initialize new initiative
---
Initialize a new initiative.`),
			},
		}

		tool := NewTool()
		tool.SetEmbeddedFS(embeddedFS)

		args := map[string]interface{}{
			"step": "init",
			"dir":  tmpDir,
			"type": "feature",
			"name": "test-feature",
		}

		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		assert.Contains(t, result, "directive")
	})
}

func TestTool_GetStringArg(t *testing.T) {
	t.Run("returns string value", func(t *testing.T) {
		args := map[string]interface{}{
			"key": "value",
		}

		result := getStringArg(args, "key")
		assert.Equal(t, "value", result)
	})

	t.Run("returns empty for missing key", func(t *testing.T) {
		args := map[string]interface{}{}

		result := getStringArg(args, "key")
		assert.Equal(t, "", result)
	})

	t.Run("returns empty for non-string value", func(t *testing.T) {
		args := map[string]interface{}{
			"key": 123,
		}

		result := getStringArg(args, "key")
		assert.Equal(t, "", result)
	})
}
