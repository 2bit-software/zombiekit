# Research Summary: MCP Command Idempotency

## Codebase Analysis

### Current MCP Tool Inventory

| Tool | Package | Idempotency Status | Risk Level |
|------|---------|-------------------|------------|
| stickymemory | `stickymemory` | Fully idempotent (database upsert) | None |
| code-reasoning | `codereasoning` | Fully idempotent (in-memory) | None |
| profile-compose | `profile` | Read-only | None |
| profile-list | `profile` | Read-only | None |
| profile-write | `profile` | Safe (overwrite flag) | None |
| feature | `zombiekit` | Read-only | None |
| **initiative** | `initiative` | **NOT idempotent** | **HIGH** |
| step | `step` | Partial (state tracked, files regenerated) | Medium |

### High-Risk Code Paths

#### 1. Initiative Creation (`internal/mcp/tools/initiative/tool.go`)

**Current flow:**
```
handleCreate()
  → generateID() // Always creates new unique ID
  → Create directory structure
  → copyTemplatesToCycle() // Overwrites without checking
  → Save INITIATIVE.md
  → Set as active
```

**Problem:** No lookup for existing initiative with same name+type. Every call creates a new initiative.

**Location:** Lines 148-159 check for active initiative but not for existing matching initiative:
```go
existing, err := initSvc.GetActive()
if existing != nil {
    return error "INITIATIVE_ALREADY_ACTIVE"
}
```

#### 2. Template Copying (`internal/mcp/tools/initiative/tool.go`)

**Current flow:**
```
copyTemplatesToCycle()
  → List template files
  → For each file:
      → os.WriteFile(dst, content, 0644) // No existence check!
```

**Problem:** Unconditionally overwrites destination files.

**Location:** Lines 315-356, specifically line 350:
```go
if err := os.WriteFile(dst, content, 0644); err != nil {
```

### Existing Patterns to Leverage

#### Profile Write Pattern (`internal/profile/service.go:216-233`)

This is the reference implementation for safe writes:

1. Check if file exists
2. If exists and `overwrite=false`, return error
3. If overwrite allowed, use atomic write (temp + rename)

#### Initiative Service Methods (`internal/initiative/service.go`)

Existing methods:
- `GetActive()` - Returns active initiative
- `List()` - Lists all initiatives
- `GetByID()` - Retrieves by ID

**Missing:** `FindByNameAndType(name, typ string)` - Needed for idempotency check

### History Directory Structure

```
history/
  {id}-{type}-{name}/
    INITIATIVE.md           # Contains Name, Type, Status fields
    {id}-{phase}-{name}/    # Cycle folder
      spec.md
      research.md
      plan.md
      tasks.md
```

### Key Findings

1. **Initiative uniqueness is undefined** - No constraint on name+type combinations
2. **Template copying is destructive** - Always overwrites without warning
3. **Profile-write is the model** - Already implements safe write pattern
4. **Step tool is borderline** - Tracks state but regenerates files

### Recommendations

1. **Add `FindByNameAndType` method** to initiative service
2. **Modify `handleCreate`** to check for existing initiative first
3. **Modify `copyTemplatesToCycle`** to skip non-empty existing files
4. **Add response fields** to indicate create-vs-found and file copy details
5. **Consider step tool** for future idempotency improvements (lower priority)

### Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Add initiative lookup | Low | Unit test the lookup logic |
| Modify create flow | Medium | Integration test with real directory |
| Skip existing files | Medium | Log what was skipped for debugging |
| Add response fields | Low | Backward compatible (new optional fields) |

### Test Coverage Needed

1. Create initiative → Create same name+type → Verify returns existing
2. Create initiative → Complete it → Create same name+type → Verify creates new
3. Create initiative → Modify spec.md → Create same → Verify spec.md unchanged
4. Create initiative with empty file → Create same → Verify template copied
