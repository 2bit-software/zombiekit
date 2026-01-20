package web

import (
	"net/http"

	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/search"
)

// PluginSearchResult groups search results with their source plugin metadata.
type PluginSearchResult struct {
	// PluginID is the plugin name from RegisteredPlugin.Name().
	PluginID string
	// PluginName is the human-readable label from SidebarItems()[0].Label.
	PluginName string
	// Items contains up to 3 results from this plugin.
	Items []search.SearchResult
}

// SearchResponse is the aggregated response for the search endpoint.
type SearchResponse struct {
	// Query is the original search query.
	Query string
	// Results are grouped by plugin.
	Results []PluginSearchResult
	// HasAny is true if any plugin returned results.
	HasAny bool
}

// getPluginLabel extracts the human-readable name from the first SidebarItem.
// Falls back to the plugin ID if no sidebar items exist.
func getPluginLabel(rp RegisteredPlugin) string {
	items := rp.Plugin().SidebarItems()
	if len(items) > 0 && items[0].Label != "" {
		return items[0].Label
	}
	return rp.Name()
}

// searchPlugins aggregates search results from all Searchable plugins.
func (s *Server) searchPlugins(query string, maxResultsPerPlugin int) SearchResponse {
	response := SearchResponse{
		Query:   query,
		Results: []PluginSearchResult{},
		HasAny:  false,
	}

	// Empty query returns empty response
	if query == "" {
		return response
	}

	for _, rp := range s.registry.All() {
		searchable, ok := rp.Plugin().(search.Searchable)
		if !ok {
			continue
		}

		items, err := searchable.Search(query, maxResultsPerPlugin, search.SortRelevance)
		if err != nil {
			logging.Logger().Error("search failed", "plugin", rp.Name(), "error", err)
			continue
		}

		// Skip plugins with no results
		if len(items) == 0 {
			continue
		}

		// Prefix URLs with plugin name
		for i := range items {
			items[i].URL = PrefixURL(rp.Name(), items[i].URL)
		}

		response.Results = append(response.Results, PluginSearchResult{
			PluginID:   rp.Name(),
			PluginName: getPluginLabel(rp),
			Items:      items,
		})
		response.HasAny = true
	}

	return response
}

// searchHandler handles the GET /search endpoint.
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	// Empty query returns empty response (no HTML)
	if query == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	response := s.searchPlugins(query, 3)

	// Render the search results template
	if err := s.renderer.RenderPartial(w, "search-results.html", response); err != nil {
		logging.Logger().Error("failed to render search results", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
