# Tasks: Profile Creator

**Initiative**: 6969926a-feature-profile-creator
**Plan**: [plan.md](./plan.md)
**Spec**: [spec.md](./spec.md)

## Task List

### Phase 1: MCP Tool Implementation

#### T001: Add WriteRequest/WriteResponse types
- [ ] **File**: `internal/mcp/tools/profile/types.go`
- **Description**: Add request and response types for profile-write MCP tool
- **Requirements**: FR-040
- **Acceptance**:
  - `WriteRequest` has fields: `Name`, `Content`, `Location`, `Overwrite`
  - `WriteResponse` has fields: `Success`, `Path`, `Error`, `Message`, `Hint`
- **Dependencies**: None

#### T002: Add Write method to profile service
- [ ] **File**: `internal/profile/service.go`
- **Description**: Add `Write(name, content, location string, overwrite bool) (string, error)` method
- **Requirements**: FR-041, FR-042, FR-043, FR-044
- **Acceptance**:
  - Creates `.brains/profiles/` or `~/.brains/profiles/` directory if needed
  - Returns error with code `PROFILE_EXISTS` if file exists and overwrite is false
  - Writes atomically using temp file + rename
  - Returns absolute path on success
  - Normalizes name (lowercase, alphanumeric + hyphens)
- **Dependencies**: None

#### T003: Add HandleWrite handler
- [ ] **File**: `internal/mcp/tools/profile/tool.go`
- **Description**: Add handler for profile-write MCP tool
- **Requirements**: FR-040
- **Acceptance**:
  - Validates required parameters (name, content, location)
  - Validates location is "local" or "global"
  - Calls service.Write() and returns structured response
  - Handles errors with appropriate error codes
- **Dependencies**: T001, T002

#### T004: Register profile-write tool
- [ ] **File**: `internal/mcp/server.go`
- **Description**: Register the profile-write tool in MCP server
- **Requirements**: FR-040
- **Acceptance**:
  - Tool appears in MCP tool list
  - Tool schema matches spec (name, content, location required; overwrite optional)
- **Dependencies**: T003

#### T005: Add Write method tests
- [ ] **File**: `internal/profile/service_test.go`
- **Description**: Add unit tests for Write() method
- **Requirements**: Test-first principle
- **Acceptance**:
  - Test local write creates file in `.brains/profiles/`
  - Test global write creates file in `~/.brains/profiles/`
  - Test directory creation when missing
  - Test error on existing file without overwrite
  - Test success with overwrite flag
  - Test name normalization
- **Dependencies**: T002

#### T006: Add HandleWrite tests
- [ ] **File**: `internal/mcp/tools/profile/tool_test.go`
- **Description**: Add tests for HandleWrite MCP handler
- **Requirements**: Test coverage
- **Acceptance**:
  - Test valid request returns success response
  - Test missing required params returns error
  - Test invalid location returns error
  - Test file exists error propagates correctly
- **Dependencies**: T003

### Phase 2: Workflow Profile

#### T007: Create profile-new.md workflow profile
- [ ] **File**: `profiles/profile-new.md`
- **Description**: Create the workflow profile for `/brains.profile.new` command
- **Requirements**: FR-001, FR-002, FR-002a, FR-003, FR-010-012, FR-020-023, FR-030-033
- **Acceptance**:
  - Valid YAML frontmatter with name, description, type: skill
  - Workflow phases documented: gather → research → create → audit → highlight → write
  - References `profile-write` tool for final write
  - References `profile-list` for research phase
- **Dependencies**: T004

#### T008: Verify profile discoverability
- [ ] **Manual verification**
- **Description**: Verify new profile appears in `profile-list` and can be composed
- **Requirements**: FR-050, FR-051
- **Acceptance**:
  - `profile-list` shows `profile-new` profile
  - `profile-compose` with `profile-new` returns content
- **Dependencies**: T007

### Phase 3: Integration Testing

#### T009: End-to-end workflow test
- [ ] **Manual verification**
- **Description**: Test complete workflow from invocation to profile creation
- **Requirements**: SC-001 through SC-006
- **Acceptance**:
  - Invoke `/brains.profile.new`
  - Provide name, description, location
  - Review generated content
  - Approve and verify file created
  - Verify new profile in `profile-list`
- **Dependencies**: T007, T008

## Dependency Graph

```
T001 (types) ─┐
              ├─→ T003 (handler) ─→ T004 (register) ─→ T007 (profile) ─→ T008 (verify) ─→ T009 (e2e)
T002 (write) ─┘
       │
       └─→ T005 (write tests)
              │
T003 ─────────┴─→ T006 (handler tests)
```

## Progress Tracking

| Task | Status | Notes |
|------|--------|-------|
| T001 | ✅ Complete | types.go created |
| T002 | ✅ Complete | Write() method added to service.go |
| T003 | ✅ Complete | HandleWrite handler added to tool.go |
| T004 | ✅ Complete | profile-write registered in server.go |
| T005 | ✅ Complete | 8 test cases in service_test.go |
| T006 | ✅ Complete | 6 test cases in tool_test.go |
| T007 | ✅ Complete | profile-new.md created |
| T008 | ⬜ Pending | Manual verification after build |
| T009 | ⬜ Pending | Manual verification after build |

## Estimated Effort

| Phase | Tasks | Complexity |
|-------|-------|------------|
| Phase 1: MCP Tool | T001-T006 | Medium |
| Phase 2: Workflow | T007-T008 | Low |
| Phase 3: Testing | T009 | Low |

**Total**: 9 tasks across 3 phases
