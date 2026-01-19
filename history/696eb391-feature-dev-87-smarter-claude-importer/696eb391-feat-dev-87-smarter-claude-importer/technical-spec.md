# Technical Specification: Smarter Import Synchronization

**Feature**: DEV-87
**Date**: 2026-01-19
**Status**: Ready for Implementation

## Design Decisions

### D1: Sync Point Strategy

**Decision**: Track last imported entry UUID per file + file mtime for change detection.

**Alternatives Considered**:
1. ❌ Content hash of entire file - requires reading entire file to check
2. ❌ Line count tracking - doesn't handle modifications
3. ❌ Backwards byte scanning - complexity not justified (see spike-results.md)
4. ✅ UUID + mtime - O(1) change detection, simple forward scan for new entries

**Rationale**: mtime check is O(1) and filters out 90%+ of files. For changed files, forward single-pass is fast enough (<200ms for 48MB file).

---

### D2: Divergence Handling

**Decision**: Mark first imported chunk with `history_gap = true` when sync point not found.

**Behavior**:
1. Attempt to find last known UUID in file
2. If not found: log warning, set `history_gap = true` on first new chunk
3. Import all entries from file (full re-import)
4. Update import state to new sync point

**Why not fail/abort?**
- User may have legitimately deleted old conversations
- Better to preserve what we can + mark the gap
- Downstream consumers can filter/warn based on `history_gap`

---

### D3: File Locking Strategy

**Decision**: Single advisory lock file at `~/.claude/.zombiekit-import.lock`

**Implementation**:
```go
func AcquireLock(lockPath string) (*ImportLock, error) {
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
    if err != nil {
        return nil, err
    }
    err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err != nil {
        f.Close()
        return nil, fmt.Errorf("import already in progress")
    }
    return &ImportLock{file: f}, nil
}
```

**Why advisory lock?**
- Cross-platform (works on macOS and Linux)
- Automatically released on process crash
- Non-blocking check prevents hangs

**Why single lock (not per-file)?**
- Simpler reasoning about state
- Import state updates should be atomic per-run
- No real benefit to parallel file processing (I/O bound)

---

### D4: mtime Precision

**Decision**: Store mtime as Unix nanoseconds (int64)

**Rationale**:
- macOS HFS+ has 1-second resolution, but APFS has nanosecond
- Linux ext4/btrfs have nanosecond resolution
- Using nanoseconds is forward-compatible
- int64 nanoseconds fit timestamps until year 2262

**Code**:
```go
mtime := stat.ModTime().UnixNano() // int64 nanoseconds
```

---

### D5: State Cleanup Strategy

**Decision**: Clean up stale import states at start of each import run.

**Algorithm**:
```go
func CleanupStaleImportStates(ctx, validPaths []string) error {
    // Delete import states where file_path NOT IN validPaths
}
```

**When**:

---

### D6: State Update Atomicity (NFR-002)

**Decision**: Import state updates use database transactions to ensure atomicity.

**Implementation**:
```go
func (s *Storage) SaveImportState(ctx context.Context, state *ImportState) error {
    // UPSERT with single statement (atomic by nature)
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO recall_import_state (file_path, last_entry_uuid, file_mtime, updated_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (file_path) DO UPDATE
        SET last_entry_uuid = $2, file_mtime = $3, updated_at = NOW()
    `, state.FilePath, state.LastEntryUUID, state.FileMtime)
    return err
}
```

**Why UPSERT?**
- Single statement is inherently atomic
- No need for explicit transaction wrapping
- Handles both create and update cases

**Failure Scenario**:
- If import succeeds but state update fails, next import will re-import those entries
- This is safe because `ON CONFLICT (source, source_id) DO NOTHING` handles duplicates
- Worst case: redundant work, not data loss or corruption

**When**:
- At start of import, after file discovery
- NOT during force import (preserve state for later)

**Why at import time?**
- Natural place to sync state with filesystem
- Doesn't require separate cleanup command
- Bounded work (one query per import run)

---

## Data Structures

### ImportState

```go
// internal/recall/types.go
type ImportState struct {
    FilePath      string    // Absolute path to JSONL file
    LastEntryUUID string    // UUID of last successfully imported entry
    FileMtime     int64     // Unix nanoseconds of file at last import
    UpdatedAt     time.Time // When this state was last updated
}
```

### Extended ChunkInput

```go
// internal/recall/types.go
type ChunkInput struct {
    Content        string
    Source         string
    SourceID       string
    ConversationID string
    Metadata       *Metadata
    HistoryGap     bool      // NEW: true if this chunk follows a sync gap
}
```

---

## Database Schema

### New Table: recall_import_state

```sql
-- internal/database/migrations/postgres/005_recall_import_state.sql

CREATE TABLE IF NOT EXISTS recall_import_state (
    file_path TEXT PRIMARY KEY,
    last_entry_uuid TEXT NOT NULL,
    file_mtime BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE recall_import_state IS 'Tracks per-file import sync state for incremental imports';
COMMENT ON COLUMN recall_import_state.file_path IS 'Absolute path to the JSONL file';
COMMENT ON COLUMN recall_import_state.last_entry_uuid IS 'UUID of last successfully imported entry';
COMMENT ON COLUMN recall_import_state.file_mtime IS 'Unix nanosecond timestamp of file modification time at last import';
```

### Schema Change: recall_chunks

```sql
-- Same migration file

ALTER TABLE recall_chunks
    ADD COLUMN IF NOT EXISTS history_gap BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN recall_chunks.history_gap IS 'True if this chunk was imported after detecting sync divergence';

CREATE INDEX IF NOT EXISTS idx_recall_chunks_history_gap
    ON recall_chunks(history_gap) WHERE history_gap = TRUE;
```

---

## API Contracts

### Parser Extension

```go
// internal/recall/claude/parser.go

// ErrSyncPointNotFound is returned when the specified UUID is not found in the file.
var ErrSyncPointNotFound = errors.New("sync point UUID not found in file")

// ParseFileFromUUID parses a JSONL file and returns importable entries after the specified UUID.
//
// If lastKnownUUID is empty, returns all importable entries (fresh import scenario).
// If lastKnownUUID is found, returns entries that come after it chronologically.
// If lastKnownUUID is not found, returns ErrSyncPointNotFound.
//
// Returns:
//   - entries: importable entries (filtered by type, non-meta, non-nil message)
//   - lastUUID: UUID of the last entry in the file (for state update)
//   - err: parsing error or ErrSyncPointNotFound
func ParseFileFromUUID(path, lastKnownUUID string) (entries []HistoryEntry, lastUUID string, err error)
```

### Storage Extension

```go
// internal/recall/storage.go

type Storage interface {
    // ... existing methods ...

    // GetImportState retrieves the import state for a file.
    // Returns nil, nil if no state exists (new file).
    GetImportState(ctx context.Context, filePath string) (*ImportState, error)

    // SaveImportState creates or updates the import state for a file.
    SaveImportState(ctx context.Context, state *ImportState) error

    // DeleteImportState removes the import state for a file.
    DeleteImportState(ctx context.Context, filePath string) error

    // CleanupStaleImportStates removes import states for files not in validPaths.
    CleanupStaleImportStates(ctx context.Context, validPaths []string) error
}
```

### Lock Interface

```go
// internal/recall/claude/lock.go

// ImportLock represents an exclusive lock for import operations.
type ImportLock struct {
    file *os.File
}

// AcquireLock attempts to acquire an exclusive import lock.
// Returns an error if another process holds the lock.
func AcquireLock(lockPath string) (*ImportLock, error)

// Release releases the import lock. Safe to call multiple times.
func (l *ImportLock) Release() error
```

---

## Error Handling

### Recoverable Errors

| Error | Recovery |
|-------|----------|
| File not found during processing | Log warning, skip file, continue |
| JSONL parse error on single line | Skip line, continue parsing |
| Import state not found | Treat as new file, full import |
| Sync point not found | Set history_gap, import all |

### Fatal Errors

| Error | Behavior |
|-------|----------|
| Cannot acquire lock | Return error, show message |
| Database connection failed | Return error |
| Cannot open file for parsing | Return error (likely permissions) |

---

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| mtime check | O(1) | Single stat() call |
| File skip (unchanged) | O(1) | No parsing |
| New entries import | O(n) | n = new entries only |
| Sync point search | O(m) | m = entries in file |

### Space Complexity

| Operation | Memory | Notes |
|-----------|--------|-------|
| File parsing | O(1) | Streaming with bufio.Scanner |
| Entry collection | O(n) | n = new entries to import |
| Import state | O(f) | f = number of files |

### Benchmarks (from spike)

| File Size | Parse Time | Notes |
|-----------|------------|-------|
| 14MB | ~60ms | 551 entries |
| 48MB | ~200ms | 88 entries (large messages) |

---

## Testing Strategy

### Unit Tests

| Test | Location | Purpose |
|------|----------|---------|
| `TestParseFileFromUUID_EmptyUUID` | parser_test.go | Fresh import returns all |
| `TestParseFileFromUUID_ValidUUID` | parser_test.go | Returns entries after sync |
| `TestParseFileFromUUID_MissingUUID` | parser_test.go | Returns ErrSyncPointNotFound |
| `TestAcquireLock_Success` | lock_test.go | Lock acquired on first try |
| `TestAcquireLock_AlreadyHeld` | lock_test.go | Error when lock held |

### Integration Tests

| Test | Location | Purpose |
|------|----------|---------|
| `TestImport_StateCreated` | import_test.go | State saved after import |
| `TestImport_UnchangedSkipped` | import_test.go | mtime check works |
| `TestImport_IncrementalImport` | import_test.go | Only new entries imported |
| `TestImport_DivergenceHandled` | import_test.go | Gap marked on sync fail |
| `TestImport_ForceBypassesState` | import_test.go | --force ignores state |
| `TestImport_ConcurrentBlocked` | import_test.go | Lock prevents concurrent |
| `TestImport_StaleStateCleanup` | import_test.go | Deleted files cleaned |

### E2E Tests

| Test | Location | Purpose |
|------|----------|---------|
| `TestE2E_IncrementalWorkflow` | e2e_test.go | Full workflow validation |

---

## Migration Notes

### Backwards Compatibility

- Existing imports will work unchanged
- First import after upgrade creates import states
- No data migration required (state built on demand)
- `history_gap` defaults to false for existing chunks

### Rollback Plan

1. Revert migration: `DROP TABLE recall_import_state; ALTER TABLE recall_chunks DROP COLUMN history_gap;`
2. Revert code changes
3. Import will work as before (slower, but functional)

---

## Open Questions

None - all questions resolved through research and spike validation.

---

## References

- Business Spec: `spec.md`
- Research: `research.md`
- Spike Results: `spike-results.md`
- Audit: `audit/2026-01-19.md`
