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
   - **Verify state transitions**: If the spec claims a component sets a status, releases a resource, or triggers a side effect, use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to verify the actual code path. Search for the specific method call in the handler/router code and confirm which values it passes. Flag any discrepancy between spec claims and actual behavior as a CRITICAL finding.
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

5. **Reuse Audit**
   - Call `mcp__zombiekit__profile-compose` with `{"profiles": ["reuse-audit"]}` and follow the returned instructions
   - This searches the codebase for existing implementations of each planned item
   - Produces `reuse-audit.md` only — does not touch the plan yet

6. **Plan Revision from Reuse Findings**
   - Read `reuse-audit.md`
   - For each DUPLICATE: replace the planned item with a direct reference to the existing code
   - For each OVERLAP marked "Extend": update the planned item to describe extending the existing code rather than creating new
   - For each OVERLAP marked "Create new": add a note in the plan explaining why new creation was chosen over reuse
   - RELATED and NONE items require no plan changes
   - The revised `implementation-plan.md` must remain self-consistent — re-check dependencies after substitutions

7. **Audit Phase** (parallel agents)
   - Verify plan completeness
   - Check alignment with spec
   - Identify gaps between spec and plan

8. **Loop or Highlight**
   - If issues found: Loop back or suggest `/brains.revise`
   - If clean: Highlight key technical decisions

9. **Report Completion**
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
  reuse-audit.md
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
