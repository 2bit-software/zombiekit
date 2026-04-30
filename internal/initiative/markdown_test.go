package initiative

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInitiativeMD(t *testing.T) {
	t.Run("parses initiative with steps", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-01-31T10:00:00-08:00
**ID**: abc12345-feature-user-auth

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:30 |
| plan | in_progress | 2026-01-31 11:00 |
| tasks | pending | - |
| implement | pending | - |

## Description

User authentication feature.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		assert.Equal(t, "user-auth", parsed.Name)
		assert.Equal(t, "feature", parsed.Type)
		assert.Equal(t, "in_progress", parsed.Status)

		require.Len(t, parsed.Steps, 4)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
		assert.Equal(t, "plan", parsed.Steps[1].Name)
		assert.Equal(t, StepInProgress, parsed.Steps[1].Status)
		assert.Equal(t, "tasks", parsed.Steps[2].Name)
		assert.Equal(t, StepPending, parsed.Steps[2].Status)
	})

	t.Run("parses legacy cycle format (backwards compat)", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		// Legacy format with ### cycle header - the parser ignores these now
		// but still parses the step table
		content := `# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-01-31

## Cycles

### 1. feat/user-auth (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:30 |
| plan | in_progress | 2026-01-31 11:00 |
| tasks | pending | - |
| implement | pending | - |

## Description

User auth.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		// Should still parse steps (cycle header is ignored)
		require.Len(t, parsed.Steps, 4)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
	})

	t.Run("handles malformed table gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: test

**Type**: bug

## Steps

| Step | Status | Updated |
|------|--------|---------|
| investigate | completed | 2026-01-31 |
some random text here
| fix | pending | - |

## Description
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		// Should parse the valid rows
		require.Len(t, parsed.Steps, 1) // Only "investigate" parsed before malformed line
	})

	t.Run("handles skipped status", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: quick-fix

**Type**: feature

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:00 |
| plan | skipped | 2026-01-31 10:05 |
| tasks | skipped | 2026-01-31 10:05 |
| implement | in_progress | 2026-01-31 10:10 |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 4)
		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
		assert.Equal(t, StepSkipped, parsed.Steps[1].Status)
		assert.Equal(t, StepSkipped, parsed.Steps[2].Status)
		assert.Equal(t, StepInProgress, parsed.Steps[3].Status)
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := ParseInitiativeMD("/nonexistent/path/INITIATIVE.md")
		assert.Error(t, err)
	})
}

func TestParsedInitiative_CurrentStep(t *testing.T) {
	t.Run("returns in_progress step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "plan", Status: StepInProgress},
				{Name: "tasks", Status: StepPending},
			},
		}

		current := parsed.CurrentStep()
		require.NotNil(t, current)
		assert.Equal(t, "plan", current.Name)
		assert.Equal(t, StepInProgress, current.Status)
	})

	t.Run("returns nil when no in_progress step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "plan", Status: StepPending},
			},
		}

		current := parsed.CurrentStep()
		assert.Nil(t, current)
	})

	t.Run("returns nil for empty steps", func(t *testing.T) {
		parsed := &ParsedInitiative{}
		current := parsed.CurrentStep()
		assert.Nil(t, current)
	})
}

func TestParsedInitiative_NextStep(t *testing.T) {
	t.Run("returns next pending after in_progress", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "plan", Status: StepInProgress},
				{Name: "tasks", Status: StepPending},
				{Name: "implement", Status: StepPending},
			},
		}

		next := parsed.NextStep()
		require.NotNil(t, next)
		assert.Equal(t, "tasks", next.Name)
	})

	t.Run("returns first pending when no in_progress", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "plan", Status: StepCompleted},
				{Name: "tasks", Status: StepPending},
				{Name: "implement", Status: StepPending},
			},
		}

		next := parsed.NextStep()
		require.NotNil(t, next)
		assert.Equal(t, "tasks", next.Name)
	})

	t.Run("returns nil when all steps completed", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "plan", Status: StepCompleted},
				{Name: "tasks", Status: StepSkipped},
				{Name: "implement", Status: StepCompleted},
			},
		}

		next := parsed.NextStep()
		assert.Nil(t, next)
	})

	t.Run("returns nil for empty steps", func(t *testing.T) {
		parsed := &ParsedInitiative{}
		next := parsed.NextStep()
		assert.Nil(t, next)
	})
}

func TestParsedInitiative_UpdateStepStatus(t *testing.T) {
	t.Run("updates step status and timestamp", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepPending, Updated: "-"},
				{Name: "plan", Status: StepPending, Updated: "-"},
			},
		}

		err := parsed.UpdateStepStatus("spec", StepCompleted, "2026-01-31 10:00")
		require.NoError(t, err)

		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
		assert.Equal(t, "2026-01-31 10:00", parsed.Steps[0].Updated)
	})

	t.Run("returns error for nonexistent step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepPending},
			},
		}

		err := parsed.UpdateStepStatus("nonexistent", StepCompleted, "2026-01-31")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step not found")
	})
}

func TestParsedInitiative_AddStep(t *testing.T) {
	t.Run("adds step after specified step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted},
				{Name: "implement", Status: StepPending},
			},
		}

		newStep := ParsedStep{Name: "plan", Status: StepPending, Updated: "-"}
		err := parsed.AddStep("spec", newStep)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 3)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, "plan", parsed.Steps[1].Name)
		assert.Equal(t, "implement", parsed.Steps[2].Name)
	})

	t.Run("adds step at beginning when afterStep is empty", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "plan", Status: StepPending},
				{Name: "implement", Status: StepPending},
			},
		}

		newStep := ParsedStep{Name: "spec", Status: StepPending, Updated: "-"}
		err := parsed.AddStep("", newStep)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 3)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, "plan", parsed.Steps[1].Name)
		assert.Equal(t, "implement", parsed.Steps[2].Name)
	})

	t.Run("returns error for nonexistent afterStep", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepPending},
			},
		}

		newStep := ParsedStep{Name: "test", Status: StepPending}
		err := parsed.AddStep("nonexistent", newStep)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step not found")
	})
}

func TestParsedInitiative_WriteTo(t *testing.T) {
	t.Run("writes updated steps to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: test

**Type**: feature
**Status**: in_progress

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | pending | - |
| plan | pending | - |

## Description

Test initiative.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		// Parse, modify, and write
		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		err = parsed.UpdateStepStatus("spec", StepCompleted, "2026-01-31 10:00")
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Read back and verify
		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		assert.Equal(t, StepCompleted, parsed2.Steps[0].Status)
		assert.Equal(t, "2026-01-31 10:00", parsed2.Steps[0].Updated)
	})

	t.Run("preserves non-step sections", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: test

**Type**: feature
**Status**: in_progress

## Source

Some source info here.

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | pending | - |

## Description

Important description.

## Notes

Some notes.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Read the file and check sections are preserved
		content, err := os.ReadFile(mdPath)
		require.NoError(t, err)

		s := string(content)
		assert.Contains(t, s, "## Source")
		assert.Contains(t, s, "Some source info here")
		assert.Contains(t, s, "## Description")
		assert.Contains(t, s, "Important description")
		assert.Contains(t, s, "## Notes")
		assert.Contains(t, s, "Some notes")
	})

	t.Run("converts legacy cycles section to steps", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: legacy

**Type**: feature

## Cycles

### 1. feat/legacy (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-30 |
| plan | in_progress | 2026-01-31 |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		// Update a step
		err = parsed.UpdateStepStatus("plan", StepCompleted, "2026-01-31 12:00")
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Verify it now has ## Steps (not ## Cycles)
		content, err := os.ReadFile(mdPath)
		require.NoError(t, err)

		s := string(content)
		assert.Contains(t, s, "## Steps")
		assert.NotContains(t, s, "## Cycles")
		assert.NotContains(t, s, "### 1. feat/legacy")

		// Verify steps preserved
		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)
		require.Len(t, parsed2.Steps, 2)
		assert.Equal(t, StepCompleted, parsed2.Steps[1].Status)
	})
}

func TestParseInitiativeMD_FourColumnTable(t *testing.T) {
	t.Run("parses 4-column table with profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: data-driven

**Type**: refactor
**Status**: in_progress

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|---------|
| spec | feature | completed | 2026-04-30 10:00 |
| plan | plan | in_progress | 2026-04-30 11:00 |
| tasks | tasks | pending | - |
| implement | implement | pending | - |

## Description

Test.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 4)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, "feature", parsed.Steps[0].Profile)
		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
		assert.Equal(t, "2026-04-30 10:00", parsed.Steps[0].Updated)

		assert.Equal(t, "plan", parsed.Steps[1].Name)
		assert.Equal(t, "plan", parsed.Steps[1].Profile)
		assert.Equal(t, StepInProgress, parsed.Steps[1].Status)

		assert.Equal(t, "tasks", parsed.Steps[2].Name)
		assert.Equal(t, "tasks", parsed.Steps[2].Profile)
		assert.Equal(t, StepPending, parsed.Steps[2].Status)
	})

	t.Run("parses 4-column with non-matching step/profile names", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: bug-fix

**Type**: bug

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|---------|
| investigate | bug | completed | 2026-04-30 |
| fix | implement | in_progress | 2026-04-30 |
| verify | audit | pending | - |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 3)
		assert.Equal(t, "investigate", parsed.Steps[0].Name)
		assert.Equal(t, "bug", parsed.Steps[0].Profile)
		assert.Equal(t, "fix", parsed.Steps[1].Name)
		assert.Equal(t, "implement", parsed.Steps[1].Profile)
		assert.Equal(t, "verify", parsed.Steps[2].Name)
		assert.Equal(t, "audit", parsed.Steps[2].Profile)
	})

	t.Run("parses 4-column with comma-separated profiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: auto-feature

**Type**: feature

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|---------|
| implement | implement,automode | in_progress | 2026-04-30 |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 1)
		assert.Equal(t, "implement", parsed.Steps[0].Name)
		assert.Equal(t, "implement,automode", parsed.Steps[0].Profile)
	})

	t.Run("3-column table still parses with empty profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: legacy

**Type**: feature

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 |
| plan | pending | - |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Steps, 2)
		assert.Equal(t, "spec", parsed.Steps[0].Name)
		assert.Equal(t, "", parsed.Steps[0].Profile)
		assert.Equal(t, StepCompleted, parsed.Steps[0].Status)
	})
}

func TestParsedInitiative_FormatSteps(t *testing.T) {
	t.Run("formats 4-column when profiles present", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Profile: "feature", Status: StepCompleted, Updated: "2026-04-30"},
				{Name: "plan", Profile: "plan", Status: StepPending, Updated: "-"},
			},
		}

		lines := parsed.formatSteps()
		assert.Contains(t, lines[0], "| Step | Profile | Status | Updated |")
		assert.Contains(t, lines[2], "| spec | feature | completed | 2026-04-30 |")
		assert.Contains(t, lines[3], "| plan | plan | pending | - |")
	})

	t.Run("formats 3-column when no profiles present", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Steps: []ParsedStep{
				{Name: "spec", Status: StepCompleted, Updated: "2026-04-30"},
				{Name: "plan", Status: StepPending, Updated: "-"},
			},
		}

		lines := parsed.formatSteps()
		assert.Contains(t, lines[0], "| Step | Status | Updated |")
		assert.NotContains(t, lines[0], "Profile")
		assert.Contains(t, lines[2], "| spec | completed | 2026-04-30 |")
	})
}

func TestParsedInitiative_RoundTrip(t *testing.T) {
	t.Run("4-column round trip preserves profiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: roundtrip

**Type**: feature
**Status**: in_progress

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|---------|
| spec | feature | completed | 2026-04-30 10:00 |
| plan | plan | in_progress | 2026-04-30 11:00 |
| tasks | tasks | pending | - |

## Description

Round trip test.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		// Modify a step
		err = parsed.UpdateStepStatus("plan", StepCompleted, "2026-04-30 12:00")
		require.NoError(t, err)

		// Write back
		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Re-parse
		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed2.Steps, 3)
		// Profile preserved
		assert.Equal(t, "feature", parsed2.Steps[0].Profile)
		assert.Equal(t, "plan", parsed2.Steps[1].Profile)
		assert.Equal(t, "tasks", parsed2.Steps[2].Profile)
		// Status updated
		assert.Equal(t, StepCompleted, parsed2.Steps[1].Status)
		assert.Equal(t, "2026-04-30 12:00", parsed2.Steps[1].Updated)
	})

	t.Run("3-column round trip still works", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: legacy

**Type**: feature
**Status**: in_progress

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 |
| plan | pending | - |

## Description

Legacy format.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		err = parsed.UpdateStepStatus("plan", StepInProgress, "2026-01-31 12:00")
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed2.Steps, 2)
		assert.Equal(t, "", parsed2.Steps[0].Profile)
		assert.Equal(t, StepInProgress, parsed2.Steps[1].Status)
	})
}

func TestParseStepStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected StepStatus
	}{
		{"pending", StepPending},
		{"Pending", StepPending},
		{"PENDING", StepPending},
		{"in_progress", StepInProgress},
		{"in-progress", StepInProgress},
		{"completed", StepCompleted},
		{"complete", StepCompleted},
		{"skipped", StepSkipped},
		{"unknown", StepPending}, // defaults to pending
		{"", StepPending},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseStepStatus(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
