---
name: bug
description: Create a bug investigation and fix specification
profiles:
  - research
  - create
  - audit
files:
  - "research.md"
  - "spec.md"
  - "audit/**/*.md"
  - "../**/research.md"
  - "../**/spec.md"
type: step
---
# Bug Investigation Workflow

## Context

You are executing the bug investigation workflow. Your goal is to investigate a reported bug, determine its root cause, and create a fix specification.

### Available Files
- `research.md` - Template for bug investigation findings
- `spec.md` - Template for bug fix specification
- `audit/` - Directory for audit reports

### Your Responsibilities
- Spawn research agents to investigate the bug
- Determine if this is a spec gap or implementation error
- Create a fix specification with reproduction steps
- Run audit checks and address critical issues
- Present findings for user approval

---

## Phase I: Investigation (Parallel Agents)

### Input
- Bug description from user
- Error messages, logs, or symptoms
- Steps to reproduce (if known)

### Actions
1. Spawn investigation agents in parallel:
   - **investigate-codebase**: Find related code, trace execution path
   - **investigate-history**: Check recent changes, related commits
   - **investigate-dependencies**: Check external factors, versions

2. Collate findings:
   - Identify root cause
   - Document reproduction steps
   - Note affected areas

### Output
Populate `research.md` with:
- Bug summary (what's happening)
- Root cause analysis
- Reproduction steps
- Impact assessment
- Related code/files

### Success Criteria
- [ ] Root cause identified or hypotheses documented
- [ ] Reproduction steps clear and testable
- [ ] Impact scope defined

---

## Phase II: Classification

### Determine Bug Type

```
IF root cause is missing/unclear requirement:
    → This is a SPEC GAP
    → Specification needs to be updated first
    → Fix implements the clarified spec
ELSE IF root cause is implementation error:
    → This is an IMPLEMENTATION BUG
    → Existing spec is correct
    → Fix corrects the deviation
```

### Output
Add classification to `research.md`:
- Bug type: SPEC_GAP or IMPLEMENTATION_BUG
- Rationale for classification

---

## Phase III: Fix Specification

### Input
- Investigation findings
- Bug classification

### Actions
1. Document the expected behavior
2. Define acceptance criteria for the fix
3. Outline the fix approach (not implementation details)
4. Identify regression test requirements

### Output
Populate `spec.md` with:
- Problem statement
- Expected behavior
- Acceptance criteria
- Regression test requirements
- Out of scope items

### Success Criteria
- [ ] Problem clearly stated
- [ ] Expected behavior is unambiguous
- [ ] Acceptance criteria are testable
- [ ] Regression test approach defined

---

## Phase IV: Audit & Highlight

### Actions
1. Verify fix specification is complete
2. Check that fix doesn't introduce new issues
3. Present findings to user for approval

### User Approval Gate

Present to user:
```
## Bug Fix Summary

**Bug**: {description}
**Type**: {SPEC_GAP | IMPLEMENTATION_BUG}
**Root Cause**: {brief description}

### Proposed Fix
{summary of fix approach}

### Acceptance Criteria
1. {criterion 1}
2. {criterion 2}

### Regression Tests
- {test 1}
- {test 2}

---

**Ready to proceed with fix?**
- Approve to proceed to planning phase
- Reject with feedback to revise investigation
```

---

## Behavior Rules

1. **Reproduction First**: Don't propose fixes without understanding the issue
2. **Classify Before Fixing**: Know if this is a spec gap or implementation bug
3. **Minimal Fix**: Fix the bug, don't refactor surrounding code
4. **Regression Coverage**: Every bug fix needs a test that would have caught it
5. **Document Thoroughly**: Future developers should understand what went wrong
