# Implementation Plan: Smarter Import Synchronization

**Feature**: DEV-87
**Date**: 2026-01-19
**Status**: Ready for Implementation

## Overview

Transform the Claude conversation importer from O(n) per-message duplicate checking to O(1) per-file change detection, with efficient incremental imports for changed files.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Import Flow (New)                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  DiscoverHistoryFiles() ──┬──► file1.jsonl                     │
│                           ├──► file2.jsonl                     │
│                           └──► file3.jsonl                     │
│                                   │                             │
│                                   ▼                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Per-File Processing                         │   │
│  │                                                          │   │
│  │  1. GetImportState(file_path)                           │   │
│  │       ├── Not found → Full import (new file)            │   │
│  │       └── Found → Check mtime                           │   │
│  │                                                          │   │
│  │  2. Compare file mtime with stored mtime                │   │
│  │       ├── Unchanged → SKIP entirely (no parsing)        │   │
│  │       └── Changed → Continue to sync                    │   │
│  │                                                          │   │
│  │  3. ParseFileFromUUID(path, last_entry_uuid)            │   │
│  │       ├── UUID found → Return entries after sync point  │   │
│  │       └── UUID not found → Divergence! Mark gap         │   │
│  │                                                          │   │
│  │  4. Import new entries (existing flow)                  │   │
│  │                                                          │   │
│  │  5. UpdateImportState(file_path, last_uuid, mtime)      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Database Schema (Day 1)

**Files to modify:**
- `internal/database/migrations/postgres/005_recall_import_state.sql` (new)
- `internal/recall/postgres/storage.go` (extend)

**Tasks:**

1.1. Create migration for `recall_import_state` table:
```sql
CREATE TABLE IF NOT EXISTS recall_import_state (
    file_path TEXT PRIMARY KEY,
    last_entry_uuid TEXT NOT NULL,
    file_mtime BIGINT NOT NULL,  -- Unix timestamp (nanoseconds)
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

1.2. Add `history_gap` column to `recall_chunks`:
```sql
ALTER TABLE recall_chunks
    ADD COLUMN IF NOT EXISTS history_gap BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_recall_chunks_history_gap
    ON recall_chunks(history_gap) WHERE history_gap = TRUE;
```

1.3. Add storage methods to `postgres.Storage`:
```go
// Import state operations
func (s *Storage) GetImportState(ctx, filePath string) (*ImportState, error)
func (s *Storage) SaveImportState(ctx, state ImportState) error
func (s *Storage) DeleteImportState(ctx, filePath string) error
func (s *Storage) ListImportStates(ctx) ([]ImportState, error)

// Extend ChunkInput to support history_gap flag
type ChunkInput struct {
    // ... existing fields
    HistoryGap bool
}
```

**Tests:**
- Unit test: CRUD operations for import state
- Integration test: State persists across connections

---

### Phase 2: Parser Extension (Day 1)

**Files to modify:**
- `internal/recall/claude/parser.go` (extend)
- `internal/recall/claude/types.go` (extend if needed)

**Tasks:**

2.1. Add `ParseFileFromUUID` function:
```go
// ParseFileFromUUID parses a JSONL file and returns entries after the specified UUID.
// If uuid is empty, returns all importable entries (fresh import).
// If uuid is not found, returns ErrSyncPointNotFound.
func ParseFileFromUUID(path, lastKnownUUID string) (entries []HistoryEntry, lastUUID string, err error)
```

Implementation:
- Single forward pass through file
- Filter importable entries (same rules as FilterImportable)
- Track whether UUID was found
- Return entries after sync point
- Return last entry's UUID for state update

2.2. Add sentinel error:
```go
var ErrSyncPointNotFound = errors.New("sync point UUID not found in file")
```

**Tests:**
- Unit: ParseFileFromUUID with valid UUID returns entries after
- Unit: ParseFileFromUUID with empty UUID returns all entries
- Unit: ParseFileFromUUID with missing UUID returns error
- Unit: ParseFileFromUUID handles empty file
- Unit: ParseFileFromUUID handles file with no importable entries

---

### Phase 3: File Lock Mechanism (Day 2)

**Files to modify:**
- `internal/recall/claude/lock.go` (new)
- `internal/cli/recall.go` (use lock)

**Tasks:**

3.1. Create file lock wrapper:
```go
// lock.go
package claude

import (
    "os"
    "syscall"
)

type ImportLock struct {
    file *os.File
}

// AcquireLock attempts to acquire exclusive import lock.
// Returns error if lock already held by another process.
func AcquireLock(lockPath string) (*ImportLock, error)

// Release releases the import lock.
func (l *ImportLock) Release() error
```

Implementation:
- Lock file at `~/.claude/.zombiekit-import.lock`
- Use `syscall.Flock` with `LOCK_EX | LOCK_NB` (non-blocking exclusive)
- Return clear error message if lock already held

3.2. Integrate into import command:
- Acquire lock at start of import
- Release on completion, error, or signal interrupt
- Show error message if another import is running

**Tests:**
- Unit: Lock acquisition succeeds on first attempt
- Integration: Second process gets lock error
- Integration: Lock released on function return

---

### Phase 4: Import Logic Refactor (Day 2-3)

**Files to modify:**
- `internal/cli/recall.go` (major refactor)
- `internal/recall/storage.go` (extend interface)

**Tasks:**

4.1. Extend Storage interface:
```go
type Storage interface {
    // ... existing methods

    // Import state operations
    GetImportState(ctx, filePath string) (*ImportState, error)
    SaveImportState(ctx, state ImportState) error
    DeleteImportState(ctx, filePath string) error
    CleanupStaleImportStates(ctx, validPaths []string) error

    // Extended save with history_gap support
    SaveWithSourceAndGap(ctx, input ChunkInput, embedding []float32) (id string, created bool, err error)
}
```

4.2. Refactor `importClaudeHistory`:

```go
func importClaudeHistory(ctx, w, storage, embedder, claudePath, projectPath string, verbose, force bool) (newCount, skipCount int, err error) {
    // 1. Acquire import lock
    lock, err := claude.AcquireLock(filepath.Join(claudePath, ".zombiekit-import.lock"))
    if err != nil {
        return 0, 0, fmt.Errorf("another import in progress: %w", err)
    }
    defer lock.Release()

    // 2. Discover files
    files := claude.DiscoverHistoryFiles(claudePath)

    // 3. Clean up stale import states for deleted files
    if !force {
        storage.CleanupStaleImportStates(ctx, files)
    }

    // 4. Process each file
    for _, filePath := range files {
        new, skip, err := processFile(ctx, storage, embedder, filePath, verbose, force)
        // ... accumulate counts, handle errors
    }

    return newCount, skipCount, nil
}
```

4.3. Create `processFile` function:
```go
func processFile(ctx, storage, embedder, filePath string, verbose, force bool) (newCount, skipCount int, err error) {
    // Get file stat
    stat, err := os.Stat(filePath)
    if err != nil {
        return 0, 0, nil // File disappeared, skip
    }
    mtime := stat.ModTime().UnixNano()

    // Check import state (unless force)
    var lastUUID string
    var isNew bool
    if !force {
        state, err := storage.GetImportState(ctx, filePath)
        if err == nil && state.FileMtime == mtime {
            // Unchanged - skip entirely
            return 0, 0, nil
        }
        if state != nil {
            lastUUID = state.LastEntryUUID
        } else {
            isNew = true
        }
    }

    // Parse file from sync point
    entries, finalUUID, err := claude.ParseFileFromUUID(filePath, lastUUID)

    // Handle divergence
    historyGap := false
    if errors.Is(err, claude.ErrSyncPointNotFound) {
        historyGap = true
        entries, finalUUID, err = claude.ParseFileFromUUID(filePath, "") // Full import
        // Log warning about divergence
    }
    if err != nil {
        return 0, 0, err
    }

    // Import entries (existing logic, but pass historyGap flag)
    for _, entry := range entries {
        // ... existing embedding/save logic
        // Mark first chunk with historyGap if divergence detected
    }

    // Update import state
    newState := &ImportState{
        FilePath:      filePath,
        LastEntryUUID: finalUUID,
        FileMtime:     mtime,
    }
    storage.SaveImportState(ctx, newState)

    return newCount, skipCount, nil
}
```

4.4. Add `--force` flag to CLI:
- Bypass import state checking
- Do not clean up stale states
- Full re-import of all files

**Tests:**
- Integration: Fresh import creates import state
- Integration: Unchanged file skipped (mtime check)
- Integration: Changed file imports only new entries
- Integration: Missing sync point triggers divergence handling
- Integration: Force flag bypasses state check
- Integration: Stale states cleaned up for deleted files

---

### Phase 5: Output Verbosity (Day 3)

**Files to modify:**
- `internal/cli/recall.go` (output changes)

**Tasks:**

5.1. Change default output behavior:
- Default: Show only summary at end (X new, Y files unchanged)
- With `--verbose`: Show per-file activity
- Divergence warnings always shown

5.2. Output format:
```
# Default output (no --verbose)
Imported 15 new messages from 3 files (7 files unchanged)

# With --verbose
Processing: ~/.claude/projects/-Users-foo-bar/abc123.jsonl
  → 5 new messages
Processing: ~/.claude/projects/-Users-foo-bar/def456.jsonl
  → unchanged (skipped)
Processing: ~/.claude/projects/-Users-foo-bar/ghi789.jsonl
  ⚠ History divergence detected - importing all entries
  → 10 new messages (marked with history gap)

Imported 15 new messages from 3 files (7 files unchanged)

# Force import
Force import enabled - reprocessing all files
Processing: ~/.claude/projects/-Users-foo-bar/abc123.jsonl
  → 50 messages (5 new, 45 existing)
...
```

**Tests:**
- Integration: Default output is summary only
- Integration: Verbose flag shows per-file output
- Integration: Divergence warning appears regardless of verbosity

---

### Phase 6: Testing & Polish (Day 4)

**Files to modify:**
- `internal/recall/claude/import_test.go` (extend)
- `internal/recall/postgres/storage_test.go` (extend)

**Tasks:**

6.1. Add tests for new functionality:
- Import state CRUD operations
- mtime-based skip logic
- Sync point detection
- Divergence handling and gap marking
- Force flag behavior
- Concurrent import blocking
- Stale state cleanup

6.2. Update existing tests:
- Ensure they still pass with new architecture
- Add history_gap assertions where relevant

6.3. Add E2E test scenario:
```go
func TestE2E_IncrementalImport(t *testing.T) {
    // 1. Fresh import - all entries imported
    // 2. No changes - zero work done
    // 3. Append to file - only new entries imported
    // 4. Delete middle entry - divergence detected, gap marked
    // 5. Force import - all entries reprocessed
}
```

6.4. Performance validation:
- Measure import time with 500+ files
- Verify <2 second completion when unchanged
- Document performance baseline

---

## Dependency Graph

```
Phase 1 (DB Schema)
    │
    ├──► Phase 2 (Parser Extension) ──┐
    │                                  │
    └──► Phase 3 (File Lock) ─────────┼──► Phase 4 (Import Logic) ──► Phase 5 (Output) ──► Phase 6 (Testing)
```

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Migration fails on existing data | Migration is additive only (new table, new column with default) |
| Lock file left behind on crash | Use `defer` for release; lock file is advisory anyway |
| mtime comparison unreliable | Use nanosecond precision; force flag as escape hatch |
| Performance regression | Spike validated 200ms worst case per file; mtime skip is O(1) |

## Success Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| No-change import time | <2 seconds | Time `recall import` with no new content |
| Skip output eliminated | Zero "skipped" messages | Count output lines |
| Incremental correctness | 100% new entries imported | Compare DB count before/after |
| Force import works | All entries reprocessed | Verify with `--force --verbose` |

## Files Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/database/migrations/postgres/005_recall_import_state.sql` | New | Schema for import state table + history_gap column |
| `internal/recall/claude/parser.go` | Modify | Add `ParseFileFromUUID` function |
| `internal/recall/claude/lock.go` | New | File locking for concurrent import prevention |
| `internal/recall/postgres/storage.go` | Modify | Add import state CRUD + history_gap support |
| `internal/recall/storage.go` | Modify | Extend interface with new methods |
| `internal/cli/recall.go` | Modify | Refactor import logic, add --force flag |
| `internal/recall/claude/import_test.go` | Modify | Add tests for new functionality |
