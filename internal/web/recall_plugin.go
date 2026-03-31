// Package web provides the HTTP server for the brains web interface.
package web

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/2bit-software/zombiekit/internal/recall"
)

// RecallPlugin implements the WebPlugin interface for conversation history.
type RecallPlugin struct {
	storage  recall.Storage
	embedder recall.Embedder
}

// NewRecallPlugin creates a new recall plugin with the given storage and embedder.
// embedder can be nil if semantic search is unavailable.
func NewRecallPlugin(storage recall.Storage, embedder recall.Embedder) *RecallPlugin {
	return &RecallPlugin{
		storage:  storage,
		embedder: embedder,
	}
}

// SidebarItems returns the navigation entries for the recall plugin.
func (p *RecallPlugin) SidebarItems() []SidebarItem {
	return []SidebarItem{
		{
			ID:    "conversations",
			Label: "Conversations",
			Path:  "/",
			Order: 30, // After memory (20)
		},
	}
}

// MountRoutes registers the HTTP handlers for the recall plugin.
func (p *RecallPlugin) MountRoutes(r chi.Router) {
	h := newRecallHandlers(p.storage, p.embedder)

	r.Get("/", h.list)
	r.Get("/search", h.search)
	r.Get("/{id}", h.view)
}

// Search implements Searchable for global search integration.
// It performs semantic search if embedder is available, returning conversation results.
func (p *RecallPlugin) Search(query string, maxResults int, _ SortOrder) ([]SearchResult, error) {
	if query == "" || p.embedder == nil {
		return []SearchResult{}, nil
	}

	ctx := context.Background()

	embedding, err := p.embedder.Embed(ctx, query, recall.PurposeQuery)
	if err != nil {
		return []SearchResult{}, nil // Silent failure
	}

	if maxResults <= 0 {
		maxResults = 10
	}
	results, err := p.storage.Search(ctx, embedding, maxResults)
	if err != nil {
		return []SearchResult{}, nil // Silent failure
	}

	seen := make(map[string]bool)
	var searchResults []SearchResult
	for _, r := range results {
		if r.Chunk.ConversationID == "" || seen[r.Chunk.ConversationID] {
			continue
		}
		seen[r.Chunk.ConversationID] = true

		title := r.Chunk.Content
		if len(title) > 100 {
			title = title[:100] + "..."
		}

		searchResults = append(searchResults, SearchResult{
			Title: title,
			URL:   "/" + r.Chunk.ConversationID,
		})
	}

	return searchResults, nil
}
