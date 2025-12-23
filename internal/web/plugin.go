// Package web provides the HTTP server for the brains web interface.
package web

import (
	"io/fs"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
)

// SidebarItem represents a navigation entry in the sidebar.
type SidebarItem struct {
	ID       string        // Unique identifier for this item
	Label    string        // Display text in navigation
	Path     string        // URL path (e.g., "/profiles")
	Icon     string        // Icon name (for future use)
	Order    int           // Sort order (lower = earlier)
	Badge    string        // Optional badge text (e.g., count)
	Children []SidebarItem // Nested items for expandable menus
}

// WebPlugin is the core interface that tools implement to participate in the web GUI.
type WebPlugin interface {
	// ID returns a unique identifier for this plugin.
	// Routes are mounted at /{ID}/...
	ID() string

	// SidebarItems returns navigation entries for this plugin.
	SidebarItems() []SidebarItem

	// MountRoutes registers HTTP handlers on the router.
	// Router is already scoped to /{ID}/
	MountRoutes(r chi.Router)
}

// TemplatePlugin is an optional extension for plugins that provide their own templates.
type TemplatePlugin interface {
	WebPlugin

	// Templates returns an fs.FS with plugin templates.
	// Templates should be in a "templates/" subdirectory.
	Templates() fs.FS
}

// PluginRegistry is the central store for registered plugins.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins []WebPlugin
	byID    map[string]WebPlugin
}

// NewPluginRegistry creates a new empty plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make([]WebPlugin, 0),
		byID:    make(map[string]WebPlugin),
	}
}

// Register adds a plugin to the registry.
// Returns an error if a plugin with the same ID already exists.
func (r *PluginRegistry) Register(p WebPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := p.ID()
	if _, exists := r.byID[id]; exists {
		return &DuplicatePluginError{ID: id}
	}

	r.plugins = append(r.plugins, p)
	r.byID[id] = p
	return nil
}

// Get retrieves a plugin by ID.
func (r *PluginRegistry) Get(id string) (WebPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.byID[id]
	return p, exists
}

// All returns all plugins in registration order.
func (r *PluginRegistry) All() []WebPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]WebPlugin, len(r.plugins))
	copy(result, r.plugins)
	return result
}

// SidebarItems returns aggregated, sorted sidebar items from all plugins.
func (r *PluginRegistry) SidebarItems() []SidebarItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []SidebarItem
	for _, p := range r.plugins {
		items = append(items, p.SidebarItems()...)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})

	return items
}

// DuplicatePluginError is returned when registering a plugin with an existing ID.
type DuplicatePluginError struct {
	ID string
}

func (e *DuplicatePluginError) Error() string {
	return "plugin already registered: " + e.ID
}
