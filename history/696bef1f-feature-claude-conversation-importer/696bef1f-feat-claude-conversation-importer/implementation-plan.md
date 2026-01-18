# Implementation Plan: Claude Conversation Importer

**Feature**: conversation-importer
**Linear Ticket**: DEV-69
**Depends On**: DEV-72 (RAG Core Infrastructure) - COMPLETED

---

## Overview

Implement a CLI command `brains recall watch claude` that imports Claude Code conversation history into the recall system for semantic search.

---

## Phase 1: Schema Migration

### Step 1.1: Create schema migration

Create migration `003_recall_chunks_source_tracking.sql` to add tracking columns:

```sql
ALTER TABLE recall_chunks
  ADD COLUMN IF NOT EXISTS source TEXT,
  ADD COLUMN IF NOT EXISTS source_id TEXT,
  ADD COLUMN IF NOT EXISTS conversation_id TEXT,
  ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Index for duplicate detection by source_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_source_id
  ON recall_chunks(source, source_id)
  WHERE source_id IS NOT NULL;

-- Index for conversation retrieval
CREATE INDEX IF NOT EXISTS idx_recall_chunks_conversation
  ON recall_chunks(conversation_id)
  WHERE conversation_id IS NOT NULL;
```

**Traces to**: Decision 2 (Schema Extension), BR-002, BR-008, BR-009

---

## Phase 2: Storage Interface Extension

### Step 2.1: Extend Chunk type

Update `internal/recall/types.go`:

```go
type Chunk struct {
    ID             string    `json:"id"`
    Content        string    `json:"content"`
    CreatedAt      time.Time `json:"created_at"`
    Source         string    `json:"source,omitempty"`          // NEW
    SourceID       string    `json:"source_id,omitempty"`       // NEW
    ConversationID string    `json:"conversation_id,omitempty"` // NEW
    Metadata       *Metadata `json:"metadata,omitempty"`        // NEW (pointer, nil when no metadata)
}

type Metadata struct {
    Role      string    `json:"role,omitempty"`       // "user" or "assistant"
    Timestamp time.Time `json:"timestamp,omitempty"`  // original timestamp
    GitBranch string    `json:"git_branch,omitempty"` // from Claude history
    CWD       string    `json:"cwd,omitempty"`        // working directory
}

// ChunkInput is used when saving new chunks with source tracking.
// Location: internal/recall/types.go (alongside Chunk)
type ChunkInput struct {
    Content        string
    Source         string
    SourceID       string
    ConversationID string
    Metadata       *Metadata
}
```

**Traces to**: Decision 2, BR-006, BR-008

### Step 2.2: Extend Storage interface

Add methods to `internal/recall/storage.go`:

```go
type Storage interface {
    // Existing methods...
    Save(ctx, content, embedding) (id, created, error)
    List(ctx, limit) ([]Chunk, error)
    Search(ctx, embedding, limit) ([]SearchResult, error)
    Close() error

    // NEW: Save with source tracking
    SaveWithSource(ctx context.Context, chunk ChunkInput, embedding []float32) (id string, created bool, err error)

    // NEW: Check if source_id already exists (for duplicate detection)
    ExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error)

    // NEW: Get all messages in a conversation
    GetByConversation(ctx context.Context, conversationID string) ([]Chunk, error)
}

```

**Traces to**: BR-002, BR-009, BR-010

**Note**: `ChunkInput` is defined in Step 2.1 above.

### Step 2.3: Implement in postgres storage

Update `internal/recall/postgres/storage.go` with new methods.

**Traces to**: Step 2.2

---

## Phase 3: Claude History Parser

### Step 3.1: Create history parser package

Create `internal/recall/claude/` package:

```
internal/recall/claude/
  parser.go     # JSONL parsing logic
  types.go      # Claude history types
  discovery.go  # Find history files
```

### Step 3.2: Define Claude history types

In `internal/recall/claude/types.go`:

```go
type HistoryEntry struct {
    Type        string          `json:"type"`
    UUID        string          `json:"uuid"`
    ParentUUID  *string         `json:"parentUuid"`
    SessionID   string          `json:"sessionId"`
    Timestamp   time.Time       `json:"timestamp"`
    Message     *MessageContent `json:"message,omitempty"`
    IsMeta      bool            `json:"isMeta"`
    IsSidechain bool            `json:"isSidechain"`
    CWD         string          `json:"cwd,omitempty"`
    GitBranch   string          `json:"gitBranch,omitempty"`
    Version     string          `json:"version,omitempty"`
}

type MessageContent struct {
    Role    string      `json:"role"`
    Content interface{} `json:"content"` // string or []ContentBlock
}

type ContentBlock struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}
```

**Traces to**: Research Summary (History File Format)

### Step 3.3: Implement parser

In `internal/recall/claude/parser.go`:

- `ParseFile(path string) ([]HistoryEntry, error)` - Parse single JSONL file
- `FilterImportable(entries []HistoryEntry) []HistoryEntry` - Filter to user/assistant messages, skip isMeta
- `ExtractContent(entry HistoryEntry) string` - Handle both string and []ContentBlock content

**Traces to**: Research Summary (Filtering Strategy)

### Step 3.4: Implement discovery

In `internal/recall/claude/discovery.go`:

- `DiscoverHistoryFiles(claudePath string) ([]string, error)` - Find all .jsonl files
- `DiscoverProjectFiles(claudePath, projectPath string) ([]string, error)` - Filter to specific project
- `EncodeProjectPath(path string) string` - Convert `/Users/foo/bar` to `-Users-foo-bar`

**Traces to**: Research Summary (Location)

---

## Phase 4: Chunking Logic

### Step 4.1: Implement message chunker

Create `internal/recall/claude/chunker.go`:

```go
const MaxChunkSize = 8000

func ChunkMessage(content string) []string {
    if len(content) <= MaxChunkSize {
        return []string{content}
    }
    // Split at sentence boundaries
    return splitAtSentences(content, MaxChunkSize)
}

// ChunkSourceID generates unique source_id for each chunk of a message.
// For single-chunk messages: returns original UUID unchanged.
// For multi-chunk messages: appends chunk index (e.g., "abc123-0", "abc123-1").
func ChunkSourceID(originalUUID string, chunkIndex int, totalChunks int) string {
    if totalChunks == 1 {
        return originalUUID
    }
    return fmt.Sprintf("%s-%d", originalUUID, chunkIndex)
}
```

**Chunked Message Strategy**:
- Original message UUID: `abc123`
- If message fits in one chunk: `source_id = "abc123"`
- If message splits into 3 chunks:
  - Chunk 0: `source_id = "abc123-0"`
  - Chunk 1: `source_id = "abc123-1"`
  - Chunk 2: `source_id = "abc123-2"`

This ensures unique constraint is satisfied while maintaining traceability to original message.

**Traces to**: Decision 1 (Chunking Strategy)

---

## Phase 5: CLI Commands

### Step 5.1: Create watch command structure

Add to `internal/cli/recall.go`:

```go
{
    Name:  "watch",
    Usage: "Watch and import content from external sources",
    Subcommands: []*cli.Command{
        {
            Name:   "claude",
            Usage:  "Import Claude Code conversation history",
            Action: recallWatchClaudeAction,
            Flags: []cli.Flag{
                &cli.BoolFlag{Name: "once", Usage: "Import once and exit"},
                &cli.StringFlag{Name: "path", Usage: "Claude config path", Value: defaultClaudePath()},
                &cli.StringFlag{Name: "project", Usage: "Filter to specific project"},
                &cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Verbose output"},
                &cli.DurationFlag{Name: "interval", Usage: "Poll interval", Value: 30 * time.Second},
            },
        },
    },
}
```

**Traces to**: Decision 5 (CLI Structure), Technical Requirements

### Step 5.2: Implement import logic

In `recallWatchClaudeAction`:

1. Discover history files
2. For each file:
   a. Parse JSONL entries
   b. Filter to importable messages (user/assistant, not isMeta)
   c. For each message:
      - **Check ExistsBySourceID first** (fast index lookup)
      - If exists → skip (no embedding generation)
      - If new → generate embedding, then SaveWithSource
   d. Report progress (new vs skipped counts)
3. Summary output

**Rationale**: Embedding generation is expensive. Pre-checking existence avoids wasted compute for already-imported messages. The unique index makes the existence check fast.

**Traces to**: BR-001, BR-002, BR-003, BR-004

### Step 5.3: Implement watch loop

For continuous mode (no `--once` flag):

```go
ticker := time.NewTicker(interval)
defer ticker.Stop()

done := make(chan os.Signal, 1)
signal.Notify(done, os.Interrupt, syscall.SIGTERM)

for {
    select {
    case <-ticker.C:
        importNewMessages(...)
    case <-done:
        return nil
    }
}
```

**Traces to**: BR-005, Decision 3 (Polling)

---

## Phase 6: Conversation Retrieval

### Step 6.1: Add conversation list command

Add subcommand `brains recall conversation <id>`:

```go
{
    Name:      "conversation",
    Usage:     "View all messages in a conversation",
    ArgsUsage: "<conversation-id>",
    Action:    recallConversationAction,
}
```

**Traces to**: BR-009, BR-010

---

## Phase 7: Testing

Testing strategy: Tests are organized by business requirement to ensure complete coverage. Each test validates specific acceptance criteria from the business spec.

### Step 7.1: Unit Tests - Parser (BR-001, BR-006)

**File**: `internal/recall/claude/parser_test.go`

Test cases for **BR-001** (Import history):
- `TestParseFile_ValidUserMessage` - Parse user message from JSONL
- `TestParseFile_ValidAssistantMessage` - Parse assistant message from JSONL
- `TestParseFile_MalformedJSON` - Gracefully skip bad lines, continue parsing
- `TestParseFile_EmptyFile` - Handle empty file without error
- `TestParseFile_LargeLine` - Handle messages up to 10MB buffer

Test cases for **BR-006** (Preserve structure):
- `TestFilterImportable_SkipsIsMeta` - Meta messages excluded
- `TestFilterImportable_IncludesSidechain` - Sidechain messages included (Decision 7)
- `TestFilterImportable_SkipsNonUserAssistant` - summary/system types excluded
- `TestExtractContent_StringContent` - Direct string content extraction
- `TestExtractContent_ContentBlocks` - Array of content blocks
- `TestExtractContent_TextBlock` - Extract text from text blocks
- `TestExtractContent_ThinkingBlock` - Extract thinking (valuable for search)
- `TestExtractContent_SkipsToolBlocks` - tool_use/tool_result not extracted

**Traces to**: BR-001, BR-006, Research Summary

### Step 7.2: Unit Tests - Chunker (BR-006)

**File**: `internal/recall/claude/chunker_test.go`

Test cases for **BR-006** (Preserve structure):
- `TestChunkMessage_ShortMessage` - Under 8000 chars returns unchanged
- `TestChunkMessage_ExactLimit` - Exactly 8000 chars returns unchanged
- `TestChunkMessage_SplitsAtSentence` - Splits at ". " boundary
- `TestChunkMessage_SplitsAtNewline` - Splits at ".\n" boundary
- `TestChunkMessage_ForceCut` - No boundary found, force cut at max
- `TestChunkMessage_MultipleChunks` - Very long message produces 3+ chunks

Test cases for **BR-008** (Unique identifiers):
- `TestChunkSourceID_SingleChunk` - Returns original UUID unchanged
- `TestChunkSourceID_MultipleChunks` - Returns "uuid-0", "uuid-1", etc.

**Traces to**: BR-006, BR-008, Decision 1

### Step 7.3: Unit Tests - Discovery (BR-001)

**File**: `internal/recall/claude/discovery_test.go`

Test cases for **BR-001** (Import history):
- `TestDiscoverHistoryFiles_FindsJSONL` - Finds .jsonl files in projects/
- `TestDiscoverHistoryFiles_IgnoresOtherFiles` - Skips non-.jsonl files
- `TestDiscoverHistoryFiles_EmptyDir` - Empty directory returns empty slice
- `TestDiscoverProjectFiles_FiltersByProject` - --project flag filters correctly
- `TestEncodeProjectPath_Basic` - `/Users/foo/bar` → `-Users-foo-bar`
- `TestEncodeProjectPath_RootPath` - `/` → `-`
- `TestDefaultClaudePath_ExpandsTilde` - `~/.claude` expands correctly

**Traces to**: BR-001

### Step 7.4: Unit Tests - Storage Methods (BR-002, BR-008, BR-009, BR-010)

**File**: `internal/recall/postgres/storage_test.go` (extend existing)

Test cases for **BR-002** (No duplicates):
- `TestExistsBySourceID_NotFound` - Returns false for new source_id
- `TestExistsBySourceID_Found` - Returns true for existing source_id
- `TestExistsBySourceID_SameIDDifferentSource` - Different sources can have same source_id
- `TestSaveWithSource_NewMessage` - Returns created=true
- `TestSaveWithSource_DuplicateMessage` - Returns created=false, no error
- `TestSaveWithSource_DuplicateRaceCondition` - Concurrent inserts handled correctly

Test cases for **BR-008** (Unique identifiers):
- `TestSaveWithSource_StoresSourceID` - source_id persisted correctly
- `TestSaveWithSource_StoresConversationID` - conversation_id persisted correctly
- `TestSaveWithSource_StoresMetadata` - JSONB metadata persisted correctly

Test cases for **BR-009** (Get messages by conversation):
- `TestGetByConversation_ReturnsAllMessages` - All chunks with matching conversation_id
- `TestGetByConversation_OrderedByTimestamp` - Chronological order
- `TestGetByConversation_EmptyResult` - Unknown conversation returns empty slice

Test cases for **BR-010** (Navigate to full conversation):
- `TestGetByConversation_PreservesMetadata` - Role, timestamp accessible
- `TestGetByConversation_IncludesChunkedMessages` - Split messages all returned

**Traces to**: BR-002, BR-008, BR-009, BR-010

### Step 7.5: Integration Tests - Import Flow (BR-001, BR-002, BR-003, BR-004)

**File**: `internal/recall/claude/import_test.go`

Test cases for **BR-001** (Import history):
- `TestImport_RealHistoryFile` - Imports actual Claude history file
- `TestImport_MultipleFiles` - Discovers and imports from multiple files
- `TestImport_LargeMessage` - Message >8000 chars chunked correctly

Test cases for **BR-002** (No duplicates):
- `TestImport_RerunSkipsDuplicates` - Second import adds no new records
- `TestImport_PartialRerun` - New messages added, existing skipped
- `TestImport_DuplicateAcrossFiles` - Same message in multiple files handled

Test cases for **BR-003** (Manual trigger):
- `TestImport_OnceMode` - --once flag runs once and exits

Test cases for **BR-004** (Progress feedback):
- `TestImport_ReportsProgress` - Output includes new/skipped counts
- `TestImport_VerboseMode` - Per-message output with --verbose

**Traces to**: BR-001, BR-002, BR-003, BR-004

### Step 7.6: Integration Tests - Watch Mode (BR-005)

**File**: `internal/recall/claude/watch_test.go`

Test cases for **BR-005** (Auto-import new conversations):
- `TestWatch_DetectsNewFile` - New history file imported on next tick
- `TestWatch_DetectsNewMessages` - New messages in existing file imported
- `TestWatch_IntervalRespected` - Polls at configured interval
- `TestWatch_GracefulShutdown` - SIGTERM stops cleanly

**Traces to**: BR-005, Decision 3

### Step 7.7: Integration Tests - Conversation Retrieval (BR-009, BR-010)

**File**: `internal/recall/claude/conversation_test.go`

Test cases for **BR-009** (Retrieve conversation messages):
- `TestConversation_AllMessagesReturned` - Complete conversation retrieved
- `TestConversation_ChronologicalOrder` - Messages ordered by timestamp
- `TestConversation_IncludesUserAndAssistant` - Both roles present

Test cases for **BR-010** (Navigate from message to conversation):
- `TestSearch_ReturnsConversationID` - Search results include conversation_id
- `TestConversation_FromSearchResult` - Can retrieve conversation from search hit

**Traces to**: BR-009, BR-010

### Step 7.8: End-to-End Test (All Scenarios)

**File**: `internal/recall/claude/e2e_test.go`

End-to-end validation of user scenarios:

**Scenario 1** (Import history):
- `TestE2E_ImportAndSearch` - Import history, search finds relevant message

**Scenario 2** (Update history):
- `TestE2E_ReimportNoDuplicates` - Import, add messages, reimport, verify no dupes

**Scenario 3** (View conversation):
- `TestE2E_SearchToConversation` - Search → result → conversation → full context

**Traces to**: All scenarios from Business Spec

---

## Testing Summary

| Business Requirement | Test Coverage |
|---------------------|---------------|
| BR-001 (Import) | 7.1, 7.3, 7.5 |
| BR-002 (No duplicates) | 7.4, 7.5 |
| BR-003 (Manual trigger) | 7.5 |
| BR-004 (Progress) | 7.5 |
| BR-005 (Watch mode) | 7.6 |
| BR-006 (Preserve structure) | 7.1, 7.2 |
| BR-007 | DEFERRED |
| BR-008 (Unique IDs) | 7.2, 7.4 |
| BR-009 (Get conversation) | 7.4, 7.7 |
| BR-010 (Navigate to conv) | 7.4, 7.7 |

| Scenario | Test Coverage |
|----------|---------------|
| S1 (Import) | 7.1-7.5, 7.8 |
| S2 (Duplicates) | 7.4, 7.5, 7.8 |
| S3 (View Conversation) | 7.7, 7.8 |
| S4 (On-Demand) | 7.5 |
| S5 (Watch Mode) | 7.6 |

---

## Dependency Graph

```
Phase 1 (Schema) ──► Phase 2 (Storage) ──► Phase 5 (CLI)
                                    │
Phase 3 (Parser) ──────────────────►│
                                    │
Phase 4 (Chunker) ─────────────────►│
                                    │
                         Phase 6 (Conversation)
                                    │
                         Phase 7 (Testing)
```

**Critical Path**: Schema → Storage → CLI

**Parallel Opportunities**:
- Phase 3 (Parser) can run in parallel with Phase 1-2
- Phase 4 (Chunker) can run in parallel with Phase 1-3

---

## Files to Create/Modify

### New Files (6)
- `internal/database/migrations/postgres/003_recall_chunks_source_tracking.sql`
- `internal/recall/claude/types.go`
- `internal/recall/claude/parser.go`
- `internal/recall/claude/discovery.go`
- `internal/recall/claude/chunker.go`
- `internal/recall/claude/parser_test.go`

### Modified Files (3)
- `internal/recall/types.go` - Add Source, SourceID, ConversationID, Metadata fields
- `internal/recall/storage.go` - Add SaveWithSource, ExistsBySourceID, GetByConversation
- `internal/recall/postgres/storage.go` - Implement new methods
- `internal/cli/recall.go` - Add watch command

---

## Estimated Complexity

| Metric | Value |
|--------|-------|
| New files | 6 |
| Modified files | 4 |
| Estimated LOC | ~600 |
| Classification | **Medium** |

---

## Risks

1. **Claude history format changes** - Mitigate with version checking and graceful degradation
2. **Large history files** - Mitigate with streaming parser, progress feedback
3. **Embedding throughput** - Mitigate with batching (future optimization)

---

## Next Step

```
/brains.tasks
```
