// Package hook implements the session-aware hook event handler for injecting
// rules into AI coding agent contexts via stdin/stdout.
package hook

// HookEvent is the JSON payload received from the agent's hook system via stdin.
type HookEvent struct {
	SessionID     string        `json:"session_id"`
	HookEventName string        `json:"hook_event_name"`
	CWD           string        `json:"cwd"`
	Source        string        `json:"source,omitempty"`
	ToolName      string        `json:"tool_name,omitempty"`
	ToolInput     *ToolInput    `json:"tool_input,omitempty"`
	ToolResponse  *ToolResponse `json:"tool_response,omitempty"`
}

// ToolInput contains the input parameters passed to the tool.
type ToolInput struct {
	FilePath    string      `json:"file_path,omitempty"`
	FilePathAlt string      `json:"filePath,omitempty"`
	Edits       []EditEntry `json:"edits,omitempty"`
	Command     string      `json:"command,omitempty"`
}

// GetFilePath returns the file path from either snake_case or camelCase field.
func (t *ToolInput) GetFilePath() string {
	if t.FilePath != "" {
		return t.FilePath
	}
	return t.FilePathAlt
}

// EditEntry represents a single file edit in a MultiEdit operation.
type EditEntry struct {
	FilePath    string `json:"file_path"`
	FilePathAlt string `json:"filePath"`
	OldString   string `json:"old_string"`
	NewString   string `json:"new_string"`
}

// GetFilePath returns the file path from either snake_case or camelCase field.
func (e *EditEntry) GetFilePath() string {
	if e.FilePath != "" {
		return e.FilePath
	}
	return e.FilePathAlt
}

// ToolResponse contains the output returned by the tool.
type ToolResponse struct {
	FilePath string `json:"filePath,omitempty"`
	Success  *bool  `json:"success,omitempty"`
}

// Agent identifies which AI coding agent is running.
type Agent string

const (
	AgentClaude   Agent = "claude"
	AgentGemini   Agent = "gemini"
	AgentOpenCode Agent = "opencode"
)

// MatchedRule records that a rule fired (or was deduped) for a specific
// trigger. File-glob rules use an empty trigger; command rules carry the
// command prefix that caused the match.
type MatchedRule struct {
	ID      string `json:"id"`
	Trigger string `json:"trigger,omitempty"`
}
