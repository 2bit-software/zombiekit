# Contract: WebPlugin Interface

**Package**: `internal/web`

## Interface Definition

```go
package web

import (
    "io/fs"

    "github.com/go-chi/chi/v5"
)

// SidebarItem represents a navigation entry in the sidebar.
type SidebarItem struct {
    ID       string        // Unique identifier
    Label    string        // Display text
    Path     string        // URL path (e.g., "/profiles")
    Icon     string        // Icon name (optional)
    Order    int           // Sort order (lower = earlier)
    Badge    string        // Badge text (optional)
    Children []SidebarItem // Nested items (optional)
}

// WebPlugin is the interface that tools implement to participate in the web GUI.
type WebPlugin interface {
    // ID returns a unique identifier for this plugin.
    // Used for route namespacing: routes are mounted at /{ID}/...
    ID() string

    // SidebarItems returns navigation items for this plugin.
    // A plugin can return multiple items if it has multiple sections.
    SidebarItems() []SidebarItem

    // MountRoutes registers HTTP handlers on the provided router.
    // The router is already mounted at the plugin's base path (/{ID}).
    // Example: r.Get("/", listHandler) becomes GET /{ID}/
    MountRoutes(r chi.Router)
}

// TemplatePlugin is an optional interface for plugins that provide templates.
type TemplatePlugin interface {
    WebPlugin

    // Templates returns an embed.FS containing the plugin's templates.
    // Templates should be in a "templates/" subdirectory.
    // Returns nil if the plugin has no custom templates.
    Templates() fs.FS
}
```

## Contract Guarantees

### Plugin ID
- MUST be unique across all registered plugins
- MUST be URL-safe (alphanumeric, hyphens)
- MUST be stable (same ID across restarts)

### SidebarItems
- MAY return empty slice (plugin won't appear in nav)
- MUST return consistent items across calls
- Order values SHOULD be in increments of 10 for insertability

### MountRoutes
- MUST NOT modify routes outside the provided router scope
- SHOULD handle errors via recovery middleware
- MUST be safe for concurrent requests

### Templates (if implemented)
- MUST use `templates/` subdirectory in embed.FS
- SHOULD use plugin ID prefix in template names
- MUST be valid html/template syntax
