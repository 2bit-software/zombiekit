# Tasks: Test Coverage Enforcement

**Input**: `plan.md`, `technical-spec.md`, `spec.md`
**Complexity**: Simple (3 files, ~50 lines of changes)
**Linear Issue**: DEV-77

---

## Phase 1: Template & Profile Updates

**Purpose**: Implement all template and profile changes

### Spec Template (FR-001, FR-002, FR-009)

- [ ] T001 [P] [US1] Add Testing Requirements section to `.brains/templates/spec-template.md`
  - Insert after line 116 (end of Success Criteria section)
  - Include integration-first testing guidance in HTML comment
  - Include Test Strategy subsection with placeholder
  - Include FR to Test Mapping table template
  - Include Edge Case Coverage subsection
  - Include opt-out instructions ("Testing Requirements: None - [justification]")
  - **Acceptance**: Template ends with Testing Requirements section containing all subsections

### Tasks Template (FR-003, FR-004)

- [ ] T002 [P] [US2] Update test requirement language in `.brains/templates/tasks-template.md` line 11
  - Change "OPTIONAL" to "REQUIRED unless specification explicitly documents 'Testing Requirements: None'"
  - **Acceptance**: Line 11 contains "REQUIRED" and references opt-out mechanism

- [ ] T003 [P] [US2] Update test section headers in `.brains/templates/tasks-template.md`
  - Line 82: Change "### Tests for User Story 1 (OPTIONAL - only if tests requested) ⚠️" to "### Tests for User Story 1 ✓"
  - Line 108: Change "### Tests for User Story 2 (OPTIONAL - only if tests requested) ⚠️" to "### Tests for User Story 2 ✓"
  - Line 130: Change "### Tests for User Story 3 (OPTIONAL - only if tests requested) ⚠️" to "### Tests for User Story 3 ✓"
  - **Acceptance**: No "OPTIONAL" text in test section headers; all use ✓ marker

### Audit Profile (FR-005, FR-006, FR-007, FR-008)

- [ ] T004 [P] [US3] Add Test Coverage Audit check to `profiles/audit.md`
  - Insert after line 53 (section 2d AI-Friendliness Audit)
  - Add section "e. Test Coverage Audit" with:
    - Check for Testing Requirements section or opt-out
    - Check for FR coverage in test mapping table
    - Check for edge case coverage
    - Severity levels: CRITICAL (missing), MAJOR (incomplete), MINOR (edge cases), INFO (legacy)
  - **Acceptance**: audit.md contains Test Coverage Audit section with all severity levels defined

- [ ] T005 [P] [US3] Update Issue Classification examples in `profiles/audit.md`
  - Add "(e.g., missing Testing Requirements)" after CRITICAL definition
  - Add "(e.g., FRs without test mappings)" after MAJOR definition
  - Add "(e.g., edge cases without explicit test coverage)" after MINOR definition
  - Add "(e.g., legacy spec warning)" after INFO definition
  - **Acceptance**: Issue Classification section includes test-related examples for each severity

---

## Phase 2: Verification

**Purpose**: Validate all changes work correctly

- [ ] T006 [US1,US2,US3] Manual verification of template changes
  - Verify spec-template.md ends with Testing Requirements section
  - Verify tasks-template.md contains "REQUIRED" not "OPTIONAL" on line 11
  - Verify tasks-template.md test headers use ✓ not "(OPTIONAL)"
  - Verify audit.md contains Test Coverage Audit check
  - Verify audit.md Issue Classification has examples
  - **Acceptance**: All 5 verification items pass

---

## Dependencies & Execution Order

### Parallel Opportunities

All Phase 1 tasks (T001-T005) are independent and can run in parallel:
- T001 modifies spec-template.md
- T002, T003 modify tasks-template.md (different sections, can be combined)
- T004, T005 modify audit.md (different sections, can be combined)

### Execution Graph

```
T001 ──┐
T002 ──┼──> T006 (verification)
T003 ──┤
T004 ──┤
T005 ──┘
```

### Suggested Execution

**Option A (Sequential)**:
1. T001 → T002 → T003 → T004 → T005 → T006

**Option B (Parallel)**:
1. T001 + T002 + T003 + T004 + T005 (all in parallel)
2. T006 (after all complete)

---

## Traceability Matrix

| Task | Plan Step | Spec FR | User Story |
|------|-----------|---------|------------|
| T001 | Step 1 | FR-001, FR-002, FR-009 | US1 |
| T002 | Step 2 | FR-003 | US2 |
| T003 | Step 2 | FR-003, FR-004 | US2 |
| T004 | Step 3 | FR-005, FR-006, FR-007, FR-008 | US3 |
| T005 | Step 3 | FR-007, FR-008 | US3 |
| T006 | Step 4 | All FRs | All |

---

## Summary

- **Total Tasks**: 6
- **Parallel Tasks**: 5 (T001-T005)
- **Critical Path**: Any single task → T006
- **Estimated Effort**: ~30 minutes total
