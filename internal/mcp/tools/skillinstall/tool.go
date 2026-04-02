// Package skillinstall provides the MCP skill-install tool implementation.
package skillinstall

import (
	"context"
	"fmt"
	"strings"

	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/skill"
)

// Tool implements the skill-install MCP tool.
type Tool struct{}

// NewTool creates a new skill-install tool.
func NewTool() *Tool {
	return &Tool{}
}

// Execute installs a named profile as a Claude Code skill.
// Args: name (string, required), scope ("local"|"global", required), working_directory (string, optional).
func (t *Tool) Execute(_ context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	scope, _ := args["scope"].(string)
	workingDir, _ := args["working_directory"].(string)

	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if scope == "" {
		return "", fmt.Errorf("scope is required (local or global)")
	}

	if err := skill.ValidateName(name); err != nil {
		return "", err
	}

	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	result, err := svc.Show(name, false)
	if err != nil {
		return "", skillProfileNotFoundError(svc, name)
	}

	targetDir, err := skill.TargetDir(scope == "global", workingDir)
	if err != nil {
		return "", err
	}

	content := skill.GenerateContent(name, result.Description)
	fullPath, err := skill.WriteSkill(targetDir, name, content)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Installed skill '%s' to %s", name, fullPath), nil
}

func skillProfileNotFoundError(svc *profile.Service, name string) error {
	entries, err := svc.List()
	if err != nil {
		return fmt.Errorf("profile %q not found", name)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, "  - "+e.Name)
	}
	return fmt.Errorf("profile %q not found. Available profiles:\n%s", name, strings.Join(names, "\n"))
}
