package step

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	t.Run("creates loader with explicit working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)
		require.NotNil(t, loader)
		assert.Equal(t, tmpDir, loader.workDir)
	})

	t.Run("uses current directory if empty", func(t *testing.T) {
		loader := NewLoader("")
		require.NotNil(t, loader)
		cwd, _ := os.Getwd()
		assert.Equal(t, cwd, loader.workDir)
	})
}

func TestLoader_Get(t *testing.T) {
	t.Run("returns embedded step when no custom exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create embedded FS with a test step
		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Create specification
profiles:
  - research
files:
  - "spec.md"
---
Create the specification.`),
			},
		}

		loader := NewLoader(tmpDir)
		loader.SetEmbeddedFS(embeddedFS)

		step, err := loader.Get("specify")
		require.NoError(t, err)
		assert.Equal(t, "specify", step.Name)
		assert.Equal(t, "Create specification", step.Description)
		assert.Equal(t, SourceEmbedded, step.Source)
		assert.Contains(t, step.Directive, "Create the specification")
	})

	t.Run("returns local step when it exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create local step
		localStepsDir := filepath.Join(tmpDir, ".brains", "steps")
		require.NoError(t, os.MkdirAll(localStepsDir, 0755))

		stepContent := `---
name: specify
description: Custom local spec
profiles:
  - custom-profile
---
Custom local directive.`
		require.NoError(t, os.WriteFile(filepath.Join(localStepsDir, "specify.md"), []byte(stepContent), 0644))

		// Also provide embedded fallback
		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Embedded spec
---
Embedded directive.`),
			},
		}

		loader := NewLoader(tmpDir)
		loader.SetEmbeddedFS(embeddedFS)

		step, err := loader.Get("specify")
		require.NoError(t, err)
		assert.Equal(t, "specify", step.Name)
		assert.Equal(t, "Custom local spec", step.Description)
		assert.Equal(t, SourceLocal, step.Source)
		assert.Contains(t, step.Directive, "Custom local directive")
	})

	t.Run("returns global step when no local exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		globalDir := t.TempDir()

		// Create global step
		globalStepsDir := filepath.Join(globalDir, "steps")
		require.NoError(t, os.MkdirAll(globalStepsDir, 0755))

		stepContent := `---
name: specify
description: Global spec
---
Global directive.`
		require.NoError(t, os.WriteFile(filepath.Join(globalStepsDir, "specify.md"), []byte(stepContent), 0644))

		loader := NewLoader(tmpDir)
		loader.SetGlobalDir(globalDir)

		step, err := loader.Get("specify")
		require.NoError(t, err)
		assert.Equal(t, "specify", step.Name)
		assert.Equal(t, "Global spec", step.Description)
		assert.Equal(t, SourceGlobal, step.Source)
	})

	t.Run("precedence: local > global > embedded", func(t *testing.T) {
		tmpDir := t.TempDir()
		globalDir := t.TempDir()

		// Create all three versions
		localStepsDir := filepath.Join(tmpDir, ".brains", "steps")
		require.NoError(t, os.MkdirAll(localStepsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(localStepsDir, "test.md"), []byte(`---
name: test
description: Local version
---
Local`), 0644))

		globalStepsDir := filepath.Join(globalDir, "steps")
		require.NoError(t, os.MkdirAll(globalStepsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(globalStepsDir, "test.md"), []byte(`---
name: test
description: Global version
---
Global`), 0644))

		embeddedFS := fstest.MapFS{
			"steps/test.md": &fstest.MapFile{
				Data: []byte(`---
name: test
description: Embedded version
---
Embedded`),
			},
		}

		loader := NewLoader(tmpDir)
		loader.SetGlobalDir(globalDir)
		loader.SetEmbeddedFS(embeddedFS)

		step, err := loader.Get("test")
		require.NoError(t, err)
		assert.Equal(t, "Local version", step.Description)
		assert.Equal(t, SourceLocal, step.Source)
	})

	t.Run("returns error for unknown step", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		_, err := loader.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNKNOWN_STEP")
	})
}

func TestLoader_List(t *testing.T) {
	t.Run("returns all available steps from all sources", func(t *testing.T) {
		tmpDir := t.TempDir()
		globalDir := t.TempDir()

		// Create local step
		localStepsDir := filepath.Join(tmpDir, ".brains", "steps")
		require.NoError(t, os.MkdirAll(localStepsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(localStepsDir, "local-only.md"), []byte(`---
name: local-only
---
Local only step`), 0644))

		// Create global step
		globalStepsDir := filepath.Join(globalDir, "steps")
		require.NoError(t, os.MkdirAll(globalStepsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(globalStepsDir, "global-only.md"), []byte(`---
name: global-only
---
Global only step`), 0644))

		// Create embedded step
		embeddedFS := fstest.MapFS{
			"steps/embedded-only.md": &fstest.MapFile{
				Data: []byte(`---
name: embedded-only
---
Embedded only step`),
			},
		}

		loader := NewLoader(tmpDir)
		loader.SetGlobalDir(globalDir)
		loader.SetEmbeddedFS(embeddedFS)

		steps, err := loader.List()
		require.NoError(t, err)

		names := make(map[string]bool)
		for _, s := range steps {
			names[s.Name] = true
		}

		assert.True(t, names["local-only"], "should include local step")
		assert.True(t, names["global-only"], "should include global step")
		assert.True(t, names["embedded-only"], "should include embedded step")
	})

	t.Run("deduplicates steps by name with precedence", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create local step that shadows embedded
		localStepsDir := filepath.Join(tmpDir, ".brains", "steps")
		require.NoError(t, os.MkdirAll(localStepsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(localStepsDir, "specify.md"), []byte(`---
name: specify
description: Local override
---
Local directive`), 0644))

		embeddedFS := fstest.MapFS{
			"steps/specify.md": &fstest.MapFile{
				Data: []byte(`---
name: specify
description: Embedded default
---
Embedded directive`),
			},
		}

		loader := NewLoader(tmpDir)
		loader.SetEmbeddedFS(embeddedFS)

		steps, err := loader.List()
		require.NoError(t, err)

		// Should only have one "specify" step
		specifyCount := 0
		for _, s := range steps {
			if s.Name == "specify" {
				specifyCount++
				assert.Equal(t, "Local override", s.Description)
				assert.Equal(t, SourceLocal, s.Source)
			}
		}
		assert.Equal(t, 1, specifyCount)
	})
}

func TestLoader_EmbeddedFS(t *testing.T) {
	t.Run("SetEmbeddedFS and HasEmbeddedSteps", func(t *testing.T) {
		loader := NewLoader("")

		assert.False(t, loader.HasEmbeddedSteps())

		embeddedFS := fstest.MapFS{
			"steps/test.md": &fstest.MapFile{
				Data: []byte(`---
name: test
---
Test`),
			},
		}
		loader.SetEmbeddedFS(embeddedFS)

		assert.True(t, loader.HasEmbeddedSteps())
	})
}

func TestParseStep(t *testing.T) {
	t.Run("parses complete step definition", func(t *testing.T) {
		content := []byte(`---
name: specify
description: Create specification
profiles:
  - research
  - spec-creator
files:
  - "spec.md"
  - "**/*.md"
type: step
---
Your task is to create the specification.

Include all requirements.`)

		step, err := ParseStep(content, "fallback-name", "/path/to/step.md", SourceLocal)
		require.NoError(t, err)

		assert.Equal(t, "specify", step.Name)
		assert.Equal(t, "Create specification", step.Description)
		assert.Equal(t, []string{"research", "spec-creator"}, step.Profiles)
		assert.Equal(t, []string{"spec.md", "**/*.md"}, step.Files)
		assert.Equal(t, "step", step.Type)
		assert.Equal(t, SourceLocal, step.Source)
		assert.Equal(t, "/path/to/step.md", step.Path)
		assert.Contains(t, step.Directive, "Your task is to create the specification")
		assert.Contains(t, step.Directive, "Include all requirements")
	})

	t.Run("uses fallback name when not in frontmatter", func(t *testing.T) {
		content := []byte(`---
description: No name in frontmatter
---
Directive text`)

		step, err := ParseStep(content, "fallback-name", "/path/to/step.md", SourceGlobal)
		require.NoError(t, err)

		assert.Equal(t, "fallback-name", step.Name)
	})

	t.Run("handles no frontmatter", func(t *testing.T) {
		content := []byte(`Just a plain directive with no frontmatter.

Multiple lines are fine.`)

		step, err := ParseStep(content, "plain", "/path/to/plain.md", SourceEmbedded)
		require.NoError(t, err)

		assert.Equal(t, "plain", step.Name)
		assert.Contains(t, step.Directive, "Just a plain directive")
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		content := []byte(`---
name: test
invalid: [yaml: missing bracket
---
Directive`)

		_, err := ParseStep(content, "test", "/path/to/test.md", SourceLocal)
		assert.Error(t, err)
	})
}

// Helper to create an in-memory embedded FS for testing
func createTestEmbeddedFS(steps map[string]string) fs.FS {
	mapFS := make(fstest.MapFS)
	for name, content := range steps {
		mapFS["steps/"+name+".md"] = &fstest.MapFile{
			Data: []byte(content),
		}
	}
	return mapFS
}
