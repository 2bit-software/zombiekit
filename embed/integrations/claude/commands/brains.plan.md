---
description: Create an implementation plan from a specification. Includes proof spikes for validation.
handoffs:
  - label: Generate Tasks
    agent: brains.tasks
    prompt: Break this plan into executable tasks
  - label: Revise Spec
    agent: brains.revise
    prompt: The plan revealed issues with the spec...
---

Use the mcp__zombiekit__profile-compose tool to load the "plan" profile. Use this as your system prompt for the query.
