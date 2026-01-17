# Feature Specification: Sticky Memory Web Plugin

**Feature Branch**: `009-sticky-memory-plugin`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Implement the frontend for the sticky memory tool with list, view, create, edit, delete, and search capabilities. Include markdown rendering toggle for viewing content as rendered or source."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse Memory List (Priority: P1) 🎯 MVP

Users navigate to the sticky memory section and see a searchable, paginated list of all stored memories with key metadata.

**Why this priority**: This is the entry point for all memory interactions. Users cannot use any other feature without first seeing what memories exist.

**Independent Test**: Can be fully tested by opening the memories page and verifying the list displays with name, size, version, and timestamps. Delivers immediate value by showing users their stored memories.

**Acceptance Scenarios**:

1. **Given** the user has no stored memories, **When** they navigate to the memories page, **Then** they see an empty state message with a prompt to create their first memory
2. **Given** the user has 25 stored memories, **When** they navigate to the memories page, **Then** they see a paginated list showing the first 20 memories sorted by most recently updated
3. **Given** the user is viewing the memory list, **When** they enter a search term, **Then** the list filters to show only memories whose name or content matches the search

---

### User Story 2 - View Memory Content (Priority: P1) 🎯 MVP

Users click on a memory from the list to view its full content and metadata in a dedicated view page.

**Why this priority**: After browsing, viewing content is the most fundamental read operation. Users need to see what they stored.

**Independent Test**: Can be tested by clicking any memory in the list and verifying content displays with toggle between rendered markdown and source view.

**Acceptance Scenarios**:

1. **Given** a memory exists with markdown content, **When** the user clicks to view it, **Then** they see the content rendered as markdown by default
2. **Given** the user is viewing a memory in rendered mode, **When** they click "View Source", **Then** the display switches to show raw markdown with syntax highlighting
3. **Given** the user is viewing a memory, **When** they view the metadata section, **Then** they see name, size, version number, created date, and last updated date

---

### User Story 3 - Create New Memory (Priority: P1) 🎯 MVP

Users create a new memory by providing a name and content through a simple form.

**Why this priority**: Without the ability to create memories, the tool has no utility. This completes the minimum viable product.

**Independent Test**: Can be tested by clicking "New Memory", filling out the form, and verifying the new memory appears in the list.

**Acceptance Scenarios**:

1. **Given** the user is on the create page, **When** they enter a valid name and content, **Then** the memory is saved and they are redirected to the memory list
2. **Given** the user submits a form with an empty name, **When** the form is submitted, **Then** they see a validation error without losing their content
3. **Given** the user creates a memory, **When** they view the list, **Then** the new memory appears at the top (most recent)

---

### User Story 4 - Edit Existing Memory (Priority: P2)

Users edit the content of an existing memory, creating a new version.

**Why this priority**: Important for maintaining accurate information but requires view and create to work first.

**Independent Test**: Can be tested by clicking edit on any memory, modifying content, saving, and verifying the version number increments.

**Acceptance Scenarios**:

1. **Given** the user views a memory and clicks "Edit", **When** the edit form loads, **Then** it is pre-populated with the current content
2. **Given** the user modifies content and saves, **When** they view the memory again, **Then** the version number has incremented and updated timestamp reflects the change
3. **Given** the user is editing, **When** they cancel without saving, **Then** no changes are made and they return to the previous view

---

### User Story 5 - Delete Memory (Priority: P2)

Users delete a memory they no longer need, with confirmation to prevent accidents.

**Why this priority**: Data cleanup is important but less critical than core CRUD operations.

**Independent Test**: Can be tested by clicking delete on a memory, confirming the action, and verifying it no longer appears in the list.

**Acceptance Scenarios**:

1. **Given** the user clicks delete on a memory, **When** the confirmation dialog appears, **Then** they must explicitly confirm before deletion proceeds
2. **Given** the user confirms deletion, **When** they return to the list, **Then** the memory no longer appears
3. **Given** the user clicks delete but cancels, **When** they return to the list, **Then** the memory remains unchanged

---

### User Story 6 - Search and Filter (Priority: P2)

Users search for specific memories by name or content to quickly find what they need.

**Why this priority**: Becomes critical as the number of memories grows, but basic browsing works without it.

**Independent Test**: Can be tested by entering a search term and verifying only matching memories appear in the results.

**Acceptance Scenarios**:

1. **Given** the user has memories named "project-notes" and "meeting-summary", **When** they search for "project", **Then** only "project-notes" appears in results
2. **Given** a memory contains the text "important deadline", **When** the user searches for "deadline", **Then** that memory appears in results (content search)
3. **Given** the user clears the search field, **When** the list refreshes, **Then** all memories are shown again

---

### User Story 7 - Toggle Markdown View Mode (Priority: P3)

Users switch between rendered markdown and raw source view when viewing memory content.

**Why this priority**: Enhances the reading experience but core functionality works without it.

**Independent Test**: Can be tested by viewing a memory with markdown, clicking the toggle button, and verifying the view switches.

**Acceptance Scenarios**:

1. **Given** a memory contains markdown headers and code blocks, **When** viewed in rendered mode, **Then** headers display as styled text and code blocks have syntax highlighting
2. **Given** the user toggles to source mode, **When** viewing the same content, **Then** they see the raw markdown syntax as plain text with monospace font
3. **Given** the user toggles view mode, **When** they navigate away and return, **Then** the default mode (rendered) is restored

---

### Edge Cases

- What happens when a memory name contains special characters? System sanitizes the name automatically using URL-safe transformations.
- How does the system handle very large content (approaching 1MB limit)? Display truncated preview in list view, full content in detail view with scroll.
- What happens if the user tries to create a memory with an existing name? The existing memory is updated (new version created), not rejected.
- How does search handle case sensitivity? Search is case-insensitive for both name and content matching.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display a list of all stored memories with name, size, version, and timestamps
- **FR-002**: System MUST allow users to view the full content of any memory
- **FR-003**: System MUST allow users to create new memories with a name and content
- **FR-004**: System MUST allow users to edit existing memories, creating new versions
- **FR-005**: System MUST allow users to delete memories with confirmation
- **FR-006**: System MUST provide search functionality that filters by name and content
- **FR-007**: System MUST render markdown content when viewing memories
- **FR-008**: System MUST provide a toggle to view raw markdown source
- **FR-009**: System MUST integrate with the existing web plugin architecture (sidebar navigation, HTMX updates)
- **FR-010**: System MUST handle empty states with appropriate messaging
- **FR-011**: System MUST display appropriate error messages when operations fail
- **FR-012**: System MUST support pagination with configurable limits (10, 20, 50, 100 items)
- **FR-013**: System MUST show metadata including size (formatted as bytes/KB/MB), version number, creation date, and last update date

### Key Entities

- **Memory**: A named piece of content with version history. Key attributes: name (unique identifier), content (markdown text up to 1MB), version (auto-incrementing integer), created timestamp, updated timestamp
- **MemoryMetadata**: Lightweight representation for list display. Key attributes: name, size in bytes, version, timestamps

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can browse, search, and view memories within 1 second of page load
- **SC-002**: Users can create a new memory in under 30 seconds (from clicking "New" to seeing it in the list)
- **SC-003**: 100% of memory operations (create, read, update, delete) complete without data loss
- **SC-004**: Users can find a specific memory among 100+ entries in under 10 seconds using search
- **SC-005**: Markdown content renders correctly for common elements (headers, code blocks, links, lists, emphasis)
- **SC-006**: All CRUD operations are accessible via keyboard navigation

## Clarifications

### Session 2025-12-22

- Q: Which client-side markdown rendering approach should be used? → A: CDN-loaded library (e.g., marked.js) - lightweight, no build step

## Assumptions

- Markdown rendering uses a CDN-loaded library (e.g., marked.js) consistent with existing CDN patterns for Tailwind and HTMX
- The existing memory storage service (`internal/memory`) is already functional and will be reused
- The web plugin architecture from feature 008 provides sidebar integration and HTMX support
- Memory content is primarily markdown text; binary content is out of scope
- Single-user local development tool; no concurrent access concerns
- Default pagination of 20 items per page is appropriate for typical use
- Client-side markdown rendering is acceptable (no server-side pre-rendering needed)

## Dependencies

- Feature 008 (Plugin Web GUI Architecture) must be complete for sidebar and HTMX integration
- Existing `internal/memory` package provides the Storage interface
- Existing MCP stickymemory tool provides the backend operations
