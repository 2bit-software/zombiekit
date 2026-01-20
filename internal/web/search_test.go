package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/search"
	"github.com/zombiekit/brains/internal/web"
)

// mockSearchablePlugin is a test plugin that implements Searchable.
type mockSearchablePlugin struct {
	name    string
	label   string
	results []search.SearchResult
}

func (p *mockSearchablePlugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{ID: p.name, Label: p.label, Path: "/"},
	}
}

func (p *mockSearchablePlugin) MountRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plugin content"))
	})
}

func (p *mockSearchablePlugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
	if query == "" {
		return []search.SearchResult{}, nil
	}

	// Filter and limit results
	var filtered []search.SearchResult
	for _, r := range p.results {
		if strings.Contains(strings.ToLower(r.Title), strings.ToLower(query)) {
			filtered = append(filtered, r)
			if maxResults > 0 && len(filtered) >= maxResults {
				break
			}
		}
	}
	return filtered, nil
}

// mockNonSearchablePlugin is a test plugin that does NOT implement Searchable.
type mockNonSearchablePlugin struct {
	name string
}

func (p *mockNonSearchablePlugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{ID: p.name, Label: "Non-Searchable", Path: "/"},
	}
}

func (p *mockNonSearchablePlugin) MountRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("non-searchable content"))
	})
}

func TestSearchHandler_EmptyQuery(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String(), "empty query should return empty response")
}

func TestSearchHandler_NoSearchablePlugins(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()
	registry.Register("nonsearchable", &mockNonSearchablePlugin{name: "nonsearchable"})

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No results found")
}

func TestSearchHandler_WithResults(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()
	registry.Register("memory", &mockSearchablePlugin{
		name:  "memory",
		label: "Memory",
		results: []search.SearchResult{
			{Title: "config-test", URL: "/config-test"},
			{Title: "test-notes", URL: "/test-notes"},
		},
	})

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Memory", "should show plugin name")
	assert.Contains(t, body, "config-test", "should show result title")
	assert.Contains(t, body, "test-notes", "should show result title")
}

func TestSearchHandler_MultiplePlugins(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	registry.Register("memory", &mockSearchablePlugin{
		name:  "memory",
		label: "Memory",
		results: []search.SearchResult{
			{Title: "memory-config", URL: "/memory-config"},
		},
	})
	registry.Register("profiles", &mockSearchablePlugin{
		name:  "profiles",
		label: "Profiles",
		results: []search.SearchResult{
			{Title: "profile-config", URL: "/profile-config"},
		},
	})

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=config", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Memory", "should show Memory plugin")
	assert.Contains(t, body, "Profiles", "should show Profiles plugin")
	assert.Contains(t, body, "memory-config", "should show memory result")
	assert.Contains(t, body, "profile-config", "should show profiles result")
}

func TestSearchHandler_LimitsToThreeResults(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	registry.Register("memory", &mockSearchablePlugin{
		name:  "memory",
		label: "Memory",
		results: []search.SearchResult{
			{Title: "item1", URL: "/item1"},
			{Title: "item2", URL: "/item2"},
			{Title: "item3", URL: "/item3"},
			{Title: "item4", URL: "/item4"},
			{Title: "item5", URL: "/item5"},
		},
	})

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=item", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "item1")
	assert.Contains(t, body, "item2")
	assert.Contains(t, body, "item3")
	assert.NotContains(t, body, "item4", "should limit to 3 results")
	assert.NotContains(t, body, "item5", "should limit to 3 results")
}

func TestSearchHandler_URLPrefixing(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	registry.Register("memory", &mockSearchablePlugin{
		name:  "memory",
		label: "Memory",
		results: []search.SearchResult{
			{Title: "config-item", URL: "/config-item"},
		},
	})

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/search?q=config", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// Verify URL has plugin prefix
	assert.Contains(t, body, "/memory/config-item", "URL should be prefixed with plugin name")
}
