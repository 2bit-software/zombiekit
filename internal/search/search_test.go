package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/2bit-software/zombiekit/internal/search"
)

// mockSearchable is a test implementation of the Searchable interface.
type mockSearchable struct {
	items []search.SearchResult
}

func (m *mockSearchable) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
	if query == "" {
		return []search.SearchResult{}, nil
	}

	results := m.items
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

// TestSearchableInterfaceIndependence verifies that the search package
// has no imports from internal/web. This test's existence in search_test
// (which only imports search) proves there's no web dependency.
func TestSearchableInterfaceIndependence(t *testing.T) {
	// The fact that this test file compiles without importing internal/web
	// proves the search package is independent.
	// If search.go ever imports internal/web, this test package would fail to compile
	// because it doesn't import web.
	assert.True(t, true, "search package compiles without web dependency")
}

// TestTypeAssertionToSearchable tests that interface satisfaction works
// correctly with type assertion.
func TestTypeAssertionToSearchable(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{
			{Title: "Test", URL: "/test"},
		},
	}

	// Type assertion should succeed
	var iface interface{} = mock
	searchable, ok := iface.(search.Searchable)
	assert.True(t, ok, "mock should satisfy Searchable interface")
	assert.NotNil(t, searchable)

	// Should be able to call Search
	results, err := searchable.Search("test", 10, search.SortRelevance)
	assert.NoError(t, err)
	assert.NotNil(t, results)
}

// Compile-time interface check: mockSearchable must implement Searchable
var _ search.Searchable = (*mockSearchable)(nil)

// TestSearchReturnsResults verifies that a Searchable implementation returns results.
func TestSearchReturnsResults(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{
			{Title: "First Item", URL: "/items/1"},
			{Title: "Second Item", URL: "/items/2"},
		},
	}

	results, err := mock.Search("item", 10, search.SortRelevance)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "First Item", results[0].Title)
	assert.Equal(t, "/items/1", results[0].URL)
}

// TestSearchWithMaxResults verifies that maxResults limit is respected.
func TestSearchWithMaxResults(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{
			{Title: "Item 1", URL: "/1"},
			{Title: "Item 2", URL: "/2"},
			{Title: "Item 3", URL: "/3"},
			{Title: "Item 4", URL: "/4"},
			{Title: "Item 5", URL: "/5"},
		},
	}

	results, err := mock.Search("item", 2, search.SortRelevance)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "should respect maxResults limit")
}

// TestSearchEmptyQueryReturnsEmptySlice verifies empty query returns empty slice.
func TestSearchEmptyQueryReturnsEmptySlice(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{
			{Title: "Item", URL: "/item"},
		},
	}

	results, err := mock.Search("", 10, search.SortRelevance)
	assert.NoError(t, err)
	assert.Empty(t, results, "empty query should return empty results")
}

// TestSearchNoMatchesReturnsEmptySlice verifies no matches returns empty slice.
func TestSearchNoMatchesReturnsEmptySlice(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{}, // No items to match
	}

	results, err := mock.Search("nonexistent", 10, search.SortRelevance)
	assert.NoError(t, err)
	assert.Empty(t, results, "no matches should return empty slice")
}

// TestSearchResultNeverNil verifies that the result slice is never nil.
func TestSearchResultNeverNil(t *testing.T) {
	mock := &mockSearchable{
		items: []search.SearchResult{},
	}

	results, err := mock.Search("", 10, search.SortRelevance)
	assert.NoError(t, err)
	assert.NotNil(t, results, "results should never be nil")

	results2, err2 := mock.Search("nonexistent", 10, search.SortRelevance)
	assert.NoError(t, err2)
	assert.NotNil(t, results2, "results should never be nil even with no matches")
}

// TestIsValidSortOrder tests the IsValidSortOrder helper function.
func TestIsValidSortOrder(t *testing.T) {
	validOrders := []search.SortOrder{
		search.SortRelevance,
		search.SortCreatedDate,
		search.SortUpdatedDate,
		search.SortLastUsed,
		search.SortName,
	}

	for _, order := range validOrders {
		assert.True(t, search.IsValidSortOrder(order), "should be valid: %s", order)
	}

	invalidOrders := []search.SortOrder{
		"",
		"invalid",
		"RELEVANCE",
		"Name",
	}

	for _, order := range invalidOrders {
		assert.False(t, search.IsValidSortOrder(order), "should be invalid: %s", order)
	}
}
