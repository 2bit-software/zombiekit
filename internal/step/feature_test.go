package step

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/initiative"
)

func TestExecuteFeatureStep_NewInitiative(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	// Create mock embedded filesystem with templates
	mockFS := fstest.MapFS{
		"steps/feature.md": &fstest.MapFile{
			Data: []byte(`---
name: feature
description: Feature step test
type: step
---
# Feature Step Directive
`),
		},
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte(`# Spec Template`),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte(`# Research Template`),
		},
	}

	svc, err := NewService(tmpDir)
	require.NoError(t, err)
	svc.SetEmbeddedFS(mockFS)

	// Execute feature step
	resp, err := svc.Execute("feature", &ExecuteOptions{
		Name: "test-feature",
		Type: "feature",
	})
	require.NoError(t, err)

	// Verify response
	assert.NotEmpty(t, resp.Directive)
	assert.NotEmpty(t, resp.InitiativeFolder)
	assert.NotEmpty(t, resp.CycleFolder)
	assert.Contains(t, resp.InitiativeFolder, "history")
	assert.Contains(t, resp.CycleFolder, "feat-test-feature")
	assert.Equal(t, resp.HistoryFolder, resp.CycleFolder) // Backward compat

	// Verify workflow phases
	assert.Len(t, resp.WorkflowPhases, 4)
	assert.Equal(t, "research", resp.WorkflowPhases[0].Name)
	assert.Equal(t, "create", resp.WorkflowPhases[1].Name)
	assert.Equal(t, "audit", resp.WorkflowPhases[2].Name)
	assert.Equal(t, "highlight", resp.WorkflowPhases[3].Name)

	// Verify folder structure
	_, err = os.Stat(resp.CycleFolder)
	assert.NoError(t, err)

	// Verify templates were copied
	specPath := filepath.Join(resp.CycleFolder, "spec.md")
	_, err = os.Stat(specPath)
	assert.NoError(t, err)

	researchPath := filepath.Join(resp.CycleFolder, "research.md")
	_, err = os.Stat(researchPath)
	assert.NoError(t, err)

	// Verify audit folder was created
	auditPath := filepath.Join(resp.CycleFolder, "audit")
	_, err = os.Stat(auditPath)
	assert.NoError(t, err)

	// Verify INITIATIVE.md was created with frontmatter
	initMDPath := filepath.Join(resp.InitiativeFolder, "INITIATIVE.md")
	content, err := os.ReadFile(initMDPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "status: active")
	assert.Contains(t, string(content), "type: feature")
}

func TestExecuteFeatureStep_AddCycleToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	mockFS := fstest.MapFS{
		"steps/feature.md": &fstest.MapFile{
			Data: []byte(`---
name: feature
type: step
---
# Feature Step
`),
		},
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte(`# Spec`),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte(`# Research`),
		},
	}

	svc, err := NewService(tmpDir)
	require.NoError(t, err)
	svc.SetEmbeddedFS(mockFS)

	// Create first initiative
	resp1, err := svc.Execute("feature", &ExecuteOptions{
		Name: "first-feature",
		Type: "feature",
	})
	require.NoError(t, err)
	initFolder1 := resp1.InitiativeFolder

	// Add a second cycle to the existing initiative
	resp2, err := svc.Execute("feature", &ExecuteOptions{
		Name: "refactor-first",
		Type: "refactor",
	})
	require.NoError(t, err)

	// Should be in the same initiative folder
	assert.Equal(t, initFolder1, resp2.InitiativeFolder)

	// But different cycle folder
	assert.NotEqual(t, resp1.CycleFolder, resp2.CycleFolder)
	assert.Contains(t, resp2.CycleFolder, "ref-refactor-first")
}

func TestExecuteFeatureStep_NewInitiativeFlag(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	mockFS := fstest.MapFS{
		"steps/feature.md": &fstest.MapFile{
			Data: []byte(`---
name: feature
type: step
---
# Feature Step
`),
		},
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte(`# Spec`),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte(`# Research`),
		},
	}

	svc, err := NewService(tmpDir)
	require.NoError(t, err)
	svc.SetEmbeddedFS(mockFS)

	// Create first initiative
	resp1, err := svc.Execute("feature", &ExecuteOptions{
		Name: "first-feature",
		Type: "feature",
	})
	require.NoError(t, err)
	initFolder1 := resp1.InitiativeFolder

	// Create new initiative with new_initiative flag
	resp2, err := svc.Execute("feature", &ExecuteOptions{
		Name:          "second-feature",
		Type:          "feature",
		NewInitiative: true,
	})
	require.NoError(t, err)

	// Should be a different initiative folder
	assert.NotEqual(t, initFolder1, resp2.InitiativeFolder)
}

func TestExecuteFeatureStep_ValidationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains"), 0755))

	mockFS := fstest.MapFS{
		"steps/feature.md": &fstest.MapFile{
			Data: []byte(`---
name: feature
type: step
---
# Feature Step
`),
		},
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte(`# Spec`),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte(`# Research`),
		},
	}

	svc, err := NewService(tmpDir)
	require.NoError(t, err)
	svc.SetEmbeddedFS(mockFS)

	t.Run("missing name", func(t *testing.T) {
		_, err := svc.Execute("feature", &ExecuteOptions{
			Type: "feature",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MISSING_NAME")
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := svc.Execute("feature", &ExecuteOptions{
			Name: "test",
			Type: "invalid",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_TYPE")
	})
}

func TestResolveTemplatePath_LocalOverride(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".brains", "templates"), 0755))

	// Create local override template
	localTemplatePath := filepath.Join(tmpDir, ".brains", "templates", "spec-template.md")
	require.NoError(t, os.WriteFile(localTemplatePath, []byte("# Local Override Spec"), 0644))

	// Create embedded template
	mockFS := fstest.MapFS{
		"steps/feature.md": &fstest.MapFile{
			Data: []byte(`---
name: feature
type: step
---
# Feature Step
`),
		},
		"templates/spec-template.md": &fstest.MapFile{
			Data: []byte(`# Embedded Spec`),
		},
		"templates/research-template.md": &fstest.MapFile{
			Data: []byte(`# Research`),
		},
	}

	svc, err := NewService(tmpDir)
	require.NoError(t, err)
	svc.SetEmbeddedFS(mockFS)

	// Execute feature step
	resp, err := svc.Execute("feature", &ExecuteOptions{
		Name: "test-feature",
		Type: "feature",
	})
	require.NoError(t, err)

	// Verify local override was used
	specPath := filepath.Join(resp.CycleFolder, "spec.md")
	content, err := os.ReadFile(specPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Local Override Spec")
}

func TestMapInitTypeToCycleType(t *testing.T) {
	tests := []struct {
		initType  initiative.InitiativeType
		cycleType initiative.CycleType
	}{
		{initiative.TypeFeature, initiative.CycleFeat},
		{initiative.TypeRefactor, initiative.CycleRef},
		{initiative.TypeBug, initiative.CycleFix},
	}

	for _, tc := range tests {
		t.Run(string(tc.initType), func(t *testing.T) {
			result := mapInitTypeToCycleType(tc.initType)
			assert.Equal(t, tc.cycleType, result)
		})
	}
}

func TestBuildWorkflowPhases(t *testing.T) {
	phases := buildWorkflowPhases()

	assert.Len(t, phases, 4)

	// Research phase
	assert.Equal(t, "research", phases[0].Name)
	assert.True(t, phases[0].Parallel)
	assert.Contains(t, phases[0].Agents, "research-codebase")
	assert.Contains(t, phases[0].Agents, "research-domain")

	// Create phase
	assert.Equal(t, "create", phases[1].Name)
	assert.False(t, phases[1].Parallel)

	// Audit phase
	assert.Equal(t, "audit", phases[2].Name)
	assert.True(t, phases[2].Parallel)

	// Highlight phase
	assert.Equal(t, "highlight", phases[3].Name)
	assert.False(t, phases[3].Parallel)
}
