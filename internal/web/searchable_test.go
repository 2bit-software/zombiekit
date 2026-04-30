package web_test

import (
	"testing"

	"github.com/2bit-software/zombiekit/internal/web"
	"github.com/stretchr/testify/assert"
)

// searchableMock is a test implementation of the Searchable interface.
type searchableMock struct {
	items []web.SearchResult
}

func (m *searchableMock) Search(query string, maxResults int, sortOrder web.SortOrder) ([]web.SearchResult, error) {
	if query == "" {
		return []web.SearchResult{}, nil
	}

	results := m.items
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

// TestTypeAssertionToSearchable tests that interface satisfaction works
// correctly with type assertion.
func TestTypeAssertionToSearchable(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{
			{Title: "Test", URL: "/test"},
		},
	}

	// Type assertion should succeed
	var iface any = mock
	searchable, ok := iface.(web.Searchable)
	assert.True(t, ok, "mock should satisfy Searchable interface")
	assert.NotNil(t, searchable)

	// Should be able to call Search
	results, err := searchable.Search("test", 10, web.SortRelevance)
	assert.NoError(t, err)
	assert.NotNil(t, results)
}

// Compile-time interface check: searchableMock must implement Searchable
var _ web.Searchable = (*searchableMock)(nil)

// TestSearchReturnsResults verifies that a Searchable implementation returns results.
func TestSearchReturnsResults(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{
			{Title: "First Item", URL: "/items/1"},
			{Title: "Second Item", URL: "/items/2"},
		},
	}

	results, err := mock.Search("item", 10, web.SortRelevance)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "First Item", results[0].Title)
	assert.Equal(t, "/items/1", results[0].URL)
}

// TestSearchWithMaxResults verifies that maxResults limit is respected.
func TestSearchWithMaxResults(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{
			{Title: "Item 1", URL: "/1"},
			{Title: "Item 2", URL: "/2"},
			{Title: "Item 3", URL: "/3"},
			{Title: "Item 4", URL: "/4"},
			{Title: "Item 5", URL: "/5"},
		},
	}

	results, err := mock.Search("item", 2, web.SortRelevance)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "should respect maxResults limit")
}

// TestSearchEmptyQueryReturnsEmptySlice verifies empty query returns empty slice.
func TestSearchEmptyQueryReturnsEmptySlice(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{
			{Title: "Item", URL: "/item"},
		},
	}

	results, err := mock.Search("", 10, web.SortRelevance)
	assert.NoError(t, err)
	assert.Empty(t, results, "empty query should return empty results")
}

// TestSearchNoMatchesReturnsEmptySlice verifies no matches returns empty slice.
func TestSearchNoMatchesReturnsEmptySlice(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{}, // No items to match
	}

	results, err := mock.Search("nonexistent", 10, web.SortRelevance)
	assert.NoError(t, err)
	assert.Empty(t, results, "no matches should return empty slice")
}

// TestSearchResultNeverNil verifies that the result slice is never nil.
func TestSearchResultNeverNil(t *testing.T) {
	mock := &searchableMock{
		items: []web.SearchResult{},
	}

	results, err := mock.Search("", 10, web.SortRelevance)
	assert.NoError(t, err)
	assert.NotNil(t, results, "results should never be nil")

	results2, err2 := mock.Search("nonexistent", 10, web.SortRelevance)
	assert.NoError(t, err2)
	assert.NotNil(t, results2, "results should never be nil even with no matches")
}

// TestIsValidSortOrder tests the IsValidSortOrder helper function.
func TestIsValidSortOrder(t *testing.T) {
	validOrders := []web.SortOrder{
		web.SortRelevance,
		web.SortCreatedDate,
		web.SortUpdatedDate,
		web.SortLastUsed,
		web.SortName,
	}

	for _, order := range validOrders {
		assert.True(t, web.IsValidSortOrder(order), "should be valid: %s", order)
	}

	invalidOrders := []web.SortOrder{
		"",
		"invalid",
		"RELEVANCE",
		"Name",
	}

	for _, order := range invalidOrders {
		assert.False(t, web.IsValidSortOrder(order), "should be invalid: %s", order)
	}
}
