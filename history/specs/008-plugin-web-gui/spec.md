# Feature Specification: Plugin-Style Web GUI Architecture

**Feature Branch**: `008-plugin-web-gui`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Plugin-style web GUI architecture with self-registering tools, implementing Chi router, html/template, Tailwind CSS, and HTMX for a single example plugin"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Registered Tools in Sidebar (Priority: P1)

A user opens the web GUI in their browser and sees a sidebar listing all registered tools. The sidebar is dynamically populated based on which plugins have been registered with the system. Each tool appears as a clickable navigation item.

**Why this priority**: The sidebar is the core navigation mechanism. Without it, users cannot discover or access any tools. This is the foundational UI element that all other functionality depends on.

**Independent Test**: Can be tested by starting the web server with one or more plugins registered and verifying the sidebar displays their navigation items in the correct order.

**Acceptance Scenarios**:

1. **Given** the web server is running with one plugin registered, **When** a user visits the home page, **Then** they see the plugin's sidebar item displayed in the navigation.
2. **Given** the web server is running with three plugins registered with different order values, **When** a user visits the home page, **Then** the sidebar items appear sorted by their configured order (lowest first).
3. **Given** the web server is running with no plugins registered, **When** a user visits the home page, **Then** they see an empty sidebar with appropriate messaging.

---

### User Story 2 - Navigate to Tool Content (Priority: P1)

A user clicks on a tool in the sidebar and the main content area updates to show that tool's content. The navigation is fast and seamless, updating only the content area without a full page reload.

**Why this priority**: Navigation between tools is the primary user interaction. Users must be able to switch between tools fluidly to accomplish their tasks.

**Independent Test**: Can be tested by clicking sidebar items and verifying the content area updates with the correct plugin's content while the sidebar remains stable.

**Acceptance Scenarios**:

1. **Given** a user is viewing the home page, **When** they click a tool in the sidebar, **Then** the content area updates to show that tool's list view and the browser URL updates to reflect the new location.
2. **Given** a user is viewing Tool A's content, **When** they click Tool B in the sidebar, **Then** the content area updates to show Tool B's content without a full page refresh.
3. **Given** a user navigates to a tool via sidebar click, **When** they use the browser's back button, **Then** the previous view is restored correctly.

---

### User Story 3 - View Tool Detail Page (Priority: P2)

A user viewing a tool's list can click on an individual item to see its detail view. The plugin controls what information is shown and how it's formatted.

**Why this priority**: Detail views are essential for working with individual items, but depend on the basic navigation being functional first.

**Independent Test**: Can be tested by navigating to a plugin's list view, clicking an item, and verifying the detail view loads with correct data.

**Acceptance Scenarios**:

1. **Given** a user is viewing a tool's list page, **When** they click on an item, **Then** the content area updates to show that item's detail view.
2. **Given** a user is viewing an item's detail page, **When** they click a "back to list" link, **Then** they return to the list view.

---

### User Story 4 - Full Page Load Support (Priority: P2)

A user can directly navigate to any URL in the application (via bookmark, shared link, or manual entry) and see the correct content with full page layout including sidebar.

**Why this priority**: Deep linking is essential for shareability and bookmarking, but the core navigation must work first.

**Independent Test**: Can be tested by directly entering a tool's URL in the browser and verifying the full page renders correctly.

**Acceptance Scenarios**:

1. **Given** a user has no page currently loaded, **When** they navigate directly to a tool's URL, **Then** they see the complete page with sidebar, header, and the tool's content.
2. **Given** a user bookmarks a specific tool page, **When** they later visit that bookmark, **Then** the correct tool content is displayed with proper navigation context.

---

### User Story 5 - Example Plugin: Profiles (Priority: P1)

A user can view, browse, and inspect profiles through the web GUI (read-only). The profiles plugin serves as the reference implementation demonstrating how to build a web plugin.

**Why this priority**: Without at least one working plugin, there is nothing to display or test. The profiles plugin is the proof of concept that validates the architecture.

**Independent Test**: Can be tested by starting the server, navigating to the profiles section, and interacting with the profile list and detail views.

**Acceptance Scenarios**:

1. **Given** the profiles plugin is registered, **When** a user navigates to the profiles section, **Then** they see a list of available profiles.
2. **Given** profiles exist in the system, **When** a user clicks on a profile name, **Then** they see the profile's details including its content.
3. **Given** no profiles exist, **When** a user navigates to the profiles section, **Then** they see an empty state message.

---

### Edge Cases

- What happens when a plugin returns an empty sidebar items list? The plugin should still be mountable but won't appear in navigation.
- How does the system handle two plugins registering with the same ID? The registry should reject the duplicate and return an error during startup.
- What happens when a user navigates to a non-existent plugin path? The system should return a 404 page.
- How does the system behave when a plugin's handler returns an error? The error should be caught by recovery middleware and display an error message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a plugin interface that tools implement to participate in the web GUI.
- **FR-002**: System MUST allow plugins to declare sidebar navigation items with label, path, icon, and sort order.
- **FR-003**: System MUST mount each plugin's routes under a namespaced path based on plugin ID.
- **FR-004**: System MUST aggregate sidebar items from all registered plugins and display them sorted by order.
- **FR-005**: System MUST render a shell layout with sidebar navigation and a content area.
- **FR-006**: System MUST detect partial update requests and return only content (no shell) for those requests.
- **FR-007**: System MUST render full page (shell + content) for direct navigation requests.
- **FR-008**: System MUST support browser history navigation (back/forward) for partial updates.
- **FR-009**: System MUST provide a plugin registry where plugins are explicitly registered at startup.
- **FR-010**: System MUST reject registration of plugins with duplicate IDs.
- **FR-011**: System MUST inject a renderer into request context for plugins to use.
- **FR-012**: System MUST include an example "profiles" plugin demonstrating the architecture with read-only operations (list and view).
- **FR-013**: System MUST serve static assets (CSS, JavaScript) embedded in the binary.
- **FR-014**: System MUST support graceful shutdown of the web server.
- **FR-015**: System MUST emit structured request logs including request path, response status code, and request duration for each HTTP request.
- **FR-016**: System MUST display a dashboard overview at the root URL (`/`) showing a welcome message and links to all registered plugins.

### Key Entities

- **WebPlugin**: An interface that tools implement; provides ID, sidebar items, and route mounting.
- **SidebarItem**: Navigation entry with ID, label, path, icon name, sort order, optional badge, and optional children.
- **PluginRegistry**: Central store of all registered plugins; provides aggregated sidebar items.
- **Renderer**: Template rendering service that handles full-page vs partial rendering based on request type.
- **PageData**: Common data structure passed to templates containing sidebar items, active path, and plugin-specific content.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can navigate between any two registered plugins in under 500ms perceived response time.
- **SC-002**: Partial navigation requests transfer less than 50% of the data compared to full page loads.
- **SC-003**: Adding a new plugin requires implementing only the plugin interface (no modifications to core web infrastructure).
- **SC-004**: The example profiles plugin demonstrates list view, detail view, and navigation within 100 lines of handler code.
- **SC-005**: Direct URL access to any valid plugin path renders correctly on first load.
- **SC-006**: Browser back/forward navigation works correctly 100% of the time for partial updates.

## Assumptions

- The existing profile service from previous features (003-profiles) is available for the example plugin to use.
- Tailwind CSS will be loaded via CDN for simplicity in the initial implementation (can be optimized later).
- HTMX will be loaded via CDN for simplicity in the initial implementation.
- The web server will run on a configurable port (default 8080).
- Template hot-reloading in development mode is out of scope for this feature but the architecture should not preclude it.
- Authentication and authorization are out of scope for this feature.

## Clarifications

### Session 2025-12-22

- Q: What logging/observability level should the web GUI provide? → A: Structured request logging (log each request with path, status, duration)
- Q: What should the home page (`/`) display? → A: Dashboard overview (welcome page with links to all plugins; foundation for future custom dashboard)
- Q: Should the profiles example plugin support write operations (create/edit/delete)? → A: Read-only (list and view profiles only)

## Scope Boundaries

**In Scope**:
- Core plugin interface definition
- Plugin registry implementation
- Shell layout with sidebar and content area
- HTMX-based partial page updates
- Chi router integration
- html/template based rendering
- Tailwind CSS styling (via CDN)
- One example plugin (profiles)
- Static asset serving via embed.FS

**Out of Scope**:
- Authentication/authorization
- WebSocket real-time updates
- Template hot-reloading
- Custom per-plugin static assets
- Plugin lifecycle hooks (Init/Shutdown)
- Multiple themes or dark mode
- Mobile-responsive design optimizations
