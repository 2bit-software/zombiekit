# Feature Specification: Simplified Plugin Registration API

**Feature Branch**: `012-plugin-registration-api`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Simplify plugin registration to webgui.Register(name, impl) with automatic URL prefixing"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register Plugin with Explicit Name (Priority: P1)

A developer wants to register a plugin with the web GUI system. They call a registration function with the plugin name and implementation, rather than relying on the plugin to return its own ID. The system uses this name for routing, sidebar display, and URL construction.

**Why this priority**: Core architectural change - all other features depend on this registration pattern working correctly.

**Independent Test**: Can be fully tested by registering a plugin with a specific name and verifying it appears in the sidebar and responds to requests at the expected URL path.

**Acceptance Scenarios**:

1. **Given** a plugin implementation, **When** registered with name "memory", **Then** the plugin is accessible at paths starting with "/memory/"
2. **Given** a plugin registered with name "profiles", **When** the sidebar is rendered, **Then** an entry appears with the plugin's configured label
3. **Given** a duplicate plugin name, **When** registration is attempted, **Then** an error is returned indicating the name conflict

---

### User Story 2 - Plugin-Relative URLs (Priority: P1)

A plugin returns search results or navigation links using URLs relative to itself (e.g., "/my-notes" instead of "/memory/my-notes"). The system automatically prefixes these with the plugin's registered name when constructing full URLs for navigation.

**Why this priority**: Enables plugins to be portable and not hardcode their own name into URLs. Critical for the search feature integration.

**Independent Test**: Can be tested by having a plugin return a relative URL and verifying the browser navigates to the fully-qualified path.

**Acceptance Scenarios**:

1. **Given** a plugin registered as "memory" returning URL "/notes", **When** the system constructs a navigation link, **Then** the full URL becomes "/memory/notes"
2. **Given** a search result with URL "/config", **When** clicked from search dropdown, **Then** the browser navigates to "/{pluginName}/config"
3. **Given** a plugin returning URL "/item/123", **When** rendered in any context, **Then** the link correctly prefixes the plugin name

---

### User Story 3 - Automatic Route Mounting (Priority: P1)

When a plugin is registered, all of its route handlers are automatically mounted under the plugin's registered name path. The plugin's handlers receive requests with the plugin prefix already stripped.

**Why this priority**: Simplifies plugin implementation - plugins don't need to know their own mount path.

**Independent Test**: Register a plugin with routes and verify requests to "/{name}/..." are handled by the plugin's handlers.

**Acceptance Scenarios**:

1. **Given** a plugin registered as "memory" with route "/list", **When** a request is made to "/memory/list", **Then** the plugin's list handler receives the request
2. **Given** a plugin's route handler, **When** it constructs a redirect URL, **Then** it uses relative paths without the plugin prefix
3. **Given** multiple plugins registered, **When** requests arrive, **Then** each plugin only receives requests for its own path prefix

---

### User Story 4 - Sidebar Configuration (Priority: P2)

A plugin provides its sidebar navigation entries with relative paths. The system automatically prefixes these paths with the plugin's registered name for proper navigation.

**Why this priority**: Consistent with URL handling pattern; reduces plugin awareness of its mount location.

**Independent Test**: Register a plugin with sidebar items and verify rendered paths include the plugin prefix.

**Acceptance Scenarios**:

1. **Given** a plugin sidebar item with path "/", **When** rendered in sidebar, **Then** the link points to "/{pluginName}/"
2. **Given** a plugin sidebar item with path "/settings", **When** clicked, **Then** navigation goes to "/{pluginName}/settings"
3. **Given** multiple sidebar items from different plugins, **When** rendered, **Then** each has the correct plugin prefix

---

### Edge Cases

- What happens when a plugin tries to register with an empty name?
  - Registration fails with a validation error; empty names are not allowed
- What happens when a plugin returns a URL that already starts with its own prefix (e.g., plugin "foo" returns "/foo/bar")?
  - The system does not double-prefix; it detects the existing prefix and uses the URL as-is
- What happens when a plugin returns an absolute URL (e.g., "https://external.com")?
  - Absolute URLs are passed through unchanged; only relative URLs are prefixed
- What happens when a plugin name contains special characters or spaces?
  - Registration fails with a validation error; names must be URL-safe (alphanumeric and hyphens only)
- How are existing plugins migrated to the new registration pattern?
  - Plugins remove their ID() method; registration call changes from `registry.Register(plugin)` to `webgui.Register("name", plugin)`
- What happens if a plugin returns a URL starting with "/" vs not?
  - URLs are normalized: "/foo" and "foo" both become "/{pluginName}/foo"

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a registration function that accepts a name string and plugin implementation
- **FR-002**: Plugin names MUST be validated as URL-safe (alphanumeric characters and hyphens only)
- **FR-003**: Duplicate plugin names MUST be rejected with an error
- **FR-004**: System MUST automatically mount plugin routes under "/{pluginName}/..."
- **FR-005**: Plugin route handlers MUST receive requests with the plugin prefix already stripped
- **FR-006**: Plugins MUST return URLs without their plugin name prefix (relative to their own mount point)
- **FR-007**: System MUST automatically prefix plugin-relative URLs with the plugin name when constructing navigation links
- **FR-008**: Sidebar item paths from plugins MUST be automatically prefixed with the plugin name
- **FR-009**: Absolute URLs (starting with http:// or https://) MUST be passed through without modification
- **FR-010**: System MUST provide the plugin name to search result aggregation for display purposes
- **FR-011**: The WebPlugin interface MUST no longer require an ID() method (breaking change)
- **FR-012**: Registration MUST fail if plugin name is empty or contains invalid characters
- **FR-013**: Registration failures (duplicate names, invalid names) MUST panic at startup to fail fast for configuration errors

### Non-Functional Requirements

- **NFR-001**: System MUST log each plugin registration at Info level, including plugin name and mount path

### Key Entities

- **PluginRegistration**: Represents the binding between a name and plugin implementation
- **WebPlugin (modified)**: Interface for plugins, now without ID() method; only requires SidebarItems() and MountRoutes()
- **PluginRegistry (modified)**: Now stores name-to-plugin mappings where name comes from registration, not the plugin

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Plugin developers can register a plugin with a single function call specifying name and implementation
- **SC-002**: 100% of plugin-generated URLs correctly navigate to the intended content
- **SC-003**: Zero URL construction errors from missing or double-prefixed plugin names
- **SC-004**: Existing plugins can be migrated in under 5 minutes per plugin (remove ID method, update registration call)
- **SC-005**: Search results display and navigate correctly with the new URL construction pattern
- **SC-006**: All plugin relative paths (sidebar, search results, redirects) work correctly after migration

## Clarifications

### Session 2025-12-22

- Q: Should registration errors (duplicate names, invalid names) panic or return errors? → A: Panic at startup - fail fast for configuration errors
- Q: Should the registration system log plugin registrations for observability? → A: Log at Info level - log each plugin registration with name and path
- Q: How should plugins access their registered name if needed (e.g., for search result metadata)? → A: Constructor injection - name passed when creating plugin instance

## Assumptions

- Existing plugins (memory, profiles) will be updated to use the new registration pattern
- The change is applied before the webgui-search feature is completed (allows search to use new pattern)
- Plugin names are typically short, descriptive, lowercase identifiers (e.g., "memory", "profiles", "settings")
- Plugins that need their registered name (e.g., for search result metadata) receive it via constructor injection when the plugin instance is created
- The sidebar Label field remains independent of the plugin name (human-readable vs URL-safe)

## Migration Notes

This is a breaking change to the plugin API:

1. **WebPlugin interface changes**: Remove `ID() string` method requirement
2. **Registration changes**: From `registry.Register(plugin)` to `webgui.Register("name", plugin)`
3. **URL changes in plugins**: Return relative URLs like "/notes" instead of "/memory/notes"
4. **Sidebar path changes**: Return relative paths like "/" instead of "/memory"
5. **Search result URLs**: Return relative URLs; system prefixes automatically

Affected existing code:
- `internal/web/plugin.go` - Modify WebPlugin interface and PluginRegistry
- `internal/webplugins/memory/plugin.go` - Update to new pattern
- `internal/webplugins/profiles/plugin.go` - Update to new pattern
- `internal/web/server.go` - Update plugin mounting logic
