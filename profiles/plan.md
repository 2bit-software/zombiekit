---
name: plan
description: Create an implementation plan from a specification. Includes proof spikes for validation.
type: skill
handoffs:
  - label: Generate Tasks
    skill: brains.tasks
    prompt: Break this plan into executable tasks
  - label: Revise Spec
    skill: brains.revise
    prompt: The plan revealed issues with the spec...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Create a detailed implementation plan informed by research and validated by proof spikes.

Execution steps:

1. **Load Context**
   - Load `business-spec.md` from current work item
   - Load `technical-requirements-research.md` for user preferences
   - Load `research-summary.md` for domain context

2. **Research Phase** (parallel agents)
   - Investigate implementation approaches
   - Validate technical assumptions from user preferences
   - Research libraries, APIs, patterns
   - Document findings with alternatives considered

3. **Spike Phase** (where needed)
   - For uncertain areas, create minimal proof-of-concept code
   - Validate interfaces, APIs, library behavior
   - Document findings in `spike-results.md`
   - Spikes are temporary - they validate, not implement

4. **Plan Creation** (single agent)
   - Create `implementation-plan.md` informed by spike results
   - Create `technical-spec.md` with implementation design
   - Order steps based on dependencies
   - Flag remaining uncertainties

5. **Audit Phase** (parallel agents)
   - Verify plan completeness
   - Check alignment with spec
   - Identify gaps between spec and plan

6. **Loop or Highlight**
   - If issues found: Loop back or suggest `/brains.revise`
   - If clean: Highlight key technical decisions

7. **Report Completion**
   - Plan summary
   - Spike results (if any)
   - Technical decisions made
   - Suggested next command (`/brains.tasks`)

## Artifact Structure

```
{work-item}/
  implementation-plan.md
  technical-spec.md
  spike-results.md (if spikes run)
```

## Spike Triggers

Create spikes when:
- External API/library usage unclear
- Performance-critical paths need validation
- Integration points with existing code uncertain
- Any area flagged "uncertain" in research

## Behavior Rules

- Always load user's technical preferences
- Spikes are for validation only, not production code
- Plan must trace back to spec requirements
