---
status: complete
updated: 2026-04-03
---

# Technical Requirements: Business-Requirement Function Comments

## Implementation Approach

Add the function comment requirement to the **implement profile** (`embed/profiles/implement.md`) as the single shared enforcement point. Mirror the requirement in spec and task templates for consistency with the existing test comment rule.

## Files to Modify

1. **`embed/profiles/implement.md`** — Add a "Function Comment Style" section with examples, exclusions, and the instruction to reuse spec verbiage.
2. **`.brains/templates/spec-template.md`** — Add a "Function Comment Style" block parallel to the existing "Test Comment Style" block.
3. **`.brains/templates/tasks-template.md`** — Add a matching "Function Comment Style" block.
4. **`STANDARDS.md`** — Add a subsection under Documentation requiring business-language doc comments.

## Technical Preferences

- Single source of truth in the implement profile; templates echo the rule for visibility
- Match the format and tone of the existing test comment requirement
- Use the same Good/Bad example pattern already established
- No linting automation in this iteration
