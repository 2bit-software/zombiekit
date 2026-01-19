---
status: complete
updated: 2026-01-19
---

# Research: Smarter Claude Import Synchronization

## Executive Summary

The current Claude conversation importer (`internal/cli/recall.go:importClaudeHistory`) processes all conversation files on every import run, checking each message UUID against the database via `ExistsBySourceID()`. This O(n) approach becomes slow with large histories. The solution is to track import state per file (last entry ID + file mtime) to skip unchanged files entirely and scan backwards in changed files to find the sync point.

## Findings

### Codebase Context

**Current Import Flow** (`internal/cli/recall.go:371-488`):
1. `DiscoverHistoryFiles()` finds all `.jsonl` files in `~/.claude/projects/`
2. For each file: `claude.ParseFile()` loads all entries into memory
3. For each entry: `storage.ExistsBySourceID(ctx, "claude", entry.UUID)` checks if already imported
4. If not exists: generate embedding, save with source tracking
5. Progress: reports "imported" or "skipped" for each message

**Key Files**:
- `internal/cli/recall.go` - CLI command and import orchestration
- `internal/recall/claude/discovery.go` - File discovery functions
- `internal/recall/claude/parser.go` - JSONL parsing
- `internal/recall/postgres/storage.go` - Database operations

**Existing Duplicate Detection**:
- Uses `source_id` (UUID) for deduplication via `ExistsBySourceID()`
- Database has unique index: `idx_recall_chunks_source_id ON recall_chunks(source, source_id)`
- Content hash also tracked but not used for skip logic

**File Structure** (Claude history):
- Path: `~/.claude/projects/{encoded-project-path}/{session-id}.jsonl`
- JSONL format with `uuid`, `sessionId`, `timestamp` per entry
- Entries appended chronologically (newest at end)

**No Existing Import Tracking**:
- No table tracking last-imported state per file
- No file modification time caching
- Each import run is stateless (relies only on DB content)

### Domain Knowledge

**Efficient Sync Strategies**:
1. **Timestamp-based change detection**: Compare file mtime to cached value; skip if unchanged
2. **Cursor-based sync**: Track last-processed entry ID; resume from cursor position
3. **Backwards scanning**: For append-only files, scan from end to find last known entry

**Similar Implementations**:
- Git fetch: tracks refs to avoid re-downloading unchanged objects
- rsync: uses mtime + size for change detection
- Database replication: uses log sequence numbers (LSN) for sync points

**JSONL Characteristics**:
- Append-only format (entries added to end)
- Line-delimited (can read backwards efficiently)
- No in-place modifications (mutations create new entries)

### Risks Identified

1. **Ordering assumption**: If history.jsonl doesn't append chronologically, backwards scanning fails
2. **Large file memory**: Loading entire file for backwards scan could be problematic
3. **Marker corruption**: If tracking table becomes inconsistent, could cause missed/duplicate imports

## Decision Points

- [x] **D1**: Sync approach - **Decided**: Tracking table with file mtime + last entry ID
- [x] **D2**: Divergence handling - **Decided**: Auto-reconcile with warning + divergence marker
- [x] **D3**: Output verbosity - **Decided**: Reduce skip noise, show only new imports by default

## Recommendations

1. **New tracking table**: `recall_import_state` with columns: `file_path`, `last_entry_id`, `file_mtime`, `updated_at`
2. **Two-phase change detection**:
   - Phase 1: Compare file mtime - if unchanged, skip entirely (no parsing)
   - Phase 2: If changed, parse file backwards to find last known entry ID
3. **Divergence markers**: Create `recall_divergence_markers` record when sync point not found
4. **Force flag**: Add `--force` option to bypass tracking and do full re-import

## Sources

- Linear ticket DEV-87: https://linear.app/heinsight/issue/DEV-87
- Business spec comment by Morgan Hein (2026-01-19)
- Work summary comment by Morgan Hein (2026-01-19)
- Codebase: `internal/cli/recall.go`, `internal/recall/claude/*.go`
