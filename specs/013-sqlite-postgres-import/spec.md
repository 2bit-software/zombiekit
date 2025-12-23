# Feature Specification: SQLite to PostgreSQL Migration Tool

**Feature Branch**: `013-sqlite-postgres-import`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Let's make an import tool to migrate from sqlite to postgres. It should be a one-way migration only. however, it should be allowed to happen more than once, and only the additions from sqlite since the last import should be imported."

## Clarifications

### Session 2025-12-22

- Q: What happens if a memory with the same name but different content exists in both databases (conflict)? → A: Compare versions; if SQLite has a higher version, import it and mark the old PostgreSQL version as deleted.
- Q: How does the tool handle concurrent access to SQLite during import? → A: Acquire exclusive lock on SQLite during import (blocks other processes).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First-time Migration (Priority: P1)

A user has been running ZombieKit with SQLite storage (the default for development/single-user scenarios) and wants to migrate their memory data to PostgreSQL for production use. They run the import tool for the first time, and all their existing memories are transferred to PostgreSQL.

**Why this priority**: This is the core use case - the initial migration that enables users to move from development to production. Without this, the feature has no value.

**Independent Test**: Can be fully tested by running the import command against a SQLite database with sample memories and verifying all data appears in PostgreSQL with matching content.

**Acceptance Scenarios**:

1. **Given** a SQLite database with 10 memory items and an empty PostgreSQL database, **When** the user runs the import command, **Then** all 10 memory items are created in PostgreSQL with their content, versions, and timestamps preserved.
2. **Given** a SQLite database with memories containing special characters and Unicode, **When** the user runs the import command, **Then** all content is preserved exactly without corruption.
3. **Given** an empty SQLite database and a PostgreSQL database, **When** the user runs the import command, **Then** the command completes successfully with a message indicating zero items imported.

---

### User Story 2 - Incremental Migration (Priority: P1)

A user has already performed an initial migration and continues to use SQLite for local development while PostgreSQL is the production database. When they add new memories to SQLite, they want to run the import tool again to sync only the new additions to PostgreSQL without duplicating existing data.

**Why this priority**: This is equally critical as the first migration - the user explicitly requested that the tool can "happen more than once" and import only additions. This enables a continuous development workflow.

**Independent Test**: Can be tested by running the initial import, adding new memories to SQLite, running import again, and verifying only new items were added to PostgreSQL.

**Acceptance Scenarios**:

1. **Given** a previous import was completed at timestamp T1, and 5 new memories were added to SQLite after T1, **When** the user runs the import command again, **Then** only the 5 new memories are imported to PostgreSQL.
2. **Given** a memory that was imported previously, and a new version of that same memory exists in SQLite, **When** the user runs the import command, **Then** the new version is imported as an additional version in PostgreSQL.
3. **Given** no new memories have been added to SQLite since the last import, **When** the user runs the import command, **Then** the command completes successfully with a message indicating zero items imported.

---

### User Story 3 - Import Status Visibility (Priority: P2)

A user wants to know what will be imported before running the migration, and wants to see progress and results during/after the import process.

**Why this priority**: While not core functionality, visibility into what's happening builds user confidence and aids troubleshooting.

**Independent Test**: Can be tested by running the import with verbose/dry-run options and verifying appropriate status information is displayed.

**Acceptance Scenarios**:

1. **Given** a SQLite database with pending items to import, **When** the user runs the import command with a preview option, **Then** they see a list of items that would be imported without actually importing them.
2. **Given** an import in progress, **When** items are being imported, **Then** the user sees progress information (e.g., "Imported 5 of 10 items").
3. **Given** an import completes, **When** the import finishes, **Then** the user sees a summary including total items imported, any errors encountered, and time taken.

---

### User Story 4 - Error Recovery (Priority: P3)

When an import fails partway through (due to network issues, database errors, etc.), the user can identify what failed and resume or retry the import without data corruption.

**Why this priority**: Error handling is important for production use but is an enhancement over the core migration functionality.

**Independent Test**: Can be tested by simulating failures during import and verifying the system handles them gracefully.

**Acceptance Scenarios**:

1. **Given** an import fails after importing 50 of 100 items, **When** the user runs the import command again, **Then** only the remaining 50 items are imported (no duplicates from already-imported items).
2. **Given** a memory item fails to import, **When** the import continues, **Then** other items are still imported and the failed item is reported to the user.
3. **Given** the PostgreSQL database is unavailable, **When** the user runs the import command, **Then** a clear error message is displayed and no data is lost from SQLite.

---

### Edge Cases

- What happens when SQLite contains memories with names that exceed PostgreSQL column limits? (Both use TEXT type - no practical limit difference)
- Soft-deleted memories in SQLite are imported with deleted status preserved (see FR-010)
- Timezone differences handled by normalizing to UTC (see FR-011)
- Concurrent access: Acquire exclusive lock on SQLite during import to ensure consistency (see FR-013)
- Version conflicts resolved by importing higher versions and soft-deleting old PostgreSQL versions (see FR-012)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST import memory items from SQLite to PostgreSQL preserving name, content, version, deleted status, and timestamps.
- **FR-002**: System MUST track the last successful import timestamp to enable incremental imports.
- **FR-003**: System MUST import only memories created or updated in SQLite after the last import timestamp.
- **FR-004**: System MUST NOT modify or delete any data in the source SQLite database.
- **FR-005**: System MUST handle connection failures gracefully without corrupting data in either database.
- **FR-006**: System MUST provide a preview/dry-run mode to show what would be imported without making changes.
- **FR-007**: System MUST report progress during import operations.
- **FR-008**: System MUST provide a summary upon completion showing items imported, skipped, and any errors.
- **FR-009**: System MUST skip importing items that already exist in PostgreSQL with matching name and version.
- **FR-012**: System MUST compare version numbers when the same memory name exists in both databases; if SQLite has a higher version, import it and soft-delete the corresponding PostgreSQL version.
- **FR-010**: System MUST import soft-deleted memories (preserving their deleted status) to maintain data consistency.
- **FR-011**: System MUST handle timezone differences by normalizing timestamps to UTC during import.
- **FR-013**: System MUST acquire an exclusive lock on the SQLite database during import to prevent concurrent modifications and ensure data consistency.

### Key Entities

- **ImportMetadata**: Tracks import history including last import timestamp, source database path, and items imported count. Stored in the target PostgreSQL database.
- **MemoryItem**: Existing entity representing a memory record (name, version, content, deleted, created_at, updated_at).
- **ImportResult**: Result of an import operation including success/failure status, items imported, items skipped, errors encountered, and duration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete initial migration of 1000 memory items in under 30 seconds on a standard connection.
- **SC-002**: Incremental imports identify and transfer only new items, completing in time proportional to new items (not total database size).
- **SC-003**: Users can preview pending imports before execution with 100% accuracy.
- **SC-004**: Zero data loss or corruption occurs during any import operation, including failed imports.
- **SC-005**: Users receive clear, actionable error messages when imports fail.
- **SC-006**: Import tracking persists across tool restarts, allowing reliable incremental imports over time.

## Assumptions

- Users have valid connection credentials for both SQLite and PostgreSQL databases.
- The PostgreSQL database has the memories table already initialized (via the existing PostgresStorage.initSchema).
- Network connectivity to PostgreSQL is reasonably stable (timeouts and retries are acceptable, but persistent unavailability is an error).
- Both databases use compatible schema versions (as defined in the existing sqlite/storage.go and postgres/storage.go implementations).
- The import is run from an environment that can access both database files/connections.
