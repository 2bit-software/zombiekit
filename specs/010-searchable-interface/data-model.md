# Data Model: Searchable Interface

**Date**: 2025-12-22
**Feature**: 010-searchable-interface

## Entities

### SearchResult

Represents a single search match returned by a Searchable implementation.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| Title | string | Display title of the matched item | Required, non-empty |
| URL | string | Deep-link URL to the matched item | Required, relative path (e.g., "/profiles/my-profile") |

**Notes**:
- Title is typically the name of the document/item
- URL must be a valid path that resolves when loaded in the web GUI
- URL format is implementation-specific (e.g., `/profiles/{name}`, `/memories/{id}`)

### SortOrder

Type alias for string representing sort options.

| Value | Description | Behavior |
|-------|-------------|----------|
| `"relevance"` | Default - closest matches first | Implementation-defined scoring |
| `"created_date"` | Sort by creation time | Newest first |
| `"updated_date"` | Sort by last modification | Most recent first |
| `"last_used"` | Sort by last access time | Most recent first |
| `"name"` | Alphabetical by title | A-Z ascending |

**Notes**:
- Empty string treated as `"relevance"`
- Invalid values should be treated as `"relevance"` (graceful fallback)
- Items lacking required metadata for a sort order appear at end of results

## Interfaces

### Searchable

Contract for types that support search functionality.

```text
Method: Search
Parameters:
  - query: string (the search text)
  - maxResults: int (maximum results to return, 0 = unlimited)
  - sortOrder: SortOrder (how to order results)
Returns:
  - []SearchResult (matching items, may be empty)
  - error (nil on success)
```

**Contract Rules**:
1. Empty query returns empty slice (no error)
2. No matches returns empty slice (no error)
3. Negative maxResults treated as 0 (unlimited)
4. Results slice is never nil
5. Search is case-insensitive by default
6. Search covers both item names and content

## Relationships

```text
┌─────────────────┐
│   Searchable    │  (Interface)
│   interface     │
└────────┬────────┘
         │ implements
         ▼
┌─────────────────┐      returns      ┌─────────────────┐
│  Plugin (e.g.,  │ ─────────────────▶│  SearchResult   │
│  profiles.Plugin│                   │  (0..n items)   │
│  memory.Plugin) │                   └─────────────────┘
└─────────────────┘
         │
         │ also implements
         ▼
┌─────────────────┐
│   WebPlugin     │  (from internal/web)
│   interface     │
└─────────────────┘
```

**Key Points**:
- `Searchable` and `WebPlugin` are independent interfaces
- A plugin can implement one, both, or neither
- Type assertion detects Searchable capability at runtime
- No package dependency between `internal/search` and `internal/web`

## State Transitions

N/A - This feature defines a stateless query interface. State management is the responsibility of each plugin implementation.

## Validation Rules

| Rule | Applies To | Description |
|------|------------|-------------|
| Non-empty title | SearchResult | Title must not be empty string |
| Valid URL format | SearchResult | URL should be a relative path starting with "/" |
| Non-nil slice | Search return | Always return empty slice `[]SearchResult{}` rather than nil |
| Bounded results | Search behavior | Respect maxResults parameter (when > 0) |
