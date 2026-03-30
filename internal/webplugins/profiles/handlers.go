package profiles

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/web"
)

// handlers contains the HTTP handlers for the profiles plugin.
type handlers struct {
	service *profile.Service
}

// listData is the data passed to the list template.
type listData struct {
	Profiles []profile.ListEntry
	Error    string
}

// list handles GET /profiles - displays all available profiles.
func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	profiles, err := h.service.List()
	if err != nil {
		data := listData{
			Error: err.Error(),
		}
		renderer.Render(w, r, "profiles/list.html", data)
		return
	}

	data := listData{
		Profiles: profiles,
	}
	if err := renderer.Render(w, r, "profiles/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// viewData is the data passed to the view template.
type viewData struct {
	Profile *profile.ShowResult
	Error   string
}

// view handles GET /profiles/{name} - displays a single profile.
func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")
	result, err := h.service.Show(name, false)
	if err != nil {
		data := viewData{
			Error: err.Error(),
		}
		renderer.Render(w, r, "profiles/view.html", data)
		return
	}

	data := viewData{
		Profile: result,
	}
	if err := renderer.Render(w, r, "profiles/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
