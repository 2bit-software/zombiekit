package web

// PromptCategory identifies the type of prompt.
type PromptCategory string

const (
	CategoryWorkflow PromptCategory = "workflow"
	CategoryProfile  PromptCategory = "profile"
	CategoryStep     PromptCategory = "step"
)

func (c PromptCategory) String() string { return string(c) }

// Label returns a human-readable label for the category.
func (c PromptCategory) Label() string {
	switch c {
	case CategoryWorkflow:
		return "Workflow"
	case CategoryProfile:
		return "Profile"
	case CategoryStep:
		return "Step"
	default:
		return string(c)
	}
}

// PromptSource identifies where the prompt is stored.
type PromptSource string

const (
	SourceLocal    PromptSource = "local"
	SourceGlobal   PromptSource = "global"
	SourceEmbedded PromptSource = "embedded"
)

func (s PromptSource) String() string { return string(s) }

// BadgeColor returns Tailwind CSS classes for styling the source badge.
func (s PromptSource) BadgeColor() string {
	switch s {
	case SourceLocal:
		return "bg-green-100 text-green-800"
	case SourceGlobal:
		return "bg-blue-100 text-blue-800"
	case SourceEmbedded:
		return "bg-gray-100 text-gray-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// Prompt is the unified representation for all prompt types.
type Prompt struct {
	Name        string
	Category    PromptCategory
	Source      PromptSource
	Description string
	Path        string
	Shadowed    bool // True if overridden by higher-precedence source

	// Profile-specific (zero values for non-profiles)
	ProfileType string   // domain, action, step, skill
	Includes    []string // Referenced profiles
	Inherits    bool     // Inherit from parent dirs
	Model       string   // Claude model override
	Color       string   // UI color

	// Step-specific (zero values for non-steps)
	Profiles []string // Profiles to compose
	Files    []string // File patterns to load

	// Full content (populated only for detail view)
	Content string
}

// PromptsListData is passed to the list template.
type PromptsListData struct {
	Prompts []Prompt

	// Current filter/sort state (for preserving in UI)
	CategoryFilter string
	SourceFilter   string
	Query          string
	SortField      string
	SortOrder      string

	// Error message if any
	Error string
}

// PromptsViewData is passed to the view template.
type PromptsViewData struct {
	Prompt *Prompt
	Error  string
}

// PromptsFilterOptions encapsulates all filter parameters.
type PromptsFilterOptions struct {
	Category string // "workflow", "profile", "step", or "" for all
	Source   string // "local", "global", "embedded", or "" for all
	Query    string // Search text
}

// PromptsSortOptions encapsulates sorting parameters.
type PromptsSortOptions struct {
	Field string // "name", "category", "source"
	Order string // "asc", "desc"
}
