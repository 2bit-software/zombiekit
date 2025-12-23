# Research: Plugin-Style Web GUI Architecture

**Feature**: 008-plugin-web-gui
**Date**: 2025-12-22

## Research Tasks

### 1. Chi Router Integration

**Decision**: Use `github.com/go-chi/chi/v5` for HTTP routing

**Rationale**:
- Lightweight, stdlib-compatible router that follows Go idioms
- Supports middleware chaining (needed for logging, recovery, renderer injection)
- URL parameter extraction via `chi.URLParam(r, "name")` for plugin routes
- Route grouping via `r.Route("/{plugin}", func(r chi.Router) {...})` for plugin namespacing
- Already familiar from mcp-genie research showing it's well-suited for SSR applications

**Alternatives Considered**:
- `http.ServeMux` (stdlib): Lacks middleware and URL parameter support needed for plugin routing
- `gin`: More opinionated, heavier, not stdlib-compatible
- `echo`: Similar to gin, more features than needed

**Best Practices**:
```go
// Mount plugins with route groups
r.Route("/"+pluginID, func(pr chi.Router) {
    pr.Use(rendererMiddleware)
    plugin.MountRoutes(pr)
})
```

### 2. HTMX Partial Update Detection

**Decision**: Check `HX-Request: true` header to distinguish partial vs full page requests

**Rationale**:
- HTMX automatically adds this header to all XHR requests
- Clean separation: same handler, different response based on header
- Supports graceful degradation when JavaScript is disabled
- `hx-push-url="true"` enables browser history for partial updates

**Implementation Pattern**:
```go
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, tmpl string, data any) {
    isHTMX := req.Header.Get("HX-Request") == "true"
    if isHTMX {
        // Render content template only
    } else {
        // Render shell with content embedded
    }
}
```

**Alternatives Considered**:
- Custom header: Requires JavaScript modification, less standard
- URL parameter: Pollutes URLs, breaks caching
- Accept header negotiation: More complex, less explicit

### 3. Template Organization for Plugins

**Decision**: Each plugin embeds its own templates via `go:embed`, renderer parses all at startup

**Rationale**:
- Keeps plugin templates co-located with plugin code
- Single template parse at startup for performance
- Namespacing via `{pluginID}/template.html` prevents collisions
- Optional `TemplatePlugin` interface for plugins that need custom templates

**Implementation Pattern**:
```go
// Plugin provides templates
type TemplatePlugin interface {
    WebPlugin
    Templates() fs.FS
}

// Renderer merges at startup
func NewRenderer(registry *PluginRegistry, shellFS fs.FS) (*Renderer, error) {
    tmpl := template.New("").ParseFS(shellFS, "templates/*.html")
    for _, p := range registry.All() {
        if tp, ok := p.(TemplatePlugin); ok {
            parsePluginTemplates(tmpl, p.ID(), tp.Templates())
        }
    }
    return &Renderer{templates: tmpl}, nil
}
```

### 4. Sidebar Aggregation Strategy

**Decision**: PluginRegistry collects SidebarItems from all plugins, sorts by Order field

**Rationale**:
- Single source of truth for navigation
- Plugins declare their own sidebar items (decentralized)
- Order field allows predictable sorting without coordination
- Children field supports nested navigation in future

**Implementation Pattern**:
```go
func (r *PluginRegistry) SidebarItems() []SidebarItem {
    var items []SidebarItem
    for _, p := range r.plugins {
        items = append(items, p.SidebarItems()...)
    }
    sort.Slice(items, func(i, j int) bool {
        return items[i].Order < items[j].Order
    })
    return items
}
```

### 5. Structured Logging Approach

**Decision**: Use Chi's middleware.Logger or custom slog-based logger

**Rationale**:
- Chi's logger provides request path, method, status, duration out of the box
- Can be replaced with slog-based logger for consistency with rest of codebase
- Existing `internal/logging` package provides slog setup

**Implementation Pattern**:
```go
// Option 1: Chi's built-in logger
r.Use(middleware.Logger)

// Option 2: Custom slog-based (if consistency needed)
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
        defer func() {
            slog.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.Status(),
                "duration", time.Since(start),
            )
        }()
        next.ServeHTTP(ww, r)
    })
})
```

### 6. Graceful Shutdown Pattern

**Decision**: Use context cancellation with timeout for graceful shutdown

**Rationale**:
- Standard Go pattern for HTTP servers
- Allows in-flight requests to complete
- 30-second timeout prevents hanging on stuck connections

**Implementation Pattern**:
```go
func (s *Server) Start(ctx context.Context) error {
    s.httpServer = &http.Server{
        Addr:    fmt.Sprintf(":%d", s.config.Port),
        Handler: s.router,
    }

    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        s.httpServer.Shutdown(shutdownCtx)
    }()

    return s.httpServer.ListenAndServe()
}
```

### 7. Static Asset Embedding

**Decision**: Use `go:embed` for all static assets with `fs.Sub` for correct path handling

**Rationale**:
- Single binary deployment (core requirement)
- No external file dependencies
- `fs.Sub` removes embed path prefix for clean URLs

**Implementation Pattern**:
```go
//go:embed static
var staticFS embed.FS

func setupStaticHandler(r chi.Router) {
    subFS, _ := fs.Sub(staticFS, "static")
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))
}
```

### 8. Profile Service Integration

**Decision**: Inject `*profile.Service` into profiles plugin via constructor

**Rationale**:
- Profile service already exists with List() and Show() methods
- Dependency injection enables testing with mocks
- No need to create new service layer

**Existing Service API**:
```go
// Already available from internal/profile
service.List() ([]ListEntry, error)
service.Show(name string, raw bool) (*ShowResult, error)
```

## Summary

All technical decisions are resolved. No NEEDS CLARIFICATION items remain. The architecture leverages:

1. **Chi router** for HTTP routing with middleware and route groups
2. **HX-Request header** for HTMX partial update detection
3. **go:embed** for template and static asset embedding
4. **Plugin interface** with optional TemplatePlugin extension
5. **PluginRegistry** for centralized plugin management and sidebar aggregation
6. **Context-based graceful shutdown** with 30-second timeout
7. **Existing profile.Service** for data access in example plugin
