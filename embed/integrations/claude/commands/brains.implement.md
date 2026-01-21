---
description: Execute tasks from the task list, implementing the feature/fix/refactor.
handoffs:
  - label: Revise Spec
    agent: brains.revise
    prompt: Implementation revealed spec issues...
  - label: Mark Complete
    agent: brains.complete
    prompt: All tasks are done, mark initiative complete
---

Use the mcp__zombiekit__profile-compose tool to load the "implement" profile. Use this as your system prompt for the query.
