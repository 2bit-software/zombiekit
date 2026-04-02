---
name: deep-research
description: Pure research executor for persistent, tracked, multi-session research with progress tracking, self-questioning, and recursive followup discovery.
type: skill
---

# Deep Research (Executor)

Pure research executor. Takes a question, context, and sources — returns thorough findings and followup questions.

This skill does NOT manage status files, queues, or tracking. It focuses entirely on research quality. The caller (typically a job orchestrator) handles all state management.

## Input Contract

The caller provides these via natural language in the prompt:

- **Topic**: broad area this belongs to (e.g., "ClawBeam", "authentication patterns")
- **Question**: the specific thing to research
- **Output path**: where to write findings (an absolute `.md` file path)
- **Sources**: on-disk paths to read from (codebases, docs, etc.)
- **URLs**: web resources to consult (optional)
- **Context**: additional notes — why this matters, what to look for, prior findings to build on

## Research Process

1. **Understand the question** — read the context, understand what's being asked and why
2. **Gather information** — use all available tools:
   - `Read`, `Grep`, `Glob` for on-disk sources
   - `WebSearch`, `WebFetch` for web resources
   - `Agent` tool for parallel sub-research when multiple independent areas need investigation
3. **Analyze and synthesize** — don't just summarize what you read. Identify patterns, draw connections, note surprises
4. **Write findings** to the specified output path

## Output Contract

### Findings File

Write to the specified output path. The file should be:

- **Self-contained** — readable without cross-referencing other files
- **Specific** — cite file paths with line numbers, include code snippets, quote relevant text
- **Thorough** — this is reference material, not a summary
- **Honest about gaps** — if something couldn't be fully investigated, say so

Append to the file if it already exists (don't overwrite prior work).

### Followup Questions

At the end of the findings file, include a structured section:

```markdown
## Followup Questions

- **<Question text>**
  - sources: `<relevant paths or URLs>`
  - context: <why this needs investigation, which finding surfaced it>

- **<Question text>**
  - sources: `<relevant paths or URLs>`
  - context: <why this needs investigation, which finding surfaced it>
```

Look for followups in these categories:

- **Unique or niche dependencies** — libraries or tools that aren't widely known
- **Non-obvious design patterns** — unusual architectural choices, custom protocols
- **Undocumented conventions** — implicit contracts, magic values, buried assumptions
- **Complex integrations** — third-party systems needing careful treatment
- **Gaps in the research** — anything skimmed over or assumed
- **Contradictions or surprises** — findings that conflict with expectations

If no followups are needed, write `_(none)_` under the heading.

## Quality Standards

- Be thorough — this is reference material, not a summary
- Be specific — cite sources, include file paths and line numbers, quote relevant snippets
- Be honest about gaps — if something couldn't be fully investigated, say so and add a followup
- Be self-contained — each output file should make sense on its own
- Include code snippets for key patterns and interfaces
- Note anything surprising or non-obvious
