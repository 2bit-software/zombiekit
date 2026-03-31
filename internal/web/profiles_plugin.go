package web

import (
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
)

// ProfilesPlugin implements the WebPlugin interface for profiles.
type ProfilesPlugin struct {
	service *profile.Service
}

// NewProfilesPlugin creates a new profiles plugin with the given profile service.
func NewProfilesPlugin(service *profile.Service) *ProfilesPlugin {
	return &ProfilesPlugin{
		service: service,
	}
}

// SidebarItems returns the navigation entries for the profiles plugin.
// Paths are relative - the system automatically prefixes with the plugin name.
func (p *ProfilesPlugin) SidebarItems() []SidebarItem {
	return []SidebarItem{
		{
			ID:    "profiles",
			Label: "Profiles",
			Path:  "/",
			Order: 10, // First in the list
		},
	}
}

// MountRoutes registers the HTTP handlers for the profiles plugin.
func (p *ProfilesPlugin) MountRoutes(r chi.Router) {
	h := &profilesHandlers{service: p.service}
	r.Get("/", h.list)
	r.Get("/{name}", h.view)
}

var _ WebPlugin = (*ProfilesPlugin)(nil)

// Ensure ProfilesPlugin implements Searchable
var _ Searchable = (*ProfilesPlugin)(nil)

// profilesSearchMatch holds a search result with its relevance score for sorting.
type profilesSearchMatch struct {
	result SearchResult
	score  int // Higher score = more relevant
	entry  profile.ListEntry
}

// Search implements Searchable.
// It searches profile names and descriptions, returning matching results.
func (p *ProfilesPlugin) Search(query string, maxResults int, sortOrder SortOrder) ([]SearchResult, error) {
	// Empty query returns empty results per contract
	if query == "" {
		return []SearchResult{}, nil
	}

	// Get all profiles
	entries, err := p.service.List()
	if err != nil {
		return nil, err
	}

	// Normalize query for case-insensitive search
	queryLower := strings.ToLower(query)

	// Find matching profiles
	var matches []profilesSearchMatch
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.Shadowed || seen[entry.Name] {
			continue
		}
		if m, ok := p.matchEntry(entry, queryLower); ok {
			seen[entry.Name] = true
			matches = append(matches, m)
		}
	}

	// Sort by the requested order
	p.sortMatches(matches, sortOrder)

	// Apply maxResults limit
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	// Extract results
	results := make([]SearchResult, len(matches))
	for i, m := range matches {
		results[i] = m.result
	}

	return results, nil
}

// matchEntry checks whether a profile entry matches the query and returns
// a profilesSearchMatch with a relevance score. Returns false if there is no match.
func (p *ProfilesPlugin) matchEntry(entry profile.ListEntry, queryLower string) (profilesSearchMatch, bool) {
	nameLower := strings.ToLower(entry.Name)
	descLower := strings.ToLower(entry.Description)

	nameMatch := strings.Contains(nameLower, queryLower)
	descMatch := strings.Contains(descLower, queryLower)

	if !nameMatch && !descMatch {
		return profilesSearchMatch{}, false
	}

	score := p.calculateRelevanceScore(nameLower, descLower, queryLower, nameMatch)
	return profilesSearchMatch{
		result: SearchResult{
			Title: entry.Name,
			URL:   "/" + entry.Name,
		},
		score: score,
		entry: entry,
	}, true
}

// calculateRelevanceScore calculates a relevance score for a match.
// Higher score = more relevant.
func (p *ProfilesPlugin) calculateRelevanceScore(nameLower, descLower, queryLower string, nameMatch bool) int {
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
func (p *ProfilesPlugin) sortMatches(matches []profilesSearchMatch, sortOrder SortOrder) {
	switch sortOrder {
	case SortName:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].entry.Name < matches[j].entry.Name
		})
	case SortCreatedDate, SortUpdatedDate, SortLastUsed:
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
