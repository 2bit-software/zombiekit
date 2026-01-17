# Task List: Semantic Search Foundation

**Feature**: semantic-search-foundation
**Linear Ticket**: DEV-72
**Generated**: 2026-01-17

---

## Complexity Analysis

| Metric | Value |
|--------|-------|
| Files affected | 10 |
| New files | 7 |
| Modified files | 3 |
| Estimated LOC | ~400 |
| Classification | **Simple** (<15 files) |

---

## Dependency Graph

```
T001 ─┬─► T003 ─► T004 ─► T005 ─┬─► T008 ─► T009 ─► T010 ─► T011 ─► T012
      │                         │
T002 ─┘                         │
                                │
T006 ───────────────────────────┤
                                │
T007 ───────────────────────────┘
```

**Critical Path**: T001 → T003 → T004 → T005 → T008 → T009 → T010 → T011 → T012

**Parallel Opportunities**:
- T001, T002 can run in parallel
- T006, T007 can run in parallel with T003-T005

---

## Tasks

### Phase 1: Infrastructure (Parallelizable)

- [ ] **T001** [P] Create pgvector migration
  - **File**: `/internal/database/migrations/postgres/002_recall_chunks.sql`
  - **Action**: Create new file with CREATE EXTENSION, CREATE TABLE, CREATE INDEX statements
  - **Acceptance**: Migration runs without error on fresh database
  - **Traces to**: Plan Step 1.1, BR-001, BR-007

- [ ] **T002** [P] Add Go dependencies
  - **File**: `/go.mod`
  - **Action**: `go get github.com/ollama/ollama/api github.com/pgvector/pgvector-go`
  - **Acceptance**: `go mod tidy` succeeds, imports resolve
  - **Traces to**: Plan Step 1.2

### Phase 2: Core Types (Sequential after T002)

- [ ] **T003** Create recall package with types
  - **File**: `/internal/recall/types.go`
  - **Action**: Create `Chunk` and `SearchResult` structs per technical spec
  - **Acceptance**: Package compiles
  - **Traces to**: Plan Step 2.2

- [ ] **T004** Create hash utility
  - **File**: `/internal/recall/hash.go`
  - **Action**: Implement `ContentHash(content string) string` using SHA-256
  - **Acceptance**: Same input produces same hash, different input produces different hash
  - **Traces to**: Plan Step 3.1, BR-008

- [ ] **T005** Create Storage interface and Embedder interface
  - **File**: `/internal/recall/storage.go`
  - **File**: `/internal/recall/embedder.go`
  - **Action**: Define `Storage` interface (Ingest, List, Search, Close) and `Embedder` interface (Embed) with `EmbedPurpose` enum
  - **Acceptance**: Interfaces compile, match technical spec
  - **Traces to**: Plan Steps 2.3, 2.4

### Phase 3: Configuration (Parallelizable with Phase 2)

- [ ] **T006** [P] Add Ollama config fields
  - **File**: `/internal/config/storage.go`
  - **Action**: Add `OllamaURL` and `EmbeddingModel` to `FileStorageConfig` and `StorageConfig`
  - **Acceptance**: Config fields exist and are accessible
  - **Traces to**: Plan Step 5.1

- [ ] **T007** [P] Add environment variable support
  - **File**: `/internal/config/loader.go`
  - **Action**: Add `BRAINS_OLLAMA_URL` and `BRAINS_EMBEDDING_MODEL` environment variable loading with defaults
  - **Acceptance**: Environment variables override config file values
  - **Traces to**: Plan Step 5.2, BR-006

### Phase 4: Implementations (Sequential)

- [ ] **T008** Implement OllamaEmbedder
  - **File**: `/internal/recall/embedder.go`
  - **Action**: Implement `OllamaEmbedder` struct with `NewOllamaEmbedder` constructor and `Embed` method using task prefixes
  - **Acceptance**: Can generate embedding for test text when Ollama is running
  - **Traces to**: Plan Step 2.4, BR-006

- [ ] **T009** Implement PostgreSQL Storage
  - **File**: `/internal/recall/postgres/storage.go`
  - **Action**: Implement `Storage` interface: `New`, `Ingest` (with ON CONFLICT), `List`, `Search`, `Close`. Register pgvector types.
  - **Acceptance**: All interface methods implemented and compile
  - **Traces to**: Plan Steps 3.1, 3.2, BR-001, BR-002, BR-003, BR-008

### Phase 5: CLI Commands (Sequential after T009)

- [ ] **T010** Create recall CLI command structure
  - **File**: `/internal/cli/recall.go`
  - **Action**: Create `newRecallCommand()` returning parent command with three subcommands
  - **File**: `/internal/cli/root.go`
  - **Action**: Register `newRecallCommand()` in Commands slice
  - **Acceptance**: `brains recall --help` shows save, list, search subcommands
  - **Traces to**: Plan Steps 4.1, 4.5

- [ ] **T011** Implement recall subcommands
  - **File**: `/internal/cli/recall.go`
  - **Action**: Implement `recallSaveAction`, `recallListAction`, `recallSearchAction` with error handling and output formatting
  - **Acceptance**:
    - `brains recall save "test"` stores content and prints confirmation
    - `brains recall save "test"` (duplicate) produces no output
    - `brains recall list` shows stored entries with timestamps
    - `brains recall search "query"` returns ranked results with similarity and timestamps
  - **Traces to**: Plan Steps 4.2, 4.3, 4.4, 6.1, 6.2, 6.3, BR-004, BR-005, BR-007

### Phase 6: Testing (Sequential after T011)

- [ ] **T012** Add integration test
  - **File**: `/internal/recall/integration_test.go`
  - **Action**: Implement round-trip test: save content → search with similar query → verify result found
  - **Acceptance**: Test passes with running PostgreSQL and Ollama
  - **Traces to**: Plan Step 7.2, SM-001, SM-002

---

## Requirements Traceability

| Requirement | Tasks |
|-------------|-------|
| BR-001 | T001, T009, T011 |
| BR-002 | T009, T011 |
| BR-003 | T009, T011 |
| BR-004 | T011 |
| BR-005 | T011 |
| BR-006 | T007, T008 |
| BR-007 | T001, T011 |
| BR-008 | T004, T009, T011 |
| SM-001 | T012 |
| SM-002 | T012 |

---

## Execution Summary

| Metric | Value |
|--------|-------|
| Total tasks | 12 |
| Parallel opportunities | 4 tasks (T001+T002, T006+T007) |
| Sequential dependencies | 8 tasks on critical path |
| Estimated complexity | Simple |

---

## Next Step

```
/brains.implement
```
