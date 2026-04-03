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

func TestService_List(t *testing.T) {
	t.Run("lists all workflows from embedded", func(t *testing.T) {
		cleanup := setupTestEmbeddedFS(t)
		defer cleanup()

		svc, err := NewService("")
		require.NoError(t, err)

		workflows, err := svc.List()
		require.NoError(t, err)

		// Should include both embedded workflows
		assert.GreaterOrEqual(t, len(workflows), 2)

		// Find workflows by name
		names := make(map[string]string)
		for _, wf := range workflows {
			names[wf.Name] = wf.Source
		}

		assert.Contains(t, names, "test-workflow")
		assert.Contains(t, names, "no-frontmatter")
	})

	t.Run("lists workflows from local directory", func(t *testing.T) {
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

		workflows, err := svc.List()
		require.NoError(t, err)

		// Find local workflow
		var localWorkflow *Workflow
		for _, wf := range workflows {
			if wf.Name == "local-workflow" {
				localWorkflow = wf
				break
			}
		}

		require.NotNil(t, localWorkflow)
		assert.Equal(t, "local", localWorkflow.Source)
	})

	t.Run("local workflow shadows embedded in list", func(t *testing.T) {
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

		workflows, err := svc.List()
		require.NoError(t, err)

		// Find test-workflow - should be local version, not embedded
		var testWorkflow *Workflow
		testWorkflowCount := 0
		for _, wf := range workflows {
			if wf.Name == "test-workflow" {
				testWorkflow = wf
				testWorkflowCount++
			}
		}

		require.NotNil(t, testWorkflow)
		assert.Equal(t, 1, testWorkflowCount, "should not have duplicate workflows")
		assert.Equal(t, "local", testWorkflow.Source)
		assert.Contains(t, testWorkflow.Content, "LOCAL override")
	})

	t.Run("returns empty slice when no workflows exist", func(t *testing.T) {
		ResetEmbeddedFS()
		defer ResetEmbeddedFS()

		// Isolate from real ~/.brains/workflows/ which may exist on the test machine.
		t.Setenv("HOME", t.TempDir())

		tempDir := t.TempDir()
		svc, err := NewService(tempDir)
		require.NoError(t, err)

		workflows, err := svc.List()
		require.NoError(t, err)
		assert.Empty(t, workflows)
	})
}
