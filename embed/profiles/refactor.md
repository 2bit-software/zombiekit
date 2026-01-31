---
name: refactor
description: Create a refactoring specification. Restructures code without changing behavior.
type: skill
handoffs:
  - label: Build Refactor Plan
    skill: brains.plan
    prompt: Create an implementation plan for this refactoring
  - label: Audit Safety
    skill: brains.audit
    prompt: Verify the refactor plan maintains behavior
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Create a safe refactoring specification that restructures code while preserving behavior.

Execution steps:

1. **Initiative Check**
   - Check for active initiative
   - Add refactor to current initiative or create new one

1.5. **Add Source Section** (if Linear ticket metadata present)
   - Check if user input contains `LINEAR_TICKET:` metadata block
   - If not present: Skip to step 2
   - If present:
     a. Extract LINEAR_TICKET, LINEAR_URL, LINEAR_TITLE from metadata
     b. Read the initiative's INITIATIVE.md file
     c. Use Edit tool to insert a Source section before "## Description":
        ```markdown
        ## Source

        **Linear Ticket**: [LINEAR_TICKET](LINEAR_URL)
        **Title**: LINEAR_TITLE

        ```
     d. Proceed to step 2

2. **Goal Definition**
   - Document improvement goal in `goal.md`
   - Identify what "better" means (readability, performance, maintainability)
   - Define success criteria

3. **Constraint Identification**
   - Document behavior that MUST NOT change in `constraints.md`
   - List all public interfaces that must remain stable
   - Identify external dependencies

4. **Dependency Analysis** (research agents)
   - Spawn research-codebase agent: Map affected code
   - Identify all files, modules, tests impacted
   - Document in `dependency-analysis.md`

5. **Safety Net Assessment**
   - Evaluate existing test coverage in `safety-net.md`
   - Identify gaps in coverage
   - Recommend additional tests before refactoring

6. **Refactor Planning**
   - Create `refactor-plan.md` with atomic steps
   - Each step should be independently committable
   - Include rollback strategy

7. **Report Completion**
   - Scope of changes
   - Test coverage assessment
   - Risk areas identified
   - Suggested next command

## Artifact Structure

```
history/{date}-{initiative}/
  refactors/{number}-{name}/
    goal.md
    constraints.md
    dependency-analysis.md
    refactor-plan.md
    safety-net.md
    progress.md
```

## Behavior Rules

- Refactors NEVER change external behavior
- Require test coverage before major restructuring
- Each step must be atomic and reversible
- Flag any behavior changes for promotion to feature
