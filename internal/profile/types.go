// Package profile provides profile composition and management for the brains CLI.
package profile

// ProfileSource indicates where a profile was loaded from.
type ProfileSource int

const (
	// SourceLocal is a profile from the project's .brains/profiles/ directory.
	SourceLocal ProfileSource = iota
	// SourceParent is a profile from an intermediate .brains/profiles/ directory
	// (between CWD and git root).
	SourceParent
	// SourceGlobal is a profile from ~/.brains/profiles/.
	SourceGlobal
	// SourceEmbedded is a profile embedded in the binary at build time.
	SourceEmbedded
)

// String returns the string representation of the profile source.
func (s ProfileSource) String() string {
	switch s {
	case SourceLocal:
		return "local"
	case SourceParent:
		return "parent"
	case SourceGlobal:
		return "global"
	case SourceEmbedded:
		return "embedded"
	default:
		return "unknown"
	}
}

// Profile represents a loaded profile with parsed frontmatter and content.
type Profile struct {
	// Identity
	Name   string        // Derived from filename if not in frontmatter
	Path   string        // Absolute path to the profile file
	Source ProfileSource // Where this profile was loaded from

	// Frontmatter fields (all optional)
	Description string   // Human-readable description
	Includes    []string // Names of profiles to include before this one
	Inherits    bool     // Whether to prepend parent directory versions (default: true)

	// Claude-specific fields (optional, only set for Claude agents)
	Model string // Claude model (e.g., "opus", "sonnet")
	Color string // UI color for Claude Code display
	Type  string // Profile type: "action", "domain", or "step"

	// Content
	Body       string // Markdown content after frontmatter
	RawContent []byte // Original file content for --raw mode
}

// ProfileFrontmatter represents the optional YAML frontmatter in a profile file.
type ProfileFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Includes    []string `yaml:"includes"`
	Inherits    *bool    `yaml:"inherits"` // Pointer to detect unset vs explicit false
	Type        string   `yaml:"type"`     // Profile type: "action", "domain", or "step"
}

// GetInherits returns the inherits value, defaulting to true if not set.
func (f ProfileFrontmatter) GetInherits() bool {
	if f.Inherits == nil {
		return true
	}
	return *f.Inherits
}

// CompositionResult contains the merged output and metadata from profile composition.
type CompositionResult struct {
	// Output
	Content string // Raw concatenated content (no separators)

	// Metadata
	ProfilesUsed    []string // Names of profiles included (in order)
	CharacterCount  int
	EstimatedTokens int // Rough estimate: CharacterCount / 4

	// Diagnostics
	Warnings      []string
	ResolutionLog []ResolutionEntry
}

// ResolutionEntry records how a profile was resolved.
type ResolutionEntry struct {
	Name       string        // Profile name
	Source     ProfileSource // Where it was loaded from
	Path       string        // Absolute path to the file
	Inherited  bool          // Whether content was inherited from parent
	IncludedBy string        // Which profile included this one (empty for top-level)
}

// InheritedFrom tracks which parent profiles content was inherited from.
type InheritedFrom struct {
	Source ProfileSource
	Path   string
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Profile     string   // Profile with the error
	Code        string   // Error code (e.g., MISSING_INCLUDE, CIRCULAR_DEPENDENCY)
	Message     string   // Human-readable error message
	Suggestions []string // Suggested fixes (e.g., similar profile names)
	Cycle       []string // For circular dependencies, the full cycle path
}

// ValidationResult contains the result of validating all profiles.
type ValidationResult struct {
	Valid           bool
	ProfilesChecked int
	Errors          []ValidationError
}

// ListEntry represents a profile in the list output.
type ListEntry struct {
	Name        string        `json:"name"`
	Source      ProfileSource `json:"-"`
	SourceStr   string        `json:"source"`
	Path        string        `json:"path"`
	Description string        `json:"description"`
	Includes    []string      `json:"includes"`
	Inherits    bool          `json:"inherits"`
	Shadowed    bool          `json:"shadowed,omitempty"` // True if shadowed by higher-precedence profile
	Model       string        `json:"model,omitempty"`    // Claude-specific: model name
	Color       string        `json:"color,omitempty"`    // Claude-specific: UI color
	Type        string        `json:"type,omitempty"`     // Profile type: "action", "domain", or "step"
}

// ShowResult contains the result of showing a single profile.
type ShowResult struct {
	Name          string          `json:"name"`
	Source        ProfileSource   `json:"-"`
	SourceStr     string          `json:"source"`
	Path          string          `json:"path"`
	Description   string          `json:"description"`
	Includes      []string        `json:"includes"`
	Inherits      bool            `json:"inherits"`
	Content       string          `json:"content"`
	RawContent    string          `json:"raw_content"`
	InheritedFrom []InheritedFrom `json:"inherited_from,omitempty"`
	Model         string          `json:"model,omitempty"` // Claude-specific: model name
	Color         string          `json:"color,omitempty"` // Claude-specific: UI color
	Type          string          `json:"type,omitempty"`  // Profile type: "action", "domain", or "step"
}

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	// Counters
	Created     int `json:"created"`     // Number of new profiles created
	Overwritten int `json:"overwritten"` // Number of existing profiles overwritten
	Failed      int `json:"failed"`      // Number of agents that failed to import

	// Details
	CreatedPaths     []string        `json:"created_paths"`     // Paths to created profiles
	OverwrittenPaths []string        `json:"overwritten_paths"` // Paths to overwritten profiles
	FailedAgents     []ImportFailure `json:"failed_agents"`     // Details of failed imports

	// Operation mode
	DryRun bool `json:"dry_run"` // True if this was a dry run (no actual writes)
}

// ImportFailure describes why an agent failed to import.
type ImportFailure struct {
	AgentName string `json:"agent_name"` // Name of the agent that failed
	AgentPath string `json:"agent_path"` // Source path
	Error     string `json:"error"`      // Error message
}
