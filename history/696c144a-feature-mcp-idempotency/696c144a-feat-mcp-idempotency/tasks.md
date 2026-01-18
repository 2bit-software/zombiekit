# Task List: MCP Command Idempotency

**Initiative:** 696c144a-feature-mcp-idempotency
**Complexity:** Simple (5 files, ~350 lines)
**Total Tasks:** 12

## Dependency Graph

```
Phase 1 (Parallel):
  T001 ──┬── T002 ──┬── T003
         │          │
Phase 2 (Tests):    │
  T004 ──┴── T005 ──┘
         │
Phase 3 (Integration):
  T006 → T007 → T008 → T009
                │
Phase 4 (Integration Tests):
  T010 → T011 → T012
```

## Tasks

### Phase 1: Independent Additions (Parallelizable)

- [ ] **T001** [P] [US-1] Add `FindActiveByNameAndType` method to `internal/initiative/service.go`
  - Location: After `GetActive()` method (around line 168)
  - Signature: `func (s *Service) FindActiveByNameAndType(name string, initType InitiativeType) (*Initiative, error)`
  - Logic: Call GetActive(), normalize name with existing normalizeName(), compare name+type
  - Returns nil if no match (not an error)
  - ~20 lines

- [ ] **T002** [P] [US-2] Add `copyTemplateIfNotExists` helper to `internal/mcp/tools/initiative/tool.go`
  - Location: Before `copyTemplatesToCycle()` method (around line 314)
  - Signature: `func copyTemplateIfNotExists(templateContent []byte, destPath string) (copied bool, err error)`
  - Logic: Check dest exists, read content, skip if non-whitespace content, else write
  - Add import: `bytes`
  - ~25 lines

- [ ] **T003** [P] [US-1,US-2] Add new fields to `CreateResponse` in `internal/mcp/tools/initiative/types.go`
  - Location: Inside `CreateResponse` struct (lines 16-26)
  - Add fields:
    ```go
    AlreadyExisted bool     `json:"already_existed"`
    SkippedFiles   []string `json:"skipped_files,omitempty"`
    CopiedFiles    []string `json:"copied_files,omitempty"`
    ```
  - ~3 lines

### Phase 2: Unit Tests (Parallel after Phase 1)

- [ ] **T004** [US-1] Add `TestService_FindActiveByNameAndType` to `internal/initiative/service_test.go`
  - Depends on: T001
  - Test cases:
    - No active initiative → returns nil
    - Active with different name → returns nil
    - Active with different type → returns nil
    - Active matching name+type → returns initiative
    - Name normalization ("User Auth" matches "user-auth")
  - ~80 lines

- [ ] **T005** [US-2] Create `internal/mcp/tools/initiative/tool_test.go` with `TestCopyTemplateIfNotExists`
  - Depends on: T002
  - Test cases:
    - File doesn't exist → copies, returns (true, nil)
    - File has content → skips, returns (false, nil)
    - File empty (0 bytes) → copies, returns (true, nil)
    - File whitespace-only → copies, returns (true, nil)
    - Write error → returns error
  - ~100 lines

### Phase 3: Integration Changes (Sequential)

- [ ] **T006** [US-1] Add `findFirstCycle` helper to `internal/mcp/tools/initiative/tool.go`
  - Depends on: T001, T002, T003
  - Location: Before `copyTemplatesToCycle()` method
  - Signature: `func (t *Tool) findFirstCycle(initiativePath string) (cyclePath, cycleID string)`
  - Logic: Scan for subdirectory with spec.md or research.md, return first alphabetically
  - Fallback to initiative path if no cycle found
  - ~25 lines

- [ ] **T007** [US-2] Modify `copyTemplatesToCycle` signature in `internal/mcp/tools/initiative/tool.go`
  - Depends on: T006
  - Change signature: `func (t *Tool) copyTemplatesToCycle(workDir, cyclePath string) (skipped, copied []string, err error)`
  - Replace `os.WriteFile` with `copyTemplateIfNotExists`
  - Track and return skipped/copied lists
  - ~15 lines changed

- [ ] **T008** [US-1,US-2] Modify `handleCreate` idempotency flow in `internal/mcp/tools/initiative/tool.go`
  - Depends on: T006, T007
  - Restructure flow:
    1. Call `FindActiveByNameAndType` first
    2. If match: call `findFirstCycle`, `copyTemplatesToCycle`, return success with `AlreadyExisted: true`
    3. If different active: return error `INITIATIVE_ALREADY_ACTIVE`
    4. If no active: create new (existing flow)
  - Update both success paths to include `SkippedFiles`/`CopiedFiles`
  - ~50 lines changed

- [ ] **T009** Run existing tests to verify no regression
  - Depends on: T008
  - Command: `go test ./internal/initiative/... ./internal/mcp/tools/initiative/...`
  - All existing tests must pass

### Phase 4: Integration Tests (Sequential)

- [ ] **T010** [US-1] Add `TestHandleCreate_Idempotent` to `internal/mcp/tools/initiative/tool_test.go`
  - Depends on: T009
  - Steps:
    1. Create initiative name="foo", type="feature"
    2. Write custom content to spec.md
    3. Call handleCreate same name+type
    4. Assert: AlreadyExisted=true, spec.md unchanged, SkippedFiles contains "spec.md"
  - ~50 lines

- [ ] **T011** [US-1] Add `TestHandleCreate_DifferentInitiativeActive` and `TestHandleCreate_SameNameDifferentType` to `internal/mcp/tools/initiative/tool_test.go`
  - Depends on: T010
  - Test 1: Create foo/feature, attempt create bar/feature → error INITIATIVE_ALREADY_ACTIVE
  - Test 2: Create foo/feature, attempt create foo/bug → error INITIATIVE_ALREADY_ACTIVE
  - ~60 lines

- [ ] **T012** [US-1] Add `TestHandleCreate_AfterComplete` to `internal/mcp/tools/initiative/tool_test.go`
  - Depends on: T011
  - Steps:
    1. Create initiative name="foo", type="feature"
    2. Complete the initiative
    3. Call handleCreate same name+type
    4. Assert: AlreadyExisted=false, new ID generated
  - ~40 lines

---

## Traceability Matrix

| User Story | Acceptance Criteria | Tasks |
|------------|---------------------|-------|
| US-1: Initiative Creation Protection | Detect existing name+type | T001, T004, T008, T010 |
| US-1 | Response indicates new/existing | T003, T008 |
| US-1 | Existing files untouched | T010 |
| US-2: Template Copy Protection | Check file existence | T002, T005, T007 |
| US-2 | Skip non-empty files | T002, T005 |
| US-2 | Overwrite empty files | T002, T005 |
| US-2 | Log skipped/copied | T003, T007, T008 |
| Success Metric 1 | Zero data loss | T010 |
| Success Metric 2 | Clear response | T003, T008 |
| Success Metric 3 | Backward compat | All (additive only) |

## Execution Summary

| Phase | Tasks | Parallelizable | Est. Lines |
|-------|-------|----------------|------------|
| 1 | T001, T002, T003 | Yes | ~50 |
| 2 | T004, T005 | Yes | ~180 |
| 3 | T006, T007, T008, T009 | No | ~90 |
| 4 | T010, T011, T012 | No | ~150 |

**Critical Path:** T001 → T004 → T006 → T007 → T008 → T009 → T010 → T011 → T012

**Next Command:** `/brains.implement` to begin implementation
