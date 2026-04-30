package web

import (
	"net/http"

	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/go-chi/chi/v5"
)

// profilesHandlers contains the HTTP handlers for the profiles plugin.
type profilesHandlers struct {
	service *profile.Service
}

// profilesListData is the data passed to the list template.
type profilesListData struct {
	Profiles []profile.ListEntry
	Error    string
}

// list handles GET /profiles - displays all available profiles.
func (h *profilesHandlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	profiles, err := h.service.List()
	if err != nil {
		data := profilesListData{
			Error: err.Error(),
		}
		renderer.Render(w, r, "profiles/list.html", data)
		return
	}

	data := profilesListData{
		Profiles: profiles,
	}
	if err := renderer.Render(w, r, "profiles/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// profilesViewData is the data passed to the view template.
type profilesViewData struct {
	Profile *profile.ShowResult
	Error   string
}

// view handles GET /profiles/{name} - displays a single profile.
func (h *profilesHandlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")
	result, err := h.service.Show(name, false)
	if err != nil {
		data := profilesViewData{
			Error: err.Error(),
		}
		renderer.Render(w, r, "profiles/view.html", data)
		return
	}

	data := profilesViewData{
		Profile: result,
	}
	if err := renderer.Render(w, r, "profiles/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
