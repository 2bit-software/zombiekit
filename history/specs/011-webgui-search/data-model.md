# Data Model: Web GUI Search Bar

**Date**: 2025-12-22
**Feature**: 011-webgui-search

## Entities

### SearchResult (Existing)

**Location**: `internal/search/search.go`
**Status**: Already implemented, no changes needed

| Field | Type   | Description                                      |
|-------|--------|--------------------------------------------------|
| Title | string | Display name of the matched item                 |
| URL   | string | Relative path to view item (e.g., "/memory/foo") |

### PluginSearchResult (New)

**Location**: `internal/web/search.go`
**Purpose**: Groups search results with their source plugin metadata

| Field      | Type           | Description                                       |
|------------|----------------|---------------------------------------------------|
| PluginID   | string         | Plugin name from RegisteredPlugin.Name()          |
| PluginName | string         | Human-readable label from SidebarItems()[0].Label |
| Items      | []SearchResult | Up to 3 results from this plugin                  |

**Validation Rules**:
- PluginID must be non-empty
- PluginName must be non-empty
- Items slice may be empty (plugin matched nothing)
- Items length capped at 3 by search caller

### SearchResponse (New)

**Location**: `internal/web/search.go`
**Purpose**: Aggregated response for the search endpoint

| Field   | Type                 | Description                          |
|---------|----------------------|--------------------------------------|
| Query   | string               | Original search query                |
| Results | []PluginSearchResult | Results grouped by plugin            |
| HasAny  | bool                 | True if any plugin returned results  |

**Validation Rules**:
- Query reflects user input (may be empty if cleared)
- Results ordered by plugin registration order
- HasAny computed: `len(Results) > 0 && any Items non-empty`

## Relationships

```text
SearchResponse
    └── []PluginSearchResult
            └── []SearchResult (from internal/search package)
```

## State Transitions

This feature is stateless from a data perspective. Each search request:
1. Receives query string
2. Queries all Searchable plugins
3. Returns aggregated results
4. No state persisted

## Notes

- SearchResult already defined in `internal/search/search.go`
- PluginSearchResult and SearchResponse are view models for the web layer
- These types are internal to `internal/web` package
- No database changes required
- All data comes from existing plugin implementations
