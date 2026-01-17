# Feature Specification: PostgreSQL Configuration with SQLite Fallback

**Feature Branch**: `015-postgres-config`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "I want to add postgres connection information to the ZombieKit configuration file. It's ok to have secrets/passwords in the file, since this is local development. If the postgres connection works, it should connect, if it doesn't, it should fall back to a sql lite db."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure PostgreSQL via Config File (Priority: P1)

A developer wants to configure ZombieKit to use their local PostgreSQL database by adding connection details to a configuration file, rather than setting environment variables. They add the postgres connection URL to their `.brains/config.toml` file and the application connects to PostgreSQL on startup.

**Why this priority**: This is the core functionality requested - allowing users to configure PostgreSQL connection details in a persistent configuration file for local development convenience.

**Independent Test**: Can be fully tested by creating a config file with valid PostgreSQL credentials and verifying the application connects to PostgreSQL.

**Acceptance Scenarios**:

1. **Given** a valid `.brains/config.toml` file with postgres connection details, **When** the application starts, **Then** it connects to the configured PostgreSQL database
2. **Given** a config file with postgres URL, **When** the user runs any command that uses storage, **Then** all data operations use PostgreSQL

---

### User Story 2 - Automatic Fallback to SQLite (Priority: P1)

A developer has PostgreSQL configured in their config file, but the database is unavailable (stopped, network issue, wrong credentials). The application automatically falls back to using SQLite so their work is not blocked.

**Why this priority**: This is equally critical as P1 since it ensures the developer experience remains smooth even when PostgreSQL is unavailable, preventing work interruption.

**Independent Test**: Can be tested by configuring invalid PostgreSQL credentials and verifying the application seamlessly uses SQLite instead.

**Acceptance Scenarios**:

1. **Given** a config file with invalid postgres credentials, **When** the application starts, **Then** it falls back to SQLite and logs a warning about the fallback
2. **Given** a config file with postgres URL but the database server is unreachable, **When** the user runs a command, **Then** operations proceed using SQLite without blocking
3. **Given** fallback has occurred, **When** the user views system status, **Then** they can see which storage backend is actually in use

---

### User Story 3 - Environment Variables Override Config File (Priority: P2)

A developer has PostgreSQL configured in their config file for local development, but wants to temporarily override it with environment variables (e.g., for testing against a different database or forcing SQLite).

**Why this priority**: Important for flexibility but secondary to the core file-based configuration.

**Independent Test**: Can be tested by setting environment variables and verifying they take precedence over config file values.

**Acceptance Scenarios**:

1. **Given** a config file with postgres URL and BRAINS_BACKEND=sqlite environment variable, **When** the application starts, **Then** it uses SQLite regardless of config file
2. **Given** a config file with postgres URL and BRAINS_POSTGRES_URL environment variable set to different value, **When** the application starts, **Then** it uses the environment variable's connection string

---

### Edge Cases

- What happens when the config file is malformed? System logs error and uses default settings (SQLite)
- What happens when fallback occurs mid-session (e.g., PostgreSQL becomes unavailable)? Current operation fails with error; subsequent operations can use fallback on restart
- What happens when PostgreSQL becomes available after fallback? SQLite remains authoritative for the session; user must restart application to re-attempt PostgreSQL connection
- What happens when both local and global config files exist? Local config takes precedence
- What happens when connection times out during startup? After timeout (configurable, default 5 seconds), system falls back to SQLite

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST read PostgreSQL connection configuration from `.brains/config.toml` (local) or `~/.config/brains/config.toml` (global)
- **FR-002**: System MUST support storing the full PostgreSQL connection URL including password in the config file
- **FR-003**: System MUST attempt to connect to PostgreSQL when configured and validate the connection at startup
- **FR-004**: System MUST automatically fall back to SQLite when PostgreSQL connection fails (unreachable, bad credentials, timeout)
- **FR-005**: System MUST log a warning message when fallback to SQLite occurs, including the reason
- **FR-006**: System MUST allow environment variables to override config file values
- **FR-007**: System MUST support configuring connection pool settings (max connections, min connections) in the config file
- **FR-008**: System MUST support configuring connection timeout in the config file
- **FR-009**: System MUST expose the current storage backend status through the existing status display
- **FR-010**: System MUST NOT automatically reconnect to PostgreSQL after fallback occurs; user must restart application to re-attempt connection

### Configuration Format

The configuration file will support the following storage section:

```toml
[storage]
backend = "postgres"  # or "sqlite" (default)
postgres_url = "postgres://user:password@localhost:5432/brains"
connection_timeout = 5  # seconds
max_connections = 10
min_connections = 2

# Optional SQLite configuration
sqlite_path = "~/.brains/memories.db"
```

### Key Entities

- **StorageConfig**: Extended to include config file source and timeout settings
- **Configuration File**: TOML file containing storage section with connection details
- **Storage Backend**: Either PostgreSQL or SQLite, determined at runtime based on configuration and availability

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can configure PostgreSQL connection in under 1 minute by editing a config file
- **SC-002**: When PostgreSQL is unavailable, fallback to SQLite occurs within the configured timeout period (default 5 seconds)
- **SC-003**: Users can see which storage backend is active through the status display
- **SC-004**: Environment variable overrides work correctly 100% of the time
- **SC-005**: Existing users with environment variable configuration experience no changes to behavior

## Clarifications

### Session 2025-12-22

- Q: When PostgreSQL is configured but fallback to SQLite occurs, what should happen when PostgreSQL becomes available again? → A: SQLite remains authoritative for that session; user must restart to re-attempt PostgreSQL connection

## Assumptions

- The user understands that storing credentials in config files is acceptable for local development
- The existing TOML configuration system (from feature 007-cli-config) is available for extension
- The PostgreSQL connection string follows standard format: `postgres://user:password@host:port/database`
- SQLite remains the default backend when no configuration is provided
- Connection validation occurs once at startup, not on every operation
