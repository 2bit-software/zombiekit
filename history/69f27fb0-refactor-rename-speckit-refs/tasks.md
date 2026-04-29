# Tasks: rename-speckit-refs

**Input**: `refactor-plan.md`
**Complexity**: Simple (4 files, 14 occurrences, zero cross-module deps)

## Phase 1: Template Files (Parallel)

- [ ] T001 [P] Replace 7 speckit references in `embed/templates/plan-template.md`
  - Line 6: `/speckit.plan` + `.specify/` path → brains workflow reference
  - Lines 42-47: `/speckit.plan command output` and `/speckit.tasks command` → brains workflow references
- [ ] T002 [P] Replace 2 speckit references in `embed/templates/checklist-template.md`
  - Line 7: `/speckit.checklist` → brains workflow (checklist step)
  - Line 13: `/speckit.checklist command` → brains workflow
- [ ] T003 [P] Replace 1 speckit reference in `embed/templates/tasks-template.md`
  - Line 36: `/speckit.tasks command` → brains workflow

## Phase 2: Profile File

- [ ] T004 [P] Replace 4 speckit references in `embed/profiles/commit-message.md`
  - Line 39: `speckit` → `zombiekit`
  - Line 79: `speckit` → `zombiekit/brains`
  - Line 80: tooling format example — remove `speckit,` from example
  - Line 92: `speckit` → `brains`

## Phase 3: Verification

- [ ] T005 Run `go build ./...` to confirm embed integrity
- [ ] T006 Grep `embed/` for remaining speckit/spec-kit/.specify references (expect zero)

## Dependencies

- T001-T004: All parallel, no dependencies
- T005-T006: Depend on T001-T004 completion

## Execution Order

All implementation tasks (T001-T004) can run in parallel, followed by verification (T005-T006).
