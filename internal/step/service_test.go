package step

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, svc)
	})
}

func TestService_GetStep(t *testing.T) {
	t.Run("returns step from loader", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Create specification
profiles:
  - research
---
Create the spec.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		step, err := svc.GetStep("specify")
		require.NoError(t, err)
		assert.Equal(t, "specify", step.Name)
		assert.Equal(t, "Create specification", step.Description)
	})

	t.Run("returns error for unknown step", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc, err := NewService(tmpDir)
		require.NoError(t, err)

		_, err = svc.GetStep("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})
}

func TestService_Execute(t *testing.T) {
	t.Run("returns step response with all fields", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder structure
		historyDir := filepath.Join(tmpDir, "history", "675d8a3f-feature-test")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte("# Spec"), 0644))

		// Create .brains directory with active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{
			"initiative": "history/675d8a3f-feature-test",
			"type": "feature",
			"name": "test",
			"status": "active"
		}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Create specification
profiles: []
files:
  - "spec.md"
---
Create the specification document.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("specify", nil)
		require.NoError(t, err)

		assert.Contains(t, resp.Directive, "Create the specification document")
		assert.Equal(t, historyDir, resp.HistoryFolder)
		assert.NotEmpty(t, resp.FilesToRead)
	})

	t.Run("returns error when no active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory without active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
---
Directive`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.Execute("specify", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NO_ACTIVE_INITIATIVE")
	})

	t.Run("allows init step without active initiative", func(t *testing.T) {
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

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		// Init step requires type and name parameters
		opts := &ExecuteOptions{
			Type: "feature",
			Name: "test-feature",
		}
		resp, err := svc.Execute("init", opts)
		require.NoError(t, err)

		assert.Contains(t, resp.Directive, "Initialize a new initiative")
		assert.NotEmpty(t, resp.HistoryFolder)
	})

	t.Run("resolves file glob patterns", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with files
		historyDir := filepath.Join(tmpDir, "history", "675d8a3f-feature-test")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte("# Spec"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "plan.md"), []byte("# Plan"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "tasks.md"), []byte("# Tasks"), 0644))

		// Create .brains with active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/675d8a3f-feature-test", "status": "active"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
files:
  - "*.md"
---
Create the plan.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("plan", nil)
		require.NoError(t, err)

		// Should resolve glob and find all .md files
		assert.GreaterOrEqual(t, len(resp.FilesToRead), 3)
	})

	t.Run("uses initiative override when provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create two history folders
		historyDir1 := filepath.Join(tmpDir, "history", "first-feature")
		historyDir2 := filepath.Join(tmpDir, "history", "second-feature")
		require.NoError(t, os.MkdirAll(historyDir1, 0755))
		require.NoError(t, os.MkdirAll(historyDir2, 0755))

		// Active initiative is first, but we'll override to second
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/first-feature", "status": "active"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
---
Directive`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		// Override to second initiative
		opts := &ExecuteOptions{
			Initiative: "second-feature",
		}
		resp, err := svc.Execute("specify", opts)
		require.NoError(t, err)

		assert.Equal(t, historyDir2, resp.HistoryFolder)
	})
}

func TestService_ListSteps(t *testing.T) {
	t.Run("returns all available steps", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/init.md": &fstest.MapFile{
				Data: []byte(`---
name: init
---
Init`),
			},
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
---
Specify`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		steps, err := svc.ListSteps()
		require.NoError(t, err)

		names := make(map[string]bool)
		for _, s := range steps {
			names[s.Name] = true
		}

		assert.True(t, names["init"])
		assert.True(t, names["specify"])
	})
}
