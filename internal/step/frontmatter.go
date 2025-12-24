package step

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/adrg/frontmatter"
)

// ParseFrontmatter parses a step definition file's content, extracting YAML
// frontmatter and the remaining markdown body (the directive).
// If no frontmatter is present, returns zero-valued frontmatter and the entire content.
func ParseFrontmatter(content []byte) (StepFrontmatter, string, error) {
	var fm StepFrontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
	if err != nil {
		return StepFrontmatter{}, "", fmt.Errorf("parsing step frontmatter: %w", err)
	}

	return fm, strings.TrimSpace(string(rest)), nil
}

// ParseStep parses a step definition file and returns a Step struct.
// The name parameter is used as the step name if not specified in frontmatter.
// The path and source are set directly on the returned Step.
func ParseStep(content []byte, name, path string, source StepSource) (*Step, error) {
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Use frontmatter name if provided, otherwise use the passed name
	stepName := fm.Name
	if stepName == "" {
		stepName = name
	}

	return &Step{
		Name:        stepName,
		Description: fm.Description,
		Profiles:    fm.Profiles,
		Files:       fm.Files,
		Directive:   body,
		Type:        fm.Type,
		Source:      source,
		Path:        path,
	}, nil
}
