# Feature Specification: Smarter Import Synchronization

**Feature Branch**: `696eb391-feature-dev-87-smarter-claude-importer`
**Created**: 2026-01-19
**Status**: Draft
**Linear**: DEV-87

## Problem Statement

The conversation importer currently scans all conversations from the beginning on every startup, checking each one against the existing database to determine if it should be skipped. For users with large conversation histories, this means hundreds or thousands of redundant "already imported" checks before reaching new content. Users experience unnecessarily long import times and see excessive skip messages for content that hasn't changed.

## Codebase Context

**Current Implementation**:
- `internal/cli/recall.go` - CLI command and import orchestration (`recallWatchClaudeAction`, `importClaudeHistory`)
- `internal/recall/claude/discovery.go` - File discovery (`DiscoverHistoryFiles`, `DiscoverProjectFiles`)
- `internal/recall/claude/parser.go` - JSONL parsing (`ParseFile`, `FilterImportable`)
- `internal/recall/claude/types.go` - Entry type definitions
- `internal/recall/claude/chunker.go` - Message chunking for large content
- `internal/recall/postgres/storage.go` - Database operations (`SaveWithSource`, `ExistsBySourceID`)

**File Locations**:
- Claude history: `~/.claude/projects/{encoded-project-path}/{session-id}.jsonl`
- Each project directory contains one or more `.jsonl` files (one per session)
- Files are discovered by walking `~/.claude/projects/` recursively

**Database**:
- Table: `recall_chunks` - stores imported conversation chunks
- Existing columns: `id`, `content`, `content_hash`, `embedding`, `source`, `source_id`, `conversation_id`, `metadata`, `created_at`

## Data Structures

### JSONL Entry Format

Each line in a Claude history file is a JSON object:

```json
{
  "type": "user|assistant",
  "uuid": "msg-abc123",           // Stable unique identifier for this entry
  "sessionId": "session-xyz",     // Groups entries into conversations
  "timestamp": "2024-01-15T10:00:00Z",
  "parentUuid": "msg-parent",     // Optional, links to previous message
  "message": {
    "role": "user|assistant",
    "content": "string or content blocks array"
  },
  "isMeta": false,
  "isSidechain": false,
  "cwd": "/path/to/project",      // Optional
  "gitBranch": "feature/x"        // Optional
}
```

The `uuid` field is the stable identifier used for sync tracking. Entries are appended chronologically (newest at end of file).

## User Scenarios & Testing

### User Story 1 - Efficient Incremental Import (Priority: P1)

As a ZombieKit user with an established conversation history, I need imports that quickly skip past already-imported conversations so that imports complete in seconds rather than minutes, and I only see activity for genuinely new content.

**Why this priority**: This is the core value proposition. Users with 500+ conversations shouldn't wait minutes for imports that have nothing new.

**Independent Test**: Can be tested with a populated database and a small number of new conversations. Measure time and output.

**Acceptance Scenarios**:

1. **Given** a user has imported 500 conversations previously, **When** they run a new import with 10 new conversations, **Then** the system processes only the 10 new items without visibly iterating through all 500 existing ones
2. **Given** the import completes, **When** the user reviews the output, **Then** they see activity only for newly imported conversations (not skip messages for everything)
3. **Given** a conversation file has not been modified since last import, **When** the import runs, **Then** that file is skipped entirely without parsing

---

### User Story 2 - Fresh Import Still Works (Priority: P1)

As a new ZombieKit user, I need to import my full conversation history on first run so that all my historical conversations become searchable.

**Why this priority**: First-run experience must work correctly. Equal priority to incremental because both are essential.

**Independent Test**: Can be tested with empty database and existing Claude history.

**Acceptance Scenarios**:

1. **Given** a user has never imported before, **When** they run an import, **Then** all conversations are processed and imported
2. **Given** a full import is running, **When** the user watches progress, **Then** they see accurate progress for the full corpus

---

### User Story 3 - History Divergence Recovery (Priority: P2)

As a user whose local history has diverged from imported state (e.g., deleted conversations, restored from backup), I need the system to detect and reconcile differences so that I maintain a consistent, complete searchable history.

**Why this priority**: Edge case that affects data integrity. Important but less common than normal incremental imports.

**Independent Test**: Can be tested by modifying history files to remove middle entries.

**Acceptance Scenarios**:

1. **Given** the user has deleted conversations from the source but they exist in the database, **When** import runs, **Then** the system handles the divergence gracefully (does not crash or corrupt data)
2. **Given** the sync point cannot be found in a changed file, **When** import runs, **Then** a divergence warning is shown and the first imported chunk is marked with `history_gap = true`
3. **Given** chunks with `history_gap = true` exist, **When** querying via MCP/tool/interface, **Then** clients can detect and surface this to users as appropriate

---

### User Story 4 - Force Full Re-import (Priority: P3)

As a user who suspects import state corruption, I need to force a full re-import so that I can recover from inconsistent states.

**Why this priority**: Recovery mechanism. Rare but necessary.

**Independent Test**: Can be tested by running import with `--force` flag on already-imported content.

**Acceptance Scenarios**:

1. **Given** the user runs import with `--force` flag, **When** import completes, **Then** all conversations are re-checked regardless of tracking state
2. **Given** a force import is running, **When** the user watches progress, **Then** they see all conversations being processed

---

### Edge Cases

- What happens when a file is deleted between discovery and processing? → Skip with warning, don't fail
- What happens when the tracking table has an entry for a non-existent file? → Clean up stale entry
- What happens when a file grows beyond memory limits? → Implementation detail (streaming or chunked reading)
- How does the system handle concurrent import processes? → System-level file lock using Go stdlib prevents concurrent imports

## Requirements

### Functional Requirements

- **FR-001**: System MUST track import state per conversation file (file path, last entry UUID, file mtime)
- **FR-002**: System MUST skip unchanged files entirely based on mtime comparison (no parsing required)
- **FR-003**: System MUST locate the sync point in changed files by finding the last known entry UUID
- **FR-004**: System MUST import only entries after the sync point in changed files
- **FR-005**: System MUST mark imported chunks with `history_gap = true` when sync point cannot be found (divergence detected)
- **FR-006**: System MUST output a warning when auto-reconciliation occurs due to divergence
- **FR-007**: System MUST support a `--force` flag to bypass tracking and do full re-import
- **FR-008**: System MUST reduce output verbosity (show only new imports, not skipped items by default)
- **FR-009**: System MUST clean up stale tracking entries for deleted files
- **FR-010**: System MUST acquire an exclusive file lock before importing to prevent concurrent imports (using Go stdlib `syscall.Flock` or equivalent)

### Key Entities

- **Import State** (new table: `recall_import_state`): Tracks per-file sync position
  - `file_path` (TEXT, PRIMARY KEY) - absolute path to the JSONL file
  - `last_entry_uuid` (TEXT) - UUID of the last successfully imported entry
  - `file_mtime` (BIGINT) - Unix timestamp of file modification time at last import
  - `updated_at` (TIMESTAMPTZ) - when this tracking record was last updated

- **History Gap Flag** (new column on `recall_chunks`):
  - `history_gap` (BOOLEAN, DEFAULT false) - when true, indicates this chunk was imported after detecting divergence; history before this point may be missing or inconsistent

### Non-Functional Requirements

- **NFR-001**: File lock MUST be released on import completion, error, or process termination
- **NFR-002**: Tracking state updates MUST be atomic (no partial state on failure)
- **NFR-003**: Memory usage during file processing SHOULD remain bounded (implementation may use streaming for large files)

## Success Criteria

### Measurable Outcomes

- **SC-001**: Import with no changes since last import completes in <2 seconds (vs current ~30+ seconds for large histories)
- **SC-002**: Import with N new entries processes only those N entries, completing in time proportional to N rather than total history size
- **SC-003**: Users see zero "skipped" messages for unchanged files
- **SC-004**: Force re-import processes all entries correctly
- **SC-005**: Concurrent import attempts are blocked with clear error message

## Testing Requirements

### Test Strategy

Integration tests are primary, using test containers for PostgreSQL (see existing patterns in `internal/recall/claude/import_test.go`). Unit tests for pure functions (mtime comparison, UUID extraction). E2E tests for CLI command behavior including lock acquisition.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify tracking table is created and populated after import |
| FR-002 | Integration | Verify unchanged files are skipped without parsing |
| FR-003 | Unit | Verify sync point location finds known UUID at various positions |
| FR-004 | Integration | Verify only new entries are imported after sync point |
| FR-005 | Integration | Verify `history_gap` flag set when UUID not found |
| FR-006 | Integration | Verify warning output when auto-reconciliation occurs |
| FR-007 | Integration | Verify `--force` bypasses tracking |
| FR-008 | Integration | Verify output shows only new imports by default |
| FR-009 | Integration | Verify stale entries cleaned up |
| FR-010 | Integration | Verify concurrent imports are blocked |

### Edge Case Coverage

- File deleted between discovery and processing → Integration test
- Sync point UUID not found in file → Integration test verifying `history_gap` flag
- Concurrent import attempt → Integration test with goroutines
- Very large file processing → Unit test with large mock data

## Assumptions

- The conversation history file (history.jsonl) has stable `uuid` identifiers that persist across sessions
- Entries in history files are appended chronologically (newest at end)
- Most import runs are incremental (small number of new conversations relative to total history)
- Single import process per system at a time (enforced by file lock)

## Out of Scope

- Real-time streaming of new conversations (still batch-based)
- Bi-directional sync (this is import-only, not export)
- Distributed locking across multiple machines (file lock is local only)
- UI/UX for displaying `history_gap` markers (deferred to interface layer)
