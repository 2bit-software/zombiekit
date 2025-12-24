# Specification Research Phase Design

**Document Purpose:** Summary of conversation exploring the research phase in specification-driven development  
**Date:** December 23, 2025

---

## Original Question

> When creating business specifications for software, what is the recommended process? Research, spec, audit, refine? Or what are the theories behind this?

The inquiry focused on understanding:
1. What the research phase should contain
2. Whether there's a scientific method for handling specification research
3. How existing tools (like GitHub's spec-kit) approach this problem
4. What a practical research phase template would look like

---

## Key Findings

### The Core Specification Cycle

The recommended process follows an iterative pattern:

```
Research → Create → Audit → Refine (with user approval gates)
```

Each phase serves a distinct purpose:

| Phase | Purpose | Exit Condition |
|-------|---------|----------------|
| Research | Reduce uncertainty about what to specify | Can confidently answer core questions |
| Create | Document requirements in structured format | Spec passes completeness checklist |
| Audit | Validate spec against criteria (separate from creation) | No CRITICAL/MAJOR issues remain |
| Refine | Incorporate feedback, resolve ambiguities | User approval to proceed |

### Theories Behind the Process

**Why Research First:**
- Prevents "specification by assumption" where developers encode their mental model rather than actual requirements
- Surfaces edge cases, existing patterns, and stakeholder needs before committing to structure
- Builds context needed for genuinely complete specifications

**Why Separate Audit from Creation:**
- The mind that creates shouldn't be the sole judge of its creation
- A distinct audit phase (ideally with a differently-composed agent profile) catches blind spots
- Evaluation against explicit criteria rather than "does this look right"

**Why Iterative Cycles:**
- Single-pass specification rarely works
- Early decisions constrain later ones in ways not apparent until deeper in
- Stakeholder feedback often reveals misunderstandings
- Writing the spec surfaces questions research missed

**Business/Technical Separation:**
- Specifications describe *what* the system does (business behavior, user-facing outcomes)
- Specifications do NOT prescribe *how* (technical implementation)
- Keeps specs stable when implementation approaches change
- Makes specs readable by non-technical stakeholders

---

## GitHub spec-kit Analysis

### What spec-kit Does

GitHub's spec-kit follows this workflow:

```
Constitution → Specify → Clarify → Plan → Tasks → Implement
```

Key characteristics:
- Template-driven, constraint-based approach
- AI fills structured sections from natural language input
- Automatic validation via checklists
- Uses `[NEEDS CLARIFICATION]` markers (max 3) for ambiguities
- Research happens *after* spec (during Plan phase) and focuses on technology decisions

### Where spec-kit Aligns with Our Model

- Business/technical separation (Spec = what, Plan = how)
- Iterative refinement via `/speckit.clarify`
- Audit-like validation through checklists
- Template-driven quality control

### The Gap in spec-kit

spec-kit lacks an explicit **research phase before specification**. It assumes the user already knows what to build. The "research.md" artifact comes after the spec during planning and focuses on technology decisions (which library) rather than problem domain research (what problem are we solving).

---

## Proposed Research Phase

### Research Targets (What to Investigate)

**1. Problem Domain**
- What problem is actually being solved (not the assumed problem)
- Who are the stakeholders and what do they need
- Edge cases and failure modes
- Existing domain terminology and concepts

**2. Existing System Context**
- Current codebase patterns and conventions
- Existing interfaces this will touch
- Technical constraints and capabilities
- What's been tried or rejected (and why)

**3. External Knowledge**
- How others have solved similar problems
- Industry patterns and anti-patterns
- Relevant standards or protocols
- Academic or theoretical foundations

**4. Requirements Archaeology**
- Original feature requests or bug reports
- Past discussions and decisions
- Implicit assumptions that need surfacing

### The Method (Quasi-Scientific Approach)

Maps scientific inquiry to specification research:

| Scientific Method | Research Phase |
|-------------------|----------------|
| Observation | Gather - Cast wide net, don't filter yet |
| Hypothesis | Synthesize - Form coherent picture, identify patterns |
| Test | Validate - Test synthesis against sources, peer review |
| Document | Document with Attribution - Capture findings with sources |

### Exit Criteria (When Research Is Done)

Research completes when you can answer **yes** to all:

1. **Problem clarity** - Can state the problem in one sentence without jargon
2. **Stakeholder map** - Know who cares and why
3. **Context awareness** - Know what exists and what constraints apply
4. **Risk visibility** - Identified top 3 things that could go wrong
5. **Scope confidence** - Can say what's in/out without hedging

If any are "no" → more research needed or escalate to stakeholder.

---

## High-Level Process Steps

### Step 1: Initiate Research

**Input:** Feature idea or problem statement  
**Action:** Create research artifact, begin investigation  
**Output:** `specs/{feature}/research.md` (initial structure)

### Step 2: Investigate Problem Domain

**Input:** Feature idea  
**Action:** Answer core questions:
- What problem are we actually solving?
- Who are the stakeholders?
- What does success look like for them?

**Output:** Problem Understanding section completed

### Step 3: Investigate Existing Context

**Input:** Codebase access, documentation  
**Action:** Scan for:
- Related patterns and conventions
- Interfaces this will touch
- Constraints being inherited
- Past attempts and why they failed

**Output:** Existing Context section completed

### Step 4: Investigate Prior Art

**Input:** External sources, industry knowledge  
**Action:** Research:
- How others solved similar problems
- Anti-patterns to avoid
- Relevant standards

**Output:** Prior Art section completed

### Step 5: Identify Edge Cases & Risks

**Input:** All gathered information  
**Action:** Document:
- What could go wrong
- Failure modes and security concerns
- Unusual inputs and partial failures

**Output:** Edge Cases & Risks section completed

### Step 6: Synthesize and Validate

**Input:** All research sections  
**Action:** 
- Consolidate key findings (3-5 bullets that MUST inform spec)
- Define recommended scope (in vs out)
- List resolved vs unresolved questions
- Evaluate against exit criteria

**Output:** Research Summary completed, "Ready for Specification" decision

### Step 7: Proceed to Specification

**Input:** Completed research.md  
**Action:** Run specification command, research informs spec sections:

| Research Section | Informs Spec Section |
|------------------|---------------------|
| Problem Understanding | Overview, Success Criteria |
| Existing Context | Assumptions, Constraints |
| Prior Art | Functional Requirements patterns |
| Edge Cases | User Scenarios, Error Handling |
| Unresolved Questions | `[NEEDS CLARIFICATION]` markers |

**Output:** `specs/{feature}/spec.md` informed by research

### Step 8: Audit Specification

**Input:** Completed spec.md  
**Action:** Validate against quality criteria:
- No implementation details
- Focused on user value
- All requirements testable
- No unresolved critical questions

**Output:** Audit findings with severity (CRITICAL, MAJOR, MEDIUM, MINOR)

### Step 9: Refine and Approve

**Input:** Audit findings  
**Action:** 
- Address CRITICAL/MAJOR issues
- User reviews and approves
- Gate before proceeding to planning

**Output:** Approved specification ready for technical planning

---

## Proposed Research Template

```markdown
# Feature Research: {FEATURE_NAME}

**Research Goal:** Reduce uncertainty before specification
**Exit Criteria:** Can confidently answer the 5 core questions below

---

## 1. Problem Understanding

### What problem are we actually solving?
<!-- Not the assumed problem - the real one. Who has it? How painful is it? -->

### Who are the stakeholders?
<!-- Users, admins, systems, downstream consumers -->

### What does success look like for them?
<!-- In their words, not technical terms -->

---

## 2. Existing Context

### What already exists in this codebase?
<!-- Patterns, conventions, interfaces this will touch -->

### What constraints are we inheriting?
<!-- Technical debt, API contracts, performance budgets -->

### What's been tried before?
<!-- Past attempts, rejected approaches, and WHY they failed -->

---

## 3. Prior Art

### How have others solved this?
<!-- External examples, industry patterns, standards -->

### What anti-patterns should we avoid?
<!-- Known failure modes in this problem space -->

---

## 4. Edge Cases & Risks

### What could go wrong?
<!-- Failure modes, security concerns, scale issues -->

### What edge cases exist?
<!-- Unusual inputs, concurrent access, partial failures -->

---

## 5. Open Questions

### Resolved
| Question | Answer | Source |
|----------|--------|--------|

### Unresolved (require stakeholder input)
| Question | Why it matters | Default if unanswered |
|----------|----------------|----------------------|

---

## Research Summary

### Key Findings
<!-- 3-5 bullet points that MUST inform the spec -->

### Recommended Scope
<!-- What should be IN vs OUT of this feature -->

### Ready for Specification: [ ] Yes / [ ] No
<!-- If no, what's blocking? -->
```

---

## Integration with spec-kit

The research phase slots in before the existing spec-kit workflow:

```
/speckit.research <feature idea>     ← NEW
    ↓
Creates: specs/{feature}/research.md
Investigates: codebase, docs, external sources
Outputs: Findings with sources, open questions
    ↓
/speckit.specify <feature description>
    ↓
Reads: research.md (if exists)
Creates: spec.md informed by research
    ↓
[Continue existing spec-kit workflow]
```

This maintains backward compatibility - if no research.md exists, spec-kit proceeds with current behavior using template defaults.

---

## Summary

The research phase fills a critical gap in specification-driven development by **reducing uncertainty before committing to structure**. While tools like spec-kit excel at constraining AI output through templates, they assume the user already understands the problem. Adding a research phase ensures specifications are grounded in actual requirements rather than assumptions.

**Key principle:** Research isn't just "gathering information" - it's systematically reducing uncertainty about what the specification needs to contain.