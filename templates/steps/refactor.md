---
name: refactor
description: Create a refactoring specification with behavior preservation focus
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
# Refactoring Specification Workflow

## Context

You are executing the refactoring workflow. Your goal is to restructure existing code while preserving its behavior. The key constraint is: **external behavior must not change**.

### Available Files
- `research.md` - Template for refactoring analysis
- `spec.md` - Template for refactoring specification
- `audit/` - Directory for audit reports

### Your Responsibilities
- Spawn research agents to analyze current structure
- Define clear before/after state
- Establish behavior verification strategy
- Create a refactoring specification
- Run audit checks for behavior preservation

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
3. **Parse `workflow_phases`**: Understand the 4-phase structure (analyze→define→specify→audit)
4. **Follow `directive`**: Execute according to this document
5. **Output to `cycle_folder`**: Save artifacts (research.md, spec.md, audit/) here
6. **Reference `composed_prompt`**: Additional context from research, create, audit profiles

### Understanding `workflow_phases`

The response includes phase definitions similar to feature workflow. Execute phases in order.

---

## Phase I: Analysis (Parallel Agents)

### Input
- Refactoring goal from user
- Target code areas
- Motivation (technical debt, performance, maintainability)

### Actions
1. Spawn analysis agents in parallel:
   - **analyze-structure**: Document current code structure
   - **analyze-dependencies**: Map dependencies and callers
   - **analyze-behavior**: Identify observable behaviors to preserve

2. Collate findings:
   - Current state documentation
   - Dependency graph
   - Behavior inventory

### Output
Populate `research.md` with:
- Current structure overview
- Dependency map
- Observable behaviors list
- Risk assessment

### Success Criteria
- [ ] Current structure documented
- [ ] All dependencies identified
- [ ] Observable behaviors catalogued
- [ ] Risks identified

---

## Phase II: Before/After Definition

### Input
- Analysis findings
- Refactoring goal

### Actions
1. Define clear BEFORE state:
   - Current structure
   - Current behavior contracts
   - Current test coverage

2. Define clear AFTER state:
   - Target structure
   - Same behavior contracts (must match!)
   - Required test additions

3. Identify invariants:
   - What MUST NOT change
   - What CAN change (internal details)

### Output
Add to `research.md`:
- BEFORE: {current state}
- AFTER: {target state}
- INVARIANTS: {behavior guarantees}
- VARIANCE ALLOWED: {what can change}

---

## Phase III: Refactoring Specification

### Input
- Before/After definitions
- Behavior inventory
- Risk assessment

### Actions
1. Document the refactoring approach
2. Define verification strategy
3. Identify migration path (if applicable)
4. Plan rollback strategy

### Output
Populate `spec.md` with:
- Refactoring scope
- Before state summary
- After state summary
- Behavior preservation criteria
- Verification strategy
- Migration path (if needed)
- Rollback plan

### Success Criteria
- [ ] Scope clearly bounded
- [ ] Before/After clearly defined
- [ ] Every behavior has a verification method
- [ ] Rollback is possible

---

## Phase IV: Audit & Highlight

### Actions
1. Verify behavior preservation strategy is complete
2. Check that no behaviors are lost
3. Confirm rollback capability
4. Present findings to user for approval

### User Approval Gate

Present to user:
```
## Refactoring Summary

**Goal**: {refactoring goal}
**Scope**: {affected areas}

### Current State (BEFORE)
{summary of current structure}

### Target State (AFTER)
{summary of target structure}

### Behavior Guarantees
These behaviors will be preserved:
1. {behavior 1}
2. {behavior 2}

### Verification Strategy
- {verification method 1}
- {verification method 2}

### Rollback Plan
{how to revert if issues arise}

---

**Ready to proceed with refactoring?**
- Approve to proceed to planning phase
- Reject with feedback to revise scope
```

---

## Behavior Rules

1. **Behavior First**: The goal is to change structure, NOT behavior
2. **Test Before Refactor**: Ensure adequate test coverage exists
3. **Small Steps**: Prefer multiple small refactors to one large change
4. **Verify Continuously**: Run tests after each refactoring step
5. **Rollback Ready**: Always be able to revert to working state
6. **Document Changes**: Update documentation to match new structure
7. **No Feature Creep**: This is refactoring, not feature development
