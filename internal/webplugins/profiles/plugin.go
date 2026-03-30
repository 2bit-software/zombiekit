// Package profiles provides a web plugin for viewing and managing profiles.
package profiles

import (
	"embed"
	"io/fs"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/search"
	"github.com/2bit-software/zombiekit/internal/web"
)

//go:embed templates
var templateFS embed.FS

// Plugin implements the WebPlugin interface for profiles.
type Plugin struct {
	service *profile.Service
}

// NewPlugin creates a new profiles plugin with the given profile service.
func NewPlugin(service *profile.Service) *Plugin {
	return &Plugin{
		service: service,
	}
}

// SidebarItems returns the navigation entries for the profiles plugin.
// Paths are relative - the system automatically prefixes with the plugin name.
func (p *Plugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{
			ID:    "profiles",
			Label: "Profiles",
			Path:  "/",
			Order: 10, // First in the list
		},
	}
}

// MountRoutes registers the HTTP handlers for the profiles plugin.
func (p *Plugin) MountRoutes(r chi.Router) {
	h := &handlers{service: p.service}
	r.Get("/", h.list)
	r.Get("/{name}", h.view)
}

// Templates returns the embedded template filesystem.
func (p *Plugin) Templates() fs.FS {
	return templateFS
}

// Ensure Plugin implements TemplatePlugin
var _ web.TemplatePlugin = (*Plugin)(nil)

// Ensure Plugin implements Searchable
var _ search.Searchable = (*Plugin)(nil)

// searchMatch holds a search result with its relevance score for sorting.
type searchMatch struct {
	result search.SearchResult
	score  int // Higher score = more relevant
	entry  profile.ListEntry
}

// Search implements search.Searchable.
// It searches profile names and descriptions, returning matching results.
func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
	// Empty query returns empty results per contract
	if query == "" {
		return []search.SearchResult{}, nil
	}

	// Get all profiles
	entries, err := p.service.List()
	if err != nil {
		return nil, err
	}

	// Normalize query for case-insensitive search
	queryLower := strings.ToLower(query)

	// Find matching profiles
	var matches []searchMatch
	seen := make(map[string]bool) // Track seen profile names to prevent duplicates

	for _, entry := range entries {
		// Skip shadowed profiles to avoid duplicates
		if entry.Shadowed {
			continue
		}

		// Skip if we've already matched this profile name
		if seen[entry.Name] {
			continue
		}

		nameLower := strings.ToLower(entry.Name)
		descLower := strings.ToLower(entry.Description)

		// Check if name or description matches
		nameMatch := strings.Contains(nameLower, queryLower)
		descMatch := strings.Contains(descLower, queryLower)

		if nameMatch || descMatch {
			seen[entry.Name] = true

			// Calculate relevance score
			score := p.calculateRelevanceScore(nameLower, descLower, queryLower, nameMatch)

			matches = append(matches, searchMatch{
				result: search.SearchResult{
					Title: entry.Name,
					URL:   "/" + entry.Name, // Relative URL - system prefixes automatically
				},
				score: score,
				entry: entry,
			})
		}
	}

	// Sort by the requested order
	p.sortMatches(matches, sortOrder)

	// Apply maxResults limit
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	// Extract results
	results := make([]search.SearchResult, len(matches))
	for i, m := range matches {
		results[i] = m.result
	}

	return results, nil
}

// calculateRelevanceScore calculates a relevance score for a match.
// Higher score = more relevant.
func (p *Plugin) calculateRelevanceScore(nameLower, descLower, queryLower string, nameMatch bool) int {
	score := 0

	// Name matches are more relevant than description matches
	if nameMatch {
		score += 100

		// Exact name match is highest relevance
		if nameLower == queryLower {
			score += 1000
		} else if strings.HasPrefix(nameLower, queryLower) {
			// Prefix match is second highest
			score += 500
		}
	}

	return score
}

// sortMatches sorts the matches according to the specified sort order.
func (p *Plugin) sortMatches(matches []searchMatch, sortOrder search.SortOrder) {
	switch sortOrder {
	case search.SortName:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].entry.Name < matches[j].entry.Name
		})
	case search.SortCreatedDate, search.SortUpdatedDate, search.SortLastUsed:
		// Profiles don't have timestamps, fall back to relevance
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
	default: // SortRelevance or empty/invalid
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
	}
}
