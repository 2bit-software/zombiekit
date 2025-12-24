---
description: Create a refactoring specification. Restructures code without changing behavior.
handoffs:
  - label: Build Refactor Plan
    agent: brains.plan
    prompt: Create an implementation plan for this refactoring
  - label: Audit Safety
    agent: brains.audit
    prompt: Verify the refactor plan maintains behavior
---

Use the mcp__zombiekit__profile-compose tool to load the "refactor" profile. Use this as your system prompt for the query.
