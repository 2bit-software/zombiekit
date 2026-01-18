# Tasks: Claude Conversation Importer

**Feature**: conversation-importer
**Linear Ticket**: DEV-69
**Total Tasks**: 21
**Classification**: Medium

---

## Dependency Graph

```
T001 (Schema) ──────────────────────────────────────► T005 (Storage Interface)
                                                              │
T002 (Claude types) ─────┐                                    │
                         ├──► T004 (Parser) ──────────────────┤
T003 (Discovery) ────────┘                                    │
                                                              │
T006 (Chunker) ───────────────────────────────────────────────┤
                                                              │
                                                              ▼
T007 (Postgres SaveWithSource) ─┐
T008 (Postgres ExistsBySourceID)├──► T010 (CLI watch command)
T009 (Postgres GetByConversation)    T011 (CLI import logic)
                                     T012 (CLI watch loop)
                                              │
                                              ▼
                                     T013 (CLI conversation cmd)
                                              │
                                              ▼
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                     T014 (Parser)    T015 (Chunker)   T016 (Discovery)
                     T017 (Storage)
                              │               │               │
                              └───────────────┼───────────────┘
                                              ▼
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                     T018 (Import)    T019 (Watch)    T020 (Conversation)
                              │               │               │
                              └───────────────┼───────────────┘
                                              ▼
                                     T021 (E2E Tests)
```

---

## Phase 1: Schema & Foundation

- [ ] **T001** [S1,S2] Create schema migration `internal/database/migrations/postgres/003_recall_chunks_source_tracking.sql`
  - Add columns: source, source_id, conversation_id, metadata (JSONB)
  - Create unique index on (source, source_id) WHERE source_id IS NOT NULL
  - Create index on conversation_id WHERE conversation_id IS NOT NULL
  - Create index on source WHERE source IS NOT NULL
  - **Acceptance**: Migration runs without error, indexes created
  - **Traces to**: BR-002, BR-008, BR-009

---

## Phase 2: Claude Parser Package (Parallel with Phase 1)

- [ ] **T002** [P] [S1] Create Claude history types `internal/recall/claude/types.go`
  - Define HistoryEntry struct (Type, UUID, ParentUUID, SessionID, Timestamp, Message, IsMeta, IsSidechain, CWD, GitBranch, Version)
  - Define MessageContent struct (Role, Content as interface{})
  - Define ContentBlock struct (Type, Text)
  - **Acceptance**: Types compile, match JSONL format from research
  - **Traces to**: Research Summary

- [ ] **T003** [P] [S1] Create history file discovery `internal/recall/claude/discovery.go`
  - Implement DiscoverHistoryFiles(claudePath) returning []string of .jsonl files
  - Implement DiscoverProjectFiles(claudePath, projectPath) for --project filter
  - Implement EncodeProjectPath converting /Users/foo to -Users-foo
  - Implement defaultClaudePath() returning ~/.claude expanded
  - **Acceptance**: Finds real .jsonl files in ~/.claude/projects/
  - **Traces to**: BR-001

- [ ] **T004** [S1] Create JSONL parser `internal/recall/claude/parser.go`
  - **Depends on**: T002
  - Implement ParseFile(path) returning []HistoryEntry
  - Use bufio.Scanner with 10MB buffer for large lines
  - Skip malformed JSON lines gracefully
  - Implement FilterImportable() - keep user/assistant, skip isMeta, include isSidechain (Decision 7)
  - Implement ExtractContent() - handle string and []ContentBlock, extract text/thinking, skip tool_use/tool_result (Decision 8)
  - **Acceptance**: Parses real Claude history file, extracts content correctly
  - **Traces to**: BR-001, BR-006

---

## Phase 3: Chunking Logic (Parallel with Phase 1-2)

- [ ] **T006** [P] [S1] Create message chunker `internal/recall/claude/chunker.go`
  - Define MaxChunkSize = 8000
  - Implement ChunkMessage(content) returning []string
  - Split at sentence boundaries (". ", ".\n", "? ", "! ")
  - Force cut at MaxChunkSize if no boundary found
  - Implement ChunkSourceID(uuid, index, total) for unique source_ids
  - **Acceptance**: Long message splits correctly, short message returns unchanged
  - **Traces to**: Decision 1

---

## Phase 4: Storage Interface Extension

- [ ] **T005** [S1,S2] Extend recall types and interface
  - **Depends on**: T001 (migration must exist first)
  - Update `internal/recall/types.go`:
    - Add Source, SourceID, ConversationID fields to Chunk
    - Add Metadata struct (Role, Timestamp, GitBranch, CWD, ParentID)
    - Add ChunkInput struct
  - Update `internal/recall/storage.go`:
    - Add SaveWithSource method to interface
    - Add ExistsBySourceID method to interface
    - Add GetByConversation method to interface
  - **Acceptance**: Interface compiles, postgres package shows unimplemented methods
  - **Traces to**: BR-002, BR-009, BR-010

- [ ] **T007** [S1,S2] Implement SaveWithSource in postgres `internal/recall/postgres/storage.go`
  - **Depends on**: T005
  - INSERT with ON CONFLICT (source, source_id) DO NOTHING
  - Store metadata as JSONB
  - Return (id, created, error) where created=false on conflict
  - **Acceptance**: Duplicate inserts return created=false
  - **Traces to**: BR-002

- [ ] **T008** [S2] Implement ExistsBySourceID in postgres `internal/recall/postgres/storage.go`
  - **Depends on**: T005
  - Fast SELECT EXISTS query using idx_recall_chunks_source_id
  - Return (exists, error)
  - **Acceptance**: Returns true for existing, false for new, fast (<10ms)
  - **Traces to**: BR-002

- [ ] **T009** [S3] Implement GetByConversation in postgres `internal/recall/postgres/storage.go`
  - **Depends on**: T005
  - SELECT all chunks WHERE conversation_id = $1
  - ORDER BY metadata->>'timestamp' ASC
  - **Acceptance**: Returns messages in chronological order
  - **Traces to**: BR-009, BR-010

---

## Phase 5: CLI Commands

- [ ] **T010** [S1,S4,S5] Create watch command structure `internal/cli/recall.go`
  - **Depends on**: T005
  - Add "watch" subcommand with "claude" nested subcommand
  - Define flags: --once, --path, --project, --verbose, --interval
  - Wire recallWatchClaudeAction stub
  - **Acceptance**: `brains recall watch claude --help` shows all flags
  - **Traces to**: BR-003, BR-005, Decision 5

- [ ] **T011** [S1,S2] Implement import logic in recallWatchClaudeAction `internal/cli/recall.go`
  - **Depends on**: T004, T006, T007, T008, T010
  - Call DiscoverHistoryFiles or DiscoverProjectFiles based on --project
  - For each file: ParseFile → FilterImportable → for each entry:
    - Call ExistsBySourceID first (short-circuit)
    - If new: ExtractContent → ChunkMessage → for each chunk: generate embedding, SaveWithSource
  - Track and report new/skipped counts
  - Support --verbose for per-message output
  - **Acceptance**: Imports real history, skips duplicates on re-run
  - **Traces to**: BR-001, BR-002, BR-004

- [ ] **T012** [S5] Implement watch loop `internal/cli/recall.go`
  - **Depends on**: T011
  - If --once: run import once and exit
  - Else: ticker loop with --interval, signal handling for graceful shutdown
  - Re-scan files each interval
  - **Acceptance**: Continuous mode runs until SIGTERM, imports new messages
  - **Traces to**: BR-005, Decision 3

- [ ] **T013** [S3] Add conversation command `internal/cli/recall.go`
  - **Depends on**: T009
  - Add "conversation" subcommand with <conversation-id> arg
  - Call GetByConversation and display messages in order
  - Format: [role] timestamp content-preview
  - **Acceptance**: Shows all messages from a conversation in order
  - **Traces to**: BR-009, BR-010

---

## Phase 6: Testing

Testing organized by business requirement for complete traceability.

### Unit Tests

- [ ] **T014** [P] Unit tests for parser `internal/recall/claude/parser_test.go`
  - **Depends on**: T004
  - **BR-001 (Import history)**:
    - TestParseFile_ValidUserMessage
    - TestParseFile_ValidAssistantMessage
    - TestParseFile_MalformedJSON (graceful skip)
    - TestParseFile_EmptyFile
    - TestParseFile_LargeLine (10MB buffer)
  - **BR-006 (Preserve structure)**:
    - TestFilterImportable_SkipsIsMeta
    - TestFilterImportable_IncludesSidechain
    - TestFilterImportable_SkipsNonUserAssistant
    - TestExtractContent_StringContent
    - TestExtractContent_ContentBlocks
    - TestExtractContent_TextBlock
    - TestExtractContent_ThinkingBlock
    - TestExtractContent_SkipsToolBlocks
  - **Acceptance**: All tests pass, parser handles edge cases
  - **Traces to**: BR-001, BR-006

- [ ] **T015** [P] Unit tests for chunker `internal/recall/claude/chunker_test.go`
  - **Depends on**: T006
  - **BR-006 (Preserve structure)**:
    - TestChunkMessage_ShortMessage
    - TestChunkMessage_ExactLimit
    - TestChunkMessage_SplitsAtSentence
    - TestChunkMessage_SplitsAtNewline
    - TestChunkMessage_ForceCut
    - TestChunkMessage_MultipleChunks
  - **BR-008 (Unique identifiers)**:
    - TestChunkSourceID_SingleChunk
    - TestChunkSourceID_MultipleChunks
  - **Acceptance**: All tests pass, chunking preserves message integrity
  - **Traces to**: BR-006, BR-008

- [ ] **T016** [P] Unit tests for discovery `internal/recall/claude/discovery_test.go`
  - **Depends on**: T003
  - **BR-001 (Import history)**:
    - TestDiscoverHistoryFiles_FindsJSONL
    - TestDiscoverHistoryFiles_IgnoresOtherFiles
    - TestDiscoverHistoryFiles_EmptyDir
    - TestDiscoverProjectFiles_FiltersByProject
    - TestEncodeProjectPath_Basic
    - TestEncodeProjectPath_RootPath
    - TestDefaultClaudePath_ExpandsTilde
  - **Acceptance**: All tests pass, discovery finds correct files
  - **Traces to**: BR-001

- [ ] **T017** [P] Unit tests for storage methods `internal/recall/postgres/storage_test.go`
  - **Depends on**: T007, T008, T009
  - **BR-002 (No duplicates)**:
    - TestExistsBySourceID_NotFound
    - TestExistsBySourceID_Found
    - TestExistsBySourceID_SameIDDifferentSource
    - TestSaveWithSource_NewMessage
    - TestSaveWithSource_DuplicateMessage
    - TestSaveWithSource_DuplicateRaceCondition
  - **BR-008 (Unique identifiers)**:
    - TestSaveWithSource_StoresSourceID
    - TestSaveWithSource_StoresConversationID
    - TestSaveWithSource_StoresMetadata
  - **BR-009 (Get messages by conversation)**:
    - TestGetByConversation_ReturnsAllMessages
    - TestGetByConversation_OrderedByTimestamp
    - TestGetByConversation_EmptyResult
  - **BR-010 (Navigate to conversation)**:
    - TestGetByConversation_PreservesMetadata
    - TestGetByConversation_IncludesChunkedMessages
  - **Acceptance**: All tests pass, storage methods behave correctly
  - **Traces to**: BR-002, BR-008, BR-009, BR-010

### Integration Tests

- [ ] **T018** Integration tests for import flow `internal/recall/claude/import_test.go`
  - **Depends on**: T011
  - **BR-001 (Import history)**:
    - TestImport_RealHistoryFile
    - TestImport_MultipleFiles
    - TestImport_LargeMessage
  - **BR-002 (No duplicates)**:
    - TestImport_RerunSkipsDuplicates
    - TestImport_PartialRerun
    - TestImport_DuplicateAcrossFiles
  - **BR-003 (Manual trigger)**:
    - TestImport_OnceMode
  - **BR-004 (Progress feedback)**:
    - TestImport_ReportsProgress
    - TestImport_VerboseMode
  - **Acceptance**: End-to-end import works, duplicates detected
  - **Traces to**: BR-001, BR-002, BR-003, BR-004

- [ ] **T019** Integration tests for watch mode `internal/recall/claude/watch_test.go`
  - **Depends on**: T012
  - **BR-005 (Auto-import)**:
    - TestWatch_DetectsNewFile
    - TestWatch_DetectsNewMessages
    - TestWatch_IntervalRespected
    - TestWatch_GracefulShutdown
  - **Acceptance**: Watch mode continuously imports new content
  - **Traces to**: BR-005

- [ ] **T020** Integration tests for conversation retrieval `internal/recall/claude/conversation_test.go`
  - **Depends on**: T013
  - **BR-009 (Retrieve messages)**:
    - TestConversation_AllMessagesReturned
    - TestConversation_ChronologicalOrder
    - TestConversation_IncludesUserAndAssistant
  - **BR-010 (Navigate to conversation)**:
    - TestSearch_ReturnsConversationID
    - TestConversation_FromSearchResult
  - **Acceptance**: Conversation retrieval works from any entry point
  - **Traces to**: BR-009, BR-010

### End-to-End Tests

- [ ] **T021** E2E tests for user scenarios `internal/recall/claude/e2e_test.go`
  - **Depends on**: T018, T019, T020
  - **Scenario 1 (Import history)**:
    - TestE2E_ImportAndSearch
  - **Scenario 2 (Update history)**:
    - TestE2E_ReimportNoDuplicates
  - **Scenario 3 (View conversation)**:
    - TestE2E_SearchToConversation
  - **Acceptance**: Full user workflows validated
  - **Traces to**: All scenarios from Business Spec

---

## Traceability Matrix

| BR | Implementation Tasks | Test Tasks |
|----|---------------------|------------|
| BR-001 (Import) | T001, T002, T003, T004, T011 | T014, T016, T018, T021 |
| BR-002 (No duplicates) | T001, T005, T007, T008, T011 | T017, T018, T021 |
| BR-003 (Manual trigger) | T010, T011 | T018 |
| BR-004 (Progress) | T011 | T018 |
| BR-005 (Watch mode) | T010, T012 | T019 |
| BR-006 (Preserve structure) | T004, T005 | T014, T015 |
| BR-007 | DEFERRED | - |
| BR-008 (Unique IDs) | T001, T005 | T015, T017 |
| BR-009 (Get conversation) | T001, T005, T009, T013 | T017, T020 |
| BR-010 (Navigate) | T005, T009, T013 | T017, T020 |

| Scenario | Implementation Tasks | Test Tasks |
|----------|---------------------|------------|
| S1 (Import) | T001-T011 | T014, T016, T018, T021 |
| S2 (Duplicates) | T001, T005, T007, T008, T011 | T017, T018, T021 |
| S3 (View Conversation) | T009, T013 | T020, T021 |
| S4 (On-Demand) | T010, T011 | T018 |
| S5 (Watch Mode) | T010, T012 | T019 |

---

## Execution Order

**Critical Path**: T001 → T005 → T007/T008/T009 → T010 → T011 → T012

**Parallel Opportunities**:
- T002, T003, T006 can run in parallel with T001
- T004 can start after T002
- T014, T015, T016 can run in parallel after their dependencies

**Suggested Order**:
1. Start: T001, T002, T003, T006 (parallel)
2. After T002: T004
3. After T001: T005
4. After T005: T007, T008, T009 (parallel)
5. After T007, T008: T010
6. After T004, T006, T010: T011
7. After T011: T012, T013 (parallel)
8. Unit Tests: T014-T017 (parallel, after implementation dependencies)
9. Integration Tests: T018, T019, T020 (after unit tests)
10. E2E Tests: T021 (after integration tests)

---

## Next Step

```
/brains.implement
```
