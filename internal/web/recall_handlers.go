package web

import (
	"net/http"
	"time"

	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/go-chi/chi/v5"
)

// recallHandlers contains the HTTP handlers for the recall plugin.
type recallHandlers struct {
	storage  recall.Storage
	embedder recall.Embedder
}

// newRecallHandlers creates a new recallHandlers instance.
func newRecallHandlers(storage recall.Storage, embedder recall.Embedder) *recallHandlers {
	return &recallHandlers{storage: storage, embedder: embedder}
}

// RecallListData is the data passed to the list template.
type RecallListData struct {
	Conversations []recall.ConversationSummary
	Pagination    RecallPaginationData
	Project       string   // Current filter
	Projects      []string // Available projects for dropdown
	Error         string
}

// RecallViewData is the data passed to the view template.
type RecallViewData struct {
	ConversationID string
	Title          string
	Messages       []recall.Chunk
	MessageCount   int
	DateRange      string
	Error          string
}

// RecallSearchData is the data passed to the search results template.
type RecallSearchData struct {
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

// RecallPaginationData contains pagination state for list views.
type RecallPaginationData struct {
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
func (h *recallHandlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
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
		data := RecallListData{Error: err.Error()}
		renderer.Render(w, r, "recall/list.html", data)
		return
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get conversations
	conversations, err := h.storage.ListConversations(r.Context(), limit+1, offset, project)
	if err != nil {
		data := RecallListData{Error: err.Error(), Projects: projects, Project: project}
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
	pagination := RecallPaginationData{
		CurrentPage:  page,
		Limit:        limit,
		HasPrev:      page > 1,
		HasNext:      hasMore,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		LimitOptions: PageLimitOptions,
	}

	data := RecallListData{
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
func (h *recallHandlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")

	chunks, err := h.storage.GetByConversation(r.Context(), id)
	if err != nil {
		data := RecallViewData{Error: "Failed to load conversation"}
		renderer.Render(w, r, "recall/view.html", data)
		return
	}

	if len(chunks) == 0 {
		data := RecallViewData{Error: "Conversation not found"}
		w.WriteHeader(http.StatusNotFound)
		renderer.Render(w, r, "recall/view.html", data)
		return
	}

	data := RecallViewData{
		ConversationID: id,
		Title:          deriveConversationTitle(chunks),
		Messages:       chunks,
		MessageCount:   len(chunks),
		DateRange:      formatDateRange(chunks),
	}

	if err := renderer.Render(w, r, "recall/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// search handles GET /recall/search - performs semantic search.
func (h *recallHandlers) search(w http.ResponseWriter, r *http.Request) {
	renderer := GetRenderer(r)
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
		data := RecallSearchData{
			Query: query,
			Error: "Search unavailable - embedding service offline. Browse conversations instead.",
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	// Generate embedding for query
	embedding, err := h.embedder.Embed(r.Context(), query, recall.PurposeQuery)
	if err != nil {
		data := RecallSearchData{
			Query: query,
			Error: "Failed to process search query: " + err.Error(),
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	// Search - get more results to allow for filtering/grouping
	results, err := h.storage.Search(r.Context(), embedding, 50)
	if err != nil {
		data := RecallSearchData{
			Query: query,
			Error: "Search failed: " + err.Error(),
		}
		renderer.Render(w, r, "recall/search-results.html", data)
		return
	}

	resultGroups := groupSearchResults(results, project)

	data := RecallSearchData{
		Query:   query,
		Results: resultGroups,
	}

	if err := renderer.Render(w, r, "recall/search-results.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper functions

// groupSearchResults groups raw search results by conversation, optionally
// filtering by project, and returns at most 10 conversation groups ordered
// by search relevance.
func groupSearchResults(results []recall.SearchResult, projectFilter string) []SearchResultGroup {
	groups := make(map[string]*SearchResultGroup)
	var groupOrder []string

	for _, result := range results {
		if !matchesProjectFilter(result, projectFilter) {
			continue
		}

		convID := result.Chunk.ConversationID
		if group, exists := groups[convID]; exists {
			if len(group.Snippets) < 3 {
				group.Snippets = append(group.Snippets, truncate(result.Chunk.Content, 150))
			}
			continue
		}

		groups[convID] = newSearchResultGroup(result)
		groupOrder = append(groupOrder, convID)
	}

	resultGroups := make([]SearchResultGroup, 0, len(groupOrder))
	for _, convID := range groupOrder {
		resultGroups = append(resultGroups, *groups[convID])
	}

	if len(resultGroups) > 10 {
		resultGroups = resultGroups[:10]
	}

	return resultGroups
}

// matchesProjectFilter reports whether the result should be included given the
// active project filter. Empty filter or missing conversation ID cause a skip.
func matchesProjectFilter(result recall.SearchResult, projectFilter string) bool {
	if result.Chunk.ConversationID == "" {
		return false
	}
	if projectFilter == "" {
		return true
	}
	return result.Chunk.Metadata != nil && result.Chunk.Metadata.CWD == projectFilter
}

// newSearchResultGroup creates a SearchResultGroup from the first search result
// encountered for a conversation.
func newSearchResultGroup(result recall.SearchResult) *SearchResultGroup {
	lastMsg := result.Chunk.CreatedAt
	if result.Chunk.Metadata != nil && !result.Chunk.Metadata.Timestamp.IsZero() {
		lastMsg = result.Chunk.Metadata.Timestamp
	}

	return &SearchResultGroup{
		ConversationID: result.Chunk.ConversationID,
		Title:          truncate(result.Chunk.Content, 100),
		Snippets:       []string{truncate(result.Chunk.Content, 150)},
		Similarity:     result.Similarity,
		LastMessage:    lastMsg,
	}
}

// deriveConversationTitle returns a display title from the first user message
// in the conversation, truncated to 100 characters.
func deriveConversationTitle(chunks []recall.Chunk) string {
	for _, chunk := range chunks {
		if chunk.Metadata != nil && chunk.Metadata.Role == "user" {
			return truncate(chunk.Content, 100)
		}
	}
	return "[No title]"
}

// formatDateRange returns a human-readable date or date range string
// spanning from the first to last chunk timestamp.
func formatDateRange(chunks []recall.Chunk) string {
	if len(chunks) == 0 {
		return ""
	}

	first := effectiveTimestamp(chunks[0])
	last := effectiveTimestamp(chunks[len(chunks)-1])

	if first.Truncate(24 * time.Hour).Equal(last.Truncate(24 * time.Hour)) {
		return first.Format("Jan 2, 2006")
	}
	return first.Format("Jan 2") + " - " + last.Format("Jan 2, 2006")
}

// effectiveTimestamp returns the metadata timestamp if available, otherwise CreatedAt.
func effectiveTimestamp(c recall.Chunk) time.Time {
	if c.Metadata != nil && !c.Metadata.Timestamp.IsZero() {
		return c.Metadata.Timestamp
	}
	return c.CreatedAt
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
