# Implementation Plan: Test Coverage Enforcement

**Feature Branch**: `feat/test-coverage-enforcement`
**Created**: 2026-01-19
**Status**: Draft
**Linear Issue**: DEV-77

## Overview

This feature enforces comprehensive test coverage through changes to three artifacts:
1. **spec-template.md** - Add mandatory Testing Requirements section
2. **tasks-template.md** - Make test tasks mandatory and test-first
3. **audit.md** - Add test coverage validation

No executable code changes. All changes are to markdown/YAML templates and profiles.

## Implementation Strategy

**Approach**: Integration-first testing strategy (selected per D4)

Changes are ordered to maintain backwards compatibility:
1. Add new sections/checks (additive)
2. Modify existing content (non-breaking)
3. No removal of existing functionality

## Step 1: Update Spec Template (FR-001, FR-002, FR-009)

**File**: `.brains/templates/spec-template.md`

**Changes**:
1. Add "Testing Requirements" section after "Success Criteria" section
2. Include guidance on FR-to-test mapping with table template
3. Include test strategy guidance (integration-first philosophy)
4. Include opt-out mechanism with required justification

**New Section Content**:
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

**Verification**: Template includes new section with guidance

---

## Step 2: Update Tasks Template (FR-003, FR-004)

**File**: `.brains/templates/tasks-template.md`

**Changes**:
1. Remove "OPTIONAL" designation from test tasks (line 11)
2. Update test section headers to remove "(OPTIONAL)" suffix
3. Add clear instruction that tests are required unless explicitly opted out in spec
4. Ensure test tasks precede implementation tasks (already the case - verify)

**Specific Edits**:

Line 11 change:
```diff
- **Tests**: The examples below include test tasks. Tests are OPTIONAL - only include them if explicitly requested in the feature specification.
+ **Tests**: The examples below include test tasks. Tests are REQUIRED unless the specification explicitly documents "Testing Requirements: None" with justification.
```

Section header changes (lines 82, 108, 130):
```diff
- ### Tests for User Story 1 (OPTIONAL - only if tests requested) ⚠️
+ ### Tests for User Story 1 ✓
```

**Verification**:
- "OPTIONAL" no longer appears in test-related content
- Test sections clearly indicate tests are expected

---

## Step 3: Update Audit Profile (FR-005, FR-006, FR-007, FR-008)

**File**: `profiles/audit.md`

**Changes**:
1. Add "Test Coverage Audit" to the Alignment Checks section (section 2)
2. Define what constitutes CRITICAL vs MAJOR test coverage issues
3. Add guidance for backwards compatibility with legacy specs

**New Audit Check (add to section 2)**:
```markdown
   e. **Test Coverage Audit**
      - Spec has Testing Requirements section (or explicit opt-out)
      - Every FR in spec has at least one test in Testing Requirements
      - Edge cases have corresponding test coverage
      - CRITICAL: No Testing Requirements section and no opt-out
      - MAJOR: Some FRs missing test mappings
      - MINOR: Edge cases not fully covered
      - INFO: Legacy spec (created before this feature) - warn only
```

**Update Issue Classification (section 3)** to add examples:
```markdown
   - CRITICAL: Missing Testing Requirements (no tests defined)
   - MAJOR: Incomplete test coverage (some FRs untested)
```

**Verification**: Audit profile includes test coverage checks with correct severity levels

---

## Step 4: Verify Changes (All FRs)

**Acceptance Testing**:
1. Create a new spec using the template - verify Testing Requirements section appears
2. Generate tasks from a spec - verify test tasks are not marked optional
3. Run audit on spec without Testing Requirements - verify CRITICAL reported
4. Run audit on spec with incomplete FR mapping - verify MAJOR reported
5. Run audit on spec with complete coverage - verify no test-related issues

---

## Dependencies

```
Step 1 (spec-template) → independent
Step 2 (tasks-template) → independent
Step 3 (audit profile) → independent
Step 4 (verification) → depends on Steps 1-3
```

Steps 1-3 can be done in parallel; Step 4 requires all previous steps.

## Files Modified

| File | Type | FR Coverage |
|------|------|-------------|
| `.brains/templates/spec-template.md` | Template | FR-001, FR-002, FR-009 |
| `.brains/templates/tasks-template.md` | Template | FR-003, FR-004 |
| `profiles/audit.md` | Profile | FR-005, FR-006, FR-007, FR-008 |

## Risk Assessment

**Low Risk**: All changes are additive or modify prompts/guidance text. No logic changes, no breaking changes to existing functionality.

**Backwards Compatibility**:
- Existing specs without Testing Requirements will get INFO-level warning (not failure)
- Existing task generation continues to work

## No Spikes Required

This implementation involves only markdown/YAML template editing. No uncertain technical areas, no external APIs, no performance concerns. Spikes are not needed.
