// Package git provides the MCP git tool for local repository operations.
package git

import (
	"encoding/json"
	"fmt"
)

// StatusResponse is returned for action=status.
type StatusResponse struct {
	Action           string   `json:"action"`
	Branch           string   `json:"branch"`
	StatusLines      []string `json:"status_lines"`
	HasStagedChanges bool     `json:"has_staged_changes"`
	TrackingInfo     string   `json:"tracking_info"`
}

// LogEntry represents a single commit in the log.
type LogEntry struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// LogResponse is returned for action=log.
type LogResponse struct {
	Action  string     `json:"action"`
	Commits []LogEntry `json:"commits"`
	Count   int        `json:"count"`
}

// DiffResponse is returned for action=diff.
type DiffResponse struct {
	Action   string `json:"action"`
	Content  string `json:"content"`
	StatOnly bool   `json:"stat_only,omitempty"`
}

// StageResponse is returned for action=stage.
type StageResponse struct {
	Action      string   `json:"action"`
	StagedFiles []string `json:"staged_files"`
}

// CommitResponse is returned for action=commit.
type CommitResponse struct {
	Action  string `json:"action"`
	Hash    string `json:"hash"`
	Branch  string `json:"branch"`
	Summary string `json:"summary"`
}

// PushResponse is returned for action=push.
type PushResponse struct {
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Remote  string `json:"remote"`
	Branch  string `json:"branch"`
	Output  string `json:"output"`
}

// ToolError represents a structured error from the git tool.
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func (e *ToolError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// marshalResponse marshals a response to indented JSON.
func marshalResponse(resp any) (string, error) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding response: %w", err)
	}
	return string(data), nil
}

// getStringArg extracts a string argument from the args map.
func getStringArg(args map[string]any, key string) string {
	if val, ok := args[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolArg extracts a boolean argument from the args map.
func getBoolArg(args map[string]any, key string) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// getIntArg extracts an integer argument from the args map.
// MCP passes numbers as float64, so we handle the conversion.
func getIntArg(args map[string]any, key string, defaultVal int) int {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultVal
}
