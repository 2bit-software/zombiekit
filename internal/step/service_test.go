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
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
description: Create implementation plan
profiles:
  - research
---
Create the plan.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		step, err := svc.GetStep("plan")
		require.NoError(t, err)
		assert.Equal(t, "plan", step.Name)
		assert.Equal(t, "Create implementation plan", step.Description)
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

		// Create history folder structure (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "675d8a3f-feature-test")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte(`---
status: approved
---
# Spec`), 0644))

		// Create .brains directory with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/675d8a3f-feature-test", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
description: Create implementation plan
profiles: []
files:
  - "spec.md"
---
Create the implementation plan.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("plan", nil)
		require.NoError(t, err)

		assert.Contains(t, resp.Directive, "Create the implementation plan")
		assert.Equal(t, historyDir, resp.HistoryFolder)
		assert.Equal(t, historyDir, resp.CycleFolder) // CycleFolder now equals HistoryFolder
		assert.NotEmpty(t, resp.FilesToRead)
	})

	t.Run("returns error when no active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory without active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		embeddedFS := fstest.MapFS{
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
---
Directive`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.Execute("plan", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NO_ACTIVE_INITIATIVE")
	})

	t.Run("feature step requires active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains directory without active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))

		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
description: Create new feature
---
Create a new feature specification.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		// Feature step without active initiative should fail
		_, err = svc.Execute("feature", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NO_ACTIVE_INITIATIVE")
	})

	t.Run("feature step works with active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder (no separate cycle folder - cycle tracking is now in INITIATIVE.md)
		historyDir := filepath.Join(tmpDir, "history", "675d8a3f-feature-test")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		// Create .brains with active initiative (new format without cycle field)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/675d8a3f-feature-test", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
description: Create new feature
---
Create a new feature specification.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("feature", nil)
		require.NoError(t, err)

		assert.Contains(t, resp.Directive, "Create a new feature specification")
		assert.Equal(t, historyDir, resp.InitiativeFolder)
		assert.Equal(t, historyDir, resp.CycleFolder) // CycleFolder now equals InitiativeFolder
		// Feature step should include workflow phases
		assert.NotEmpty(t, resp.WorkflowPhases)
	})

	t.Run("resolves file glob patterns", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with files (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "675d8a3f-feature-test")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte(`---
status: approved
---
# Spec`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "research.md"), []byte("# Research"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "notes.md"), []byte("# Notes"), 0644))

		// Create .brains with active initiative (new format without cycle field)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/675d8a3f-feature-test", "status": "in_progress"}`
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

		// Should resolve glob and find all .md files in initiative folder
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
			"steps/audit.md": &fstest.MapFile{
				Data: []byte(`---
name: audit
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
		resp, err := svc.Execute("audit", opts)
		require.NoError(t, err)

		assert.Equal(t, historyDir2, resp.HistoryFolder)
	})
}

func TestService_FeatureStep(t *testing.T) {
	t.Run("feature step returns workflow phases", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		// Create .brains directory with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
description: Create new feature
---
Create a new feature specification.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("feature", nil)
		require.NoError(t, err)

		// Verify response includes workflow phases
		assert.NotEmpty(t, resp.WorkflowPhases)
		assert.Equal(t, historyDir, resp.InitiativeFolder)
		assert.Equal(t, historyDir, resp.CycleFolder) // CycleFolder now equals InitiativeFolder
		assert.Contains(t, resp.Directive, "Create a new feature specification")
	})
}

func TestService_ListSteps(t *testing.T) {
	t.Run("returns all available steps", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
---
Feature`),
			},
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
---
Plan`),
			},
			"steps/implement.md": &fstest.MapFile{
				Data: []byte(`---
name: implement
---
Implement`),
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

		assert.True(t, names["feature"])
		assert.True(t, names["plan"])
		assert.True(t, names["implement"])
	})
}

func TestService_BugStep(t *testing.T) {
	t.Run("bug step returns workflow phases with active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-bug")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		// Create .brains directory with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-bug", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/bug.md": &fstest.MapFile{
				Data: []byte(`---
name: bug
description: Bug investigation
---
Investigate and fix the bug.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("bug", nil)
		require.NoError(t, err)

		assert.Equal(t, historyDir, resp.InitiativeFolder)
		assert.Equal(t, historyDir, resp.CycleFolder) // CycleFolder now equals InitiativeFolder
		assert.Contains(t, resp.Directive, "Investigate and fix the bug")
		// Bug step should include workflow phases
		assert.NotEmpty(t, resp.WorkflowPhases)
	})
}

func TestService_RefactorStep(t *testing.T) {
	t.Run("refactor step returns workflow phases with active initiative", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-refactor")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		// Create .brains directory with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-refactor", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/refactor.md": &fstest.MapFile{
				Data: []byte(`---
name: refactor
description: Refactoring specification
---
Refactor the code with behavior preservation.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("refactor", nil)
		require.NoError(t, err)

		assert.Equal(t, historyDir, resp.InitiativeFolder)
		assert.Equal(t, historyDir, resp.CycleFolder) // CycleFolder now equals InitiativeFolder
		assert.Contains(t, resp.Directive, "Refactor the code with behavior preservation")
		// Refactor step should include workflow phases
		assert.NotEmpty(t, resp.WorkflowPhases)
	})
}

func TestService_PrerequisiteEnforcement(t *testing.T) {
	t.Run("plan step blocks when spec.md not approved", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with draft spec (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte(`---
status: draft
---
# Spec`), 0644))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
---
Create the plan.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.Execute("plan", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PREREQUISITE_NOT_MET")
	})

	t.Run("plan step allows when spec.md is approved", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with approved spec (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "spec.md"), []byte(`---
status: approved
---
# Spec`), 0644))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/plan.md": &fstest.MapFile{
				Data: []byte(`---
name: plan
---
Create the plan.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("plan", nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("tasks step blocks when plan.md not approved", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with draft plan (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "plan.md"), []byte(`---
status: draft
---
# Plan`), 0644))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/tasks.md": &fstest.MapFile{
				Data: []byte(`---
name: tasks
---
Generate tasks.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.Execute("tasks", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PREREQUISITE_NOT_MET")
	})

	t.Run("tasks step allows when plan.md is approved", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with approved plan (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "plan.md"), []byte(`---
status: approved
---
# Plan`), 0644))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/tasks.md": &fstest.MapFile{
				Data: []byte(`---
name: tasks
---
Generate tasks.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("tasks", nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("implement step blocks when tasks.md is missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder without tasks.md (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/implement.md": &fstest.MapFile{
				Data: []byte(`---
name: implement
---
Execute implementation.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.Execute("implement", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PREREQUISITE_NOT_MET")
	})

	t.Run("implement step allows when tasks.md exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create history folder with tasks.md (no separate cycle folder)
		historyDir := filepath.Join(tmpDir, "history", "test-feature")
		require.NoError(t, os.MkdirAll(historyDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(historyDir, "tasks.md"), []byte("# Tasks\n- [ ] Task 1"), 0644))

		// Create .brains with active initiative (new format)
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "in_progress"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		embeddedFS := fstest.MapFS{
			"steps/implement.md": &fstest.MapFile{
				Data: []byte(`---
name: implement
---
Execute implementation.`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		resp, err := svc.Execute("implement", nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestService_LegacySteps(t *testing.T) {
	t.Run("init step returns UNKNOWN_STEP error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Empty embedded FS - no legacy steps
		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
---
Feature`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.GetStep("init")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})

	t.Run("specify step returns UNKNOWN_STEP error", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
---
Feature`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.GetStep("specify")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})

	t.Run("eat step returns UNKNOWN_STEP error (replaced by implement)", func(t *testing.T) {
		tmpDir := t.TempDir()

		embeddedFS := fstest.MapFS{
			"steps/implement.md": &fstest.MapFile{
				Data: []byte(`---
name: implement
---
Implement`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		_, err = svc.GetStep("eat")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})
}

func TestService_CompleteStep(t *testing.T) {
	t.Run("complete step returns UNKNOWN_STEP (now initiative action)", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .brains with active initiative
		brainsDir := filepath.Join(tmpDir, ".brains")
		require.NoError(t, os.MkdirAll(brainsDir, 0755))
		activeJSON := `{"initiative": "history/test-feature", "status": "active"}`
		require.NoError(t, os.WriteFile(filepath.Join(brainsDir, "active.json"), []byte(activeJSON), 0644))

		// Empty embedded FS - no complete step (it's been removed)
		embeddedFS := fstest.MapFS{
			"steps/feature.md": &fstest.MapFile{
				Data: []byte(`---
name: feature
---
Feature`),
			},
		}

		svc, err := NewService(tmpDir)
		require.NoError(t, err)
		svc.SetEmbeddedFS(embeddedFS)

		// complete is now an initiative action, not a step
		_, err = svc.GetStep("complete")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})
}
