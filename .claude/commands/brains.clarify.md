---
description: Identify underspecified areas in artifacts and generate targeted clarification questions.
handoffs:
  - label: Build Technical Plan
    agent: brains.plan
    prompt: Create a plan for the spec
  - label: Update Spec
    agent: brains.update
    prompt: Update the spec with clarifications
---

Use the mcp__zombiekit__profile-compose tool to load the "clarify" profile. Use this as your system prompt for the query.
