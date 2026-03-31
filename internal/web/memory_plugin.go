package web

import (
	"context"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/memory"
)

// MemoryPlugin implements the WebPlugin interface for sticky memories.
type MemoryPlugin struct {
	storage memory.Storage
}

// NewMemoryPlugin creates a new memory plugin with the given storage.
func NewMemoryPlugin(storage memory.Storage) *MemoryPlugin {
	return &MemoryPlugin{
		storage: storage,
	}
}

// SidebarItems returns the navigation entries for the memory plugin.
// Paths are relative - the system automatically prefixes with the plugin name.
func (p *MemoryPlugin) SidebarItems() []SidebarItem {
	return []SidebarItem{
		{
			ID:    "memory",
			Label: "Memory",
			Path:  "/",
			Order: 20, // After profiles (10)
		},
	}
}

// MountRoutes registers the HTTP handlers for the memory plugin.
func (p *MemoryPlugin) MountRoutes(r chi.Router) {
	h := newMemoryHandlers(p.storage)

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

var _ WebPlugin = (*MemoryPlugin)(nil)

// Ensure MemoryPlugin implements Searchable
var _ Searchable = (*MemoryPlugin)(nil)

// memoryMatch holds a search result with metadata for sorting.
type memoryMatch struct {
	result   SearchResult
	score    int // Higher score = more relevant
	metadata memory.MemoryMetadata
}

// Search implements Searchable.
// It searches memory item names and content, returning matching results.
func (p *MemoryPlugin) Search(query string, maxResults int, sortOrder SortOrder) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	ctx := context.Background()

	allItems, err := p.storage.List(ctx, "")
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	matches := p.findMatches(ctx, allItems, queryLower)

	p.sortMatches(matches, sortOrder)

	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	results := make([]SearchResult, len(matches))
	for i, m := range matches {
		results[i] = m.result
	}

	return results, nil
}

// findMatches tests each item for name or content match and returns scored results.
func (p *MemoryPlugin) findMatches(ctx context.Context, items []memory.MemoryMetadata, queryLower string) []memoryMatch {
	var matches []memoryMatch

	for _, meta := range items {
		if m, ok := p.matchItem(ctx, meta, queryLower); ok {
			matches = append(matches, m)
		}
	}
	return matches
}

// matchItem checks whether a single item matches the query by name or content.
func (p *MemoryPlugin) matchItem(ctx context.Context, meta memory.MemoryMetadata, queryLower string) (memoryMatch, bool) {
	nameLower := strings.ToLower(meta.Name)
	nameMatch := strings.Contains(nameLower, queryLower)

	if !nameMatch && !p.contentMatches(ctx, meta.Name, queryLower) {
		return memoryMatch{}, false
	}

	score := p.calculateRelevanceScore(nameLower, queryLower, nameMatch)
	return memoryMatch{
		result: SearchResult{
			Title: meta.Name,
			URL:   "/" + meta.Name,
		},
		score:    score,
		metadata: meta,
	}, true
}

// contentMatches reports whether the item's content contains the query string.
func (p *MemoryPlugin) contentMatches(ctx context.Context, name, queryLower string) bool {
	item, err := p.storage.Get(ctx, name)
	if err != nil || !item.HasValue() {
		return false
	}
	return strings.Contains(strings.ToLower(item.Value().Content), queryLower)
}

// calculateRelevanceScore calculates a relevance score for a match.
func (p *MemoryPlugin) calculateRelevanceScore(nameLower, queryLower string, nameMatch bool) int {
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
func (p *MemoryPlugin) sortMatches(matches []memoryMatch, sortOrder SortOrder) {
	switch sortOrder {
	case SortName:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.Name < matches[j].metadata.Name
		})
	case SortCreatedDate:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.CreatedAt.After(matches[j].metadata.CreatedAt)
		})
	case SortUpdatedDate:
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].metadata.UpdatedAt.After(matches[j].metadata.UpdatedAt)
		})
	case SortLastUsed:
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
