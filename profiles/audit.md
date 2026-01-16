---
name: audit
description: Cross-artifact alignment check. Verifies consistency between specs, plans, and tasks.
type: skill
handoffs:
  - label: Fix Issues
    skill: brains.update
    prompt: Fix the audit issues found...
  - label: Full Revision
    skill: brains.revise
    prompt: Significant misalignment requires revision...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Verify alignment and consistency across all artifacts in a work item.

Execution steps:

1. **Load Artifacts**
   - Load all artifacts for current work item
   - Identify artifact chain (spec -> plan -> tasks)
   - Note which artifacts exist

2. **Alignment Checks** (parallel auditors)

   a. **Spec-Plan Alignment**
      - Every spec requirement has plan coverage
      - Plan doesn't introduce unspecified features
      - Technical decisions align with constraints

   b. **Plan-Tasks Alignment**
      - Every plan step has corresponding task(s)
      - Task dependencies match plan order
      - No orphan tasks

   c. **Completeness Audit**
      - All mandatory sections present
      - No TODO/TBD markers remaining
      - Success criteria are measurable

   d. **AI-Friendliness Audit**
      - Clear structure for LLM consumption
      - Unambiguous instructions
      - Appropriate context included

3. **Issue Classification**
   - CRITICAL: Blocking, must fix before proceeding
   - MAJOR: Significant gap, should fix
   - MINOR: Nice to fix, low impact
   - INFO: Observation, no action needed

4. **Conflict Detection**
   - If auditors disagree, surface conflict
   - Present both perspectives
   - Request user decision

5. **Report Generation**
   - Create audit report with findings
   - Group by severity
   - Include specific references
   - Provide fix suggestions

6. **Report Completion**
   - Total issues by severity
   - Alignment score
   - Suggested fixes
   - Recommended next command

## Report Format

```markdown
# Audit Report: {Work Item}

## Summary
- Critical: {count}
- Major: {count}
- Minor: {count}

## Findings

### CRITICAL

#### [C1] {Title}
- **Location**: {artifact}:{section}
- **Issue**: {description}
- **Fix**: {suggestion}

### MAJOR
...

## Alignment Matrix
| Spec Requirement | Plan Step | Tasks |
|-----------------|-----------|-------|
| R1: User login  | P2.1      | T003  |

## Recommendations
{Next steps based on findings}
```

## Behavior Rules

- Never pass CRITICAL issues silently
- Conflicting auditor findings go to user
- Provide actionable fix suggestions
- Include specific artifact references
