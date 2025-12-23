# Go Interface Contracts: Plugin Registration API

**Feature**: 012-plugin-registration-api
**Date**: 2025-12-22

## Package: internal/web

### WebPlugin Interface (Modified)

```go
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
```

### TemplatePlugin Interface (Unchanged)

```go
// TemplatePlugin is an optional extension for plugins that provide their own templates.
type TemplatePlugin interface {
    WebPlugin

    // Templates returns an fs.FS with plugin templates.
    // Templates should be in a "templates/" subdirectory.
    Templates() fs.FS
}
```

### PluginRegistry Type (Modified)

```go
// PluginRegistry is the central store for registered plugins.
type PluginRegistry struct {
    mu      sync.RWMutex
    plugins []registeredPlugin    // Maintains registration order
    byName  map[string]WebPlugin  // Fast lookup by name
    logger  *slog.Logger          // Registration logging
}

// registeredPlugin pairs a plugin with its registered name.
type registeredPlugin struct {
    name   string
    plugin WebPlugin
}
```

### Registration Function

```go
// Register adds a plugin to the registry with the given name.
//
// The name MUST:
//   - Be non-empty
//   - Match the pattern [a-z0-9]+(-[a-z0-9]+)* (lowercase alphanumeric with hyphens)
//   - Not already be registered
//
// Panics if validation fails (configuration error - fail fast at startup).
// Logs successful registration at Info level.
//
// Example:
//   web.Register("memory", memory.NewPlugin(storage))
//   web.Register("profiles", profiles.NewPlugin(service))
func (r *PluginRegistry) Register(name string, plugin WebPlugin)
```

### URL Helper Functions

```go
// PrefixURL prepends the plugin name to a relative URL path.
//
// Behavior:
//   - Absolute URLs (http://, https://) are returned unchanged
//   - URLs already prefixed with /{name}/ are returned unchanged
//   - Relative URLs are prefixed: "/foo" → "/{name}/foo"
//   - Bare paths are normalized: "foo" → "/{name}/foo"
//
// Example:
//   PrefixURL("memory", "/notes")     → "/memory/notes"
//   PrefixURL("memory", "/")          → "/memory/"
//   PrefixURL("memory", "https://x")  → "https://x"
//   PrefixURL("foo", "/foo/bar")      → "/foo/bar" (no double prefix)
func PrefixURL(pluginName, url string) string

// ValidatePluginName returns an error if the name is invalid.
// Returns nil if the name is valid for plugin registration.
func ValidatePluginName(name string) error
```

### Registry Methods

```go
// Get retrieves a plugin by its registered name.
func (r *PluginRegistry) Get(name string) (WebPlugin, bool)

// All returns all plugins with their names in registration order.
// Returns a slice of (name, plugin) pairs.
func (r *PluginRegistry) All() []registeredPlugin

// SidebarItems returns aggregated sidebar items from all plugins.
// Paths are automatically prefixed with each plugin's name.
// Items are sorted by Order field.
func (r *PluginRegistry) SidebarItems() []SidebarItem
```

---

## Package: internal/search

### Searchable Interface (Unchanged but Clarified)

```go
// Searchable is the interface for types that support search functionality.
//
// URL CONTRACT:
// When a WebPlugin implements Searchable, the Search() method MUST return
// plugin-relative URLs (e.g., "/notes" not "/memory/notes").
// The search aggregator prefixes URLs with the plugin name.
//
// This interface is independent of WebPlugin and can be implemented by any type.
type Searchable interface {
    // Search finds items matching the query string.
    //
    // URL REQUIREMENT: Return plugin-relative URLs (e.g., "/notes").
    // The search aggregator will prefix with the plugin name.
    Search(query string, maxResults int, sortOrder SortOrder) ([]SearchResult, error)
}
```

---

## Error Types

### InvalidPluginNameError (New)

```go
// InvalidPluginNameError is returned/used when a plugin name fails validation.
type InvalidPluginNameError struct {
    Name   string
    Reason string // e.g., "empty", "invalid characters", "must be lowercase"
}

func (e *InvalidPluginNameError) Error() string {
    return fmt.Sprintf("invalid plugin name %q: %s", e.Name, e.Reason)
}
```

### DuplicatePluginError (Renamed Field)

```go
// DuplicatePluginError is used when registering a plugin with an existing name.
type DuplicatePluginError struct {
    Name string  // Renamed from ID for consistency
}

func (e *DuplicatePluginError) Error() string {
    return "plugin already registered: " + e.Name
}
```

---

## Usage Examples

### Registering Plugins

```go
// In cmd/brains/main.go or similar

func setupPlugins(registry *web.PluginRegistry, storage memory.Storage, profileSvc *profile.Service) {
    // Registration panics on error (fail-fast for configuration errors)
    registry.Register("memory", memory.NewPlugin(storage))
    registry.Register("profiles", profiles.NewPlugin(profileSvc))
}
```

### Implementing a Plugin

```go
// In internal/webplugins/memory/plugin.go

type Plugin struct {
    storage memory.Storage
}

func NewPlugin(storage memory.Storage) *Plugin {
    return &Plugin{storage: storage}
}

// No more ID() method!

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{
        {
            ID:    "memory",
            Label: "Memory",
            Path:  "/",  // Relative! Will become "/memory/" after prefixing
            Order: 20,
        },
    }
}

func (p *Plugin) MountRoutes(r chi.Router) {
    h := newHandlers(p.storage)
    r.Get("/", h.list)           // Handles /memory/
    r.Get("/{name}", h.view)     // Handles /memory/{name}
}

func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
    // ... find matches ...
    return []search.SearchResult{
        {Title: "My Note", URL: "/my-note"},  // Relative! Will become "/memory/my-note"
    }, nil
}
```
