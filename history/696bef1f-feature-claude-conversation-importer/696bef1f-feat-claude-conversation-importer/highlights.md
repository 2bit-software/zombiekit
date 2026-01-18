# Highlights: Key Decisions

**Feature**: Claude Conversation Importer
**Linear Ticket**: DEV-69

---

## Decisions Approved

| # | Decision | Choice |
|---|----------|--------|
| 1 | Chunking | Whole message with 8000 char max split |
| 2 | Schema | Add source, source_id, conversation_id, metadata columns |
| 3 | Watch mode | Polling-based (simple) |
| 4 | Progress tracking | Track source IDs for duplicate detection |
| 5 | CLI structure | `brains recall watch claude` with flags |

---

## Decision 6: File Locking (Research Result)

**Research Findings**:

Based on web research and local investigation:

- Claude Code uses `.update.lock` in `~/.claude/` for version/update coordination
- The lock file contains a PID (e.g., "31397")
- History JSONL files are opened in **read mode** (`r`) by Claude (verified via `lsof`)
- **No specific lock file exists for individual history.jsonl files**
- [GitHub issue #13287](https://github.com/anthropics/claude-code/issues/13287) documents multi-instance locking problems but these relate to version directories, not history files

**Decision**: No lock file needed for our use case

**Rationale**:
1. We only **read** history files (never write)
2. Claude opens them in read mode too - concurrent reads are safe
3. JSONL append-only format means partial lines only occur at file end
4. We'll implement graceful handling:
   - Retry with backoff if file temporarily unavailable
   - Skip/ignore partial JSON lines at end of file
   - Handle file rotation gracefully

---

## Original Decision Details

### Decision 1: Chunking Strategy

**Approved**: Whole message with 8000 char max split

Each user or assistant message becomes one chunk. If a message exceeds 8000 characters, split it at sentence boundaries. This maintains message coherence while avoiding embedding dimension issues.

### Decision 2: Database Schema Extension

**Approved**: Add tracking columns to `recall_chunks`

New columns:
- `source` TEXT (e.g., "claude", "slack", "notion")
- `source_id` TEXT (e.g., message UUID from Claude)
- `conversation_id` TEXT (e.g., session ID from Claude)
- `metadata` JSONB (source-specific data like timestamp, role, git branch)

This enables:
- Proper conversation reconstruction
- Source tracking for multi-source imports
- Duplicate detection by source_id

### Decision 3: Watch Mode Implementation

**Approved**: Polling-based

Simple polling mechanism checking for new files/content periodically. Default interval: 30 seconds. Can upgrade to fsnotify later if performance becomes an issue.

### Decision 4: Import Progress Tracking

**Approved**: Track source IDs

Store Claude message UUIDs in the `source_id` column. Before importing, check if source_id already exists. This enables:
- Definitive duplicate detection
- Progress reporting ("X new messages imported")
- Incremental imports

### Decision 5: CLI Subcommand Structure

**Approved**: `brains recall watch claude`

Flags:
- `--once`: Import once and exit (no continuous watch)
- `--path <dir>`: Override ~/.claude location
- `--project <path>`: Filter to specific project only
- `--verbose`: Show detailed import progress

### Decision 7: Sidechain Message Handling

**Approved**: Import sidechain messages

Claude Code marks alternate/branched responses with `isSidechain: true`. These represent valid conversation branches the user explored and may contain valuable information.

**Decision**: Import sidechain messages alongside main conversation flow.

**Rationale**:
- Sidechains contain real user interactions worth searching
- The `parentUuid` field preserves the branching structure
- Excluding would lose potentially valuable context
- Storage cost is minimal compared to information value

### Decision 8: Content Block Type Handling

**Approved**: Extract text and thinking blocks only

| Block Type | Handling |
|------------|----------|
| text | Extract → include in corpus |
| thinking | Extract → include (valuable for search) |
| tool_use | Skip (function calls, not prose) |
| tool_result | Skip (structured output) |
| image | Skip (binary, not searchable) |

---

## Audit Results

### Completeness Check

| BR | Requirement | Covered | Notes |
|----|-------------|---------|-------|
| BR-001 | Import Claude history | ✓ | Phase 5 |
| BR-002 | No duplicates | ✓ | Unique index on (source, source_id) |
| BR-003 | Manual trigger | ✓ | `--once` flag |
| BR-004 | Progress feedback | ✓ | Verbose output mode |
| BR-005 | Watch mode | ✓ | Polling loop |
| BR-006 | Preserve context | ✓ | Metadata struct |
| BR-007 | Import history | ⚠️ | **Deferred** - implicit via chunk timestamps |
| BR-008 | Unique conversation ID | ✓ | conversation_id column |
| BR-009 | Retrieve conversation | ✓ | GetByConversation method |
| BR-010 | Navigate to conversation | ✓ | conversation command |

### BR-007 Deferral Rationale

The business spec requires viewing import history, but explicit tracking adds schema complexity for limited value in MVP. Users can infer import timing from chunk `created_at` timestamps. A dedicated `import_runs` table can be added in a future iteration if explicit tracking is needed.

### All Scenarios Covered

- S1: Import history → Phase 5
- S2: Re-import without duplicates → Phase 1, 2
- S3: View full conversation → Phase 6
- S4: On-demand import → `--once` flag
- S5: Watch mode → Phase 5.3

---

## Next Step

```
/brains.tasks
```
