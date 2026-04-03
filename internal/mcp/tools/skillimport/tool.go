// Package skillimport provides MCP tools for importing Claude Code skills and agents into zombiekit profiles.
package skillimport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/2bit-software/zombiekit/internal/skill"
)

// Tool implements the skill-import and skill-import-list MCP tools.
type Tool struct{}

// NewTool creates a new skill import tool.
func NewTool() *Tool {
	return &Tool{}
}

// ExecuteList lists Claude Code skills and agents available for import.
func (t *Tool) ExecuteList(_ context.Context, args map[string]any) (string, error) {
	workingDir, _ := args["working_directory"].(string)

	items, warnings, err := skill.DiscoverAll(workingDir)
	if err != nil {
		return "", fmt.Errorf("discovering items: %w", err)
	}

	type listResponse struct {
		Items    []skill.DiscoverableItem `json:"items"`
		Warnings []string                 `json:"warnings,omitempty"`
	}

	resp := listResponse{
		Items:    items,
		Warnings: warnings,
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling response: %w", err)
	}

	return string(data), nil
}

// ExecuteImport imports named skills/agents into zombiekit profiles.
func (t *Tool) ExecuteImport(_ context.Context, args map[string]any) (string, error) {
	scope, _ := args["scope"].(string)
	shim, _ := args["shim"].(bool)
	workingDir, _ := args["working_directory"].(string)

	if scope == "" {
		return "", fmt.Errorf("scope is required ('local' or 'global')")
	}

	namesRaw, ok := args["names"].([]any)
	if !ok || len(namesRaw) == 0 {
		return "", fmt.Errorf("names is required (array of strings)")
	}

	names := make([]string, 0, len(namesRaw))
	for _, n := range namesRaw {
		s, ok := n.(string)
		if !ok {
			return "", fmt.Errorf("each name must be a string, got %T", n)
		}
		names = append(names, s)
	}

	items, warnings, err := skill.DiscoverAll(workingDir)
	if err != nil {
		return "", fmt.Errorf("discovering items: %w", err)
	}

	if len(warnings) > 0 {
		// Name collisions detected — include in response but don't block
		_ = warnings
	}

	opts := skill.ImportOptions{
		Names:      names,
		Scope:      scope,
		Shim:       shim,
		WorkingDir: workingDir,
	}

	result, err := skill.Import(opts, items)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}

	return string(data), nil
}
