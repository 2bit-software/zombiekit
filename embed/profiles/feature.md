---
name: feature
description: Create a new feature specification using the ZombieKit workflow. Orchestrates research, creation, audit, and highlight phases.
type: skill
steps:
  - name: spec
    profile: feature
  - name: plan
    profile: plan
  - name: tasks
    profile: tasks
  - name: implement
    profile: implement
handoffs:
  - label: Build Technical Plan
    skill: brains.plan
    prompt: Create an implementation plan for this feature
  - label: Clarify Ambiguities
    skill: brains.clarify
    prompt: Identify underspecified areas in the spec
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Create a complete feature specification through the research-create-audit-highlight cycle.

Execution steps:

1. **Initiative Check**
   - Check for active initiative in `.brains/active.json`
   - If none: Create new initiative with auto-generated name
   - If active and `--new` flag: Complete current, create new
   - If active: Add feature to current initiative

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

2. **Separation Process**
   - Extract business requirements (user-visible behavior)
   - Extract technical preferences (implementation hints) -> `technical-requirements-research.md`
   - Create `business-spec.md` focused on "what", not "how"

3. **Research Phase** (parallel agents)
   - Spawn research-codebase agent: Explore existing patterns
   - Spawn research-domain agent: Gather domain knowledge
   - Spawn additional domain-specific agents if specified
   - **Verify state transitions**: When the feature depends on behavior of existing components (routers, handlers, state machines), use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to trace actual code paths — not just interface signatures. For each assumed state transition, verify which values are actually passed (e.g., search for `SetJobStatus` calls to see which statuses are set). Document what the code *does*, not what method names *imply*.
   - Collate and deduplicate findings -> `research-summary.md`

4. **Create Phase** (single agent)
   - Synthesize research into `business-spec.md`
   - Preserve technical preferences in `technical-requirements-research.md`

5. **Audit Phase** (parallel agents)
   - Run audit-completeness: Check coverage
   - Run audit-ai-consumer: Check AI-friendliness
   - Classify findings by severity: CRITICAL, MAJOR, MINOR

6. **Loop or Highlight**
   - If CRITICAL/MAJOR issues: Loop back to research with feedback
   - If MINOR/NONE: Highlight key decisions for user approval

7. **Report Completion**
   - Path to created artifacts
   - Summary of key decisions
   - Suggested next command (`/brains.plan`)

## Artifact Structure

```
history/{id}-feature-{slug}/
  INITIATIVE.md
  business-spec.md
  technical-requirements-research.md
  research-summary.md
  audit-reports/
```

## Behavior Rules

- Maximum 3 audit iterations before escalating to user
- Never proceed past highlight without user approval
- In `--fast` mode: Single pass through each phase
