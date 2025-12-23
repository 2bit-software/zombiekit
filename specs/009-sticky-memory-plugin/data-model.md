# Data Model: Sticky Memory Web Plugin

**Feature**: 009-sticky-memory-plugin
**Date**: 2025-12-22

## Entities

### MemoryItem (Existing - internal/memory/types.go)

Primary entity representing a stored memory with full content.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| Name | string | Required, max 255 chars, URL-safe | Unique identifier for the memory |
| Content | string | Optional, max 1MB | Markdown text content |
| Version | int | Auto-increment, >= 1 | Version number (increments on update) |
| Deleted | bool | Default false | Soft delete flag |
| CreatedAt | time.Time | Auto-set on create | Creation timestamp |
| UpdatedAt | time.Time | Auto-set on create/update | Last modification timestamp |

**Validation Rules**:
- Name: `^[a-zA-Z0-9._-]+$` pattern, invalid chars replaced with underscore
- Empty names become "unnamed"
- Content size checked before storage

**State Transitions**:
```
[New] --create--> [Active] --update--> [Active (version++)]
                     |
                     +--delete--> [Deleted (soft)]
```

### MemoryMetadata (Existing - internal/memory/types.go)

Lightweight representation for list display (excludes content).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| Name | string | Same as MemoryItem | Memory identifier |
| Size | int | Calculated field | Content size in bytes |
| Version | int | Same as MemoryItem | Current version number |
| CreatedAt | time.Time | Same as MemoryItem | Creation timestamp |
| UpdatedAt | time.Time | Same as MemoryItem | Last modification timestamp |

## View Models (New for Plugin)

### ListData

Data passed to list.html template.

| Field | Type | Description |
|-------|------|-------------|
| Memories | []MemoryMetadata | List of memories for current page |
| Pagination | PaginationData | Pagination state |
| Query | string | Current search query (empty if none) |
| Error | string | Error message (empty if none) |

### ViewData

Data passed to view.html template.

| Field | Type | Description |
|-------|------|-------------|
| Memory | *MemoryItem | Full memory with content |
| FormattedSize | string | Human-readable size (e.g., "1.2 KB") |
| Error | string | Error message (empty if none) |

### FormData

Data passed to form.html template (create/edit).

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Current name value |
| Content | string | Current content value |
| Error | string | Validation error message |
| IsEdit | bool | True if editing existing memory |

### DeleteData

Data passed to delete.html template.

| Field | Type | Description |
|-------|------|-------------|
| Memory | *MemoryMetadata | Memory being deleted (name, metadata) |
| Error | string | Error message if deletion failed |

### PaginationData

Pagination state for list view.

| Field | Type | Description |
|-------|------|-------------|
| CurrentPage | int | Current page number (1-indexed) |
| TotalPages | int | Total number of pages |
| TotalItems | int | Total number of items |
| Limit | int | Items per page |
| HasPrev | bool | Whether previous page exists |
| HasNext | bool | Whether next page exists |
| PrevPage | int | Previous page number |
| NextPage | int | Next page number |
| LimitOptions | []int | Available limit options [10, 20, 50, 100] |

## Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                        Web Layer                            │
├─────────────────────────────────────────────────────────────┤
│  ListData ──contains──> []MemoryMetadata                    │
│  ListData ──contains──> PaginationData                      │
│  ViewData ──contains──> *MemoryItem                         │
│  FormData ──represents──> MemoryItem (name, content only)   │
│  DeleteData ──references──> *MemoryMetadata                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Uses
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Storage Layer                           │
├─────────────────────────────────────────────────────────────┤
│  memory.Storage interface                                   │
│    ├── Set(name, content) -> creates/updates MemoryItem     │
│    ├── Get(name) -> Maybe[MemoryItem]                       │
│    ├── List(query) -> []MemoryMetadata                      │
│    └── Delete(name) -> soft-deletes MemoryItem              │
└─────────────────────────────────────────────────────────────┘
```

## Storage Interface (Existing)

The plugin reuses the existing `memory.Storage` interface:

```go
type Storage interface {
    Set(ctx context.Context, name, content string) error
    Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error)
    Delete(ctx context.Context, name string) error
    List(ctx context.Context, search string) ([]MemoryMetadata, error)
    Clear(ctx context.Context) (int, error)
    Close() error
}
```

**Notes**:
- `Set` is upsert: creates if not exists, updates with version++ if exists
- `Get` returns Maybe monad (handles not-found gracefully)
- `List` with empty search returns all; with query filters by name/content
- `Delete` is soft delete (sets Deleted=true)

## Formatting Helpers

### FormatSize

Converts bytes to human-readable format.

```go
func FormatSize(bytes int) string {
    if bytes < 1024 {
        return fmt.Sprintf("%d B", bytes)
    } else if bytes < 1024*1024 {
        return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
    }
    return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
```

### FormatTime

Formats timestamps for display (consider relative time for recent items).

```go
func FormatTime(t time.Time) string {
    return t.Format("Jan 2, 2006 3:04 PM")
}
```

## Constants

```go
const (
    DefaultPageLimit = 20
    MaxPageLimit     = 100
    PageLimitOptions = []int{10, 20, 50, 100}

    MaxNameLength    = 255      // bytes (from memory package)
    MaxContentSize   = 1048576  // 1MB (from memory package)
)
```
