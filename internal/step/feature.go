package step

// buildWorkflowPhases returns the workflow phase definitions for the feature step.
func buildWorkflowPhases() []Phase {
	return []Phase{
		{
			Name:        "research",
			Description: "Gather context and domain knowledge through parallel research agents",
			Agents:      []string{"research-codebase", "research-domain"},
			Outputs:     []string{"research.md"},
			Parallel:    true,
		},
		{
			Name:        "create",
			Description: "Synthesize specification from research findings",
			Agents:      []string{"spec-writer"},
			Outputs:     []string{"spec.md"},
			Parallel:    false,
		},
		{
			Name:        "audit",
			Description: "Check specification quality and completeness with severity classification",
			Agents:      []string{"audit-completeness", "audit-ai-readiness"},
			Outputs:     []string{"audit/{date}.md"},
			Parallel:    true,
		},
		{
			Name:        "highlight",
			Description: "Present key decisions for user approval before proceeding",
			Agents:      []string{"highlighter"},
			Outputs:     []string{},
			Parallel:    false,
		},
	}
}
