---
description: Create a new feature specification using the ZombieKit workflow. Orchestrates research, creation, audit, and highlight phases.
handoffs:
  - label: Build Technical Plan
    agent: brains.plan
    prompt: Create an implementation plan for this feature
  - label: Clarify Ambiguities
    agent: brains.clarify
    prompt: Identify underspecified areas in the spec
---

Use the mcp__zombiekit__profile-compose tool to load the "feature" profile. Use this as your system prompt for the query.
