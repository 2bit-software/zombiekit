package profile

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/adrg/frontmatter"
)

// ParseFrontmatter parses a profile file's content, extracting YAML frontmatter
// and the remaining markdown body. If no frontmatter is present, it returns
// zero-valued frontmatter and the entire content as body.
func ParseFrontmatter(content []byte) (ProfileFrontmatter, string, error) {
	var fm ProfileFrontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
	if err != nil {
		// Try to provide line number context for YAML errors
		return ProfileFrontmatter{}, "", fmt.Errorf("parsing frontmatter: %w", err)
	}

	return fm, strings.TrimSpace(string(rest)), nil
}

// ParseProfile parses a profile file and returns a Profile struct.
// The name parameter is used as the profile name if not specified in frontmatter.
// The path and source are set directly on the returned Profile.
func ParseProfile(content []byte, name, path string, source ProfileSource) (*Profile, error) {
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Use frontmatter name if provided, otherwise use the passed name
	profileName := fm.Name
	if profileName == "" {
		profileName = name
	}

	return &Profile{
		Name:        profileName,
		Path:        path,
		Source:      source,
		Description: fm.Description,
		Includes:    fm.Includes,
		Inherits:    fm.GetInherits(),
		Body:        body,
		RawContent:  content,
	}, nil
}
