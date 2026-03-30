package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

// === Filter Tests ===

func TestFilterPromptsByCategory(t *testing.T) {
	prompts := []Prompt{
		{Name: "new", Category: CategoryWorkflow, Source: SourceEmbedded},
		{Name: "feature", Category: CategoryProfile, Source: SourceLocal},
		{Name: "plan", Category: CategoryStep, Source: SourceGlobal},
	}

	// Filter workflows only
	result := filterPrompts(prompts, FilterOptions{Category: "workflow"})
	assert.Len(t, result, 1)
	assert.Equal(t, "new", result[0].Name)

	// Filter profiles only
	result = filterPrompts(prompts, FilterOptions{Category: "profile"})
	assert.Len(t, result, 1)
	assert.Equal(t, "feature", result[0].Name)

	// Filter steps only
	result = filterPrompts(prompts, FilterOptions{Category: "step"})
	assert.Len(t, result, 1)
	assert.Equal(t, "plan", result[0].Name)

	// No filter returns all
	result = filterPrompts(prompts, FilterOptions{})
	assert.Len(t, result, 3)
}

func TestFilterPromptsBySource(t *testing.T) {
	prompts := []Prompt{
		{Name: "local-profile", Source: SourceLocal},
		{Name: "global-profile", Source: SourceGlobal},
		{Name: "embedded-profile", Source: SourceEmbedded},
	}

	// Filter local only
	result := filterPrompts(prompts, FilterOptions{Source: "local"})
	assert.Len(t, result, 1)
	assert.Equal(t, "local-profile", result[0].Name)

	// Filter global only
	result = filterPrompts(prompts, FilterOptions{Source: "global"})
	assert.Len(t, result, 1)
	assert.Equal(t, "global-profile", result[0].Name)

	// Filter embedded only
	result = filterPrompts(prompts, FilterOptions{Source: "embedded"})
	assert.Len(t, result, 1)
	assert.Equal(t, "embedded-profile", result[0].Name)
}

func TestFilterPromptsByQuery(t *testing.T) {
	prompts := []Prompt{
		{Name: "feature-dev", Description: "Development workflow"},
		{Name: "bug-fix", Description: "Bug fixing process"},
		{Name: "deploy", Description: "Deployment automation"},
	}

	// Search by name
	result := filterPrompts(prompts, FilterOptions{Query: "feat"})
	assert.Len(t, result, 1)
	assert.Equal(t, "feature-dev", result[0].Name)

	// Search by description
	result = filterPrompts(prompts, FilterOptions{Query: "automation"})
	assert.Len(t, result, 1)
	assert.Equal(t, "deploy", result[0].Name)

	// Case insensitive search
	result = filterPrompts(prompts, FilterOptions{Query: "BUG"})
	assert.Len(t, result, 1)
	assert.Equal(t, "bug-fix", result[0].Name)

	// No matches
	result = filterPrompts(prompts, FilterOptions{Query: "nonexistent"})
	assert.Empty(t, result)
}

func TestFilterPromptsCombined(t *testing.T) {
	prompts := []Prompt{
		{Name: "feature", Category: CategoryProfile, Source: SourceLocal, Description: "Feature profile"},
		{Name: "feature", Category: CategoryProfile, Source: SourceEmbedded, Description: "Embedded feature"},
		{Name: "feature", Category: CategoryStep, Source: SourceLocal, Description: "Feature step"},
	}

	// Filter by category AND source
	result := filterPrompts(prompts, FilterOptions{Category: "profile", Source: "local"})
	assert.Len(t, result, 1)
	assert.Equal(t, CategoryProfile, result[0].Category)
	assert.Equal(t, SourceLocal, result[0].Source)

	// Filter by category AND source AND query
	result = filterPrompts(prompts, FilterOptions{Category: "profile", Source: "embedded", Query: "embed"})
	assert.Len(t, result, 1)
	assert.Equal(t, SourceEmbedded, result[0].Source)
}

// === Sort Tests ===

func TestSortPromptsByName(t *testing.T) {
	prompts := []Prompt{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "middle"},
	}

	// Ascending
	sortPrompts(prompts, SortOptions{Field: "name", Order: "asc"})
	assert.Equal(t, "alpha", prompts[0].Name)
	assert.Equal(t, "middle", prompts[1].Name)
	assert.Equal(t, "zebra", prompts[2].Name)

	// Descending
	sortPrompts(prompts, SortOptions{Field: "name", Order: "desc"})
	assert.Equal(t, "zebra", prompts[0].Name)
	assert.Equal(t, "middle", prompts[1].Name)
	assert.Equal(t, "alpha", prompts[2].Name)
}

func TestSortPromptsByCategory(t *testing.T) {
	prompts := []Prompt{
		{Name: "step-a", Category: CategoryStep},
		{Name: "profile-a", Category: CategoryProfile},
		{Name: "workflow-a", Category: CategoryWorkflow},
		{Name: "profile-b", Category: CategoryProfile},
	}

	// Sort by category, then by name within category
	sortPrompts(prompts, SortOptions{Field: "category", Order: "asc"})

	// Profiles should come first (alphabetically)
	assert.Equal(t, CategoryProfile, prompts[0].Category)
	assert.Equal(t, CategoryProfile, prompts[1].Category)
	// Then step, then workflow
	assert.Equal(t, CategoryStep, prompts[2].Category)
	assert.Equal(t, CategoryWorkflow, prompts[3].Category)

	// Within profiles, should be sorted by name
	assert.Equal(t, "profile-a", prompts[0].Name)
	assert.Equal(t, "profile-b", prompts[1].Name)
}

func TestSortPromptsBySource(t *testing.T) {
	prompts := []Prompt{
		{Name: "local-b", Source: SourceLocal},
		{Name: "embedded-a", Source: SourceEmbedded},
		{Name: "global-a", Source: SourceGlobal},
		{Name: "local-a", Source: SourceLocal},
	}

	// Sort by source, then by name within source
	sortPrompts(prompts, SortOptions{Field: "source", Order: "asc"})

	// Embedded first (alphabetically before global, local)
	assert.Equal(t, SourceEmbedded, prompts[0].Source)
	assert.Equal(t, SourceGlobal, prompts[1].Source)
	// Then local sources, sorted by name
	assert.Equal(t, SourceLocal, prompts[2].Source)
	assert.Equal(t, SourceLocal, prompts[3].Source)
	assert.Equal(t, "local-a", prompts[2].Name)
	assert.Equal(t, "local-b", prompts[3].Name)
}

func TestSortPromptsDefaultsToName(t *testing.T) {
	prompts := []Prompt{
		{Name: "zebra"},
		{Name: "alpha"},
	}

	// Unknown field defaults to name
	sortPrompts(prompts, SortOptions{Field: "unknown", Order: "asc"})
	assert.Equal(t, "alpha", prompts[0].Name)
	assert.Equal(t, "zebra", prompts[1].Name)
}

// === Converter Tests ===

func TestConvertWorkflow(t *testing.T) {
	wf := &workflow.Workflow{
		Name:        "new",
		Description: "Start new work",
		Path:        "/path/to/new.md",
		Source:      "local",
		Content:     "# New Workflow\n\nContent here",
	}

	prompt := convertWorkflow(wf)

	assert.Equal(t, "new", prompt.Name)
	assert.Equal(t, CategoryWorkflow, prompt.Category)
	assert.Equal(t, SourceLocal, prompt.Source)
	assert.Equal(t, "Start new work", prompt.Description)
	assert.Equal(t, "/path/to/new.md", prompt.Path)
	assert.Empty(t, prompt.Content, "convertWorkflow should not include content")
}

func TestConvertWorkflowFull(t *testing.T) {
	wf := &workflow.Workflow{
		Name:        "new",
		Description: "Start new work",
		Path:        "/path/to/new.md",
		Source:      "local",
		Content:     "# New Workflow\n\nContent here",
	}

	prompt := convertWorkflowFull(wf)

	assert.Equal(t, "new", prompt.Name)
	assert.Equal(t, "# New Workflow\n\nContent here", prompt.Content)
}

func TestConvertProfile(t *testing.T) {
	entry := profile.ListEntry{
		Name:        "feature",
		Source:      profile.SourceLocal,
		Path:        "/path/to/feature.md",
		Description: "Feature development",
		Includes:    []string{"research", "base"},
		Inherits:    true,
		Shadowed:    false,
		Model:       "opus",
		Color:       "#ff0000",
		Type:        "domain",
	}

	prompt := convertProfile(entry)

	assert.Equal(t, "feature", prompt.Name)
	assert.Equal(t, CategoryProfile, prompt.Category)
	assert.Equal(t, SourceLocal, prompt.Source)
	assert.Equal(t, "Feature development", prompt.Description)
	assert.Equal(t, "/path/to/feature.md", prompt.Path)
	assert.Equal(t, []string{"research", "base"}, prompt.Includes)
	assert.True(t, prompt.Inherits)
	assert.False(t, prompt.Shadowed)
	assert.Equal(t, "opus", prompt.Model)
	assert.Equal(t, "#ff0000", prompt.Color)
	assert.Equal(t, "domain", prompt.ProfileType)
}

func TestConvertProfileFull(t *testing.T) {
	result := &profile.ShowResult{
		Name:        "feature",
		Source:      profile.SourceGlobal,
		Path:        "/path/to/feature.md",
		Description: "Feature development",
		Includes:    []string{"base"},
		Content:     "# Feature Profile\n\nBody content",
	}

	prompt := convertProfileFull(result)

	assert.Equal(t, "feature", prompt.Name)
	assert.Equal(t, CategoryProfile, prompt.Category)
	assert.Equal(t, SourceGlobal, prompt.Source)
	assert.Equal(t, "# Feature Profile\n\nBody content", prompt.Content)
}

func TestConvertStep(t *testing.T) {
	s := &step.Step{
		Name:        "plan",
		Description: "Create implementation plan",
		Path:        "/path/to/plan.md",
		Source:      step.SourceEmbedded,
		Profiles:    []string{"architect", "planner"},
		Files:       []string{"spec.md", "*.go"},
		Directive:   "Create a detailed plan...",
	}

	prompt := convertStep(s)

	assert.Equal(t, "plan", prompt.Name)
	assert.Equal(t, CategoryStep, prompt.Category)
	assert.Equal(t, SourceEmbedded, prompt.Source)
	assert.Equal(t, "Create implementation plan", prompt.Description)
	assert.Equal(t, "/path/to/plan.md", prompt.Path)
	assert.Equal(t, []string{"architect", "planner"}, prompt.Profiles)
	assert.Equal(t, []string{"spec.md", "*.go"}, prompt.Files)
	assert.Empty(t, prompt.Content, "convertStep should not include content")
}

func TestConvertStepFull(t *testing.T) {
	s := &step.Step{
		Name:        "plan",
		Description: "Create implementation plan",
		Path:        "/path/to/plan.md",
		Source:      step.SourceLocal,
		Directive:   "Create a detailed plan...",
	}

	prompt := convertStepFull(s)

	assert.Equal(t, "plan", prompt.Name)
	assert.Equal(t, "Create a detailed plan...", prompt.Content)
}

// === Source Mapping Tests ===

func TestMapWorkflowSource(t *testing.T) {
	assert.Equal(t, SourceLocal, mapWorkflowSource("local"))
	assert.Equal(t, SourceGlobal, mapWorkflowSource("global"))
	assert.Equal(t, SourceEmbedded, mapWorkflowSource("embedded"))
	assert.Equal(t, SourceEmbedded, mapWorkflowSource("unknown"))
}

func TestMapProfileSource(t *testing.T) {
	assert.Equal(t, SourceLocal, mapProfileSource(profile.SourceLocal))
	assert.Equal(t, SourceGlobal, mapProfileSource(profile.SourceGlobal))
	assert.Equal(t, SourceEmbedded, mapProfileSource(profile.SourceEmbedded))
}

func TestMapStepSource(t *testing.T) {
	assert.Equal(t, SourceLocal, mapStepSource(step.SourceLocal))
	assert.Equal(t, SourceGlobal, mapStepSource(step.SourceGlobal))
	assert.Equal(t, SourceEmbedded, mapStepSource(step.SourceEmbedded))
}

// === Type Helper Tests ===

func TestPromptCategoryLabel(t *testing.T) {
	assert.Equal(t, "Workflow", CategoryWorkflow.Label())
	assert.Equal(t, "Profile", CategoryProfile.Label())
	assert.Equal(t, "Step", CategoryStep.Label())
	assert.Equal(t, "unknown", PromptCategory("unknown").Label())
}

func TestPromptSourceBadgeColor(t *testing.T) {
	assert.Equal(t, "bg-green-100 text-green-800", SourceLocal.BadgeColor())
	assert.Equal(t, "bg-blue-100 text-blue-800", SourceGlobal.BadgeColor())
	assert.Equal(t, "bg-gray-100 text-gray-800", SourceEmbedded.BadgeColor())
	assert.Equal(t, "bg-gray-100 text-gray-800", PromptSource("unknown").BadgeColor())
}

// === Plugin Interface Tests ===

func TestPluginSidebarItems(t *testing.T) {
	plugin := NewPlugin(nil, nil, nil)
	items := plugin.SidebarItems()

	require.Len(t, items, 1)
	assert.Equal(t, "prompts", items[0].ID)
	assert.Equal(t, "Prompts", items[0].Label)
	assert.Equal(t, "/", items[0].Path)
	assert.Equal(t, 15, items[0].Order)
}

func TestPluginTemplates(t *testing.T) {
	plugin := NewPlugin(nil, nil, nil)
	templates := plugin.Templates()

	assert.NotNil(t, templates, "templates FS should not be nil")
}
