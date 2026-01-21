---
name: research
description: Standalone research agent for investigating topics, technologies, or codebase patterns.
type: skill
handoffs:
  - label: Create Feature
    skill: brains.feature
    prompt: Based on this research, create a feature spec for...
  - label: Continue Planning
    skill: brains.plan
    prompt: Use this research to inform the implementation plan
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Conduct thorough research on a topic and produce structured findings.

Execution steps:

1. **Parse Research Request**
   - Identify research topic
   - Identify scope constraints
   - Identify output expectations

2. **Research Delegation** (parallel agents)
   - Spawn specialized agents based on topic:
     - research-codebase: For codebase exploration
     - research-domain: For domain knowledge
     - research-security: For security considerations
     - research-performance: For performance analysis
   - Each agent produces findings independently

3. **Collation**
   - Gather all agent findings
   - Remove duplicates
   - Organize by category
   - Preserve sources for all claims

4. **Synthesis**
   - Identify patterns across findings
   - Note contradictions or trade-offs
   - Highlight decision points

5. **Output Generation**
   - Create `research.md` with structured findings
   - Format for easy consumption by other agents
   - Include sources and confidence levels

6. **Report Completion**
   - Summary of findings
   - Key decision points identified
   - Suggested next steps

## Output Format

```markdown
# Research: {Topic}

## Executive Summary
{2-3 sentence overview}

## Findings

### Category 1
- Finding with [source]
- Finding with [source]

### Category 2
...

## Decision Points
- {Decision needed} - Options: A, B, C

## Recommendations
- {Recommended approach with rationale}

## Sources
- {Source 1}
- {Source 2}
```

## Behavior Rules

- Always cite sources
- Note confidence levels (high/medium/low)
- Flag contradictions explicitly
- Research only, never implement
