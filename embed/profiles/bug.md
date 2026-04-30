---
name: bug
description: Create a bug investigation and fix specification. Determines if issue is a spec gap or implementation error.
type: skill
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Investigate a bug, determine root cause, and create a fix specification.

Execution steps:

1. **Initiative Check**
   - Check for active initiative
   - Add bug to current initiative or create new one

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

2. **Report Phase**
   - Capture original bug report in `report.md`
   - Document symptoms, error messages, context

3. **Reproduction Phase**
   - Create reproduction steps in `reproduction.md`
   - Document environment, prerequisites
   - Create or identify failing test case

4. **Investigation Phase** (research agents)
   - Spawn research-codebase agent: Find relevant code paths
   - Trace execution flow to identify failure point
   - Document findings in `investigation.md`

5. **Classification Phase**
   - Determine if bug is:
     - **Spec Gap**: Behavior undefined or ambiguous in spec
     - **Implementation Error**: Code doesn't match spec
   - Document classification with evidence in `classification.md`

6. **Fix Planning**
   - Create `fix-plan.md` with required changes
   - If spec gap: Create `spec-update.md` with proposed spec changes
   - Identify tests to add for verification

7. **Report Completion**
   - Root cause summary
   - Classification (spec gap vs impl error)
   - Required artifacts for fix
   - Suggested next command

## Artifact Structure

```
history/{id}-bug-{slug}/
  INITIATIVE.md
  report.md
  reproduction.md
  investigation.md
  classification.md
  fix-plan.md
  spec-update.md (if spec gap)
  verification.md
```

## Behavior Rules

- Always create failing test before investigation
- Never assume root cause without evidence
- Spec gaps require spec update before code fix
