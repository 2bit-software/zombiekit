# Feature Specification: WebGUI Status Page

**Feature Branch**: `014-webgui-status`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "let's add status information to the home page of the webgui. I want stats on which database backend is being used, what version/git commit is being used, and as much other debug information as possible that we can provide."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View System Version Information (Priority: P1)

A developer or system administrator visits the home page of the WebGUI and immediately sees the current version of Brains, including the git commit hash, build date, and Go version used to compile the application.

**Why this priority**: Version information is the most critical debug information as it helps users understand exactly what software they're running, essential for bug reports and troubleshooting.

**Independent Test**: Can be fully tested by visiting the home page and verifying version details match the output of `brains version` command.

**Acceptance Scenarios**:

1. **Given** the WebGUI is running, **When** a user visits the home page, **Then** they see the application version prominently displayed
2. **Given** the application was built with version tags, **When** viewing the home page, **Then** the version, git commit, and build date are all visible
3. **Given** the application is a dev build, **When** viewing the home page, **Then** "dev" version and commit hash are shown appropriately

---

### User Story 2 - View Database Backend Status (Priority: P1)

A developer or administrator views the home page to quickly determine which database backend (SQLite or PostgreSQL) is currently in use, along with connection status.

**Why this priority**: Database backend information is equally critical as it affects how data is stored, what features are available, and helps troubleshoot data-related issues.

**Independent Test**: Can be fully tested by starting the GUI with different database configurations and verifying the correct backend is displayed.

**Acceptance Scenarios**:

1. **Given** the WebGUI is running with SQLite storage, **When** viewing the home page, **Then** "SQLite" is shown as the database backend with the database file path
2. **Given** the WebGUI is running with PostgreSQL storage, **When** viewing the home page, **Then** "PostgreSQL" is shown as the database backend with connection info (host/database name, not credentials)
3. **Given** the database connection is healthy, **When** viewing the home page, **Then** a "Connected" or healthy status indicator is shown

---

### User Story 3 - View Runtime Environment Information (Priority: P2)

A developer views the home page to see runtime environment details including the Go version, operating system, architecture, and process information.

**Why this priority**: Runtime environment helps diagnose platform-specific issues and provides context for bug reports.

**Independent Test**: Can be fully tested by viewing the home page and verifying runtime information matches the actual system environment.

**Acceptance Scenarios**:

1. **Given** the WebGUI is running, **When** viewing the home page, **Then** the Go runtime version is displayed
2. **Given** the WebGUI is running, **When** viewing the home page, **Then** the operating system and architecture (e.g., "darwin/arm64") are displayed
3. **Given** the WebGUI is running, **When** viewing the home page, **Then** the process start time or uptime is displayed

---

### User Story 4 - View Plugin Status (Priority: P2)

A developer views the home page to see a summary of registered plugins and their status.

**Why this priority**: Plugin information helps understand what features are active and aids in troubleshooting plugin-related issues.

**Independent Test**: Can be fully tested by viewing the home page and verifying the plugin list matches the registered plugins.

**Acceptance Scenarios**:

1. **Given** multiple plugins are registered, **When** viewing the home page, **Then** a count and list of registered plugins is shown
2. **Given** a plugin failed to initialize, **When** viewing the home page, **Then** an indication of the plugin's error state is visible
3. **Given** all plugins are healthy, **When** viewing the home page, **Then** a healthy status indicator is shown for each plugin

---

### User Story 5 - View Configuration Summary (Priority: P3)

A developer views the home page to see key configuration values that affect the application's behavior.

**Why this priority**: Configuration visibility helps understand non-default settings that may affect behavior.

**Independent Test**: Can be fully tested by viewing the home page with custom configuration and verifying settings are reflected.

**Acceptance Scenarios**:

1. **Given** the WebGUI is running on a custom port, **When** viewing the home page, **Then** the configured port is displayed
2. **Given** the WebGUI is running, **When** viewing the home page, **Then** the log level is displayed
3. **Given** custom profile directories are configured, **When** viewing the home page, **Then** the profile search paths are shown

---

### Edge Cases

- What happens when database connection is lost after startup? The status should show the last known state or indicate connection issues.
- How does the system handle when build information wasn't injected? Default values like "dev" and "unknown" should display gracefully.
- What happens when the profile service failed to initialize? Status should indicate the error without crashing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display the application version (version number, git commit hash, build date) on the home page
- **FR-002**: System MUST display the Go runtime version used to compile the application
- **FR-003**: System MUST display the current database backend type (SQLite or PostgreSQL)
- **FR-004**: System MUST display database-specific information (file path for SQLite, host/database name for PostgreSQL)
- **FR-005**: System MUST display a database connection status indicator (healthy/unhealthy)
- **FR-006**: System MUST display the operating system and architecture the application is running on
- **FR-007**: System MUST display process uptime or start time
- **FR-008**: System MUST display a count and summary of registered plugins
- **FR-009**: System MUST display the current log level configuration
- **FR-010**: System MUST display the HTTP server port
- **FR-011**: System MUST NOT expose sensitive information (database passwords, connection strings with credentials, API keys)
- **FR-012**: System MUST display all status information within the existing home page layout, maintaining visual consistency with the current design
- **FR-013**: System MUST continue to function even if some status information cannot be gathered (graceful degradation)

### Key Entities

- **StatusInfo**: Aggregate of all system status information including version details, database status, runtime info, plugin status, and configuration
- **DatabaseStatus**: Backend type, connection status, and sanitized connection details
- **PluginStatus**: Name, registration status, and health of each plugin

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can view all system status information without navigating away from the home page
- **SC-002**: The home page loads and displays status information within 500ms under normal conditions
- **SC-003**: 100% of status information is visible without scrolling on a standard desktop screen (above the fold for most common viewport sizes)
- **SC-004**: Users report the status page provides sufficient information to include in bug reports (version, commit, database type, OS)
- **SC-005**: Zero sensitive credentials are exposed in the status display
- **SC-006**: Status information remains accurate and updates if the user refreshes the page

## Assumptions

- The existing `internal/version` package provides version information via ldflags at build time
- The WebGUI server has access to all necessary status information through existing service references
- Database connection health can be determined through a simple query or ping
- Plugin registration status is available through the existing PluginRegistry
- The Go runtime package provides sufficient system information (OS, architecture, etc.)
