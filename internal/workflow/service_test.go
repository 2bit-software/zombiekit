package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEmbeddedFS(t *testing.T) func() {
	t.Helper()
	originalFS := GetEmbeddedFS()

	mockFS := fstest.MapFS{
		"workflows/test-workflow.md": &fstest.MapFile{
			Data: []byte(`---
name: test-workflow
description: Test workflow for testing
---

Test workflow content.
`),
		},
		"workflows/no-frontmatter.md": &fstest.MapFile{
			Data: []byte(`Just plain content without frontmatter.`),
		},
	}
	SetEmbeddedFS(mockFS)

	return func() {
		if originalFS != nil {
			SetEmbeddedFS(originalFS)
		} else {
			ResetEmbeddedFS()
		}
	}
}

func TestService_Load(t *testing.T) {
	t.Run("loads workflow from embedded", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
		defer cleanup()

		svc, err := NewService("")
		require.NoError(t, err)

		wf, err := svc.Load("test-workflow")
		require.NoError(t, err)

		assert.Equal(t, "test-workflow", wf.Name)
		assert.Equal(t, "Test workflow for testing", wf.Description)
		assert.Contains(t, wf.Content, "Test workflow content")
		assert.Equal(t, "embedded", wf.Source)
	})

	t.Run("loads workflow without frontmatter", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
		defer cleanup()

		svc, err := NewService("")
		require.NoError(t, err)

		wf, err := svc.Load("no-frontmatter")
		require.NoError(t, err)

		assert.Equal(t, "no-frontmatter", wf.Name)
		assert.Contains(t, wf.Content, "Just plain content")
	})

	t.Run("loads workflow from local directory", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
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

		svc, err := NewService(tempDir)
		require.NoError(t, err)

		wf, err := svc.Load("local-workflow")
		require.NoError(t, err)

		assert.Equal(t, "local-workflow", wf.Name)
		assert.Contains(t, wf.Content, "Local workflow content")
		assert.Equal(t, "local", wf.Source)
	})

	t.Run("local workflow shadows embedded", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
		defer cleanup()

		tempDir := t.TempDir()
		workflowDir := filepath.Join(tempDir, ".brains", "workflows")
		require.NoError(t, os.MkdirAll(workflowDir, 0o755))

		// Create local override with same name as embedded
		content := `---
name: test-workflow
description: Local override
---

This is the LOCAL override.
`
		require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "test-workflow.md"), []byte(content), 0o644))

		svc, err := NewService(tempDir)
		require.NoError(t, err)

		wf, err := svc.Load("test-workflow")
		require.NoError(t, err)

		assert.Contains(t, wf.Content, "LOCAL override")
		assert.NotContains(t, wf.Content, "Test workflow content")
		assert.Equal(t, "local", wf.Source)
	})

	t.Run("returns error for non-existent workflow", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
		defer cleanup()

		svc, err := NewService("")
		require.NoError(t, err)

		_, err = svc.Load("nonexistent")
		require.Error(t, err)

		var notFoundErr *WorkflowNotFoundError
		assert.ErrorAs(t, err, &notFoundErr)
		assert.Equal(t, "nonexistent", notFoundErr.Name)
	})
}

func TestWorkflowNotFoundError(t *testing.T) {
	err := &WorkflowNotFoundError{Name: "test"}
	assert.Contains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), "not found")
}
