# Tasks: Enhanced Initiative Lifecycle with Linear Integration

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 8 |
| Parallelizable | 6 (within feature groups) |
| Files affected | 6 |
| Complexity | Simple |

## Dependency Graph

```
F1: Ticket Capture
  T001 (new.md) → T002, T003, T004 (profiles can run in parallel after T001)

F2: Commit Offer
  T005 (complete.md) ──┐
                       ├── T007 (complete profile, after both)
F3: Linear Update      │
  T006 (complete.md) ──┘

T008: Test all features (after all)
```

## Critical Path

```
T001 → T002 → T005 → T006 → T007 → T008
```

---

## Feature 1: Ticket Capture on Creation

- [ ] T001 [US1] Add ticket detection section to `embed/workflows/new.md`
  - Parse user input for `[A-Z]+-[0-9]+` pattern after classification
  - Fetch ticket via `mcp__linear-server__get_issue` if found
  - Append LINEAR_TICKET/LINEAR_URL/LINEAR_TITLE to arguments
  - Handle errors gracefully (skip on failure)

- [ ] T002 [P] [US1] Add Source section step to `embed/profiles/feature.md`
  - Check arguments for LINEAR_TICKET metadata
  - Use Edit tool to insert Source section before "## Description"
  - Include ticket ID, URL, and title

- [ ] T003 [P] [US1] Add Source section step to `embed/profiles/bug.md`
  - Same implementation as T002

- [ ] T004 [P] [US1] Add Source section step to `embed/profiles/refactor.md`
  - Same implementation as T002

---

## Feature 2: Commit Offer on Completion

- [ ] T005 [US2] Add commit offer step to `embed/workflows/complete.md`
  - Insert new step 5 after "Update INITIATIVE.md"
  - Run `git status --porcelain` to detect changes
  - Use AskUserQuestion to offer commit
  - On accept: `git add -A` then invoke `commit-message` skill
  - Handle errors, proceed on failure

---

## Feature 3: Linear Ticket Update on Completion

- [ ] T006 [US3] Add Linear update step to `embed/workflows/complete.md`
  - Insert new step 6 after commit offer
  - Read INITIATIVE.md for Source section
  - Fallback: parse initiative name for ticket pattern
  - Use AskUserQuestion to offer update
  - On accept: post comment and update status to Done
  - Handle errors, proceed on failure

---

## Finalization

- [ ] T007 Update `embed/profiles/complete.md` to match workflow
  - Sync profile content with updated complete.md workflow
  - Ensure steps 5-6 (commit + Linear) are documented

- [ ] T008 Test all features end-to-end
  - Test F1: `/brains.new "work on DEV-XXX"` creates Source section
  - Test F1: `/brains.new "add feature"` has no Source section
  - Test F2: `/brains.complete` with uncommitted changes shows offer
  - Test F3: `/brains.complete` with Source section shows Linear offer

---

## Execution Order

**Recommended sequence:**

1. T001 (new.md ticket detection) - foundation for F1
2. T002, T003, T004 (profiles) - can run in parallel
3. T005 (commit offer) - F2
4. T006 (Linear update) - F3
5. T007 (sync complete profile)
6. T008 (testing)

**Parallel opportunities:**
- T002 + T003 + T004 (all profile changes)
- T005 + T006 could be combined (same file, adjacent sections)
