package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/go-chi/chi/v5"
)

// DefaultPageLimit is the default number of items per page.
const DefaultPageLimit = 20

// PageLimitOptions are the available pagination limit choices.
var PageLimitOptions = []int{10, 20, 50, 100}

// memoryHandlers contains the HTTP handlers for the memory plugin.
type memoryHandlers struct {
	storage memory.Storage
}

// newMemoryHandlers creates a new memoryHandlers instance.
func newMemoryHandlers(storage memory.Storage) *memoryHandlers {
	return &memoryHandlers{storage: storage}
}

// MemoryListData is the data passed to the list template.
type MemoryListData struct {
	Memories   []memory.MemoryMetadata
	Pagination MemoryPaginationData
	Query      string
	Error      string
}

// MemoryViewData is the data passed to the view template.
type MemoryViewData struct {
	Memory        *memory.MemoryItem
	FormattedSize string
	Error         string
}

// MemoryFormData is the data passed to the form template.
type MemoryFormData struct {
	Name    string
	Content string
	Error   string
	IsEdit  bool
}

// MemoryDeleteData is the data passed to the delete confirmation template.
type MemoryDeleteData struct {
	Memory *memory.MemoryMetadata
	Error  string
}

// MemoryPaginationData contains pagination state for list views.
type MemoryPaginationData struct {
	CurrentPage  int
	TotalPages   int
	TotalItems   int
	Limit        int
	HasPrev      bool
	HasNext      bool
	PrevPage     int
	NextPage     int
	LimitOptions []int
}

// FormatSize converts bytes to human-readable format.
func FormatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

// list handles GET /memory - displays all memories with pagination and search.
func (h *memoryHandlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", DefaultPageLimit)

	// Validate and clamp limit
	if !isValidLimit(limit) {
		limit = DefaultPageLimit
	}
	if page < 1 {
		page = 1
	}

	// Get all memories from storage
	memories, err := h.storage.List(r.Context(), query)
	if err != nil {
		data := MemoryListData{Error: err.Error()}
		renderer.Render(w, r, "memory/list.html", data)
		return
	}

	// Calculate pagination
	totalItems := len(memories)
	totalPages := (totalItems + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Slice for current page
	start := (page - 1) * limit
	end := start + limit
	if end > totalItems {
		end = totalItems
	}
	pageMemories := memories[start:end]

	pagination := MemoryPaginationData{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   totalItems,
		Limit:        limit,
		HasPrev:      page > 1,
		HasNext:      page < totalPages,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		LimitOptions: PageLimitOptions,
	}

	data := MemoryListData{
		Memories:   pageMemories,
		Pagination: pagination,
		Query:      query,
	}

	if err := renderer.Render(w, r, "memory/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// view handles GET /memory/{name} - displays a single memory.
func (h *memoryHandlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")
	result, err := h.storage.Get(r.Context(), name)
	if err != nil {
		data := MemoryViewData{Error: err.Error()}
		renderer.Render(w, r, "memory/view.html", data)
		return
	}

	if !result.HasValue() {
		data := MemoryViewData{Error: "Memory not found"}
		renderer.Render(w, r, "memory/view.html", data)
		return
	}
	item := result.Value()

	data := MemoryViewData{
		Memory:        &item,
		FormattedSize: FormatSize(len(item.Content)),
	}

	if err := renderer.Render(w, r, "memory/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// createForm handles GET /memory/new - displays the create form.
func (h *memoryHandlers) createForm(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := MemoryFormData{IsEdit: false}
	if err := renderer.Render(w, r, "memory/form.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// create handles POST /memory - creates a new memory.
func (h *memoryHandlers) create(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		data := MemoryFormData{Error: "Invalid form data", IsEdit: false}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	name := r.FormValue("name")
	content := r.FormValue("content")

	// Validate name
	if name == "" {
		data := MemoryFormData{Name: name, Content: content, Error: "Name is required", IsEdit: false}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	if len(name) > 255 {
		data := MemoryFormData{Name: name, Content: content, Error: "Name must be 255 characters or less", IsEdit: false}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	// Validate content size (1MB limit)
	if len(content) > 1048576 {
		data := MemoryFormData{Name: name, Content: content, Error: "Content must be 1MB or less", IsEdit: false}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	// Create memory
	if err := h.storage.Set(r.Context(), name, content); err != nil {
		data := MemoryFormData{Name: name, Content: content, Error: err.Error(), IsEdit: false}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	// Redirect to list (relative path - works because router is scoped to /memory)
	w.Header().Set("HX-Redirect", "/memory")
	w.WriteHeader(http.StatusOK)
}

// editForm handles GET /memory/{name}/edit - displays the edit form.
func (h *memoryHandlers) editForm(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")
	result, err := h.storage.Get(r.Context(), name)
	if err != nil {
		data := MemoryFormData{Error: err.Error(), IsEdit: true}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	if !result.HasValue() {
		data := MemoryFormData{Error: "Memory not found", IsEdit: true}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}
	item := result.Value()

	data := MemoryFormData{
		Name:    item.Name,
		Content: item.Content,
		IsEdit:  true,
	}

	if err := renderer.Render(w, r, "memory/form.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// update handles PUT /memory/{name} - updates an existing memory.
func (h *memoryHandlers) update(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")

	if err := r.ParseForm(); err != nil {
		data := MemoryFormData{Name: name, Error: "Invalid form data", IsEdit: true}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	content := r.FormValue("content")

	// Validate content size (1MB limit)
	if len(content) > 1048576 {
		data := MemoryFormData{Name: name, Content: content, Error: "Content must be 1MB or less", IsEdit: true}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	// Update memory (creates new version)
	if err := h.storage.Set(r.Context(), name, content); err != nil {
		data := MemoryFormData{Name: name, Content: content, Error: err.Error(), IsEdit: true}
		renderer.Render(w, r, "memory/form.html", data)
		return
	}

	// Redirect to view
	w.Header().Set("HX-Redirect", "/memory/"+name)
	w.WriteHeader(http.StatusOK)
}

// deleteConfirm handles GET /memory/{name}/delete - displays delete confirmation.
func (h *memoryHandlers) deleteConfirm(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")
	result, err := h.storage.Get(r.Context(), name)
	if err != nil {
		// Redirect to list on error
		w.Header().Set("HX-Redirect", "/memory")
		w.WriteHeader(http.StatusOK)
		return
	}

	if !result.HasValue() {
		// Redirect to list if not found
		w.Header().Set("HX-Redirect", "/memory")
		w.WriteHeader(http.StatusOK)
		return
	}
	item := result.Value()

	metadata := &memory.MemoryMetadata{
		Name:      item.Name,
		Size:      len(item.Content),
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}

	data := MemoryDeleteData{Memory: metadata}
	if err := renderer.Render(w, r, "memory/delete.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// delete handles DELETE /memory/{name} - deletes a memory.
func (h *memoryHandlers) delete(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "name")

	if err := h.storage.Delete(r.Context(), name); err != nil {
		// Get memory metadata for error display
		result, _ := h.storage.Get(r.Context(), name)
		if result.HasValue() {
			item := result.Value()
			metadata := &memory.MemoryMetadata{
				Name:      item.Name,
				Size:      len(item.Content),
				Version:   item.Version,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
			data := MemoryDeleteData{Memory: metadata, Error: err.Error()}
			renderer.Render(w, r, "memory/delete.html", data)
			return
		}
		// Fall through to redirect if we can't get the item
	}

	// Redirect to list
	w.Header().Set("HX-Redirect", "/memory")
	w.WriteHeader(http.StatusOK)
}

// Helper functions

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func isValidLimit(limit int) bool {
	for _, opt := range PageLimitOptions {
		if limit == opt {
			return true
		}
	}
	return false
}
