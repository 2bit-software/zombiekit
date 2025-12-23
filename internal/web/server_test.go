package web_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zombiekit/brains/internal/web"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := web.NewPluginRegistry(logger)

	server, err := web.NewServer(registry, web.DefaultServerConfig(), logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("server is nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := web.NewPluginRegistry(logger)

	server, err := web.NewServer(registry, web.DefaultServerConfig(), logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestHomeEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := web.NewPluginRegistry(logger)

	server, err := web.NewServer(registry, web.DefaultServerConfig(), logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestNotFoundEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := web.NewPluginRegistry(logger)

	server, err := web.NewServer(registry, web.DefaultServerConfig(), logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestPluginRegistry(t *testing.T) {
	registry := web.NewPluginRegistry(nil)

	// Test empty registry
	if len(registry.All()) != 0 {
		t.Errorf("expected empty registry, got %d plugins", len(registry.All()))
	}

	if len(registry.SidebarItems()) != 0 {
		t.Errorf("expected empty sidebar items, got %d", len(registry.SidebarItems()))
	}
}
