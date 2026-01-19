package recall

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/web"
)

// Pagination constants
const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// PageLimitOptions are the available pagination limit choices.
var PageLimitOptions = []int{10, 20, 50, 100}

// handlers contains the HTTP handlers for the recall plugin.
type handlers struct {
	storage  recall.Storage
	embedder recall.Embedder
}

// newHandlers creates a new handlers instance.
func newHandlers(storage recall.Storage, embedder recall.Embedder) *handlers {
	return &handlers{storage: storage, embedder: embedder}
}

// ListData is the data passed to the list template.
type ListData struct {
	Conversations []recall.ConversationSummary
	Pagination    PaginationData
	Project       string   // Current filter
	Projects      []string // Available projects for dropdown
	Error         string
}

// ViewData is the data passed to the view template.
type ViewData struct {
	ConversationID string
	Title          string
	Messages       []recall.Chunk
	MessageCount   int
	DateRange      string
	Error          string
}

// SearchData is the data passed to the search results template.
type SearchData struct {
	Query   string
	Results []SearchResultGroup
	Error   string
}

// SearchResultGroup groups search results by conversation.
type SearchResultGroup struct {
	ConversationID string
	Title          string
	Snippets       []string
	Similarity     float64
	LastMessage    time.Time
}

// PaginationData contains pagination state for list views.
type PaginationData struct {
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

// list handles GET /recall - displays all conversations with pagination.
func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	project := r.URL.Query().Get("project")
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", DefaultPageLimit)

	// Validate and clamp limit
	if !isValidLimit(limit) {
		limit = DefaultPageLimit
	}
	if page < 1 {
		page = 1
	}

	// Get distinct projects for filter dropdown
	projects, err := h.storage.ListDistinctProjects(r.Context())
	if err != nil {
		data := ListData{Error: err.Error()}
		renderer.Render(w, r, "recall/list.html", data)
		return
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get conversations
	conversations, err := h.storage.ListConversations(r.Context(), limit+1, offset, project)
	if err != nil {
		data := ListData{Error: err.Error(), Projects: projects, Project: project}
		renderer.Render(w, r, "recall/list.html", data)
		return
	}

	// Check if there are more results (we fetched limit+1)
	hasMore := len(conversations) > limit
	if hasMore {
		conversations = conversations[:limit]
	}

	// Calculate pagination
	// Note: We don't have total count, so we use hasMore to determine HasNext
	pagination := PaginationData{
		CurrentPage:  page,
		Limit:        limit,
		HasPrev:      page > 1,
		HasNext:      hasMore,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		LimitOptions: PageLimitOptions,
	}

	data := ListData{
		Conversations: conversations,
		Pagination:    pagination,
		Project:       project,
		Projects:      projects,
	}

	if err := renderer.Render(w, r, "recall/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// view handles GET /recall/{id} - displays a single conversation.
func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")

	chunks, err := h.storage.GetByConversation(r.Context(), id)
	if err != nil {
		data := ViewData{Error: "Failed to load conversation"}
		renderer.Render(w, r, "recall/view.html", data)
		return
	}

	if len(chunks) == 0 {
		data := ViewData{Error: "Conversation not found"}
		w.WriteHeader(http.StatusNotFound)
		renderer.Render(w, r, "recall/view.html", data)
		return
	}

	// Derive title from first user message
	title := "[No title]"
	for _, chunk := range chunks {
		if chunk.Metadata != nil && chunk.Metadata.Role == "user" {
			title = chunk.Content
			if len(title) > 100 {
				title = title[:100] + "..."
			}
			break
		}
	}

	// Calculate date range
	var dateRange string
	if len(chunks) > 0 {
		first := chunks[0].CreatedAt
		last := chunks[len(chunks)-1].CreatedAt
		if chunks[0].Metadata != nil && !chunks[0].Metadata.Timestamp.IsZero() {
			first = chunks[0].Metadata.Timestamp
		}
		if chunks[len(chunks)-1].Metadata != nil && !chunks[len(chunks)-1].Metadata.Timestamp.IsZero() {
			last = chunks[len(chunks)-1].Metadata.Timestamp
		}

		if first.Truncate(24*time.Hour).Equal(last.Truncate(24 * time.Hour)) {
			dateRange = first.Format("Jan 2, 2006")
		} else {
			dateRange = first.Format("Jan 2") + " - " + last.Format("Jan 2, 2006")
		}
	}

	data := ViewData{
		ConversationID: id,
		Title:          title,
		Messages:       chunks,
		MessageCount:   len(chunks),
		DateRange:      dateRange,
	}

	if err := renderer.Render(w, r, "recall/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// search handles GET /recall/search - performs semantic search.
func (h *handlers) search(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query().Get("q")
	project := r.URL.Query().Get("project")

	// Empty query - redirect to list
	if query == "" {
		w.Header().Set("HX-Redirect", "/recall")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check embedder availability
	if h.embedder == nil {
		data := SearchData{
			Query: query,
			Error: "Search unavailable - embedding service offline. Browse conversations instead.",
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	// Generate embedding for query
	embedding, err := h.embedder.Embed(r.Context(), query, recall.PurposeQuery)
	if err != nil {
		data := SearchData{
			Query: query,
			Error: "Failed to process search query: " + err.Error(),
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	// Search - get more results to allow for filtering/grouping
	results, err := h.storage.Search(r.Context(), embedding, 50)
	if err != nil {
		data := SearchData{
			Query: query,
			Error: "Search failed: " + err.Error(),
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	// Group by conversation, filter by project if specified
	groups := make(map[string]*SearchResultGroup)
	var groupOrder []string

	for _, result := range results {
		if result.Chunk.ConversationID == "" {
			continue
		}

		// Filter by project if specified
		if project != "" {
			if result.Chunk.Metadata == nil || result.Chunk.Metadata.CWD != project {
				continue
			}
		}

		convID := result.Chunk.ConversationID
		if group, exists := groups[convID]; exists {
			// Add snippet if we have room
			if len(group.Snippets) < 3 {
				snippet := result.Chunk.Content
				if len(snippet) > 150 {
					snippet = snippet[:150] + "..."
				}
				group.Snippets = append(group.Snippets, snippet)
			}
		} else {
			// New conversation group
			title := result.Chunk.Content
			if len(title) > 100 {
				title = title[:100] + "..."
			}

			snippet := result.Chunk.Content
			if len(snippet) > 150 {
				snippet = snippet[:150] + "..."
			}

			lastMsg := result.Chunk.CreatedAt
			if result.Chunk.Metadata != nil && !result.Chunk.Metadata.Timestamp.IsZero() {
				lastMsg = result.Chunk.Metadata.Timestamp
			}

			groups[convID] = &SearchResultGroup{
				ConversationID: convID,
				Title:          title,
				Snippets:       []string{snippet},
				Similarity:     result.Similarity,
				LastMessage:    lastMsg,
			}
			groupOrder = append(groupOrder, convID)
		}
	}

	// Convert to slice maintaining relevance order
	var resultGroups []SearchResultGroup
	for _, convID := range groupOrder {
		resultGroups = append(resultGroups, *groups[convID])
	}

	// Limit to top 10 conversations
	if len(resultGroups) > 10 {
		resultGroups = resultGroups[:10]
	}

	data := SearchData{
		Query:   query,
		Results: resultGroups,
	}

	if err := renderer.Render(w, r, "recall/search-results.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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
