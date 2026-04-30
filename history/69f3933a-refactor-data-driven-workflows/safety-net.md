# Safety Net Assessment

## Risk Level: Medium

Multiple Go packages are affected, plus markdown files consumed by both Go code and LLMs. The step table format change requires backwards compatibility. Existing test coverage is moderate but has gaps in the exact areas we're changing.

## Existing Test Coverage

### Well-Covered (safe to change)

| Component | Test File | What's Tested |
|-----------|-----------|---------------|
| Initiative creation | `initiative/service_test.go` | Folder creation, state persistence, INITIATIVE.md generation |
| INITIATIVE.md parsing | `initiative/markdown_test.go` | Step table parsing, legacy format compat, step status tracking |
| Step loading | `step/loader_test.go` | Local/global precedence, deduplication, frontmatter parsing |
| Profile composition | `profile/service_test.go` | Atomic writes, composition, listing |
| Workflow loading | `workflow/service_test.go` | Resolution precedence, frontmatter parsing, shadowing |
| MCP initiative tool | `mcp/tools/initiative/tool_test.go` | Create, status, complete, abandon, template copying |
| MCP workflow tool | `mcp/tools/workflow/tool_test.go` | Load, parameter validation, error formatting |

### Gaps (need new tests)

| Gap | Why It Matters | Priority |
|-----|----------------|----------|
| `GetWorkflowSteps()` reading from profiles | This function changes its data source entirely | CRITICAL |
| 4-column step table parsing | New format, must coexist with 3-column | CRITICAL |
| `createInitiativeMD()` Profile column output | New format written by initiative creation | HIGH |
| Workflow frontmatter `steps:` array parsing | New frontmatter field | HIGH |
| End-to-end: create initiative → read step table → resolve profile | Integration path | HIGH |
| Profile files without `steps:` frontmatter | Verify graceful handling | MEDIUM |
| Backwards compat: old 3-column INITIATIVE.md still parses | Regression protection | CRITICAL |

## Recommended Tests to Add BEFORE Refactoring

### 1. Snapshot test for current behavior (regression baseline)

```go
// internal/initiative/markdown_test.go
func TestParseInitiativeMD_ThreeColumnFormat(t *testing.T) {
    // Freeze current 3-column parsing behavior as a regression test
    // This test MUST still pass after the refactor
}
```

### 2. `GetWorkflowSteps` integration test

```go
// internal/step/service_test.go
func TestGetWorkflowSteps_ReadsFromWorkflow(t *testing.T) {
    // After refactor: verify steps come from workflow file, not profile
}
```

### 3. INITIATIVE.md round-trip tests

```go
// internal/initiative/markdown_test.go
func TestParsedInitiative_RoundTrip_FourColumn(t *testing.T) {
    // Parse 4-column table → modify → write → re-parse → verify
}

func TestParsedInitiative_RoundTrip_ThreeColumn(t *testing.T) {
    // Parse 3-column table → modify → write → re-parse → verify (backwards compat)
}
```

## Exhaustive Test Plan

### Unit Tests (per function)

| Function | Test Cases |
|----------|-----------|
| `parseWorkflow()` | No frontmatter; name+description only; name+description+steps; malformed steps; empty steps array |
| `ParseInitiativeMD()` | 3-column table; 4-column table; mixed (shouldn't happen but defensive); empty table; no table section |
| `parseStepRow()` | 3-column row; 4-column row; header row (skip); separator row (skip); malformed |
| `formatSteps()` | Steps with profiles; steps without profiles (legacy); empty steps |
| `createInitiativeMD()` | With steps+profiles; empty steps; nil steps |
| `GetWorkflowSteps()` | Valid workflow; missing workflow; workflow without steps; invalid YAML |
| `loadWorkflowSteps()` | Success path; step service error; workflow not found (returns nil) |

### Integration Tests

| Scenario | Covers |
|----------|--------|
| Create feature initiative → verify INITIATIVE.md has 4-column table | End-to-end create flow |
| Create initiative → parse back → verify steps have profiles | Round-trip integrity |
| Load workflow → extract steps → create initiative → verify step names match | Workflow→Initiative pipeline |
| Parse old 3-column INITIATIVE.md → advance step → write back | Backwards compat under mutation |

### Edge Cases

| Edge Case | Expected Behavior |
|-----------|-------------------|
| Workflow with no `steps:` field | Return empty step list (not an error) |
| Profile still has `steps:` frontmatter (not yet cleaned) | Ignored — workflow is authoritative |
| INITIATIVE.md with 3-column table + new code | Parse succeeds, Profile field is empty string |
| Step name = profile name (e.g., "plan" → "plan") | Works, profile column is explicit anyway |
| Step name ≠ profile name (e.g., "spec" → "feature") | Profile column resolves correctly |
| Multiple profiles per step (e.g., `[implement, automode]`) | Stored as comma-separated or JSON in Profile column |

## Rollback Strategy

1. Each step is independently committable
2. Backwards-compatible 3-column parsing means old initiatives keep working
3. If something breaks: revert the workflow frontmatter changes, `GetWorkflowSteps()` change, and step table format change independently
4. Profile frontmatter removal is the LAST step — can be deferred indefinitely without breaking anything
