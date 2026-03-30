// Package web provides the HTTP server for the brains web interface.
package web

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/logging"
)

// pluginNamePattern validates plugin names: lowercase alphanumeric with hyphens.
// Examples: "memory", "my-plugin", "plugin2"
var pluginNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

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

// WebPlugin is the core interface that plugins implement to participate in the web GUI.
// Plugins receive their registered name via constructor injection, not via an interface method.
type WebPlugin interface {
	// SidebarItems returns navigation entries for this plugin.
	// Paths MUST be relative to the plugin's mount point (e.g., "/" or "/settings").
	// The system automatically prefixes paths with the plugin name.
	SidebarItems() []SidebarItem

	// MountRoutes registers HTTP handlers on the router.
	// The router is already scoped to /{pluginName}/.
	// Handlers should use relative paths (e.g., "/" for the plugin root).
	MountRoutes(r chi.Router)
}

// TemplatePlugin is an optional extension for plugins that provide their own templates.
type TemplatePlugin interface {
	WebPlugin

	// Templates returns an fs.FS with plugin templates.
	// Templates should be in a "templates/" subdirectory.
	Templates() fs.FS
}

// RegisteredPlugin pairs a plugin with its registered name.
type RegisteredPlugin struct {
	name   string
	plugin WebPlugin
}

// Name returns the registered name of the plugin.
func (rp RegisteredPlugin) Name() string {
	return rp.name
}

// Plugin returns the plugin instance.
func (rp RegisteredPlugin) Plugin() WebPlugin {
	return rp.plugin
}

// SidebarItems returns the plugin's sidebar items with paths prefixed.
func (rp RegisteredPlugin) SidebarItems() []SidebarItem {
	items := rp.plugin.SidebarItems()
	result := make([]SidebarItem, len(items))
	for i, item := range items {
		item.Path = PrefixURL(rp.name, item.Path)
		result[i] = item
	}
	return result
}

// PluginRegistry is the central store for registered plugins.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins []RegisteredPlugin   // Maintains registration order
	byName  map[string]WebPlugin // Fast lookup by name
}

// NewPluginRegistry creates a new empty plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make([]RegisteredPlugin, 0),
		byName:  make(map[string]WebPlugin),
	}
}

// Register adds a plugin to the registry with the given name.
//
// The name MUST:
//   - Be non-empty
//   - Match the pattern [a-z][a-z0-9]*(-[a-z0-9]+)* (lowercase alphanumeric with hyphens)
//   - Not already be registered
//
// Panics if validation fails (configuration error - fail fast at startup).
// Logs successful registration at Info level.
func (r *PluginRegistry) Register(name string, plugin WebPlugin) {
	if err := ValidatePluginName(name); err != nil {
		panic(err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byName[name]; exists {
		panic(&DuplicatePluginError{Name: name})
	}

	r.plugins = append(r.plugins, RegisteredPlugin{name: name, plugin: plugin})
	r.byName[name] = plugin

	logging.Logger().Info("registered plugin", "name", name, "path", "/"+name)
}

// Get retrieves a plugin by its registered name.
func (r *PluginRegistry) Get(name string) (WebPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.byName[name]
	return p, exists
}

// All returns all plugins with their names in registration order.
func (r *PluginRegistry) All() []RegisteredPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]RegisteredPlugin, len(r.plugins))
	copy(result, r.plugins)
	return result
}

// SidebarItems returns aggregated, sorted sidebar items from all plugins.
// Paths are automatically prefixed with each plugin's name.
func (r *PluginRegistry) SidebarItems() []SidebarItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []SidebarItem
	for _, rp := range r.plugins {
		for _, item := range rp.plugin.SidebarItems() {
			// Prefix the path with the plugin name
			item.Path = PrefixURL(rp.name, item.Path)
			items = append(items, item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})

	return items
}

// DuplicatePluginError is used when registering a plugin with an existing name.
type DuplicatePluginError struct {
	Name string
}

func (e *DuplicatePluginError) Error() string {
	return "plugin already registered: " + e.Name
}

// InvalidPluginNameError is returned/used when a plugin name fails validation.
type InvalidPluginNameError struct {
	Name   string
	Reason string
}

func (e *InvalidPluginNameError) Error() string {
	return fmt.Sprintf("invalid plugin name %q: %s", e.Name, e.Reason)
}

// ValidatePluginName returns an error if the name is invalid.
// Returns nil if the name is valid for plugin registration.
func ValidatePluginName(name string) error {
	if name == "" {
		return &InvalidPluginNameError{Name: name, Reason: "name cannot be empty"}
	}
	if !pluginNamePattern.MatchString(name) {
		return &InvalidPluginNameError{
			Name:   name,
			Reason: "must be lowercase alphanumeric with hyphens (e.g., 'my-plugin')",
		}
	}
	return nil
}

// PrefixURL prepends the plugin name to a relative URL path.
//
// Behavior:
//   - Absolute URLs (http://, https://) are returned unchanged
//   - URLs already prefixed with /{name}/ are returned unchanged
//   - Relative URLs are prefixed: "/foo" → "/{name}/foo"
//   - Bare paths are normalized: "foo" → "/{name}/foo"
func PrefixURL(pluginName, url string) string {
	// Absolute URLs pass through unchanged
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	prefix := "/" + pluginName

	// Already prefixed - return unchanged
	if strings.HasPrefix(url, prefix+"/") || url == prefix {
		return url
	}

	// Normalize: ensure URL starts with /
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	return prefix + url
}
