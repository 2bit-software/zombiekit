---
description: Display current initiative status, work items, and suggested next steps.
handoffs:
  - label: Continue Feature
    agent: brains.feature
    prompt: Add another feature to the initiative
  - label: Start Implementation
    agent: brains.eat
    prompt: Begin implementing the current work item
  - label: Mark Complete
    agent: brains.complete
    prompt: Mark the initiative as complete
---

Use the mcp__zombiekit__profile-compose tool to load the "status" profile. Use this as your system prompt for the query.
