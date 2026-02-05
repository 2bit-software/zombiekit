package initiative

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInitiativeMD(t *testing.T) {
	t.Run("parses single cycle initiative", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-01-31T10:00:00-08:00
**ID**: abc12345-feature-user-auth

## Cycles

### 1. feat/user-auth (active)

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

		require.Len(t, parsed.Cycles, 1)
		cycle := parsed.Cycles[0]
		assert.Equal(t, 1, cycle.Number)
		assert.Equal(t, "feat", cycle.Type)
		assert.Equal(t, "user-auth", cycle.Name)
		assert.Equal(t, "active", cycle.Status)

		require.Len(t, cycle.Steps, 4)
		assert.Equal(t, "spec", cycle.Steps[0].Name)
		assert.Equal(t, StepCompleted, cycle.Steps[0].Status)
		assert.Equal(t, "plan", cycle.Steps[1].Name)
		assert.Equal(t, StepInProgress, cycle.Steps[1].Status)
		assert.Equal(t, "tasks", cycle.Steps[2].Name)
		assert.Equal(t, StepPending, cycle.Steps[2].Status)
	})

	t.Run("parses multiple cycles", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-01-31

## Cycles

### 1. feat/user-auth (completed)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:30 |
| plan | completed | 2026-01-31 11:00 |
| tasks | completed | 2026-01-31 12:00 |
| implement | completed | 2026-01-31 14:00 |

### 2. ref/user-auth (active)

| Step | Status | Updated |
|------|--------|---------|
| analyze | in_progress | 2026-01-31 15:00 |
| plan | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

User auth with refactor.
`
		require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed.Cycles, 2)

		// First cycle (completed)
		assert.Equal(t, 1, parsed.Cycles[0].Number)
		assert.Equal(t, "completed", parsed.Cycles[0].Status)
		assert.Len(t, parsed.Cycles[0].Steps, 4)

		// Second cycle (active)
		assert.Equal(t, 2, parsed.Cycles[1].Number)
		assert.Equal(t, "active", parsed.Cycles[1].Status)
		assert.Len(t, parsed.Cycles[1].Steps, 4)
		assert.Equal(t, "analyze", parsed.Cycles[1].Steps[0].Name)
	})

	t.Run("handles malformed table gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: test

**Type**: bug

## Cycles

### 1. fix/test (active)

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
		require.Len(t, parsed.Cycles, 1)
		assert.Len(t, parsed.Cycles[0].Steps, 1) // Only "investigate" parsed before malformed line
	})

	t.Run("handles skipped status", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		content := `# Initiative: quick-fix

**Type**: feature

## Cycles

### 1. feat/quick-fix (active)

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

		require.Len(t, parsed.Cycles[0].Steps, 4)
		assert.Equal(t, StepCompleted, parsed.Cycles[0].Steps[0].Status)
		assert.Equal(t, StepSkipped, parsed.Cycles[0].Steps[1].Status)
		assert.Equal(t, StepSkipped, parsed.Cycles[0].Steps[2].Status)
		assert.Equal(t, StepInProgress, parsed.Cycles[0].Steps[3].Status)
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := ParseInitiativeMD("/nonexistent/path/INITIATIVE.md")
		assert.Error(t, err)
	})
}

func TestParsedInitiative_ActiveCycle(t *testing.T) {
	t.Run("returns active cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{Number: 1, Status: "completed"},
				{Number: 2, Status: "active"},
				{Number: 3, Status: "pending"},
			},
		}

		active := parsed.ActiveCycle()
		require.NotNil(t, active)
		assert.Equal(t, 2, active.Number)
	})

	t.Run("returns nil when no active cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{Number: 1, Status: "completed"},
				{Number: 2, Status: "completed"},
			},
		}

		active := parsed.ActiveCycle()
		assert.Nil(t, active)
	})

	t.Run("returns nil for empty cycles", func(t *testing.T) {
		parsed := &ParsedInitiative{}
		active := parsed.ActiveCycle()
		assert.Nil(t, active)
	})
}

func TestParsedInitiative_CurrentStep(t *testing.T) {
	t.Run("returns in_progress step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "plan", Status: StepInProgress},
						{Name: "tasks", Status: StepPending},
					},
				},
			},
		}

		current := parsed.CurrentStep()
		require.NotNil(t, current)
		assert.Equal(t, "plan", current.Name)
		assert.Equal(t, StepInProgress, current.Status)
	})

	t.Run("returns nil when no in_progress step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "plan", Status: StepPending},
					},
				},
			},
		}

		current := parsed.CurrentStep()
		assert.Nil(t, current)
	})

	t.Run("returns nil when no active cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "completed",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepInProgress},
					},
				},
			},
		}

		current := parsed.CurrentStep()
		assert.Nil(t, current)
	})
}

func TestParsedInitiative_NextStep(t *testing.T) {
	t.Run("returns next pending after in_progress", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "plan", Status: StepInProgress},
						{Name: "tasks", Status: StepPending},
						{Name: "implement", Status: StepPending},
					},
				},
			},
		}

		next := parsed.NextStep()
		require.NotNil(t, next)
		assert.Equal(t, "tasks", next.Name)
	})

	t.Run("returns first pending when no in_progress", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "plan", Status: StepCompleted},
						{Name: "tasks", Status: StepPending},
						{Name: "implement", Status: StepPending},
					},
				},
			},
		}

		next := parsed.NextStep()
		require.NotNil(t, next)
		assert.Equal(t, "tasks", next.Name)
	})

	t.Run("returns nil when all steps completed", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "plan", Status: StepCompleted},
						{Name: "tasks", Status: StepSkipped},
						{Name: "implement", Status: StepCompleted},
					},
				},
			},
		}

		next := parsed.NextStep()
		assert.Nil(t, next)
	})

	t.Run("returns nil when no active cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{Number: 1, Status: "completed"},
			},
		}

		next := parsed.NextStep()
		assert.Nil(t, next)
	})
}

func TestParsedInitiative_UpdateStepStatus(t *testing.T) {
	t.Run("updates step status and timestamp", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepPending, Updated: "-"},
						{Name: "plan", Status: StepPending, Updated: "-"},
					},
				},
			},
		}

		err := parsed.UpdateStepStatus(1, "spec", StepCompleted, "2026-01-31 10:00")
		require.NoError(t, err)

		assert.Equal(t, StepCompleted, parsed.Cycles[0].Steps[0].Status)
		assert.Equal(t, "2026-01-31 10:00", parsed.Cycles[0].Steps[0].Updated)
	})

	t.Run("returns error for nonexistent cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{Number: 1, Status: "active"},
			},
		}

		err := parsed.UpdateStepStatus(99, "spec", StepCompleted, "2026-01-31")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cycle not found")
	})

	t.Run("returns error for nonexistent step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepPending},
					},
				},
			},
		}

		err := parsed.UpdateStepStatus(1, "nonexistent", StepCompleted, "2026-01-31")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step not found")
	})
}

func TestParsedInitiative_AddStep(t *testing.T) {
	t.Run("adds step after specified step", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepCompleted},
						{Name: "implement", Status: StepPending},
					},
				},
			},
		}

		newStep := ParsedStep{Name: "plan", Status: StepPending, Updated: "-"}
		err := parsed.AddStep(1, "spec", newStep)
		require.NoError(t, err)

		require.Len(t, parsed.Cycles[0].Steps, 3)
		assert.Equal(t, "spec", parsed.Cycles[0].Steps[0].Name)
		assert.Equal(t, "plan", parsed.Cycles[0].Steps[1].Name)
		assert.Equal(t, "implement", parsed.Cycles[0].Steps[2].Name)
	})

	t.Run("adds step at beginning when afterStep is empty", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "plan", Status: StepPending},
						{Name: "implement", Status: StepPending},
					},
				},
			},
		}

		newStep := ParsedStep{Name: "spec", Status: StepPending, Updated: "-"}
		err := parsed.AddStep(1, "", newStep)
		require.NoError(t, err)

		require.Len(t, parsed.Cycles[0].Steps, 3)
		assert.Equal(t, "spec", parsed.Cycles[0].Steps[0].Name)
		assert.Equal(t, "plan", parsed.Cycles[0].Steps[1].Name)
		assert.Equal(t, "implement", parsed.Cycles[0].Steps[2].Name)
	})

	t.Run("returns error for nonexistent cycle", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{Number: 1, Status: "active"},
			},
		}

		newStep := ParsedStep{Name: "test", Status: StepPending}
		err := parsed.AddStep(99, "", newStep)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cycle not found")
	})

	t.Run("returns error for nonexistent afterStep", func(t *testing.T) {
		parsed := &ParsedInitiative{
			Cycles: []ParsedCycle{
				{
					Number: 1,
					Status: "active",
					Steps: []ParsedStep{
						{Name: "spec", Status: StepPending},
					},
				},
			},
		}

		newStep := ParsedStep{Name: "test", Status: StepPending}
		err := parsed.AddStep(1, "nonexistent", newStep)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "step not found")
	})
}

func TestParsedInitiative_WriteTo(t *testing.T) {
	t.Run("writes updated cycles to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: test

**Type**: feature
**Status**: in_progress

## Cycles

### 1. feat/test (active)

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

		err = parsed.UpdateStepStatus(1, "spec", StepCompleted, "2026-01-31 10:00")
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Read back and verify
		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		assert.Equal(t, StepCompleted, parsed2.Cycles[0].Steps[0].Status)
		assert.Equal(t, "2026-01-31 10:00", parsed2.Cycles[0].Steps[0].Updated)
	})

	t.Run("preserves non-cycle sections", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: test

**Type**: feature
**Status**: in_progress

## Source

Some source info here.

## Cycles

### 1. feat/test (active)

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

	t.Run("handles multiple cycles", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := filepath.Join(tmpDir, "INITIATIVE.md")

		original := `# Initiative: multi

**Type**: feature

## Cycles

### 1. feat/multi (completed)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-30 |

### 2. ref/multi (active)

| Step | Status | Updated |
|------|--------|---------|
| analyze | in_progress | 2026-01-31 |
`
		require.NoError(t, os.WriteFile(mdPath, []byte(original), 0644))

		parsed, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		// Update second cycle
		err = parsed.UpdateStepStatus(2, "analyze", StepCompleted, "2026-01-31 12:00")
		require.NoError(t, err)

		err = parsed.WriteTo(mdPath)
		require.NoError(t, err)

		// Verify both cycles preserved
		parsed2, err := ParseInitiativeMD(mdPath)
		require.NoError(t, err)

		require.Len(t, parsed2.Cycles, 2)
		assert.Equal(t, "completed", parsed2.Cycles[0].Status)
		assert.Equal(t, StepCompleted, parsed2.Cycles[1].Steps[0].Status)
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
