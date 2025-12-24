// Package step provides step definitions and execution for the initiative framework.
// A step represents a workflow step (specify, plan, implement, etc.) that can be
// executed within an initiative context.
package step

// StepSource represents where a step definition was loaded from.
type StepSource int

const (
	// SourceEmbedded indicates the step was loaded from embedded defaults.
	SourceEmbedded StepSource = iota
	// SourceGlobal indicates the step was loaded from ~/.brains/steps/.
	SourceGlobal
	// SourceLocal indicates the step was loaded from .brains/steps/.
	SourceLocal
)

// String returns a human-readable string for the step source.
func (s StepSource) String() string {
	switch s {
	case SourceEmbedded:
		return "embedded"
	case SourceGlobal:
		return "global"
	case SourceLocal:
		return "local"
	default:
		return "unknown"
	}
}

// Step represents a workflow step definition.
type Step struct {
	// Name is the step identifier (e.g., "specify", "plan").
	Name string `json:"name"`
	// Description is a human-readable description of the step.
	Description string `json:"description,omitempty"`
	// Profiles is the list of profile names to compose for this step.
	Profiles []string `json:"profiles,omitempty"`
	// Files is the list of glob patterns for files to read.
	Files []string `json:"files,omitempty"`
	// Directive is the instruction text for this step.
	Directive string `json:"directive"`
	// Type is the profile type marker (always "step" for step definitions).
	Type string `json:"type,omitempty"`
	// Source indicates where the step definition was loaded from.
	Source StepSource `json:"-"`
	// Path is the absolute path if loaded from file.
	Path string `json:"-"`
}

// StepFrontmatter represents the YAML frontmatter in a step definition file.
type StepFrontmatter struct {
	// Name is the step identifier.
	Name string `yaml:"name"`
	// Description is a human-readable description.
	Description string `yaml:"description,omitempty"`
	// Profiles is the list of profile names to compose.
	Profiles []string `yaml:"profiles,omitempty"`
	// Files is the list of glob patterns for files to read.
	Files []string `yaml:"files,omitempty"`
	// Type is the profile type marker.
	Type string `yaml:"type,omitempty"`
}

// StepResponse is the structured output from executing a step via MCP.
type StepResponse struct {
	// Directive is the step directive/instruction text.
	Directive string `json:"directive"`
	// HistoryFolder is the absolute path to the initiative's history folder.
	HistoryFolder string `json:"history_folder"`
	// FilesToRead is the list of files the agent should read.
	FilesToRead []string `json:"files_to_read"`
	// ComposedPrompt is the pre-composed profile prompt for this step.
	ComposedPrompt string `json:"composed_prompt"`
}
