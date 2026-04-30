---
name: refactor
description: Create a safe refactoring specification. Analyzes dependencies, assesses safety nets, and produces an atomic refactor plan.
type: skill
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
   - Check for active initiative in `.brains/active.json`
   - If none: Create new initiative with auto-generated name
   - If active and `--new` flag: Complete current, create new
   - If active: Add refactor to current initiative

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
   - Spawn a research-codebase agent: Map affected code
   - Use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to follow execution paths through affected modules
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
   - List of artifacts created
   - Suggested next command (`/brains.next`)

## Artifact Structure

```
history/{id}-refactor-{slug}/
  INITIATIVE.md
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
- Never assume safety — evidence required before proceeding
- Do not implement the refactor in this step — that is `/brains.next`
