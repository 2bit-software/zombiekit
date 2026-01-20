package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/web"
)

func TestNewServer(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	server, err := web.NewServer(registry, web.DefaultServerConfig())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("server is nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	server, err := web.NewServer(registry, web.DefaultServerConfig())
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
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	server, err := web.NewServer(registry, web.DefaultServerConfig())
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
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	server, err := web.NewServer(registry, web.DefaultServerConfig())
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
	logging.InitLogger("info", false, os.Stderr)
	defer logging.ResetLogger()

	registry := web.NewPluginRegistry()

	// Test empty registry
	if len(registry.All()) != 0 {
		t.Errorf("expected empty registry, got %d plugins", len(registry.All()))
	}

	if len(registry.SidebarItems()) != 0 {
		t.Errorf("expected empty sidebar items, got %d", len(registry.SidebarItems()))
	}
}
