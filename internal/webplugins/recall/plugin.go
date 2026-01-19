// Package recall provides a web plugin for viewing and searching conversation history.
package recall

import (
	"context"
	"embed"
	"io/fs"

	"github.com/go-chi/chi/v5"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/search"
	"github.com/zombiekit/brains/internal/web"
)

//go:embed templates
var templateFS embed.FS

// Ensure Plugin implements TemplatePlugin
var _ web.TemplatePlugin = (*Plugin)(nil)

// Ensure Plugin implements Searchable
var _ search.Searchable = (*Plugin)(nil)

// Plugin implements the WebPlugin interface for conversation history.
type Plugin struct {
	storage  recall.Storage
	embedder recall.Embedder
}

// NewPlugin creates a new recall plugin with the given storage and embedder.
// embedder can be nil if semantic search is unavailable.
func NewPlugin(storage recall.Storage, embedder recall.Embedder) *Plugin {
	return &Plugin{
		storage:  storage,
		embedder: embedder,
	}
}

// SidebarItems returns the navigation entries for the recall plugin.
func (p *Plugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{
			ID:    "conversations",
			Label: "Conversations",
			Path:  "/",
			Order: 30, // After memory (20)
		},
	}
}

// MountRoutes registers the HTTP handlers for the recall plugin.
func (p *Plugin) MountRoutes(r chi.Router) {
	h := newHandlers(p.storage, p.embedder)

	r.Get("/", h.list)
	r.Get("/search", h.search)
	r.Get("/{id}", h.view)
}

// Templates returns the embedded template filesystem.
func (p *Plugin) Templates() fs.FS {
	return templateFS
}

// Search implements search.Searchable for global search integration.
// It performs semantic search if embedder is available, returning conversation results.
func (p *Plugin) Search(query string, maxResults int, _ search.SortOrder) ([]search.SearchResult, error) {
	if query == "" {
		return []search.SearchResult{}, nil
	}

	// Check embedder availability - silent failure for global search
	if p.embedder == nil {
		return []search.SearchResult{}, nil
	}

	ctx := context.Background()

	// Generate embedding for query
	embedding, err := p.embedder.Embed(ctx, query, recall.PurposeQuery)
	if err != nil {
		return []search.SearchResult{}, nil // Silent failure
	}

	// Search
	limit := maxResults
	if limit <= 0 {
		limit = 10
	}
	results, err := p.storage.Search(ctx, embedding, limit)
	if err != nil {
		return []search.SearchResult{}, nil // Silent failure
	}

	// Convert to SearchResult, grouping by conversation (deduplicate)
	seen := make(map[string]bool)
	var searchResults []search.SearchResult

	for _, r := range results {
		if r.Chunk.ConversationID == "" || seen[r.Chunk.ConversationID] {
			continue
		}
		seen[r.Chunk.ConversationID] = true

		// Use first user message as title, truncated
		title := r.Chunk.Content
		if len(title) > 100 {
			title = title[:100] + "..."
		}

		// URL is relative to plugin root - framework auto-prefixes with "/recall"
		searchResults = append(searchResults, search.SearchResult{
			Title: title,
			URL:   "/" + r.Chunk.ConversationID,
		})
	}

	return searchResults, nil
}
