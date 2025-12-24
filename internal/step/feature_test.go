package step

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
