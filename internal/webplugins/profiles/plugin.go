// Package profiles provides a web plugin for viewing and managing profiles.
package profiles

import (
	"embed"
	"io/fs"

	"github.com/go-chi/chi/v5"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/web"
)

//go:embed templates
var templateFS embed.FS

// Plugin implements the WebPlugin interface for profiles.
type Plugin struct {
	service *profile.Service
}

// NewPlugin creates a new profiles plugin with the given profile service.
func NewPlugin(service *profile.Service) *Plugin {
	return &Plugin{
		service: service,
	}
}

// ID returns the unique identifier for this plugin.
func (p *Plugin) ID() string {
	return "profiles"
}

// SidebarItems returns the navigation entries for the profiles plugin.
func (p *Plugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{
			ID:    "profiles",
			Label: "Profiles",
			Path:  "/profiles",
			Order: 10, // First in the list
		},
	}
}

// MountRoutes registers the HTTP handlers for the profiles plugin.
func (p *Plugin) MountRoutes(r chi.Router) {
	h := &handlers{service: p.service}
	r.Get("/", h.list)
	r.Get("/{name}", h.view)
}

// Templates returns the embedded template filesystem.
func (p *Plugin) Templates() fs.FS {
	return templateFS
}

// Ensure Plugin implements TemplatePlugin
var _ web.TemplatePlugin = (*Plugin)(nil)
