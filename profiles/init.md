---
name: init
description: Initialize ZombieKit in the current project or create a new initiative.
type: skill
handoffs:
  - label: Start Feature
    skill: brains.feature
    prompt: Create a new feature specification for...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Initialize ZombieKit configuration and optionally start a new initiative.

Execution steps:

1. Check if `.brains/` directory exists in the current project
   - If exists: Report current configuration and active initiative (if any)
   - If not exists: Create `.brains/` directory structure

2. Create default configuration if needed:
   - `.brains/config.toml` with sensible defaults
   - `.brains/profiles/` directory for local profile overrides
   - `.brains/templates/` directory for custom templates

3. Check for active initiative in `.brains/active.json`
   - If active: Display status and suggest next steps
   - If none: Offer to create new initiative or wait for `/brains.feature`

4. Report completion:
   - Configuration location
   - Available commands
   - Suggested next steps

## Behavior Rules

- Never overwrite existing configuration without explicit user confirmation
- Respect gitignore patterns (`.brains/active.json` should be gitignored)
- Create `history/` directory for artifact storage if it doesn't exist
