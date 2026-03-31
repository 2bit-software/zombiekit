// Package codereasoning provides the MCP code-reasoning tool implementation.
package codereasoning

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool implements the code-reasoning MCP tool.
type Tool struct {
	manager *SessionManager
}

// NewTool creates a new code-reasoning tool with the given session manager.
func NewTool(manager *SessionManager) *Tool {
	return &Tool{manager: manager}
}

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "code-reasoning",
		Description: "Sequential thinking tool for problem-solving with branching and revision capabilities",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"thought": map[string]any{
					"type":        "string",
					"description": "Your current reasoning step",
					"minLength":   1,
				},
				"thought_number": map[string]any{
					"type":        "integer",
					"description": "Current number in sequence (1-indexed)",
					"minimum":     1,
				},
				"total_thoughts": map[string]any{
					"type":        "integer",
					"description": "Estimated final count (can adjust as needed)",
					"minimum":     1,
					"maximum":     MaxThoughts,
				},
				"next_thought_needed": map[string]any{
					"type":        "boolean",
					"description": "Set to FALSE ONLY when completely done",
				},
				"is_revision": map[string]any{
					"type":        "boolean",
					"description": "When correcting earlier thinking",
					"default":     false,
				},
				"revises_thought": map[string]any{
					"type":        "integer",
					"description": "Which thought to revise (required if is_revision is true)",
					"minimum":     1,
				},
				"branch_from_thought": map[string]any{
					"type":        "integer",
					"description": "When exploring alternative approaches",
					"minimum":     1,
				},
				"branch_id": map[string]any{
					"type":        "string",
					"description": "Branch identifier (required if branch_from_thought is set)",
					"pattern":     "^[a-zA-Z0-9._-]+$",
				},
			},
			"required": []string{"thought", "thought_number", "total_thoughts", "next_thought_needed"},
		},
	}
}

// Execute runs the tool with the given arguments.
func (t *Tool) Execute(ctx context.Context, sessionID string, args map[string]any) (string, error) {
	req, err := parseRequest(args)
	if err != nil {
		return "", err
	}

	session := t.manager.GetOrCreate(sessionID)

	if err := session.AddThought(req); err != nil {
		return "", fmt.Errorf("failed to add thought: %w", err)
	}

	response := ThoughtResponse{
		ThoughtNumber: session.GetCurrentThoughtNumber(),
		TotalThoughts: session.GetTotalThoughts(),
		Chain:         session.Format(),
		Status:        session.GetStatus(),
		Branches:      session.GetBranchIDs(),
	}

	if req.IsRevision {
		response.RevisedNumber = req.RevisesThought
	}

	if req.BranchID != "" {
		response.BranchID = req.BranchID
		response.BranchedFrom = req.BranchFromThought
	}

	return toJSON(response)
}

func parseRequest(args map[string]any) (ThoughtRequest, error) {
	var req ThoughtRequest

	if err := parseRequiredFields(args, &req); err != nil {
		return req, err
	}

	if err := parseRevisionFields(args, &req); err != nil {
		return req, err
	}

	if err := parseBranchFields(args, &req); err != nil {
		return req, err
	}

	return req, nil
}

func parseRequiredFields(args map[string]any, req *ThoughtRequest) error {
	thought, ok := args["thought"].(string)
	if !ok || thought == "" {
		return fmt.Errorf("thought is required")
	}
	req.Thought = thought

	thoughtNum, ok := args["thought_number"].(float64)
	if !ok {
		return fmt.Errorf("thought_number is required")
	}
	req.ThoughtNumber = int(thoughtNum)

	totalThoughts, ok := args["total_thoughts"].(float64)
	if !ok {
		return fmt.Errorf("total_thoughts is required")
	}
	req.TotalThoughts = int(totalThoughts)

	nextNeeded, ok := args["next_thought_needed"].(bool)
	if !ok {
		return fmt.Errorf("next_thought_needed is required")
	}
	req.NextThoughtNeeded = nextNeeded

	return nil
}

func parseRevisionFields(args map[string]any, req *ThoughtRequest) error {
	isRevision, ok := args["is_revision"].(bool)
	if !ok || !isRevision {
		return nil
	}
	req.IsRevision = true

	revisesThought, ok := args["revises_thought"].(float64)
	if !ok {
		return fmt.Errorf("revises_thought is required when is_revision is true")
	}
	req.RevisesThought = int(revisesThought)
	return nil
}

func parseBranchFields(args map[string]any, req *ThoughtRequest) error {
	branchFrom, ok := args["branch_from_thought"].(float64)
	if !ok {
		return nil
	}
	req.BranchFromThought = int(branchFrom)

	branchID, ok := args["branch_id"].(string)
	if !ok || branchID == "" {
		return fmt.Errorf("branch_id is required when branch_from_thought is set")
	}
	req.BranchID = branchID
	return nil
}

func toJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(data), nil
}
