// Package search defines the Searchable interface and related types.
// This package is independent of internal/web and can be used by any
// component that wants to provide search functionality.
package search

// SortOrder defines how search results should be ordered.
type SortOrder string

// Sort order constants.
const (
	SortRelevance   SortOrder = "relevance"    // Default: closest matches first
	SortCreatedDate SortOrder = "created_date" // Newest created first
	SortUpdatedDate SortOrder = "updated_date" // Most recently updated first
	SortLastUsed    SortOrder = "last_used"    // Most recently accessed first
	SortName        SortOrder = "name"         // Alphabetical A-Z
)

// IsValidSortOrder returns true if the given sort order is a known valid value.
func IsValidSortOrder(order SortOrder) bool {
	switch order {
	case SortRelevance, SortCreatedDate, SortUpdatedDate, SortLastUsed, SortName:
		return true
	default:
		return false
	}
}

// SearchResult represents a single search match.
type SearchResult struct {
	// Title is the display name of the matched item.
	Title string

	// URL is the relative path to view this item (e.g., "/profiles/my-profile").
	URL string
}

// Searchable is the interface for types that support search functionality.
// This interface is independent of WebPlugin and can be implemented by any type.
//
// Implementation requirements:
//   - Search MUST be case-insensitive by default
//   - Search MUST check both item names and content
//   - Empty query MUST return empty slice (not error)
//   - No matches MUST return empty slice (not nil)
//   - maxResults <= 0 means unlimited
//   - Empty or invalid sortOrder defaults to SortRelevance
type Searchable interface {
	// Search finds items matching the query string.
	//
	// Parameters:
	//   - query: The search text (case-insensitive)
	//   - maxResults: Maximum results to return (0 or negative = unlimited)
	//   - sortOrder: How to order results (empty = relevance)
	//
	// Returns:
	//   - Slice of matching results (empty if no matches, never nil)
	//   - Error only for actual failures (not for "no results")
	Search(query string, maxResults int, sortOrder SortOrder) ([]SearchResult, error)
}
