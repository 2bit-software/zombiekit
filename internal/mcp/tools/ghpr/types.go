// Package ghpr provides the MCP tool for GitHub PR operations via the gh CLI.
package ghpr

import (
	"encoding/json"
	"fmt"
)

// ViewResponse is returned for action=view.
type ViewResponse struct {
	Action string `json:"action"`
	Exists bool   `json:"exists"`
	URL    string `json:"url,omitempty"`
	Title  string `json:"title,omitempty"`
	Number int    `json:"number,omitempty"`
	State  string `json:"state,omitempty"`
}

// CreateResponse is returned for action=create.
type CreateResponse struct {
	Action string `json:"action"`
	URL    string `json:"url"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// CommentResponse is returned for action=comment.
type CommentResponse struct {
	Action   string `json:"action"`
	Success  bool   `json:"success"`
	PRNumber int    `json:"pr_number"`
}

// EditResponse is returned for action=edit.
type EditResponse struct {
	Action   string `json:"action"`
	Success  bool   `json:"success"`
	PRNumber int    `json:"pr_number"`
	URL      string `json:"url,omitempty"`
}

// ToolError represents a structured error from the gh-pr tool.
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
