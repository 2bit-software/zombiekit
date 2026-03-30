# Quickstart: Searchable Interface

**Feature**: 010-searchable-interface

## Overview

This guide shows how to make a web GUI plugin searchable by implementing the `Searchable` interface.

## Prerequisites

- Existing plugin implementing `web.WebPlugin`
- Go 1.24+

## Implementation Steps

### 1. Import the search package

```go
import "github.com/2bit-software/zombiekit/internal/search"
```

### 2. Add the Search method to your plugin

```go
// Search implements search.Searchable
func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
    // Handle empty query
    if query == "" {
        return []search.SearchResult{}, nil
    }

    // Normalize for case-insensitive search
    query = strings.ToLower(query)

    // Get your items (implementation-specific)
    items := p.getAllItems()

    var results []search.SearchResult
    for _, item := range items {
        // Check name and content
        if strings.Contains(strings.ToLower(item.Name), query) ||
           strings.Contains(strings.ToLower(item.Content), query) {
            results = append(results, search.SearchResult{
                Title: item.Name,
                URL:   fmt.Sprintf("/%s/%s", p.ID(), item.ID),
            })
        }
    }

    // Apply sorting
    results = sortResults(results, sortOrder, items)

    // Apply limit
    if maxResults > 0 && len(results) > maxResults {
        results = results[:maxResults]
    }

    return results, nil
}
```

### 3. Add compile-time interface check

```go
// Ensure Plugin implements Searchable
var _ search.Searchable = (*Plugin)(nil)
```

## Sort Order Implementation

```go
func sortResults(results []search.SearchResult, order search.SortOrder, items []Item) []search.SearchResult {
    switch order {
    case search.SortName:
        sort.Slice(results, func(i, j int) bool {
            return results[i].Title < results[j].Title
        })
    case search.SortCreatedDate:
        // Sort by creation date (requires item lookup)
        // Items without dates go to end
    case search.SortUpdatedDate:
        // Sort by update date
    case search.SortLastUsed:
        // Sort by last access time
    default: // SortRelevance or empty
        // Keep original order (relevance from search)
    }
    return results
}
```

## Testing Your Implementation

```go
func TestSearch(t *testing.T) {
    p := NewPlugin(mockService)

    // Test basic search
    results, err := p.Search("test", 10, search.SortRelevance)
    require.NoError(t, err)
    assert.NotNil(t, results)

    // Test empty query
    results, err = p.Search("", 10, search.SortRelevance)
    require.NoError(t, err)
    assert.Empty(t, results)

    // Test max results
    results, err = p.Search("common", 2, search.SortRelevance)
    require.NoError(t, err)
    assert.LessOrEqual(t, len(results), 2)
}
```

## Type Assertion (for consumers)

```go
// Check if a plugin supports search
if searchable, ok := plugin.(search.Searchable); ok {
    results, err := searchable.Search("query", 10, search.SortRelevance)
    // handle results
}
```

## Key Rules

1. **Never return nil** - always return `[]search.SearchResult{}`
2. **Case-insensitive** - normalize query and content for comparison
3. **Search both** - check names AND content
4. **Handle empty query** - return empty results, no error
5. **Respect limits** - honor maxResults when > 0
6. **Graceful sort fallback** - unknown sort orders default to relevance
