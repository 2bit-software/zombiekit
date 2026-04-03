# Tasks: Business-Requirement Function Comments

**Complexity**: Simple (4 files, ~50 lines added, no code changes)

## Task List

- [ ] T001 [P] [US1] Add "Function Comment Style" section to `embed/profiles/implement.md` after the "Behavior Rules" section. This is the single enforcement point. Include: business-language definition, litmus test, good/bad examples, exclusions for generated code, reference to existing test comment conventions, note about reusing spec verbiage, scope of "creates or updates", interface method declarations.

- [ ] T002 [P] [US1] Add "Function Comment Style" block to `.brains/templates/spec-template.md` inside the Testing Requirements HTML comment, after the "Test Comment Style" block (after line 138, before "If this feature requires NO tests"). Use indented format matching the surrounding HTML comment style.

- [ ] T003 [P] [US1] Add "Function Comment Style" section to `.brains/templates/tasks-template.md` after the "Test Comment Style" section (after line 266). Use `### Function Comment Style` heading matching the existing `### Test Comment Style` heading format.

- [ ] T004 [P] [US1] Add "Business-Language Framing" subsection to `STANDARDS.md` in the Documentation section (after line 318, before the `---` separator). Include litmus test and Go code examples.

## Dependency Graph

All tasks are independent — T001-T004 can execute in parallel.

## FR Traceability

| Task | FRs Covered |
|------|-------------|
| T001 | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008 |
| T002 | FR-006 |
| T003 | FR-006 |
| T004 | FR-002 |

## Notes

- [P] tasks = different files, no dependencies
- [US1] = User Story 1 (workflow-generated functions have business comments)
- All tasks are template/documentation edits — no Go code changes
- Canonical text block is in `technical-spec.md` for each file's specific content

### Test Comment Style

When writing tests, comments and section headers in code **MUST** use short
business-language descriptions of what the scenario verifies. Do NOT reference
task IDs (T010), story tags ([US1]), spec IDs (FR-001, BR-002), or any planning
artifact identifiers in test code comments.

- Good: `// Re-importing the same file skips already-stored messages`
- Good: `// Unicode and special characters survive the migration round-trip`
- Bad:  `// T009 [P] [US1]`
- Bad:  `// BR-002 (No duplicates)`

The task/story/FR mapping exists for planning traceability only and must not
leak into generated source code.

### Function Comment Style

When creating or modifying functions and methods, doc comments **MUST** describe
the outcome from the caller's perspective — not how it works internally. Litmus
test: would the comment still be true after a complete reimplementation?

- Good: `// ImportMessages brings external conversation history into the system.`
- Good: `// ResolveConflict picks the most recent version when two edits collide.`
- Bad: `// ImportMessages iterates over the input slice and calls db.Insert for each.`
- Bad: `// ResolveConflict compares timestamps and returns the newer struct.`

Reuse spec language where it fits naturally. Generated code and test code are
excluded (test code has its own rules above).
