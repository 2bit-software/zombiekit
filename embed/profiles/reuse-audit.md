---
name: reuse-audit
description: Audits a planned implementation against the existing codebase to find reuse opportunities and prevent duplication.
type: skill
handoffs:
  - label: Update Plan
    skill: brains.update
    prompt: Update the implementation plan to incorporate reuse findings
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: For each planned item in `implementation-plan.md`, determine whether equivalent or overlapping code already exists in the codebase. Produce concrete reuse decisions to keep the plan DRY.

Execution steps:

1. **Parse the Plan**
   - Load `implementation-plan.md`
   - Extract every discrete planned artifact: types, interfaces, functions, utilities, packages, patterns, constants

2. **Codebase Search** (parallel agents, one per planned item cluster)
   - For each extracted item, search using `codebase-memory-mcp`:
     - `search_code` — semantic search for similar logic and naming
     - `search_graph` — find matching symbols, types, interfaces
     - `trace_call_path` — follow related call chains when a candidate is found
   - Each agent returns: candidate files, symbols, confidence level

3. **Classify Each Finding**
   - **DUPLICATE**: Existing code satisfies the planned item's contract. Use directly.
   - **OVERLAP**: Existing code is substantially similar but not identical. Evaluate extending vs. creating new.
   - **RELATED**: Same domain, different contract. Note for naming and consistency only.
   - **NONE**: No existing equivalent found. Proceed as planned.

4. **Reuse Decision** (for DUPLICATE and OVERLAP only)

   Extend existing if:
   - Extension doesn't break current consumers
   - Existing code is in an appropriate package/layer for this new use
   - Responsibility stays coherent after the change

   Create new if:
   - Extension requires changes at 3+ call sites
   - Would violate the existing type's single responsibility
   - Package/layer boundary makes it a wrong fit

5. **Produce Artifact**
   - Write `reuse-audit.md` with all findings
   - Do not modify `implementation-plan.md` — plan revision is a separate step

6. **Report Completion**
   - Count by classification
   - Which plan items changed and how
   - Any RELATED findings worth noting for naming consistency

## Output Format

```markdown
# Reuse Audit

## Summary
- Duplicates: {count} (redirected to existing)
- Overlaps: {count} ({extend_count} extend, {new_count} create new)
- Related: {count} (noted for consistency)
- No match: {count}

## Findings

### DUPLICATE

#### {Planned Item}
- **Existing**: `{file}:{symbol}`
- **Decision**: Use existing directly
- **Plan change**: {what was updated in implementation-plan.md}

### OVERLAP

#### {Planned Item}
- **Existing**: `{file}:{symbol}`
- **Similarity**: {description of overlap}
- **Decision**: Extend / Create new
- **Rationale**: {why}
- **Plan change**: {what was updated in implementation-plan.md}

### RELATED

#### {Planned Item}
- **Related code**: `{file}:{symbol}`
- **Note**: {naming or consistency observation}

## Plan Changes
{Summary of all modifications made to implementation-plan.md}
```

## Behavior Rules

- Search broadly — false positives are cheaper than missed reuse opportunities
- When uncertain between OVERLAP and RELATED, classify as OVERLAP and decide explicitly
- Never remove a planned item without confirming the existing code fully satisfies its contract
- RELATED findings are informational only — do not modify the plan for them
