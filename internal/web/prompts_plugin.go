package web

import (
	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

// PromptsPlugin implements the WebPlugin interface for the unified prompts view.
type PromptsPlugin struct {
	profileSvc  *profile.Service
	stepSvc     *step.Service
	workflowSvc *workflow.Service
}

// NewPromptsPlugin creates a new prompts plugin with the given services.
func NewPromptsPlugin(profileSvc *profile.Service, stepSvc *step.Service, workflowSvc *workflow.Service) *PromptsPlugin {
	return &PromptsPlugin{
		profileSvc:  profileSvc,
		stepSvc:     stepSvc,
		workflowSvc: workflowSvc,
	}
}

// SidebarItems returns the navigation entries for the prompts plugin.
func (p *PromptsPlugin) SidebarItems() []SidebarItem {
	return []SidebarItem{
		{
			ID:    "prompts",
			Label: "Prompts",
			Path:  "/",
			Order: 15, // After profiles
		},
	}
}

// MountRoutes registers the HTTP handlers for the prompts plugin.
func (p *PromptsPlugin) MountRoutes(r chi.Router) {
	h := newPromptsHandlers(p.profileSvc, p.stepSvc, p.workflowSvc)
	r.Get("/", h.list)
	r.Get("/{category}/{name}", h.view)
}
