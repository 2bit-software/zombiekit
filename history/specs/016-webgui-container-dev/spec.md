# Feature Specification: WebGUI Container Development Environment

**Feature Branch**: `016-webgui-container-dev`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "add a taskfile entry that starts up the webgui in a container. Inside the container, it should be started using wgo (a file watcher) that reloads on changes. Use port 9981. Mount a local directory for the sqlite database in the container."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start WebGUI Development Server (Priority: P1)

As a developer, I want to start the webgui in a containerized development environment with a single command so that I can develop and test changes with automatic hot-reloading without affecting my local machine setup.

**Why this priority**: This is the core functionality - developers need to be able to start the development environment to do any work. Without this, no other features matter.

**Independent Test**: Can be fully tested by running the taskfile command and verifying the webgui is accessible at http://localhost:9981 with file watching active.

**Acceptance Scenarios**:

1. **Given** the project is cloned and Docker is running, **When** the developer runs the taskfile command for webgui development, **Then** a container starts with the webgui accessible on port 9981
2. **Given** the webgui container is running, **When** the developer makes a change to any Go source file, **Then** wgo detects the change and automatically rebuilds and restarts the webgui within the container
3. **Given** the webgui container is running, **When** the developer accesses http://localhost:9981 from their browser, **Then** they see the webgui interface functioning normally

---

### User Story 2 - Persistent SQLite Data (Priority: P1)

As a developer, I want SQLite database files to persist between container restarts so that my test data is not lost during development sessions.

**Why this priority**: Data persistence is essential for effective development - losing test data on every restart would severely impact productivity and testing capability.

**Independent Test**: Can be fully tested by creating data in the webgui, stopping the container, restarting it, and verifying the data still exists.

**Acceptance Scenarios**:

1. **Given** the webgui container is running and data has been stored in SQLite, **When** the container is stopped and restarted, **Then** all previously stored data is still accessible
2. **Given** the local SQLite data directory exists with an existing database, **When** the webgui container starts, **Then** it uses the existing database file instead of creating a new one
3. **Given** the local SQLite data directory does not exist, **When** the webgui container starts for the first time, **Then** the directory is created and a new database is initialized

---

### User Story 3 - Stop Development Server (Priority: P2)

As a developer, I want to cleanly stop the webgui development container so that resources are freed and ports are released.

**Why this priority**: While starting is more critical, clean shutdown is necessary for proper resource management and avoiding port conflicts.

**Independent Test**: Can be fully tested by starting the container, then stopping it and verifying port 9981 is released and no orphaned processes remain.

**Acceptance Scenarios**:

1. **Given** the webgui container is running, **When** the developer uses standard container stop commands (Ctrl+C or docker stop), **Then** the container stops gracefully and port 9981 is released
2. **Given** the webgui container has been stopped, **When** the developer runs the start command again, **Then** a new container starts without conflicts

---

### Edge Cases

- What happens when port 9981 is already in use by another process?
  - The container should fail to start with a clear error message indicating the port conflict
- What happens when Docker is not running?
  - The taskfile command should fail with a clear error message indicating Docker is required
- What happens when the SQLite database file is corrupted?
  - The webgui should fail to start with an error message; developer can delete and recreate the database
- What happens when wgo fails to detect file changes?
  - This is a wgo limitation; developer can manually restart the container
- What happens when disk space runs out in the mounted volume?
  - SQLite operations will fail with appropriate error messages; developer needs to free space

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a Taskfile entry named appropriately (e.g., `webgui:dev` or `webgui-dev`) that invokes Docker Compose to start the webgui development container
- **FR-002**: System MUST define a Docker Compose service with a Dockerfile that includes wgo for file watching and hot-reloading
- **FR-003**: System MUST expose port 9981 from the container to the host machine
- **FR-004**: System MUST mount the `.data/` directory to the container for SQLite database persistence
- **FR-005**: System MUST configure wgo inside the container to watch for Go source file changes and rebuild/restart the webgui automatically
- **FR-006**: System MUST mount the source code directory into the container so that wgo can detect changes made on the host
- **FR-007**: System MUST ensure the container has all necessary Go dependencies to build and run the webgui
- **FR-008**: System MUST provide clear console output showing when the webgui has started and is ready to accept connections
- **FR-009**: System MUST handle graceful shutdown when the container receives stop signals

### Key Entities

- **Taskfile Entry**: A task definition in Taskfile.yml that invokes Docker Compose to start the development environment
- **Docker Compose Service**: A service definition in docker-compose.yml for the webgui development container
- **Docker Container**: The runtime environment running wgo and the webgui application
- **SQLite Database Volume**: The `.data/` directory mounted into the container to persist SQLite database files between restarts
- **Source Code Volume**: A mounted directory containing Go source files for wgo to watch

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developer can start the webgui development environment with a single command in under 30 seconds (cold start with image build may take longer on first run)
- **SC-002**: File changes are detected and the webgui is rebuilt and restarted within 5 seconds of saving a source file
- **SC-003**: Data stored in SQLite persists across 100% of container restart cycles
- **SC-004**: WebGUI is accessible at http://localhost:9981 immediately after the container reports ready status
- **SC-005**: No manual configuration or environment setup is required beyond having Docker installed and running

## Clarifications

### Session 2025-12-22

- Q: Where should the SQLite database be persisted on the host machine? → A: `.data/` directory (hidden, gitignored by convention)
- Q: Should this use Docker Compose or standalone Docker commands? → A: Docker Compose (docker-compose.yml with `task` invoking compose)

## Assumptions

- Docker and Docker Compose are installed and running on the developer's machine
- The developer has access to the Taskfile CLI (task command)
- wgo is available as a Go module or will be installed within the container during build
- The existing webgui application can be started with a standard Go command
- The local data directory is `.data/` in the project root (gitignored)
- The existing Docker infrastructure (if any) in the project will be leveraged or extended
