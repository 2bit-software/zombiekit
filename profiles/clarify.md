---
name: clarify
description: Identify underspecified areas in artifacts and generate targeted clarification questions.
type: skill
handoffs:
  - label: Build Technical Plan
    skill: brains.plan
    prompt: Create a plan for the spec
  - label: Update Spec
    skill: brains.update
    prompt: Update the spec with clarifications
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Surface ambiguities and missing information in specifications through targeted questions.

Execution steps:

1. **Load Artifact**
   - Load target spec/plan/tasks
   - Identify artifact type for appropriate analysis

2. **Ambiguity Scan**
   - Scan for vague terms ("robust", "intuitive", "fast")
   - Identify missing decision points
   - Find undefined edge cases
   - Locate inconsistencies

   Categories:
   - Functional scope & behavior
   - Domain & data model
   - Interaction & UX flow
   - Non-functional requirements
   - Integration & dependencies
   - Edge cases & failure handling
   - Constraints & trade-offs

3. **Question Generation**
   - Generate max 5 targeted questions
   - Each question must be answerable with:
     - Multiple choice (2-5 options), OR
     - Short answer (<=5 words)
   - Prioritize by impact on implementation

4. **Interactive Questioning**
   - Present ONE question at a time
   - Provide recommended answer with reasoning
   - Accept user response or recommendation
   - Record answer before next question

5. **Integration**
   - Update spec with each clarification
   - Add to `## Clarifications` section
   - Apply to appropriate spec sections
   - Save after each integration

6. **Report Completion**
   - Questions asked and answered
   - Sections updated
   - Remaining ambiguities (if any)
   - Suggested next command

## Question Format

```markdown
## Question 1: {Topic}

**Context**: {Quote from spec}

**What we need to know**: {Specific question}

**Recommended**: Option A - {reasoning}

| Option | Description |
|--------|-------------|
| A | {Option A} |
| B | {Option B} |
| C | {Option C} |

Reply with option letter, "recommended", or your own answer.
```

## Behavior Rules

- Maximum 5 questions per session
- Only ask high-impact questions
- Always provide recommendation
- Save after each answer
- Stop early if user signals done
