# Quickstart: Plugin-Style Web GUI

**Feature**: 008-plugin-web-gui

## Prerequisites

- Go 1.24+
- Existing `brains` CLI with profile service

## Running the Web GUI

```bash
# Start the web server (default port 8080)
brains gui

# With custom port
brains gui --port 3000

# Open in browser
open http://localhost:8080
```

## Creating a New Plugin

### 1. Create Plugin Package

```bash
mkdir -p internal/webplugins/myplugin
```

### 2. Implement WebPlugin Interface

```go
// internal/webplugins/myplugin/plugin.go
package myplugin

import (
    "embed"
    "io/fs"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/2bit-software/zombiekit/internal/web"
)

//go:embed templates
var templateFS embed.FS

type Plugin struct {
    // Your dependencies here
}

func NewPlugin() *Plugin {
    return &Plugin{}
}

func (p *Plugin) ID() string {
    return "myplugin"
}

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{
        {
            ID:    "myplugin",
            Label: "My Plugin",
            Path:  "/myplugin",
            Order: 20, // Use increments of 10
        },
    }
}

func (p *Plugin) MountRoutes(r chi.Router) {
    r.Get("/", p.listHandler)
    r.Get("/{id}", p.viewHandler)
}

func (p *Plugin) Templates() fs.FS {
    return templateFS
}

func (p *Plugin) listHandler(w http.ResponseWriter, r *http.Request) {
    renderer := web.GetRenderer(r)
    data := map[string]any{
        "Items": []string{"Item 1", "Item 2"},
    }
    renderer.Render(w, r, "myplugin/list.html", data)
}

func (p *Plugin) viewHandler(w http.ResponseWriter, r *http.Request) {
    renderer := web.GetRenderer(r)
    id := chi.URLParam(r, "id")
    data := map[string]any{
        "ID": id,
    }
    renderer.Render(w, r, "myplugin/view.html", data)
}
```

### 3. Create Templates

```html
<!-- internal/webplugins/myplugin/templates/list.html -->
<div class="bg-white rounded-lg shadow p-6">
    <h1 class="text-xl font-semibold mb-4">My Plugin</h1>
    <ul class="divide-y">
        {{range .Content.Items}}
        <li class="py-2">
            <a href="/myplugin/{{.}}"
               hx-get="/myplugin/{{.}}"
               hx-target="#content"
               hx-push-url="true"
               class="text-blue-600 hover:underline">
                {{.}}
            </a>
        </li>
        {{end}}
    </ul>
</div>
```

```html
<!-- internal/webplugins/myplugin/templates/view.html -->
<div class="bg-white rounded-lg shadow p-6">
    <a href="/myplugin"
       hx-get="/myplugin"
       hx-target="#content"
       hx-push-url="true"
       class="text-gray-500 hover:text-gray-700 mb-4 inline-block">
        ← Back to list
    </a>
    <h1 class="text-xl font-semibold">Item: {{.Content.ID}}</h1>
</div>
```

### 4. Register Plugin

```go
// internal/cli/serve.go (in setupWebServer function)
import (
    "github.com/2bit-software/zombiekit/internal/webplugins/myplugin"
)

func setupWebServer() (*web.Server, error) {
    registry := web.NewPluginRegistry()

    // Register your plugin
    registry.Register(myplugin.NewPlugin())

    return web.NewServer(registry, web.ServerConfig{Port: 8080})
}
```

## HTMX Patterns

### Partial Navigation (sidebar clicks)

```html
<a href="/profiles"
   hx-get="/profiles"
   hx-target="#content"
   hx-push-url="true">
    Profiles
</a>
```

### List Item Click

```html
<a href="/profiles/{{.Name}}"
   hx-get="/profiles/{{.Name}}"
   hx-target="#content"
   hx-push-url="true">
    {{.Name}}
</a>
```

### Form Submission

```html
<form hx-post="/myplugin/create"
      hx-target="#content"
      hx-push-url="true">
    <input name="name" type="text">
    <button type="submit">Create</button>
</form>
```

## Template Data

All templates receive `PageData`:

```go
type PageData struct {
    Title        string        // Page title
    SidebarItems []SidebarItem // For navigation
    ActivePath   string        // Current path
    Content      any           // Your handler data
    IsHTMX       bool          // True if partial request
}
```

Access your data via `.Content`:

```html
{{range .Content.Profiles}}
    <div>{{.Name}}</div>
{{end}}
```

## Testing

```go
func TestPluginRoutes(t *testing.T) {
    registry := web.NewPluginRegistry()
    registry.Register(myplugin.NewPlugin())

    server, _ := web.NewServer(registry, web.ServerConfig{Port: 0})

    // Test list endpoint
    req := httptest.NewRequest("GET", "/myplugin/", nil)
    w := httptest.NewRecorder()
    server.Router().ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
}
```
