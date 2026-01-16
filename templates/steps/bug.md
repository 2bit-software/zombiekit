---
name: bug
description: Create a bug investigation and fix specification
profiles:
  - research
  - bug
  - audit
files:
  - "../INITIATIVE.md"
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

### System Responsibilities (handled by MCP tool)
- Folder creation
- Template copying
- Git branch management
- State updates

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load research.md, spec.md, and any previous cycle artifacts
3. **Parse `workflow_phases`**: Understand the 4-phase structure (investigate→classify→specify→audit)
4. **Follow `directive`**: Execute according to this document
5. **Output to `cycle_folder`**: Save artifacts (research.md, spec.md, audit/) here
6. **Reference `composed_prompt`**: Additional context from research, create, audit profiles

### Understanding `workflow_phases`

The response includes phase definitions similar to feature workflow. Execute phases in order.

---

## Phase 0: Initialize Initiative

### Input
- User's bug description
- `INITIATIVE.md` template in initiative folder

### Actions
1. Read the user's bug description from the input
2. Open `INITIATIVE.md` in the initiative folder (parent of cycle folder)
3. Fill in the Description section with a summary of the bug being investigated
4. Fill in the Goals section with investigation and fix goals

### Output
Update `INITIATIVE.md` with:
- Description: Clear summary of the bug (symptoms, affected area)
- Goals: Investigation goals (find root cause, determine fix approach, verify fix)

### Success Criteria
- [ ] INITIATIVE.md Description section is filled (no placeholder comments)
- [ ] INITIATIVE.md Goals section has at least 2 goals
- [ ] Goals reflect bug investigation objectives

**IMPORTANT**: Complete this phase BEFORE starting investigation. The initiative context informs all subsequent phases.

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
**CRITICAL**: Write all investigation findings to `research.md` in the cycle folder. Replace placeholder content with actual findings.

Populate `research.md` with:
- Bug summary (what's happening)
- Root cause analysis
- Reproduction steps
- Impact assessment
- Related code/files

### Success Criteria
- [ ] research.md has been written (not just template placeholders)
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
