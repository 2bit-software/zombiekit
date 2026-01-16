// Package step contracts define the interface boundaries for step operations.
// This is a design contract, not production code.

package contracts

// StepName enumerates valid step names.
type StepName string

const (
	StepFeature  StepName = "feature"
	StepBug      StepName = "bug"
	StepRefactor StepName = "refactor"
	StepPlan     StepName = "plan"
	StepTasks    StepName = "tasks"
	StepEat      StepName = "eat"
	StepAudit    StepName = "audit"
	StepClarify  StepName = "clarify"
	StepComplete StepName = "complete"
)

// ValidStepNames returns all valid step names.
func ValidStepNames() []StepName {
	return []StepName{
		StepFeature,
		StepBug,
		StepRefactor,
		StepPlan,
		StepTasks,
		StepEat,
		StepAudit,
		StepClarify,
		StepComplete,
	}
}

// StepPrerequisite defines what must exist before a step can execute.
type StepPrerequisite struct {
	// RequiredArtifact is the file that must exist (e.g., "spec.md").
	RequiredArtifact string
	// RequiredStatus is the status in frontmatter (empty = no status check).
	RequiredStatus string
	// Hint is shown when prerequisite not met.
	Hint string
	// BlockingStep is the step that produces the required artifact.
	BlockingStep StepName
}

// Prerequisites maps steps to their requirements.
var Prerequisites = map[StepName]StepPrerequisite{
	StepPlan: {
		RequiredArtifact: "spec.md",
		RequiredStatus:   "approved",
		Hint:             "Run feature, bug, or refactor first and approve the spec",
		BlockingStep:     StepFeature,
	},
	StepTasks: {
		RequiredArtifact: "plan.md",
		RequiredStatus:   "approved",
		Hint:             "Run plan first and approve the plan",
		BlockingStep:     StepPlan,
	},
	StepEat: {
		RequiredArtifact: "tasks.md",
		RequiredStatus:   "", // No status check, just existence
		Hint:             "Run tasks first to generate the task list",
		BlockingStep:     StepTasks,
	},
}

// StepService defines the interface for step execution.
type StepService interface {
	// Execute runs a step by name with options.
	// Returns StepResponse on success.
	// Returns error with code UNKNOWN_STEP if step not found.
	// Returns error with code PREREQUISITE_NOT_MET if prerequisite fails.
	Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error)

	// GetStep retrieves a step definition by name.
	GetStep(name string) (*Step, error)

	// ListSteps returns all available step definitions.
	ListSteps() ([]*Step, error)

	// CheckPrerequisite validates if a step's prerequisite is met.
	// Returns nil if met, StepError if not.
	CheckPrerequisite(stepName string, cyclePath string) error
}

// ExecuteOptions provides parameters for step execution.
type ExecuteOptions struct {
	// Initiative overrides the active initiative.
	Initiative string
	// Type is required for feature, bug, refactor steps.
	Type string
	// Name is required for feature, bug, refactor steps.
	Name string
	// Description is optional description for the initiative.
	Description string
	// NewInitiative forces creation of new initiative.
	NewInitiative bool
}

// StepResponse is the result of step execution.
type StepResponse struct {
	// Directive is the step instruction text.
	Directive string
	// InitiativeFolder is the absolute path to initiative folder.
	InitiativeFolder string
	// CycleFolder is the absolute path to active cycle folder.
	CycleFolder string
	// FilesToRead is the list of files the agent should read.
	FilesToRead []string
	// ComposedPrompt is the pre-composed profile prompt.
	ComposedPrompt string
	// WorkflowPhases describes phases in multi-phase steps.
	WorkflowPhases []Phase
}

// Step represents a workflow step definition.
type Step struct {
	Name        string
	Description string
	Profiles    []string
	Files       []string
	Directive   string
}

// Phase represents a workflow phase in a multi-phase step.
type Phase struct {
	Name        string
	Description string
	Agents      []string
	Outputs     []string
	Parallel    bool
}

// StepError represents an error with structured code.
type StepError struct {
	Code    string // e.g., "UNKNOWN_STEP", "PREREQUISITE_NOT_MET"
	Message string
	Hint    string
}

func (e *StepError) Error() string {
	return e.Message
}

// Error codes
const (
	ErrUnknownStep        = "UNKNOWN_STEP"
	ErrPrerequisiteNotMet = "PREREQUISITE_NOT_MET"
	ErrNoActiveInitiative = "NO_ACTIVE_INITIATIVE"
	ErrMissingName        = "MISSING_NAME"
	ErrInvalidType        = "INVALID_TYPE"
)
