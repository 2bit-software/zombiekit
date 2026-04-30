package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/go-chi/chi/v5"
)

//go:embed static
var staticFS embed.FS

// ServerConfig holds configuration for the web server.
type ServerConfig struct {
	Port         int           // HTTP listen port (default: 8080)
	ReadTimeout  time.Duration // Max time to read request (default: 15s)
	WriteTimeout time.Duration // Max time to write response (default: 15s)
	IdleTimeout  time.Duration // Max keep-alive idle time (default: 60s)
	StatusConfig StatusConfig  // Status page configuration
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Server is the HTTP server for the web interface.
type Server struct {
	config     ServerConfig
	router     chi.Router
	registry   *PluginRegistry
	renderer   *Renderer
	httpServer *http.Server
	startTime  time.Time
}

// NewServer creates a new web server with the given registry and configuration.
func NewServer(registry *PluginRegistry, config ServerConfig) (*Server, error) {
	// Apply defaults for zero values
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 15 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 60 * time.Second
	}

	// Create renderer
	renderer, err := NewRenderer(registry)
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	s := &Server{
		config:    config,
		registry:  registry,
		renderer:  renderer,
		startTime: time.Now(),
	}

	s.setupRouter()
	return s, nil
}

// setupRouter configures the Chi router with middleware and routes.
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Apply middleware
	r.Use(s.loggingMiddleware)
	r.Use(s.recoveryMiddleware)
	r.Use(s.rendererMiddleware)

	// Static assets
	s.setupStaticHandler(r)

	// Health check
	r.Get("/health", s.healthHandler)

	// Home page
	r.Get("/", s.homeHandler)

	// Search endpoint
	r.Get("/search", s.searchHandler)

	// Mount plugins
	for _, rp := range s.registry.All() {
		name := rp.Name()
		plugin := rp.Plugin()
		r.Route("/"+name, func(pr chi.Router) {
			plugin.MountRoutes(pr)
		})
	}

	// 404 handler
	r.NotFound(s.notFoundHandler)

	s.router = r
}

// setupStaticHandler configures the static file handler.
func (s *Server) setupStaticHandler(r chi.Router) {
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		logging.Logger().Error("failed to create static sub-filesystem", "error", err)
		return
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))
}

// healthHandler returns a simple health check response.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// homeHandler renders the dashboard home page.
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	// Build status config with current start time
	statusCfg := s.config.StatusConfig
	statusCfg.StartTime = s.startTime

	// Gather status information
	status := GatherStatus(r.Context(), statusCfg, s.registry)

	data := map[string]any{
		"Plugins": s.registry.All(),
		"Status":  status,
	}
	if err := s.renderer.Render(w, r, "home.html", data); err != nil {
		logging.Logger().Error("failed to render home page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// notFoundHandler renders the 404 page.
func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := s.renderer.Render(w, r, "404.html", nil); err != nil {
		logging.Logger().Error("failed to render 404 page", "error", err)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// Router returns the underlying Chi router for testing.
func (s *Server) Router() chi.Router {
	return s.router
}

// Start starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Graceful shutdown goroutine
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logging.Logger().Error("server shutdown error", "error", err)
		}
	}()

	logging.Logger().Info("starting web server", "port", s.config.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
