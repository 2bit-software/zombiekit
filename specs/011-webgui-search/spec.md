# Feature Specification: Web GUI Search Bar

**Feature Branch**: `011-webgui-search`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Add web GUI search bar with debounced autocomplete across all plugins"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Search Across Plugins (Priority: P1)

A user wants to quickly find content from any registered plugin. They type a query into the search bar at the top of the page and see real-time autocomplete results from all plugins that implement the Searchable interface, limited to 3 results per plugin. Results appear in a dropdown below the search bar with relevant information displayed.

**Why this priority**: Core value proposition - without a working search bar, the entire feature has no purpose. This enables users to discover content across the entire application.

**Independent Test**: Can be fully tested by typing a search query and verifying that results appear from plugins implementing Searchable, with correct titles and URLs displayed.

**Acceptance Scenarios**:

1. **Given** at least one plugin implementing Searchable with matching content, **When** the user types a query, **Then** matching results appear in a dropdown within 500ms of typing pause
2. **Given** multiple plugins implementing Searchable, **When** the user searches, **Then** results from each plugin are displayed (up to 3 per plugin)
3. **Given** a search query with no matches, **When** the user types, **Then** the dropdown shows "No results found" message

---

### User Story 2 - Debounced Search Input (Priority: P1)

As users type quickly, the system waits for a brief pause before executing the search to avoid excessive server requests. This provides a smooth experience without overwhelming the server or causing UI flickering.

**Why this priority**: Essential for usability - without debouncing, rapid typing would cause poor performance and jarring UI updates.

**Independent Test**: Type multiple characters quickly and verify that only one search request is made after typing stops, not one per character.

**Acceptance Scenarios**:

1. **Given** a user typing rapidly, **When** they type 5 characters within 200ms, **Then** only one search request is made after typing stops
2. **Given** a user pausing while typing, **When** they pause for 300ms or more, **Then** a search request is initiated
3. **Given** a user typing, **When** they type additional characters before debounce delay expires, **Then** the previous pending search is cancelled

---

### User Story 3 - Navigate to Search Result (Priority: P1)

A user sees a search result they want and clicks on it. The application navigates to that item's page, loading the content while preserving the full page layout (sidebar, search bar, etc.). The URL in the browser updates to reflect the new location.

**Why this priority**: Critical path - if users cannot navigate to results, the search feature is useless. Navigation must maintain app context.

**Independent Test**: Click a search result and verify the content area updates with the correct page while sidebar and search remain visible, and browser URL changes.

**Acceptance Scenarios**:

1. **Given** a search result from the memory plugin showing "/memory/my-notes", **When** the user clicks it, **Then** the content area loads the memory view page for "my-notes"
2. **Given** a search result clicked, **When** navigation completes, **Then** the browser URL reflects the result's URL (e.g., "/memory/my-notes")
3. **Given** a search result clicked, **When** navigation completes, **Then** the sidebar remains visible and the search bar remains functional
4. **Given** a search result clicked, **When** navigation completes, **Then** the search dropdown closes

---

### User Story 4 - Search Result Display (Priority: P2)

Each search result in the dropdown shows the item title and a visual indicator of which plugin it came from (e.g., plugin name or icon). This helps users identify the type of result before clicking.

**Why this priority**: Enhances usability by providing context about results, but basic functionality works without it.

**Independent Test**: Execute a search and verify each result displays title and plugin source indicator.

**Acceptance Scenarios**:

1. **Given** a search result from the memory plugin, **When** displayed in dropdown, **Then** it shows the memory title and indicates it's from "Memory"
2. **Given** results from multiple plugins, **When** displayed together, **Then** results are visually grouped or labeled by plugin source
3. **Given** a search result title that is very long, **When** displayed, **Then** it is truncated gracefully with ellipsis

---

### User Story 5 - Empty State and Loading (Priority: P2)

When the user begins typing, a loading indicator appears until results arrive. If no plugins implement Searchable or the search bar is empty, appropriate messaging is displayed.

**Why this priority**: Good UX practice but not blocking core functionality.

**Independent Test**: Type a query and observe loading state before results appear; clear the input and verify empty state.

**Acceptance Scenarios**:

1. **Given** a user typing a query, **When** the search is in progress, **Then** a loading indicator is visible
2. **Given** no plugins implement Searchable, **When** user tries to search, **Then** a message indicates search is unavailable
3. **Given** the search input is empty, **When** the user focuses it, **Then** no dropdown appears until they type

---

### User Story 6 - Keyboard Navigation (Priority: P3)

Users can navigate search results using keyboard arrows and select with Enter, providing accessibility and power-user efficiency.

**Why this priority**: Accessibility enhancement; basic mouse interaction provides equivalent functionality.

**Independent Test**: Use arrow keys to navigate results and Enter to select; verify correct behavior.

**Acceptance Scenarios**:

1. **Given** search results are displayed, **When** user presses Down arrow, **Then** the next result is highlighted
2. **Given** a result is highlighted, **When** user presses Enter, **Then** navigation to that result occurs
3. **Given** a result is highlighted, **When** user presses Escape, **Then** the dropdown closes

---

### Edge Cases

- What happens when a search query matches thousands of items across multiple plugins?
  - Each plugin returns at most 3 results (per maxResults); total dropdown shows up to 3 × number of searchable plugins
- How are plugins that do not implement Searchable handled?
  - They are silently skipped; only plugins implementing Searchable are queried
- What happens if a plugin's Search method returns an error?
  - The error is logged; other plugins' results are still displayed; the failing plugin shows no results
- What happens when the user navigates to a search result that no longer exists?
  - Standard 404 handling; the plugin's view handler determines the error response
- How does the search bar behave on mobile/narrow viewports?
  - The search bar should be responsive and accessible on mobile (implementation detail)
- What happens if a user submits the form (presses Enter) with text in the input but no result selected?
  - No action is taken; Enter only navigates when a result is highlighted

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display a search bar in the page header/top area, visible on all pages
- **FR-002**: System MUST debounce search input with a configurable delay (default 300ms)
- **FR-003**: System MUST query all registered plugins that implement the Searchable interface
- **FR-004**: System MUST limit results to 3 items per plugin
- **FR-005**: System MUST display search results in a dropdown below the search bar
- **FR-006**: Each search result MUST display the item title and plugin source indicator
- **FR-007**: Clicking a search result MUST navigate to the result's URL using HTMX partial page updates
- **FR-008**: Navigation MUST preserve the shell layout (sidebar, header, search bar)
- **FR-009**: Navigation MUST update the browser URL to match the result's destination
- **FR-010**: System MUST close the search dropdown after a result is selected
- **FR-011**: System MUST display a loading indicator while search is in progress
- **FR-012**: System MUST display "No results found" when a query matches no items
- **FR-013**: Search results MUST return URLs relative to the plugin's mount point (e.g., "/memory/my-notes")
- **FR-014**: System MUST support keyboard navigation (arrows to move, Enter to select, Escape to close)

### Key Entities

- **SearchResult**: Contains Title (string) and URL (string) - already defined in internal/search package
- **SearchResponse**: Aggregated results from multiple plugins, grouped by plugin ID/name
- **SearchBar Component**: UI component in page header handling input, debouncing, and result display

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can find content across all searchable plugins within 3 keystrokes of their target
- **SC-002**: Search results appear within 500ms of the last keystroke (debounce + query time)
- **SC-003**: 100% of displayed results link to valid, accessible pages
- **SC-004**: Users can complete a search-and-navigate flow in under 5 seconds
- **SC-005**: Search bar is accessible on 100% of pages in the web GUI (present in shell template)
- **SC-006**: Zero search requests are made while the user is actively typing (debounce functioning)

## Assumptions

- The Searchable interface is already implemented by the memory plugin (verified from codebase review)
- Profiles plugin search implementation will be done in a future iteration
- HTMX is available and used for partial page updates (present in shell.html)
- The existing shell.html layout can be modified to include the search bar
- Plugin-relative URLs (e.g., "/memory/my-notes") work correctly with the existing routing
- 300ms is an appropriate debounce delay (industry standard for search inputs)
- 3 results per plugin provides sufficient context without overwhelming the dropdown
