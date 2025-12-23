---
name: update
description: Modify existing artifacts (specs, plans, tasks) without full re-research cycle.
type: skill
handoffs:
  - label: Full Revision
    skill: brains.revise
    prompt: This change is too significant, need full revision...
  - label: Re-audit
    skill: brains.audit
    prompt: Verify the update maintains consistency
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Apply targeted modifications to existing artifacts without triggering full workflow cycles.

Execution steps:

1. **Parse Update Request**
   - Identify target artifact (spec, plan, tasks)
   - Identify change type (clarification, correction, addition)
   - Identify scope of change

2. **Scope Assessment**
   - Evaluate if change is truly "minor"
   - Minor: Typo, wording clarification, small addition
   - Major: Scope change, new requirement, architectural shift
   - If major: Recommend `/brains.revise` instead

3. **Load Artifact**
   - Load current artifact content
   - Load related artifacts for context
   - Identify sections affected

4. **Apply Changes**
   - Make targeted modifications
   - Preserve document structure
   - Update modification timestamp

5. **Consistency Check**
   - Verify change doesn't contradict other sections
   - Verify downstream artifacts still align
   - Flag if cascade updates needed

6. **Version Tracking**
   - Note change in revision history
   - Don't create full version archive (that's for `/brains.revise`)

7. **Report Completion**
   - Changes made
   - Sections affected
   - Consistency status
   - Suggested next steps

## Change Categories

| Type | Description | Action |
|------|-------------|--------|
| Typo | Spelling, grammar | Direct fix |
| Clarification | Wording unclear | Rewrite section |
| Small Addition | Missing detail | Add content |
| Correction | Factual error | Fix with note |
| Scope Change | New requirement | Redirect to /brains.revise |

## Behavior Rules

- Never silently change scope
- Always check downstream impacts
- Preserve document structure
- Redirect major changes to /brains.revise
