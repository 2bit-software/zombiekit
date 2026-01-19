# Technical Specification: Test Coverage Enforcement

## Summary

Three template/profile files are modified to enforce test coverage in the ZombieKit workflow.

## Change 1: spec-template.md

**Location**: `.brains/templates/spec-template.md`
**Insert after**: Line 116 (end of Success Criteria section)

### New Content to Add

```markdown
## Testing Requirements *(mandatory)*

<!--
  IMPORTANT: This section defines what tests are required for this feature.
  Every functional requirement (FR) must have at least one associated test.

  Testing Strategy (Integration-First):
  - Prefer integration tests at module boundaries
  - Use E2E tests for critical user journeys
  - Add unit tests only for complex pure functions with many edge cases
  - Test contracts/behavior, not implementation details

  If this feature requires NO tests (e.g., documentation-only change):
  - Replace this section with: "Testing Requirements: None - [justification]"
  - The justification must explain why no tests are needed
-->

### Test Strategy

[Describe the testing approach for this feature:
- What types of tests will be written?
- What test frameworks/tools will be used?
- Any special testing considerations?]

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | [What the test verifies] |
| FR-002 | E2E | [What the test verifies] |
| FR-003 | Unit | [What the test verifies - only if complex logic] |

### Edge Case Coverage

- [Edge case 1] → [How it will be tested]
- [Edge case 2] → [How it will be tested]
```

---

## Change 2: tasks-template.md

**Location**: `.brains/templates/tasks-template.md`

### Edit 1: Line 11

**Before**:
```markdown
**Tests**: The examples below include test tasks. Tests are OPTIONAL - only include them if explicitly requested in the feature specification.
```

**After**:
```markdown
**Tests**: The examples below include test tasks. Tests are REQUIRED unless the specification explicitly documents "Testing Requirements: None" with justification.
```

### Edit 2: Line 82

**Before**:
```markdown
### Tests for User Story 1 (OPTIONAL - only if tests requested) ⚠️
```

**After**:
```markdown
### Tests for User Story 1 ✓
```

### Edit 3: Line 108

**Before**:
```markdown
### Tests for User Story 2 (OPTIONAL - only if tests requested) ⚠️
```

**After**:
```markdown
### Tests for User Story 2 ✓
```

### Edit 4: Line 130

**Before**:
```markdown
### Tests for User Story 3 (OPTIONAL - only if tests requested) ⚠️
```

**After**:
```markdown
### Tests for User Story 3 ✓
```

---

## Change 3: audit.md

**Location**: `profiles/audit.md`

### Edit 1: Add Test Coverage Audit (after line 53, section 2d)

**Insert**:
```markdown

   e. **Test Coverage Audit**
      - Spec has Testing Requirements section (or explicit "None - [reason]" opt-out)
      - Every FR in spec has at least one test in Testing Requirements table
      - Edge cases have corresponding test coverage entries
      - Severity levels:
        - CRITICAL: No Testing Requirements section and no valid opt-out
        - MAJOR: Some FRs missing from test mapping table
        - MINOR: Edge cases listed but not mapped to tests
        - INFO: Legacy spec (pre-dates this feature) - warn but don't fail
```

### Edit 2: Update Issue Classification (line 56-59 area)

**Before**:
```markdown
3. **Issue Classification**
   - CRITICAL: Blocking, must fix before proceeding
   - MAJOR: Significant gap, should fix
   - MINOR: Nice to fix, low impact
   - INFO: Observation, no action needed
```

**After**:
```markdown
3. **Issue Classification**
   - CRITICAL: Blocking, must fix before proceeding (e.g., missing Testing Requirements)
   - MAJOR: Significant gap, should fix (e.g., FRs without test mappings)
   - MINOR: Nice to fix, low impact (e.g., edge cases without explicit test coverage)
   - INFO: Observation, no action needed (e.g., legacy spec warning)
```

---

## Verification Checklist

- [ ] spec-template.md has Testing Requirements section at end
- [ ] tasks-template.md line 11 says "REQUIRED" not "OPTIONAL"
- [ ] tasks-template.md test section headers don't say "OPTIONAL"
- [ ] audit.md includes Test Coverage Audit check
- [ ] audit.md issue classification includes test-related examples
