# Tasks: Recall MCP Conversations

**Generated**: 2026-01-21
**Total Tasks**: 12
**Complexity**: Medium (8 files affected)
**Critical Path**: T001 → T003/T004 → T005/T006 → T007 → T008/T009/T010

## Dependency Graph

```
T001 (storage interface) ──┬──► T003 (list tool)
T002 (types) ──────────────┤
                           └──► T004 (read tool)
                                      │
                                      ▼
                               T005 (server integration)
                                      │
                                      ▼
                               T006 (serve.go deps)
                                      │
                                      ▼
                               T007 (config defaults)
                                      │
                                      ▼
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
              T008 (list tests)  T009 (read tests)  T010 (edge tests)
                                      │
                                      ▼
                               T011 (verify build)
                                      │
                                      ▼
                               T012 (manual smoke test)
```

## Tasks

### Foundation Layer

- [ ] T001 [US1,US2] Add `GetConversationChunks` method to storage interface and implement PostgreSQL query
  - File: `internal/recall/storage.go` - Add interface method
  - File: `internal/recall/postgres/storage.go` - Implement query with timestamp+id ordering
  - Acceptance: Method returns chunks ordered by timestamp ASC, then ID ASC

- [ ] T002 [P] Create response types for MCP tools
  - File: `internal/mcp/tools/recall/types.go` (create)
  - Types: `ListResponse`, `ReadResponse`, `ChunkOutput`, `ErrorResponse`
  - Acceptance: Types compile and match spec JSON structures

### Tool Implementation

- [ ] T003 [US1] Implement `recall-list-conversations` tool logic
  - File: `internal/mcp/tools/recall/tool.go` (create)
  - Implement: `ListConversations(ctx, args)` with pagination normalization
  - Uses has_more detection pattern (fetch limit+1)
  - Acceptance: Returns paginated conversation summaries

- [ ] T004 [US2] Implement `recall-read-conversation` tool logic
  - File: `internal/mcp/tools/recall/tool.go`
  - Implement: `ReadConversation(ctx, args)` with UUID validation
  - Implement: conversation existence check
  - Acceptance: Returns paginated chunks, validates conversation_id

### Integration Layer

- [ ] T005 [US1,US2] Register recall tools in MCP server
  - File: `internal/mcp/server.go`
  - Add: `recallTool *recall.Tool` field
  - Add: `registerRecallTools()` method
  - Register both tools with schemas from technical spec
  - Acceptance: Tools appear in MCP server tool list

- [ ] T006 Update serve command to pass recall storage to MCP server
  - File: `cmd/brains/serve.go`
  - Update: `NewServer` call to pass recall storage
  - Acceptance: Recall storage is wired to MCP server

- [ ] T007 Add recall tools to default enabled tools list
  - File: `internal/config/config.go`
  - Add: `recall-list-conversations`, `recall-read-conversation` to defaults
  - Acceptance: Tools enabled by default in config

### Test Implementation

- [ ] T008 [P] [US1] Integration tests for list-conversations tool
  - File: `internal/mcp/tools/recall/tool_test.go` (create)
  - Tests: DefaultPagination, CustomPage, ProjectFilter, LimitCapped, EmptyResult
  - Acceptance: Tests pass against PostgreSQL test database

- [ ] T009 [P] [US2] Integration tests for read-conversation tool
  - File: `internal/mcp/tools/recall/tool_test.go`
  - Tests: DefaultPagination, CustomPage, InvalidUUID, NotFound, ChronologicalOrder
  - Acceptance: Tests pass against PostgreSQL test database

- [ ] T010 [P] Edge case tests for both tools
  - File: `internal/mcp/tools/recall/tool_test.go`
  - Tests: PageBeyondData, LimitZero, NegativePage, IdenticalTimestamps
  - Acceptance: All edge cases handled per spec

### Verification

- [ ] T011 Verify project builds and lints cleanly
  - Run: `task lint` and `task build`
  - Acceptance: No build errors, no lint warnings

- [ ] T012 Manual smoke test via MCP
  - Start MCP server, call both tools via Claude
  - Verify JSON responses match spec
  - Acceptance: Tools functional end-to-end

## Execution Order

**Phase 1 (Foundation)**: T001, T002 (parallel)
**Phase 2 (Tools)**: T003, T004 (parallel after Phase 1)
**Phase 3 (Integration)**: T005 → T006 → T007 (sequential)
**Phase 4 (Testing)**: T008, T009, T010 (parallel after Phase 3)
**Phase 5 (Verification)**: T011 → T012 (sequential)

## Traceability

| Spec FR | Tasks |
|---------|-------|
| FR-001 | T003, T005, T008 |
| FR-002 | T004, T005, T009 |
| FR-003 | T003, T008 |
| FR-004 | T003, T004, T008, T009 |
| FR-005 | T003, T004, T008, T009 |
| FR-006 | T003, T008 |
| FR-007 | T001, T004, T009 |
| FR-008 | T002, T004, T009 |
| FR-009 | T003, T004, T010 |
| FR-010 | T003, T004, T010 |
