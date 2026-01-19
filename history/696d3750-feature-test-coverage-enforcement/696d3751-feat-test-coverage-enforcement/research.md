---
status: complete
updated: 2026-01-19
---

# Research: Test Coverage Enforcement

## Executive Summary

ZombieKit currently treats testing as an optional bolt-on rather than a workflow requirement. Testing guidance is fragmented across three locations (spec-template, tasks-template, CLAUDE.md) with no enforcement mechanism. The opportunity is to make testing a first-class workflow concern while respecting the existing artifact-based architecture.

## Findings

### Codebase Context

**Current Testing Touchpoints:**

1. **Spec Template** (`spec-template.md`)
   - "User Scenarios & Testing" section exists
   - BDD acceptance scenarios (Given/When/Then)
   - "Independent Test" field per user story
   - No test strategy or coverage requirements

2. **Tasks Template** (`tasks-template.md`)
   - Tests explicitly marked OPTIONAL (line 11-12)
   - Sample test tasks exist (contract, integration, unit)
   - Test-first mentioned once (line 84): "Write tests FIRST, ensure they FAIL"
   - No mandatory testing gate

3. **CLAUDE.md** (lines 150-174)
   - Testing philosophy: E2E → Integration → Unit preference
   - "Test contracts, not implementation"
   - Never test: private methods, getters/setters, framework code
   - No coverage targets or workflow integration

**Current Workflow (no testing enforcement):**
```
Research → Create Spec → Plan → Tasks → Implement
                           ↓
                    Tests are optional
```

**Profile/Template Architecture:**
- Profiles are composable markdown with YAML frontmatter
- Resolution: project `.brains/profiles/` → git root → `~/.brains/profiles/` → embedded
- Templates define artifact structure
- Step tool executes workflow phases

### Domain Knowledge

**Industry Best Practices:**

1. **Test Pyramid** (Martin Fowler)
   - Many unit tests (fast, isolated)
   - Fewer integration tests (module boundaries)
   - Few E2E tests (user journeys)

2. **Contract Testing**
   - Test public interfaces, not internals
   - Enables refactoring without breaking tests

3. **Test-Driven Development (TDD)**
   - Write test first → watch fail → implement → watch pass
   - Forces testable design

4. **Behavior-Driven Development (BDD)**
   - Given/When/Then scenarios (already in spec template)
   - Maps directly to acceptance tests

**What "Comprehensive Testing" Means:**

- Every functional requirement has at least one test
- Every acceptance scenario maps to an automated test
- Edge cases have explicit test coverage
- Error paths are tested, not just happy paths

## Decision Points

- [x] **D1**: Where should testing requirements live?
  - **Decision**: In the spec template as a mandatory section, with enforcement in audit phase

- [x] **D2**: Should testing be mandatory or configurable?
  - **Decision**: Mandatory by default with explicit opt-out (documented in spec)

- [x] **D3**: How to enforce testing in workflow?
  - **Decision**: Audit phase checks FR→test mapping, tasks phase generates test tasks automatically

- [x] **D4**: What coverage strategy to recommend?
  - Options:
    - A) Test pyramid (unit-heavy)
    - B) Integration-first (CLAUDE.md preference) **SELECTED**
    - C) Acceptance-test-driven (BDD scenarios)
  - **Decision**: Integration-first - aligns with existing CLAUDE.md philosophy (E2E → Integration → Unit preference)

## Recommendations

1. **Add "Testing Requirements" section to spec-template.md**
   - Mandatory section after Success Criteria
   - Explicitly maps FRs to required tests
   - Defines test strategy for the feature

2. **Modify tasks-template.md to make tests non-optional**
   - Remove "OPTIONAL" designation
   - Generate test tasks from acceptance scenarios automatically
   - Test tasks precede implementation tasks (test-first)

3. **Add testing audit to audit.md profile**
   - Check: Every FR has at least one test task
   - Check: Every acceptance scenario maps to a test
   - Check: Edge cases have test coverage
   - Severity: CRITICAL if no tests, MAJOR if incomplete

4. **Update CLAUDE.md testing philosophy**
   - Clarify when each test type applies
   - Add guidance on acceptance test → integration test mapping
   - Reinforce test-first principle

5. **Create test-strategy profile (optional)**
   - Composable profile for test methodology guidance
   - Language-specific patterns (Go, Python, TypeScript)
   - Could be included by implement.md

## Sources

- ZombieKit codebase exploration (profiles/, .brains/templates/)
- DEV-77 Linear issue description
- CLAUDE.md testing philosophy section
- Martin Fowler's Test Pyramid concept
- BDD/TDD industry practices
