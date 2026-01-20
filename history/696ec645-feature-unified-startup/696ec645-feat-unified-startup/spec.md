# Feature Specification: Unified Startup Command

**Feature Branch**: `696ec645-feature-unified-startup`
**Created**: 2026-01-19
**Status**: Draft
**Linear Ticket**: DEV-89
**Input**: "ZombieKit should have a start command that reads from configuration and starts all configured services"

## User Scenarios & Testing

### User Story 1 - Start All Services with One Command (Priority: P1)

A developer wants to start their ZombieKit environment quickly without opening multiple terminals or remembering individual commands.

**Why this priority**: This is the core value proposition - single command startup. Without this, the feature has no value.

**Independent Test**: Run `brains start` with a valid configuration and verify all services are running and logs appear in the terminal.

**Acceptance Scenarios**:

1. **Given** a valid startup configuration exists, **When** user runs `brains start`, **Then** all configured services begin running and the terminal displays their combined output
2. **Given** services are running, **When** user views the terminal, **Then** they see interleaved logs from all services with each line prefixed by service name
3. **Given** a service is not configured (disabled), **When** user runs `brains start`, **Then** that service is not started but others proceed normally

---

### User Story 2 - Graceful Shutdown (Priority: P1)

A developer wants to stop all services cleanly when they're done working.

**Why this priority**: Critical for developer experience - ungraceful shutdown leaves orphan processes and corrupted state.

**Independent Test**: Start services, press Ctrl+C, verify all services stop and terminal is released within a reasonable timeout.

**Acceptance Scenarios**:

1. **Given** services are running, **When** user presses Ctrl+C, **Then** all services stop gracefully and the terminal is released
2. **Given** services are running, **When** user sends SIGTERM, **Then** all services stop gracefully
3. **Given** a service hangs during shutdown, **When** shutdown timeout expires (10s), **Then** the command exits and logs a warning about the hung service

---

### User Story 3 - Service Failure Notification (Priority: P1)

A developer needs to know when a service fails to start so they can diagnose and fix the issue.

**Why this priority**: Essential for debuggability - silent failures create frustrating debugging sessions.

**Independent Test**: Intentionally misconfigure a service (e.g., invalid port), run start, verify error is clearly reported.

**Acceptance Scenarios**:

1. **Given** a service fails to start, **When** the failure occurs, **Then** the user sees an error message indicating which service failed and why
2. **Given** service A fails, **When** startup runs, **Then** all services stop and the command exits with a non-zero status (fail-fast)
3. **Given** the configuration file is invalid/missing, **When** user runs start, **Then** a clear error message explains how to create/fix the configuration

---

### User Story 4 - Configure Services (Priority: P2)

A developer wants to customize which services start and their settings.

**Why this priority**: Important for flexibility, but users can work with defaults initially.

**Independent Test**: Create a configuration file, verify services respect the configuration (e.g., custom port).

**Acceptance Scenarios**:

1. **Given** user creates a `.brains/config.yml` file, **When** they specify `gui.port: 9999`, **Then** the GUI starts on port 9999
2. **Given** user sets `recall.enabled: false`, **When** they run start, **Then** the recall service is not started
3. **Given** no configuration file exists, **When** user runs start, **Then** services start with sensible defaults and a message suggests creating a config file

---

### Edge Cases

- What happens when the configuration file has syntax errors? → Clear YAML parse error with line number
- How does system handle port already in use? → Report which service failed and the port conflict
- What if database isn't running when recall starts? → Log connection error, recall service reports failure but doesn't crash other services
- What if user runs `brains start` when services are already running? → Detect port in use, fail fast with message

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide a `brains start` CLI command that starts configured services
- **FR-002**: System MUST read service configuration from `.brains/config.yml` (local) with fallback to `~/.brains/config.yml` (global), plus env var overrides
- **FR-003**: System MUST display combined log output from all services in the terminal
- **FR-004**: System MUST prefix each log line with the service name (e.g., `[gui]`, `[recall]`)
- **FR-005**: System MUST stop all services gracefully when receiving SIGINT or SIGTERM
- **FR-006**: System MUST report clear error messages when services fail to start
- **FR-007**: System MUST exit when any service fails, reporting which service failed and why (fail-fast behavior)
- **FR-008**: System MUST validate configuration before attempting to start any services
- **FR-009**: System MUST provide a shutdown timeout (10 seconds) after which forced exit occurs
- **FR-010**: System MUST support disabling individual services via configuration

### Key Entities

- **Service**: A runnable component (GUI, Recall) with name, enabled status, and service-specific settings
- **StartupConfiguration**: Collection of service configurations loaded from YAML file
- **ServiceRunner**: Abstraction for starting/stopping a service with lifecycle hooks

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can start all services with a single `brains start` command
- **SC-002**: Ctrl+C reliably stops all services within 10 seconds (or forces exit)
- **SC-003**: Service logs are clearly distinguishable by their prefix
- **SC-004**: Invalid configuration produces actionable error messages

## Testing Requirements

### Test Strategy

- Integration tests for the CLI command behavior
- Unit tests for configuration parsing and validation
- Manual testing for signal handling and log interleaving readability

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify `brains start` command exists and is callable |
| FR-002 | Unit | Test YAML config loading with various inputs |
| FR-003, FR-004 | Manual | Verify log output format in terminal |
| FR-005 | Integration | Send SIGINT, verify clean shutdown |
| FR-006, FR-007 | Integration | Start with invalid service config, verify error handling |
| FR-008 | Unit | Test configuration validation logic |

### Edge Case Coverage

- Invalid YAML syntax → Unit test for parse error handling
- Port conflict → Integration test with pre-bound port
- Missing config file → Integration test for default behavior
