---
name: feature
description: Execute the research-create-audit-highlight workflow for a new feature specification
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
# Feature Specification Workflow

## Context

You are executing the feature specification workflow. Your goal is to create a complete, AI-readable feature specification through a structured research→create→audit→highlight cycle.

### Available Files
- `research.md` - Template for research findings (populate during Research phase)
- `spec.md` - Template for feature specification (populate during Create phase)
- `audit/` - Directory for audit reports (populate during Audit phase)

### Your Responsibilities
- Spawn research agents to gather context
- Synthesize findings into a specification
- Run audit checks and address critical/major issues
- Present highlights for user approval

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
3. **Parse `workflow_phases`**: Understand the 4-phase structure (research→create→audit→highlight)
4. **Follow `directive`**: Execute according to this document
5. **Output to `cycle_folder`**: Save artifacts (research.md, spec.md, audit/) here
6. **Reference `composed_prompt`**: Additional context from research, create, audit profiles

### Understanding `workflow_phases`

The response includes phase definitions:

```json
{
  "workflow_phases": [
    {"name": "research", "parallel": true, "outputs": ["research.md"]},
    {"name": "create", "parallel": false, "outputs": ["spec.md"]},
    {"name": "audit", "parallel": true, "outputs": ["audit/{date}.md"]},
    {"name": "highlight", "parallel": false, "outputs": []}
  ]
}
```

Execute phases in order. Parallel phases can spawn multiple agents.

---

## Phase I: Research (Parallel Agents)

### Input
- User's feature description
- Codebase context
- Previous cycle artifacts (if available)

### Actions
1. Spawn research agents in parallel:
   - **research-codebase**: Explore existing patterns, dependencies, constraints
   - **research-domain**: Gather domain knowledge, best practices, standards
   - Additional agents as needed based on description

2. Collate findings:
   - Remove duplicates
   - Organize by category
   - Preserve sources for all claims

### Output
Populate `research.md` with:
- Executive summary (2-3 sentences)
- Findings organized by category
- Decision points identified
- Recommendations with rationale
- Sources cited

### Success Criteria
- [ ] research.md has non-empty Executive Summary
- [ ] At least 2 finding categories populated
- [ ] All claims have sources cited
- [ ] Decision points clearly marked

---

## Phase II: Create (Single Agent)

### Input
- Populated `research.md`
- `spec.md` template
- Previous cycle specs (if refactoring)

### Actions
1. Synthesize specification from research findings
2. Focus on **what**, not **how** (no implementation details)
3. Fill all mandatory sections:
   - User Scenarios & Testing
   - Requirements (Functional)
   - Success Criteria
4. Mark any unclear areas as NEEDS CLARIFICATION

### Output
Populate `spec.md` with:
- Complete feature specification
- Testable user stories
- Measurable success criteria
- No TODOs or placeholder text

### Success Criteria
- [ ] All mandatory sections filled
- [ ] No "TODO", "TBD", or placeholder text
- [ ] Each user story has acceptance scenarios
- [ ] Success criteria are measurable
- [ ] No implementation details present

---

## Phase III: Audit (Parallel Agents)

### Input
- Populated `spec.md`
- `research.md` for context
- Previous audit reports (if retry)

### Actions
1. Spawn audit agents in parallel:
   - **audit-completeness**: Check all sections filled, no gaps
   - **audit-ai-readiness**: Check specification is unambiguous, testable

2. Classify findings by severity:
   - **CRITICAL**: Blocking, must fix before proceeding
   - **MAJOR**: Significant gap, should fix
   - **MINOR**: Nice to fix, low impact
   - **INFO**: Observation, no action needed

### Conditional Transition
```
IF CRITICAL or MAJOR issues found:
    - Document findings in audit/{YYYY-MM-DD}.md
    - Prepare feedback for research phase
    - Track iteration count (current: N of 3)
    - IF iteration < 3: Return to Phase I with feedback
    - IF iteration >= 3: Escalate to user for intervention
ELSE:
    - Document findings (if any)
    - Proceed to Phase IV
```

### Output
Create `audit/{YYYY-MM-DD}.md` with:
- Issue counts by severity
- Detailed findings with fix suggestions
- Alignment matrix (requirements → coverage)
- Recommendation for next step

### Success Criteria
- [ ] All sections audited
- [ ] Findings properly classified
- [ ] Critical/major issues have fix suggestions
- [ ] Audit report follows template format

---

## Phase IV: Highlight (Single Agent)

### Input
- Finalized `spec.md`
- `research.md` findings
- Audit results

### Actions
1. Summarize key specification decisions
2. Highlight any trade-offs or assumptions made
3. Note minor issues or open questions
4. Prepare presentation for user review

### User Approval Gate

Present to user:
```
## Feature Specification Summary

**Feature**: {name}
**Cycles**: {current cycle number}

### Key Decisions
1. {Decision 1}
2. {Decision 2}
3. {Decision 3}

### Assumptions
- {Assumption 1}
- {Assumption 2}

### Minor Issues (if any)
- {Minor issue 1}

---

**Ready for approval?**
- Approve to proceed to planning phase
- Reject with feedback to revise specification
```

### Conditional Transition
```
IF user approves:
    - Mark spec.md status as "approved"
    - Report completion with artifact paths
    - Suggest next step: Call `step` MCP tool with `step: "plan"`
ELSE IF user rejects with feedback:
    - Incorporate feedback into research
    - Return to Phase I (counts toward 3-loop limit)
```

---

## Behavior Rules

1. **Maximum Iterations**: 3 research→audit cycles before requiring user intervention
2. **Never Skip Phases**: Each phase must execute (may be fast if no issues)
3. **Always Cite Sources**: Research findings must reference sources
4. **Mark Timestamps**: All artifacts get updated timestamps
5. **Update State**: Initiative state updated after each phase
6. **No Implementation Details**: Specification focuses on "what", not "how"
7. **Parallel When Possible**: Research and Audit spawn parallel agents
8. **Single Agent for Synthesis**: Create and Highlight use single agent

---

## Phase Flow Diagram

```
                    ┌───────────────┐
                    │   START       │
                    └───────┬───────┘
                            ▼
              ┌─────────────────────────┐
              │  Phase I: RESEARCH      │◄──────────────────────┐
              │  (Parallel Agents)      │                       │
              └───────────┬─────────────┘                       │
                          ▼                                     │
              ┌─────────────────────────┐                       │
              │  Phase II: CREATE       │                       │
              │  (Single Agent)         │                       │
              └───────────┬─────────────┘                       │
                          ▼                                     │
              ┌─────────────────────────┐                       │
              │  Phase III: AUDIT       │                       │
              │  (Parallel Agents)      │                       │
              └───────────┬─────────────┘                       │
                          │                                     │
            ┌─────────────┴─────────────┐                       │
            ▼                           ▼                       │
    ┌───────────────┐           ┌───────────────┐               │
    │ CRITICAL or   │           │ MINOR or      │               │
    │ MAJOR issues  │           │ No issues     │               │
    └───────┬───────┘           └───────┬───────┘               │
            │                           │                       │
            ▼                           ▼                       │
    ┌───────────────┐           ┌─────────────────────────┐     │
    │ iteration < 3?│           │  Phase IV: HIGHLIGHT    │     │
    └───────┬───────┘           │  (Single Agent)         │     │
            │                   └───────────┬─────────────┘     │
    ┌───────┴───────┐                       │                   │
    ▼               ▼                       ▼                   │
 ┌──────┐     ┌──────────┐          ┌───────────────┐           │
 │ Yes  │     │ No       │          │ User Approval │           │
 └───┬──┘     └────┬─────┘          └───────┬───────┘           │
     │             │                        │                   │
     │             ▼                ┌───────┴───────┐           │
     │     ┌─────────────┐          ▼               ▼           │
     │     │ ESCALATE    │    ┌──────────┐   ┌──────────┐       │
     │     │ to User     │    │ Approved │   │ Rejected │───────┘
     │     └─────────────┘    └────┬─────┘   └──────────┘
     │                              │
     └─────────────────────────────►▼
                            ┌───────────────┐
                            │     END       │
                            │ (spec.md =    │
                            │  approved)    │
                            └───────────────┘
```
