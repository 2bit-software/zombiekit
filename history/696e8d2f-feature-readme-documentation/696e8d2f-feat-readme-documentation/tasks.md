---
status: draft
type: tasks
name: readme-documentation
created: 2026-01-19
plan: plan.md
spec: spec.md
---

# Task List: README Documentation

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 2 |
| Parallelizable | 0 |
| Complexity | Simple |
| Estimated files | 1 |

## Dependency Graph

```
T001 ──────> T002
(write)      (validate)
```

No parallel opportunities - T002 depends on T001 completion.

---

## Tasks

### Phase 1: Create README

- [ ] T001 [US1,US2,US3,US4] Create README.md at repository root with all 7 sections
  - **File:** `README.md`
  - **Sections to include:**
    1. Header (project name, tagline, one-sentence description)
    2. What Is ZombieKit? (3 paragraphs: core insight, capabilities, integration)
    3. Quick Start (prerequisites with version checks, clone/install, MCP config JSON, init, verification)
    4. The Workflow Cycle (ASCII-7 diagram, 4-line explanation)
    5. Core Skills (9 skills in markdown table, grouped by purpose)
    6. Example Workflow (terminal vs Claude Code distinction, numbered steps)
    7. Learn More (links to DESIGN.md and .claude/commands/)
  - **Acceptance criteria:**
    - [ ] File exists at `/README.md`
    - [ ] Contains all 7 sections in order
    - [ ] Under 200 lines
    - [ ] No badges
    - [ ] ASCII-7 only for diagrams
    - [ ] Git URL: `https://github.com/morganhein/zombiekit.git`
    - [ ] Skills table has exactly 9 core skills

---

### Phase 2: Validation

- [ ] T002 [FR-010,SC-001,SC-002,SC-003,SC-004] Validate README meets all requirements
  - **Commands:**
    - `wc -l README.md` (must be < 200)
    - Validate MCP config JSON parses correctly
    - Verify DESIGN.md exists
    - Verify .claude/commands/ exists
    - Count sections (7 expected)
    - Count skills in table (9 expected)
  - **Acceptance criteria:**
    - [ ] Line count < 200
    - [ ] MCP JSON block is valid
    - [ ] All referenced files exist
    - [ ] All 7 sections present
    - [ ] Exactly 9 skills in table

---

## Requirement Traceability

| Task | Requirements Covered |
|------|---------------------|
| T001 | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011 |
| T002 | SC-001, SC-002, SC-003, SC-004 |

| User Story | Task |
|------------|------|
| US1 (Quick Start) | T001 |
| US2 (Capabilities) | T001 |
| US3 (Workflow) | T001 |
| US4 (Skill Reference) | T001 |

---

## Suggested Execution Order

1. **T001** - Create the README.md file with all content
2. **T002** - Validate all requirements are met

---

## Next Step

After task approval: `/brains.implement`
