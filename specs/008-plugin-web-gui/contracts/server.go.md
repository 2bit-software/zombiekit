# Contract: Server

**Package**: `internal/web`

## Interface Definition

```go
package web

import (
    "context"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
    Port         int           // HTTP listen port (default: 8080)
    ReadTimeout  time.Duration // Max request read time (default: 15s)
    WriteTimeout time.Duration // Max response write time (default: 15s)
    IdleTimeout  time.Duration // Max keep-alive idle time (default: 60s)
}

// Server is the web GUI server.
type Server struct {
    config     ServerConfig
    registry   *PluginRegistry
    renderer   *Renderer
    router     chi.Router
    httpServer *http.Server
}

// NewServer creates a new web server with the given plugins and config.
func NewServer(registry *PluginRegistry, config ServerConfig) (*Server, error)

// Start starts the server and blocks until context is cancelled.
// Returns http.ErrServerClosed on graceful shutdown.
func (s *Server) Start(ctx context.Context) error

// Router returns the underlying chi.Router for testing.
func (s *Server) Router() chi.Router
```

## Route Structure

```
GET  /                    → Dashboard (home.html)
GET  /health              → Health check (JSON)
GET  /static/*            → Static assets (embed.FS)
GET  /{pluginID}/*        → Plugin routes (delegated)
```

## Contract Guarantees

### Startup
- Parses all templates at creation time
- Returns error if templates invalid
- Does not start listening until Start() called

### Shutdown
- Context cancellation triggers graceful shutdown
- 30-second timeout for in-flight requests
- Returns `http.ErrServerClosed` on successful shutdown

### Middleware Chain
- RequestID (generates unique ID per request)
- RealIP (extracts client IP from headers)
- Logger (structured request logging)
- Recoverer (panic recovery, returns 500)
- Compress (gzip compression)
- RendererContext (injects renderer for handlers)

### Plugin Mounting
- Each plugin mounted at `/{pluginID}/`
- Plugin's MountRoutes receives scoped router
- Renderer available via `web.GetRenderer(r)`

## Context Values

```go
// GetRenderer retrieves the renderer from request context.
// Must be called within a plugin handler.
func GetRenderer(r *http.Request) *Renderer

// Context key for renderer
type contextKey string
const rendererKey contextKey = "renderer"
```
