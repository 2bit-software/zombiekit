// Package memory provides a web plugin for viewing and managing sticky memories.
package memory

import (
	"context"
	"embed"
	"io/fs"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/2bit-software/zombiekit/internal/search"
	"github.com/2bit-software/zombiekit/internal/web"
)

//go:embed templates
var templateFS embed.FS

// Plugin implements the WebPlugin interface for sticky memories.
type Plugin struct {
	storage memory.Storage
}

// NewPlugin creates a new memory plugin with the given storage.
func NewPlugin(storage memory.Storage) *Plugin {
	return &Plugin{
		storage: storage,
	}
}

// SidebarItems returns the navigation entries for the memory plugin.
// Paths are relative - the system automatically prefixes with the plugin name.
func (p *Plugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{
			ID:    "memory",
			Label: "Memory",
			Path:  "/",
			Order: 20, // After profiles (10)
		},
	}
}

// MountRoutes registers the HTTP handlers for the memory plugin.
func (p *Plugin) MountRoutes(r chi.Router) {
	h := newHandlers(p.storage)

	// List and create
	r.Get("/", h.list)
	r.Get("/new", h.createForm)
	r.Post("/", h.create)

	// View, edit, delete by name
	r.Get("/{name}", h.view)
	r.Get("/{name}/edit", h.editForm)
	r.Put("/{name}", h.update)
	r.Get("/{name}/delete", h.deleteConfirm)
	r.Delete("/{name}", h.delete)
}

// Templates returns the embedded template filesystem.
func (p *Plugin) Templates() fs.FS {
	return templateFS
}

// Ensure Plugin implements TemplatePlugin
var _ web.TemplatePlugin = (*Plugin)(nil)

// Ensure Plugin implements Searchable
var _ search.Searchable = (*Plugin)(nil)

// memoryMatch holds a search result with metadata for sorting.
type memoryMatch struct {
	result   search.SearchResult
	score    int // Higher score = more relevant
	metadata memory.MemoryMetadata
}

// Search implements search.Searchable.
// It searches memory item names and content, returning matching results.
func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
	// Empty query returns empty results per contract
	if query == "" {
		return []search.SearchResult{}, nil
	}

	// Use storage's List with search query for name matching
	// Then we need to also search content for each item
	ctx := context.Background()

	// Get all items (empty search returns all)
	allItems, err := p.storage.List(ctx, "")
	if err != nil {
		return nil, err
	}

	// Normalize query for case-insensitive search
	queryLower := strings.ToLower(query)

	// Find matching items
	var matches []memoryMatch

	for _, meta := range allItems {
		nameLower := strings.ToLower(meta.Name)

		// Check name match
		nameMatch := strings.Contains(nameLower, queryLower)

		// Check content match by fetching the full item
		var contentMatch bool
		if !nameMatch {
			// Only fetch content if name didn't match (optimization)
			item, err := p.storage.Get(ctx, meta.Name)
			if err != nil {
				continue // Skip items we can't read
			}
			if item.HasValue() {
				contentLower := strings.ToLower(item.Value().Content)
				contentMatch = strings.Contains(contentLower, queryLower)
			}
		}

		if nameMatch || contentMatch {
			// Calculate relevance score
			score := p.calculateRelevanceScore(nameLower, queryLower, nameMatch)

			matches = append(matches, memoryMatch{
				result: search.SearchResult{
					Title: meta.Name,
					URL:   "/" + meta.Name, // Relative URL - system prefixes automatically
				},
				score:    score,
				metadata: meta,
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
func (p *Plugin) calculateRelevanceScore(nameLower, queryLower string, nameMatch bool) int {
	score := 0

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
func (p *Plugin) sortMatches(matches []memoryMatch, sortOrder search.SortOrder) {
	switch sortOrder {
	case search.SortName:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.Name < matches[j].metadata.Name
		})
	case search.SortCreatedDate:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.CreatedAt.After(matches[j].metadata.CreatedAt)
		})
	case search.SortUpdatedDate:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.UpdatedAt.After(matches[j].metadata.UpdatedAt)
		})
	case search.SortLastUsed:
		// Memory doesn't track last used, fall back to updated date
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.UpdatedAt.After(matches[j].metadata.UpdatedAt)
		})
	default: // SortRelevance or empty/invalid
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
	}
}
