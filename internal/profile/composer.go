package profile

import (
	"fmt"
	"strings"
)

// Composer handles DAG building, cycle detection, and profile composition.
type Composer struct {
	profiles map[string]*Profile    // All available profiles keyed by name
	resolver *Resolver              // For loading inheritance chains (deprecated)
	source   ProfileSourceInterface // For loading inheritance chains
}

// NewComposerWithSource creates a new Composer with the given profiles and source.
func NewComposerWithSource(profiles map[string]*Profile, source ProfileSourceInterface) *Composer {
	return &Composer{
		profiles: profiles,
		source:   source,
	}
}

// Compose combines the specified profiles into a single result.
// It resolves includes, handles inheritance, and deduplicates content.
func (c *Composer) Compose(profileNames []string) (*CompositionResult, error) {
	profileNames = c.deduplicateNames(profileNames)

	if err := c.validateProfileNames(profileNames); err != nil {
		return nil, err
	}

	if err := c.checkForCycles(profileNames); err != nil {
		return nil, err
	}

	orderedProfiles, resolutionLog, err := c.resolveAllIncludes(profileNames)
	if err != nil {
		return nil, err
	}

	return c.composeContent(orderedProfiles, resolutionLog), nil
}

// validateProfileNames checks that every requested profile exists, returning a
// ProfileNotFoundError with suggestions for the first missing name.
func (c *Composer) validateProfileNames(names []string) error {
	for _, name := range names {
		if _, exists := c.profiles[name]; !exists {
			return &ProfileNotFoundError{
				Name:        name,
				Suggestions: c.findSimilar(name),
			}
		}
	}
	return nil
}

// checkForCycles runs DFS cycle detection across all requested profile names.
func (c *Composer) checkForCycles(names []string) error {
	visited := make(map[string]bool)
	pathSet := make(map[string]bool)
	for _, name := range names {
		if err := c.detectCycle(name, visited, pathSet, nil); err != nil {
			return err
		}
	}
	return nil
}

// resolveAllIncludes performs depth-first include resolution across all
// requested profile names, deduplicating along the way.
func (c *Composer) resolveAllIncludes(names []string) ([]*Profile, []ResolutionEntry, error) {
	resolved := make(map[string]bool)
	var orderedProfiles []*Profile
	var resolutionLog []ResolutionEntry

	for _, name := range names {
		profiles, log, err := c.resolveIncludes(name, resolved, "")
		if err != nil {
			return nil, nil, err
		}
		orderedProfiles = append(orderedProfiles, profiles...)
		resolutionLog = append(resolutionLog, log...)
	}

	return orderedProfiles, resolutionLog, nil
}

// composeContent resolves inheritance for each profile, concatenates content,
// and assembles the final CompositionResult.
func (c *Composer) composeContent(orderedProfiles []*Profile, resolutionLog []ResolutionEntry) *CompositionResult {
	var contentParts []string
	var profilesUsed []string
	var warnings []string

	for _, p := range orderedProfiles {
		content, inherited, err := c.resolveContent(p)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("warning resolving %s: %v", p.Name, err))
			content = p.Body
		}
		if inherited {
			for i := range resolutionLog {
				if resolutionLog[i].Name == p.Name {
					resolutionLog[i].Inherited = true
					break
				}
			}
		}
		contentParts = append(contentParts, content)
		profilesUsed = append(profilesUsed, p.Name)
	}

	finalContent := strings.Join(contentParts, "\n\n")
	charCount := len(finalContent)

	return &CompositionResult{
		Content:         finalContent,
		ProfilesUsed:    profilesUsed,
		CharacterCount:  charCount,
		EstimatedTokens: charCount / 4,
		Warnings:        warnings,
		ResolutionLog:   resolutionLog,
	}
}

// detectCycle uses DFS with path tracking to detect cycles in the include graph.
func (c *Composer) detectCycle(name string, visited, pathSet map[string]bool, path []string) error {
	pathSet[name] = true
	path = append(path, name)

	profile, exists := c.profiles[name]
	if !exists {
		// Profile not found is handled elsewhere; skip for cycle detection
		delete(pathSet, name)
		return nil
	}

	for _, included := range profile.Includes {
		if pathSet[included] {
			// Cycle detected - include the full path in error
			cyclePath := append(path, included)
			return &CycleError{
				Cycle: cyclePath,
			}
		}
		if visited[included] {
			continue // Already fully processed in another branch
		}
		if err := c.detectCycle(included, visited, pathSet, path); err != nil {
			return err
		}
	}

	delete(pathSet, name)
	visited[name] = true
	return nil
}

// resolveIncludes performs depth-first resolution of includes.
// Each profile's includes are resolved before the profile itself.
// Already-resolved profiles are skipped to prevent duplicates.
func (c *Composer) resolveIncludes(name string, resolved map[string]bool, includedBy string) ([]*Profile, []ResolutionEntry, error) {
	if resolved[name] {
		return nil, nil, nil
	}

	profile, exists := c.profiles[name]
	if !exists {
		suggestions := c.findSimilar(name)
		return nil, nil, &ProfileNotFoundError{
			Name:        name,
			Suggestions: suggestions,
		}
	}

	var profiles []*Profile
	var log []ResolutionEntry

	// First, resolve all includes (depth-first)
	for _, includeName := range profile.Includes {
		included, incLog, err := c.resolveIncludes(includeName, resolved, name)
		if err != nil {
			return nil, nil, err
		}
		profiles = append(profiles, included...)
		log = append(log, incLog...)
	}

	// Then add self
	profiles = append(profiles, profile)
	log = append(log, ResolutionEntry{
		Name:       name,
		Source:     profile.Source,
		Path:       profile.Path,
		IncludedBy: includedBy,
	})

	resolved[name] = true
	return profiles, log, nil
}

// resolveContent gets the final content for a profile, handling inheritance.
// Returns the content and whether inheritance was applied.
func (c *Composer) resolveContent(p *Profile) (string, bool, error) {
	if !p.Inherits {
		return p.Body, false, nil
	}

	// Try source first, fall back to resolver for backward compatibility
	var chain []*Profile
	var err error
	if c.source != nil {
		chain, err = c.source.GetInheritanceChain(p.Name)
	} else if c.resolver != nil {
		chain, err = c.resolver.GetInheritanceChain(p.Name)
	} else {
		return p.Body, false, nil
	}

	if err != nil {
		return p.Body, false, err
	}

	if len(chain) <= 1 {
		// No parent versions to inherit from
		return p.Body, false, nil
	}

	// Concatenate content from all versions: global first, local last
	var parts []string
	for _, version := range chain {
		if version.Body != "" {
			parts = append(parts, version.Body)
		}
	}

	if len(parts) <= 1 {
		return p.Body, false, nil
	}

	return strings.Join(parts, "\n\n"), true, nil
}

// deduplicateNames removes duplicate profile names while preserving order.
func (c *Composer) deduplicateNames(names []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}

// findSimilar finds profile names similar to the given name (for suggestions).
func (c *Composer) findSimilar(name string) []string {
	var suggestions []string
	nameLower := strings.ToLower(name)

	for profileName := range c.profiles {
		profileLower := strings.ToLower(profileName)
		// Simple similarity: contains or prefix match
		if strings.Contains(profileLower, nameLower) ||
			strings.Contains(nameLower, profileLower) ||
			strings.HasPrefix(profileLower, nameLower[:min(3, len(nameLower))]) {
			suggestions = append(suggestions, profileName)
		}
	}

	// Limit suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions
}

// ProfileNotFoundError is returned when a requested profile doesn't exist.
type ProfileNotFoundError struct {
	Name        string
	Suggestions []string
}

func (e *ProfileNotFoundError) Error() string {
	msg := fmt.Sprintf("profile %q not found", e.Name)
	if len(e.Suggestions) > 0 {
		msg += fmt.Sprintf(" (did you mean: %s?)", strings.Join(e.Suggestions, ", "))
	}
	return msg
}

// CycleError is returned when a circular dependency is detected.
type CycleError struct {
	Cycle []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Cycle, " -> "))
}
