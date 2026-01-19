# Initiative: test-coverage-enforcement

**Type**: feature
**Status**: complete
**Created**: 2026-01-18T11:41:04-08:00
**Completed**: 2026-01-19
**ID**: 696d3750-feature-test-coverage-enforcement
**Linear Issue**: DEV-77

## Description

Add prompts and instructions to ZombieKit workflow to ensure comprehensive test coverage for every business requirement. Previously, tests were marked as "OPTIONAL" in the tasks template, leading to insufficient test coverage.

## Goals

- Make testing a first-class workflow concern
- Ensure every functional requirement has at least one associated test
- Add audit checks for test coverage validation
- Provide clear guidance on integration-first testing strategy

## Outcomes

### Files Modified

| File | Change |
|------|--------|
| `.brains/templates/spec-template.md` | Added mandatory "Testing Requirements" section |
| `.brains/templates/tasks-template.md` | Changed tests from OPTIONAL to REQUIRED |
| `profiles/audit.md` | Added Test Coverage Audit check |

### Features Delivered

1. **Spec Template Enhancement** (FR-001, FR-002, FR-009)
   - New "Testing Requirements" section with integration-first guidance
   - FR-to-test mapping table template
   - Explicit opt-out mechanism with justification

2. **Tasks Template Update** (FR-003, FR-004)
   - Tests now REQUIRED by default
   - Test section headers updated (removed OPTIONAL markers)

3. **Audit Enhancement** (FR-005, FR-006, FR-007, FR-008)
   - Test Coverage Audit added to alignment checks
   - CRITICAL severity for missing test requirements
   - MAJOR severity for incomplete FR coverage
   - Legacy spec backwards compatibility (INFO warning)

## Artifacts

```
history/696d3750-feature-test-coverage-enforcement/
  696d3751-feat-test-coverage-enforcement/
    spec.md          - Business specification
    research.md      - Codebase analysis
    plan.md          - Implementation plan
    technical-spec.md - Detailed change specifications
    tasks.md         - Task breakdown
    progress.md      - Implementation log
    audit/
      2026-01-19.md      - Spec audit report
      2026-01-19-plan.md - Plan audit report
```

## Notes

- All 6 implementation tasks completed successfully
- All FR acceptance criteria verified
- Integration-first testing strategy selected (user decision)
- No blockers encountered
