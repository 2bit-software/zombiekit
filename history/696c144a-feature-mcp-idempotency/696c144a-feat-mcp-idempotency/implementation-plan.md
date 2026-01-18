# Implementation Plan: MCP Command Idempotency

## Overview

This plan implements idempotency for the `initiative create` MCP command, preventing data loss when the command is called multiple times with the same name and type.

## Requirements Traceability

| Requirement | Implementation Task |
|-------------|---------------------|
| US-1: Initiative Creation Protection | Tasks 1, 3, 5 |
| US-2: Template Copy Protection | Tasks 2, 4 |
| Success Metric 1: Zero data loss | Task 6 (integration test) |
| Success Metric 2: Clear response messaging | Tasks 3, 5 |
| Success Metric 3: Backward compatibility | All tasks (additive changes only) |

## Implementation Tasks

### Task 1: Add `FindActiveByNameAndType` to Initiative Service

**File:** `internal/initiative/service.go`

**Location:** After `GetActive()` method (around line 168)

**Change:** Add new method to check if active initiative matches given name+type

**Logic:**
1. Call `GetActive()` to get active initiative
2. If no active initiative, return nil
3. Normalize the input name using existing `normalizeName()` function
4. Compare normalized name and type with active initiative
5. Return initiative if matches, nil otherwise

**Estimated lines:** ~20

---

### Task 2: Add `copyTemplateIfNotExists` Helper

**File:** `internal/mcp/tools/initiative/tool.go`

**Location:** Before `copyTemplatesToCycle()` method (around line 314)

**Change:** Add new helper function for safe template copying

**Logic:**
1. Check if destination file exists using `os.Stat`
2. If exists, read content with `os.ReadFile`
3. If content has non-whitespace (`len(bytes.TrimSpace(content)) > 0`), return (false, nil)
4. Otherwise, read template and write to destination
5. Return (true, nil) on successful copy

**New import:** `bytes`

**Estimated lines:** ~25

---

### Task 3: Modify `handleCreate` for Idempotency Check

**File:** `internal/mcp/tools/initiative/tool.go`

**Location:** In `handleCreate()` method, after line 159 (after current active check)

**Change:** Add idempotency check before creating new initiative

**Sub-tasks:**
1. Add `findFirstCycle` helper method to locate cycle folder within an initiative
2. Restructure handleCreate to:
   - First check if active initiative matches name+type (idempotent case)
   - Then check if different initiative is active (error case)
   - Finally create new if no active initiative

**Logic:**
1. Call `FindActiveByNameAndType` to check for idempotent case
2. If active initiative matches name+type:
   - Get existing cycle path using `findFirstCycle` helper
   - Call modified `copyTemplatesToCycle` to get skipped/copied files
   - Build response with `AlreadyExisted: true`
   - Return success response (not error)
3. If no match but initiative is active, return error "INITIATIVE_ALREADY_ACTIVE"
4. If no active, proceed with normal creation flow

**`findFirstCycle` behavior:**
- Scans initiative directory for subdirectories
- Returns first directory containing spec.md or research.md
- Falls back to initiative path if no cycle found
- Selects first cycle alphabetically when multiple exist

**Key insight:** Current code returns error for "INITIATIVE_ALREADY_ACTIVE". New code should:
- Return **success** with `AlreadyExisted: true` when name+type matches (idempotent)
- Keep error behavior when name+type differs (different initiative is active)

**Estimated lines changed:** ~50 (including findFirstCycle helper)

---

### Task 4: Modify `copyTemplatesToCycle` for Safe Copying

**File:** `internal/mcp/tools/initiative/tool.go`

**Location:** `copyTemplatesToCycle()` method (lines 315-356)

**Change:** Return lists of skipped and copied files

**New signature:**
```go
func (t *Tool) copyTemplatesToCycle(workDir, cyclePath string) (skipped, copied []string, err error)
```

**Logic:**
1. For each template, call `copyTemplateIfNotExists` instead of `os.WriteFile`
2. Track which files were skipped vs copied
3. Return both lists

**Callers to update:**
- `handleCreate` (line 182) - handle new return values

**Estimated lines changed:** ~15

---

### Task 5: Update Response Types

**File:** `internal/mcp/tools/initiative/types.go`

**Location:** `CreateResponse` struct (lines 16-26)

**Change:** Add new optional fields

```go
type CreateResponse struct {
    // ... existing fields ...
    AlreadyExisted bool     `json:"already_existed"`
    SkippedFiles   []string `json:"skipped_files,omitempty"`
    CopiedFiles    []string `json:"copied_files,omitempty"`
}
```

**Estimated lines added:** 3

---

### Task 6: Add Unit Tests

**File:** `internal/initiative/service_test.go`

**Tests to add:**

```go
func TestService_FindActiveByNameAndType(t *testing.T) {
    // Test cases:
    // - No active initiative -> returns nil
    // - Active initiative with different name -> returns nil
    // - Active initiative with different type -> returns nil
    // - Active initiative matching name+type -> returns initiative
    // - Name normalization works (e.g., "User Auth" matches "user-auth")
}
```

**Estimated lines:** ~80

---

### Task 7: Add Helper Function Tests

**File:** `internal/mcp/tools/initiative/tool_test.go` (create new file)

**Tests to add:**

```go
func TestCopyTemplateIfNotExists(t *testing.T) {
    // Test cases:
    // - File doesn't exist -> copies template, returns (true, nil)
    // - File exists with content -> skips, returns (false, nil)
    // - File exists but empty (0 bytes) -> copies, returns (true, nil)
    // - File exists with only whitespace -> copies, returns (true, nil)
    // - Template source missing -> returns error
}
```

**Estimated lines:** ~100

---

### Task 8: Add Integration Tests

**File:** `internal/mcp/tools/initiative/tool_test.go`

**Tests to add:**

```go
func TestHandleCreate_Idempotent(t *testing.T) {
    // 1. Create initiative with name="foo", type="feature"
    // 2. Write custom content to spec.md
    // 3. Call handleCreate again with same name+type
    // 4. Verify: returns existing (AlreadyExisted=true)
    // 5. Verify: spec.md content unchanged
    // 6. Verify: SkippedFiles includes "spec.md"
}

func TestHandleCreate_DifferentInitiativeActive(t *testing.T) {
    // 1. Create initiative with name="foo", type="feature"
    // 2. Call handleCreate with name="bar", type="feature"
    // 3. Verify: returns error INITIATIVE_ALREADY_ACTIVE (existing behavior preserved)
}

func TestHandleCreate_AfterComplete(t *testing.T) {
    // 1. Create initiative with name="foo", type="feature"
    // 2. Complete the initiative
    // 3. Call handleCreate with same name+type
    // 4. Verify: creates new initiative (AlreadyExisted=false)
    // Note: This covers edge case "Initiative exists but NOT in active state"
}

func TestHandleCreate_SameNameDifferentType(t *testing.T) {
    // 1. Create initiative with name="foo", type="feature"
    // 2. Call handleCreate with name="foo", type="bug"
    // 3. Verify: returns error INITIATIVE_ALREADY_ACTIVE (not idempotent - type differs)
}
```

**Estimated lines:** ~180

---

## Implementation Order

```
Task 1 (FindActiveByNameAndType)
    ↓
Task 6 (Unit tests for Task 1) ← Run tests, verify green
    ↓
Task 2 (copyTemplateIfNotExists)
    ↓
Task 7 (Unit tests for Task 2) ← Run tests, verify green
    ↓
Task 5 (Update response types) ← No tests needed, trivial
    ↓
Task 4 (Modify copyTemplatesToCycle)
    ↓
Task 3 (Modify handleCreate) ← Most complex, depends on 1,2,4,5
    ↓
Task 8 (Integration tests) ← Run tests, verify all green
```

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing `handleCreate` callers | Response changes are additive only; existing fields unchanged |
| Incorrect idempotency detection | Use same `normalizeName` function as Create uses |
| Template detection edge cases | Comprehensive tests for empty, whitespace, content cases |
| Finding cycle path for existing initiative | Parse initiative directory for first matching cycle folder |

## Files Modified Summary

| File | Type of Change |
|------|----------------|
| `internal/initiative/service.go` | Add method |
| `internal/initiative/service_test.go` | Add tests |
| `internal/mcp/tools/initiative/tool.go` | Add helper, modify methods |
| `internal/mcp/tools/initiative/tool_test.go` | Create new test file |
| `internal/mcp/tools/initiative/types.go` | Add fields |

## Definition of Done

- [ ] All new tests pass
- [ ] Existing tests continue to pass
- [ ] Creating initiative with same name+type returns existing (AlreadyExisted=true)
- [ ] Creating initiative with different name/type while another is active returns error (existing behavior)
- [ ] Template files with content are not overwritten
- [ ] Empty/whitespace-only files are overwritten with templates
- [ ] Response includes SkippedFiles/CopiedFiles when applicable
