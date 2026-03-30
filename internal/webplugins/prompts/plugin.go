package prompts

import (
	"embed"
	"io/fs"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/web"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

//go:embed templates
var templateFS embed.FS

// Plugin implements the WebPlugin interface for the unified prompts view.
type Plugin struct {
	profileSvc  *profile.Service
	stepSvc     *step.Service
	workflowSvc *workflow.Service
}

// NewPlugin creates a new prompts plugin with the given services.
func NewPlugin(profileSvc *profile.Service, stepSvc *step.Service, workflowSvc *workflow.Service) *Plugin {
	return &Plugin{
		profileSvc:  profileSvc,
		stepSvc:     stepSvc,
		workflowSvc: workflowSvc,
	}
}

// SidebarItems returns the navigation entries for the prompts plugin.
func (p *Plugin) SidebarItems() []web.SidebarItem {
	return []web.SidebarItem{
		{
			ID:    "prompts",
			Label: "Prompts",
			Path:  "/",
			Order: 15, // After profiles
		},
	}
}

// MountRoutes registers the HTTP handlers for the prompts plugin.
func (p *Plugin) MountRoutes(r chi.Router) {
	h := newHandlers(p.profileSvc, p.stepSvc, p.workflowSvc)
	r.Get("/", h.list)
	r.Get("/{category}/{name}", h.view)
}

// Templates returns the embedded template filesystem.
func (p *Plugin) Templates() fs.FS {
	return templateFS
}

// Ensure Plugin implements TemplatePlugin
var _ web.TemplatePlugin = (*Plugin)(nil)
