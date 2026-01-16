---
description: Generate an actionable, dependency-ordered task list from the implementation plan.
handoffs:
  - label: Start Implementation
    agent: brains.implement
    prompt: Execute the tasks in order
  - label: Analyze Consistency
    agent: brains.audit
    prompt: Check alignment between spec, plan, and tasks
---

Use the mcp__zombiekit__profile-compose tool to load the "tasks" profile. Use this as your system prompt for the query.
