# Feature Specification: Searchable Interface

**Feature Branch**: `010-searchable-interface`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Implement a new web GUI concept - a Searchable interface that plugins can implement. Includes predefined SearchResult type with title and URL, max_results parameter, and sort_order options (created_date, updated_date, last_used, name). Searches both names and content. Designed as a composable interface separate from WebPlugin for potential reuse elsewhere. No CLI/MCP exposure or search form UI in this iteration."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin Developer Implements Search (Priority: P1)

A plugin developer wants to make their plugin's content searchable through the web GUI. They implement the Searchable interface on their existing plugin, providing search functionality that returns results matching user queries with proper titles and deep-link URLs.

**Why this priority**: Core value proposition - without plugins implementing this interface, search cannot function. This enables the entire search ecosystem.

**Independent Test**: Can be fully tested by creating a mock plugin that implements Searchable, executing a search query, and verifying results contain expected titles and valid URLs.

**Acceptance Scenarios**:

1. **Given** a plugin that implements the Searchable interface, **When** the Search method is called with a query string, **Then** it returns SearchResult items containing titles and URLs matching the query
2. **Given** a plugin that implements Searchable, **When** Search is called with max_results=5, **Then** at most 5 results are returned
3. **Given** a plugin that implements Searchable, **When** Search is called with an empty query, **Then** it returns an empty result set (no error)

---

### User Story 2 - Search Results Sorted by Relevance (Priority: P2)

When a user (through future UI) searches for content, results are returned in order of closest match by default. This provides the most useful results first without requiring additional configuration.

**Why this priority**: Default behavior that provides immediate value - users expect best matches first. Required for any practical search implementation.

**Independent Test**: Execute search with a query that matches multiple items with varying relevance. Verify results are ordered with closest matches appearing first.

**Acceptance Scenarios**:

1. **Given** multiple searchable items with varying match quality, **When** Search is called without specifying sort_order, **Then** results are returned with closest matches first (relevance-based ordering)
2. **Given** a search query that exactly matches one item's name, **When** Search is executed, **Then** the exact match appears before partial matches

---

### User Story 3 - Search Results Sorted by User-Specified Order (Priority: P2)

Users can request results sorted by different criteria to find specific content more easily - such as recently created items, recently modified items, most recently used items, or alphabetically by name.

**Why this priority**: Adds flexibility for different use cases. Equal priority to P2 since sort options are equally important for usability.

**Independent Test**: Execute same search with different sort_order values and verify result ordering changes according to the specified criterion.

**Acceptance Scenarios**:

1. **Given** searchable items with different creation dates, **When** Search is called with sort_order="created_date", **Then** results are ordered by creation date (newest first)
2. **Given** searchable items with different update times, **When** Search is called with sort_order="updated_date", **Then** results are ordered by last update time (newest first)
3. **Given** searchable items with usage tracking, **When** Search is called with sort_order="last_used", **Then** results are ordered by last access time (most recent first)
4. **Given** searchable items with names, **When** Search is called with sort_order="name", **Then** results are ordered alphabetically by name (A-Z)

---

### User Story 4 - Search Across Names and Content (Priority: P1)

Searches match against both the name/title of items and their content. Users can find items by either remembering the name or something within the content.

**Why this priority**: Fundamental search behavior - users expect to find items regardless of whether they remember the title or content details.

**Independent Test**: Create items where one matches by name and another matches only by content. Execute search and verify both items appear in results.

**Acceptance Scenarios**:

1. **Given** an item with the word "report" in its name, **When** Search is called with query="report", **Then** the item appears in results
2. **Given** an item with the word "budget" only in its content (not name), **When** Search is called with query="budget", **Then** the item appears in results
3. **Given** an item matching both in name and content, **When** Search is executed, **Then** the item appears once (not duplicated)

---

### User Story 5 - Interface Composition with WebPlugin (Priority: P1)

The Searchable interface is designed as a separate, composable interface that can be composed with WebPlugin. This allows plugins to optionally implement search capability while keeping concerns separated for potential reuse in CLI or MCP contexts later.

**Why this priority**: Architectural foundation - proper interface design enables future extensibility and clean implementation patterns.

**Independent Test**: Create a plugin that implements both WebPlugin and Searchable. Verify both interfaces work independently and the plugin can be used in both contexts.

**Acceptance Scenarios**:

1. **Given** a type that implements both WebPlugin and Searchable, **When** type-asserting to Searchable, **Then** the assertion succeeds and search methods are callable
2. **Given** a type that implements only WebPlugin (not Searchable), **When** type-asserting to Searchable, **Then** the assertion fails gracefully
3. **Given** the Searchable interface, **When** reviewing its dependencies, **Then** it has no imports or dependencies on WebPlugin or web-specific types

---

### Edge Cases

- What happens when a search query matches thousands of items but max_results is set low?
  - Only max_results items are returned; total match count is not provided in this iteration
- How does the system handle a plugin that returns malformed URLs in search results?
  - URLs are returned as-is; validation is the plugin's responsibility
- What happens when sort_order is specified but items lack the required metadata (e.g., last_used not tracked)?
  - Items without the metadata are sorted to the end of results
- What happens when max_results is 0 or negative?
  - A max_results of 0 means "no limit" (return all matches); negative values are treated as 0
- How are case-sensitive vs case-insensitive searches handled?
  - Searches are case-insensitive by default; plugin implementations handle matching

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST define a SearchResult type containing at minimum: Title (string) and URL (string)
- **FR-002**: System MUST define a Searchable interface with a Search method that accepts query string, max_results integer, and sort_order parameter
- **FR-003**: Searchable interface MUST be independent of WebPlugin with no shared dependencies
- **FR-004**: Search method MUST support sort_order values: "relevance" (default), "created_date", "updated_date", "last_used", "name"
- **FR-005**: Search method MUST search across both item names and item content
- **FR-006**: Search method MUST respect the max_results parameter, returning at most that many results
- **FR-007**: Search method MUST return an empty slice (not nil) when no matches are found
- **FR-008**: System MUST allow WebPlugin implementations to optionally implement Searchable via interface composition
- **FR-009**: Search results MUST contain URLs that uniquely identify and link to the specific matched item
- **FR-010**: When sort_order is not specified or empty, results MUST be sorted by relevance (closest matches first)

### Key Entities

- **SearchResult**: Represents a single search match. Contains the title of the matched item and a URL that can be used to navigate directly to that item.
- **Searchable**: Interface contract for types that can be searched. Defines the Search method with query, pagination, and sorting parameters.
- **SortOrder**: Enumeration or string type representing valid sorting options (relevance, created_date, updated_date, last_used, name).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Plugins can implement search capability by adding a single interface (Searchable) without modifying existing WebPlugin implementation
- **SC-002**: Search results can be retrieved and sorted within 100ms for typical datasets (< 10,000 items)
- **SC-003**: 100% of search results contain valid, navigable URLs that load the correct item when accessed
- **SC-004**: Developers can understand and implement the Searchable interface in under 30 minutes using documentation and examples
- **SC-005**: The Searchable interface can be type-asserted from any compatible type without runtime errors

## Assumptions

- Relevance scoring is implementation-specific and left to each plugin (e.g., substring match, fuzzy match, weighted fields)
- Plugins are responsible for tracking metadata (created_date, updated_date, last_used) if they want to support those sort orders
- The search form UI will be implemented in a future iteration (not part of this feature)
- No CLI or MCP exposure is planned for this iteration
- URL format follows existing patterns in the web GUI (e.g., /profiles/{name}, /memories/{id})
