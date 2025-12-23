# Tasks: ZombieKit MCP Tool

**Input**: Design documents from `/specs/017-zombiekit-mcp/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests will be written alongside implementation following Go conventions.

**Organization**: Single user story (US1) - minimal feature with straightforward implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Create package structure for ZombieKit tool

- [x] T001 Create zombiekit tool package directory at internal/mcp/tools/zombiekit/

---

## Phase 2: Foundational (Configuration)

**Purpose**: Register new tool in config system

- [x] T002 Add "feature" to KnownTools slice in internal/config/tools.go

**Checkpoint**: Tool is configurable via --enable-tool/--disable-tool flags

---

## Phase 3: User Story 1 - Invoke Feature Tool via MCP (Priority: P1)

**Goal**: AI assistants can invoke the "feature" tool to retrieve step feature template contents

**Independent Test**: Start MCP server, call "feature" tool, verify file contents returned (or error if file missing)

### Implementation for User Story 1

- [x] T003 [P] [US1] Create Tool struct with NewTool() constructor in internal/mcp/tools/zombiekit/tool.go
- [x] T004 [P] [US1] Implement Definition() method returning tool name and description in internal/mcp/tools/zombiekit/tool.go
- [x] T005 [US1] Implement Execute() method with home dir expansion and file read in internal/mcp/tools/zombiekit/tool.go
- [x] T006 [US1] Add error handling for file not found, permission denied, and read errors in internal/mcp/tools/zombiekit/tool.go
- [x] T007 [P] [US1] Write unit tests for Execute() in internal/mcp/tools/zombiekit/tool_test.go
- [x] T008 [US1] Add zombiekit field to Server struct in internal/mcp/server.go
- [x] T009 [US1] Create zombiekit tool instance in NewServer() in internal/mcp/server.go
- [x] T010 [US1] Implement handleFeature() handler method in internal/mcp/server.go
- [x] T011 [US1] Register feature tool in registerTools() with enablement check in internal/mcp/server.go

**Checkpoint**: Feature tool is fully functional and appears in MCP tool list

---

## Phase 4: Polish & Validation

**Purpose**: Final verification and cleanup

- [x] T012 Build and verify no compilation errors with go build ./...
- [x] T013 Run all tests with go test ./internal/mcp/...
- [x] T014 Manual test: start server and invoke feature tool via MCP inspector

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - creates package structure
- **Foundational (Phase 2)**: Can run in parallel with Phase 1
- **User Story 1 (Phase 3)**: Depends on Phase 1 and Phase 2 completion
- **Polish (Phase 4)**: Depends on Phase 3 completion

### Task Dependencies within User Story 1

```text
T003 (struct) ─┬─> T005 (Execute) ─> T006 (errors) ─> T007 (tests)
T004 (def)   ──┘
                                                          │
T008 (field) ─> T009 (instance) ─> T010 (handler) ─> T011 (register)
                                                          │
                                                          v
                                                    T012, T013, T014
```

### Parallel Opportunities

Tasks T003 and T004 can run in parallel (separate methods, no dependency)
Task T007 can run in parallel with T008-T011 (different files)

---

## Parallel Example: User Story 1 Tool Implementation

```bash
# Launch tool struct and definition in parallel:
Task: "Create Tool struct in internal/mcp/tools/zombiekit/tool.go"
Task: "Implement Definition() in internal/mcp/tools/zombiekit/tool.go"
```

---

## Implementation Strategy

### MVP (This Feature IS the MVP)

1. Complete Phase 1: Create package
2. Complete Phase 2: Add to KnownTools
3. Complete Phase 3: Full tool implementation
4. Complete Phase 4: Validate

### Execution Order

```text
T001 → T002 → T003/T004 (parallel) → T005 → T006 → T007
                                              ↓
                               T008 → T009 → T010 → T011
                                                      ↓
                                            T012 → T013 → T014
```

---

## Notes

- All paths are relative to repository root
- Follow existing tool patterns (stickymemory, codereasoning) for consistency
- Use `os.UserHomeDir()` for cross-platform home directory resolution
- Return descriptive errors with resolved file path for debugging
