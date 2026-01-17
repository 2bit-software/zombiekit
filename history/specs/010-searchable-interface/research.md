# Research: Searchable Interface

**Date**: 2025-12-22
**Feature**: 010-searchable-interface

## Overview

This feature defines a new `Searchable` interface for the web GUI plugin system. Research focuses on Go interface design patterns for optional capability composition.

## Research Topics

### 1. Interface Composition Pattern in Go

**Decision**: Use Go's implicit interface satisfaction with a separate package

**Rationale**:
- Go interfaces are satisfied implicitly - no "implements" keyword needed
- Placing `Searchable` in a separate `internal/search` package ensures zero coupling with `internal/web`
- Plugins can import `internal/search` and implement the interface alongside `WebPlugin`
- Type assertion (`plugin.(search.Searchable)`) provides runtime capability detection

**Alternatives Considered**:
1. **Embed Searchable in WebPlugin**: Rejected - would require all plugins to implement search
2. **Optional method on WebPlugin**: Rejected - Go doesn't support optional interface methods
3. **Separate interface in same package**: Rejected - would create dependency cycle risk

### 2. Search Result Type Design

**Decision**: Simple struct with Title and URL fields

**Rationale**:
- Matches spec requirements exactly (FR-001)
- URL as string allows flexibility for different URL patterns across plugins
- Title as string is universally applicable

**Alternatives Considered**:
1. **Rich result with snippet/excerpt**: Rejected - adds complexity; can be added later if needed
2. **Result with metadata map**: Rejected - over-engineering for current requirements
3. **Separate result types per plugin**: Rejected - defeats purpose of unified search

### 3. Sort Order Representation

**Decision**: String constants with a `SortOrder` type alias

**Rationale**:
- String type allows easy serialization in future API exposure
- Type alias provides compile-time documentation
- Constants ensure typo-free usage

**Alternatives Considered**:
1. **Integer enum (iota)**: Rejected - less readable, harder to debug
2. **Separate type with methods**: Rejected - over-engineering for 5 fixed values
3. **Interface parameter**: Rejected - unnecessary complexity

### 4. Search Method Signature

**Decision**: `Search(query string, maxResults int, sortOrder SortOrder) ([]SearchResult, error)`

**Rationale**:
- Single method keeps interface minimal
- Error return allows implementations to report failures
- Slice return (not pointer) follows Go conventions
- Parameters match spec requirements exactly

**Alternatives Considered**:
1. **Options struct parameter**: Rejected - only 3 parameters, struct adds boilerplate
2. **Functional options pattern**: Rejected - over-engineering for simple use case
3. **Context parameter**: Deferred - can be added if cancellation needed later

### 5. Package Location

**Decision**: `internal/search/` as new top-level internal package

**Rationale**:
- Parallel to `internal/web/` maintaining clean separation
- Can be imported by `internal/webplugins/*` without cycles
- Future CLI/MCP code can import without web dependencies

**Alternatives Considered**:
1. **`internal/web/search/`**: Rejected - creates dependency on web package
2. **`pkg/search/`**: Rejected - not a public API
3. **`internal/plugin/search/`**: Rejected - no plugin package exists

## Implementation Notes

1. **Empty query behavior**: Return empty slice, not error (per edge case spec)
2. **max_results=0**: Means "no limit" (per edge case spec)
3. **Negative max_results**: Treat as 0 (per edge case spec)
4. **Missing sort metadata**: Items without metadata sort to end (per edge case spec)
5. **Case sensitivity**: Case-insensitive matching by default (per edge case spec)

## No Unresolved Items

All technical decisions made. Ready for Phase 1 design.
