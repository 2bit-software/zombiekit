# Contract: Renderer

**Package**: `internal/web`

## Interface Definition

```go
package web

import (
    "html/template"
    "net/http"
)

// PageData is passed to all templates.
type PageData struct {
    Title        string        // Page title
    SidebarItems []SidebarItem // Navigation items
    ActivePath   string        // Current URL for highlighting
    Content      any           // Plugin-specific data
    IsHTMX       bool          // True if partial update request
}

// Renderer handles template rendering with HTMX detection.
type Renderer struct {
    templates *template.Template
    registry  *PluginRegistry
}

// NewRenderer creates a renderer with shell and plugin templates.
func NewRenderer(registry *PluginRegistry, shellFS fs.FS) (*Renderer, error)

// Render renders a template with automatic HTMX handling.
// - For HTMX requests (HX-Request header): renders content template only
// - For full page requests: wraps content in shell layout
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, tmplName string, data any) error

// RenderPartial renders a template fragment (always partial, ignores HX-Request).
// Useful for HTMX responses that update specific elements.
func (r *Renderer) RenderPartial(w http.ResponseWriter, tmplName string, data any) error
```

## Contract Guarantees

### HTMX Detection
- Checks `HX-Request: true` header
- Header present → content template only
- Header absent → shell + content

### Template Resolution
- Shell templates: `{name}.html` (e.g., "shell.html", "home.html")
- Plugin templates: `{pluginID}/{name}.html` (e.g., "profiles/list.html")

### Error Handling
- Template not found → returns error
- Template execution error → returns error (caller should handle)
- Always sets `Content-Type: text/html; charset=utf-8`

### PageData Population
- SidebarItems: populated from registry
- ActivePath: extracted from request URL
- IsHTMX: detected from header
- Content: passed through from caller
- Title: extracted from data if implements `PageTitle() string`

## Template Functions

```go
// Built-in template functions
funcMap := template.FuncMap{
    "isActive": func(current, path string) bool {
        // Returns true if current starts with path
    },
}
```
